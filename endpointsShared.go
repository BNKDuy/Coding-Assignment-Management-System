package main

import (
	"archive/zip"
	"compress/flate"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"server/Types"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func listAssignments(w http.ResponseWriter, r *http.Request) {
	// Get the sign in user
	user, err := GetAuthorizedUser(w, r)
	if err != nil {
		return
	}

	// If this is an authorized user, return all the assignments that is associated with this user
	index := 1
	res := make(map[string]map[string]any)
	assignments := user.GetAssignments()
	for k := range assignments {
		assignmentData.RLock()
		assignment := assignmentData.m[k]
		assignmentData.RUnlock()

		languages := assignment.GetAllowedLanguages()
		allows := make([]string, 0, len(languages))
		for k := range languages {
			allows = append(allows, k)
		}
		res[strconv.Itoa(index)] = make(map[string]any)
		res[strconv.Itoa(index)]["ID"] = assignment.GetId()
		res[strconv.Itoa(index)]["Name"] = assignment.GetName()
		res[strconv.Itoa(index)]["Allowed languages"] = allows

		if user.GetRole() == Types.RoleInstructor {
			index += 1
			continue
		}
		res[strconv.Itoa(index)]["Grade"] = assignment.GetAllSubmissions()[user.GetUsername()]

		index += 1
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(res)
}

func getAssignment(w http.ResponseWriter, r *http.Request) {
	// Retrieve the assignment id from the url
	assignment_id := chi.URLParam(r, "id")

	// Get the authorized user
	user, err := GetAuthorizedUser(w, r)
	if err != nil {
		return
	}

	// Get the assignment from db by
	assignmentData.RLock()
	assignment, ok := assignmentData.m[assignment_id]
	assignmentData.RUnlock()

	// Assignment not found, return 404
	if !ok {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	// Check if this user is the owner of the assignment
	// If this is the owner of the assignment, return all the submission
	owner := assignment.GetOwner()
	if user.IsEqual(owner) {
		instructorResponse := make(map[string]any)
		languages := assignment.GetAllowedLanguages()
		allows := make([]string, 0, len(languages))
		for k := range languages {
			allows = append(allows, k)
		}
		instructorResponse["ID"] = assignment.GetId()
		instructorResponse["Name"] = assignment.GetName()
		instructorResponse["Allowed languages"] = allows
		instructorResponse["Submission"] = assignment.GetAllSubmissions()

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.Encode(instructorResponse)
		return
	}

	// If this is not the owner, check if they have permission to this assignment
	if !assignment.HasPermission(user.GetUsername()) {
		http.Error(w, "Permission required", http.StatusForbidden)
		return
	}

	// Return the approrriate resul to the student
	// Only include their submission, not everyone
	studentResult := make(map[string]any)
	languages := assignment.GetAllowedLanguages()
	allows := make([]string, 0, len(languages))
	for k := range languages {
		allows = append(allows, k)
	}
	studentResult["ID"] = assignment.GetId()
	studentResult["Name"] = assignment.GetName()
	studentResult["Allowed languages"] = allows
	studentResult["Grade"] = assignment.GetAllSubmissions()[user.GetUsername()]

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(studentResult)
}

func getSubmissions(w http.ResponseWriter, r *http.Request) {
	user, err := GetAuthorizedUser(w, r)
	if err != nil {
		return
	}

	assignmentId := chi.URLParam(r, "id")
	assignmentData.RLock()
	assignment, ok := assignmentData.m[assignmentId]
	assignmentData.RUnlock()

	if !ok {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	if !user.IsEqual(assignment.GetOwner()) && !assignment.HasPermission(user.GetUsername()) {
		http.Error(w, "Permission required", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, user.GetUsername()))

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

	if !user.IsEqual(assignment.GetOwner()) {
		submissionDirPath := filepath.Join(assignment.GetPath(), user.GetUsername())
		submissionDir, err := os.ReadDir(submissionDirPath)
		if err != nil {
			log.Println("Failed to read submission dir when student try to retrieve their latest submission")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		for _, file := range submissionDir {
			f, err := zipWriter.Create(file.Name())
			if err != nil {
				log.Println("Failed to create file in zip writer when student request submission")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			studentFile, err := os.ReadFile(filepath.Join(submissionDirPath, file.Name()))
			if err != nil {
				log.Println("Failed to read student submission when student get their submission")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			_, err = f.Write(studentFile)
			if err != nil {
				log.Println("Failed to write the content of student submission to the zip")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
		return
	}

	assignmentDir, err := os.ReadDir(assignment.GetPath())
	if err != nil {
		log.Println("Failed to Read the assignment dir when the instructor retrieve the submissions")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	for _, folder := range assignmentDir {
		if !folder.IsDir() {
			continue
		}

		if folder.Name() == "_input" || folder.Name() == "_expected" {
			continue
		}

		_, err = zipWriter.Create(folder.Name() + "/")
		if err != nil {
			log.Println("Failed to create folder for a student in zip file")
			return
		}

		studentDir, err := os.ReadDir(filepath.Join(assignment.GetPath(), folder.Name()))
		if err != nil {
			return
		}
		for _, file := range studentDir {
			f, err := zipWriter.Create(folder.Name() + "/" + file.Name())
			if err != nil {
				return
			}

			studentSubmission, err := os.ReadFile(filepath.Join(assignment.GetPath(), folder.Name(), file.Name()))
			if err != nil {
				return
			}

			_, err = f.Write(studentSubmission)
			if err != nil {
				return
			}
		}

	}
}
