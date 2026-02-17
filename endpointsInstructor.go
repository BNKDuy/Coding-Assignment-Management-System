package main

import (
	"archive/zip"
	"compress/flate"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"server/Types"
	"server/ZipUtil"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func instructorSignUp(w http.ResponseWriter, r *http.Request) {
	var req Types.RequestSignUp

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	userData.RLock()
	_, ok := userData.m[req.Username]
	userData.RUnlock()

	if ok {
		http.Error(w, "User already Exist", http.StatusBadRequest)
		return
	}

	newUser := Types.NewUser(req.Username, req.Password, Types.RoleInstructor)

	userData.Lock()
	userData.m[req.Username] = *newUser
	userData.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func createAssignment(w http.ResponseWriter, r *http.Request) {
	// Check if this user is a logged in user
	user, err := GetAuthorizedUser(w, r)
	if err != nil {
		return
	}

	// If the user is not an instructor, dont alow them to cretae an assignment
	if user.GetRole() != Types.RoleInstructor {
		http.Error(w, "You cannot create an assignment as a student", http.StatusForbidden)
		return
	}

	// parse multipart form
	// Limit the size to 1MB (avoid DOS attack hopefully)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "Bad multipart form", http.StatusBadRequest)
		return
	}

	// Get the metadata of the new assignment
	var req Types.RequestCreateAssignment
	meta := r.FormValue("metadata")
	if meta == "" {
		http.Error(w, "Missing metadata", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal([]byte(meta), &req); err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	// Some sanity checks
	// Make sure the student list is not empty
	if len(req.Students) == 0 {
		http.Error(w, "Bad request, students list cannot be empty", http.StatusBadRequest)
		return
	}

	// Make sure that all the languages in the request in this request is supported
	if len(req.Students) != 0 {
		for _, language := range req.Languages {
			_, ok := Types.SupportedLanguage[language]
			if !ok {
				supportedLanguageList := "Supported langagues:\n"
				for k := range Types.SupportedLanguage {
					supportedLanguageList += fmt.Sprintf("- %s\n", k)
				}
				http.Error(w, fmt.Sprintf("Unsupported language %s\n%s", language, supportedLanguageList), http.StatusBadRequest)
				return
			}
		}
	}

	studentList := make([]Types.User, 0, len(req.Students))
	// Check to make sure that all the student in the request are exist
	for _, studentUsername := range req.Students {
		userData.RLock()
		student, ok := userData.m[studentUsername]
		userData.RUnlock()
		if !ok {
			http.Error(w, fmt.Sprintf("Student %s does not exists", studentUsername), http.StatusBadRequest)
			return
		}
		studentList = append(studentList, student)
	}

	// Retrieve te testcases from the request
	file, _, err := r.FormFile("Testcases")
	if err != nil {
		http.Error(w, "Invalid testcases", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a new assignemnt
	f, err := os.CreateTemp("", "testcases.*.zip")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Copy the zip from the request to the server
	_, err = io.Copy(f, file)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create a new Assignment object
	assignment := Types.NewAssignment(*user, req.Name, "", req.Students, req.Languages)
	problemPath := fmt.Sprintf("ProblemStorage/%s", assignment.GetId())
	err = os.MkdirAll(problemPath, os.ModePerm)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Unzip the test case and place it in the ProblemStorage
	// Think of it as an Object storage similar to AWS S3
	err = ZipUtil.UnzipTests(f.Name(), problemPath)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update the path to the testcase
	assignment.SetPath(problemPath)

	// Sanity check
	// Check number of unput match the number of expected output
	inputDirPath := filepath.Join(problemPath, "_input")
	inputDir, err := os.ReadDir(inputDirPath)
	if err != nil {
		log.Println("Failed to read _expected dir when creating assignment for sanity check")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	expectedDirPath := filepath.Join(problemPath, "_expected")
	expectedDir, err := os.ReadDir(expectedDirPath)
	if err != nil {
		log.Println("Failed to read _expected dir when creating assignment for sanity check")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// This mean the number of file in _input and number of file in _expected does not match
	if len(inputDir) != len(expectedDir) {
		http.Error(w, "The number of input test cases does not match the number of expected output testcases", http.StatusBadRequest)
		return
	}

	// Store the assignment info in the KV db
	assignmentData.Lock()
	assignmentData.m[assignment.GetId()] = *assignment
	assignmentData.Unlock()

	// Assign the new assignment to the instructor account
	user.AddAssignment(assignment.GetId(), assignment.GetName())
	username := user.GetUsername()
	userData.Lock()
	userData.m[username] = *user
	userData.Unlock()

	// Assign the new assignment to the student account
	for _, student := range studentList {
		student.AddAssignment(assignment.GetId(), assignment.GetName())
		userData.Lock()
		userData.m[student.GetUsername()] = student
		userData.Unlock()
	}

	res := make(map[string]string)
	res["id"] = assignment.GetId()
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Println("Failed to encode json when creating assignment")
	}
}

func getGradebook(w http.ResponseWriter, r *http.Request) {
	user, err := GetAuthorizedUser(w, r)
	if err != nil {
		return
	}

	if user.GetRole() != Types.RoleInstructor {
		http.Error(w, "Permission required", http.StatusForbidden)
		return
	}

	assignmentId := chi.URLParam(r, "id")
	assignmentData.RLock()
	assignemnt, ok := assignmentData.m[assignmentId]
	assignmentData.RUnlock()

	if !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if !user.IsEqual(assignemnt.GetOwner()) {
		http.Error(w, "Permission required", http.StatusForbidden)
		return
	}

	inputDir, err := os.ReadDir(filepath.Join(assignemnt.GetPath(), "_input"))
	if err != nil {
		log.Println("Failed to read input dir when instrucotr retrieving gradebook")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	f, err := os.CreateTemp("", "Gradebook_*.csv")
	if err != nil {
		log.Println("Failed to create gradebook.csv when instrucotr retrieving gradebook")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	defer os.Remove(f.Name())

	csvWriter := csv.NewWriter(f)
	header := []string{"Student username", fmt.Sprintf("Tests passed (out of %d)", len(inputDir)), "Grade (percent)"}
	csvWriter.Write(header)

	submissions := assignemnt.GetAllSubmissions()

	for k, v := range submissions {
		if v == "No submission" {
			csvWriter.Write([]string{k, "-", "-"})
			continue
		}
		gradeStrLen := len(v)
		passCases := 0
		for i := 0; i < gradeStrLen; i++ {
			if v[i] == '/' {
				passCases, err = strconv.Atoi(v[0:i])
				if err != nil {
					passCases = 0
				}
				break
			}
		}

		gradePercent := strconv.Itoa((passCases * 100 / len(inputDir)))
		csvWriter.Write([]string{k, strconv.Itoa(passCases), gradePercent})
	}

	// Write any buffered data to the underlying writer (standard output).
	csvWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		log.Println("Failed to close csv writer when getting gradebook")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="gradebook-%s.zip"`, assignemnt.GetName()))

	// Create a new zip archive.
	zipWriter := zip.NewWriter(w)

	// Register a custom Deflate compressor.
	zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})

	defer func() {
		if err := zipWriter.Close(); err != nil {
			log.Println("Failed to close zip writer when student request submission:", err)
		}
	}()

	csvContent, err := os.ReadFile(f.Name())
	if err != nil {
		log.Println("Failed to read temp csv file when instructor get gradebook")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	zf, err := zipWriter.Create(fmt.Sprintf("gradebook-%s.csv", assignemnt.GetName()))
	if err != nil {
		log.Println("Failed to generate the gradebook inside the zip folder")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = zf.Write(csvContent)
	if err != nil {
		log.Println("Failed to write the csv content to zip file")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
