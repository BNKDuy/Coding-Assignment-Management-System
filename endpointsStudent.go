package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"server/ExecUtil"
	"server/GradingUtil"
	"server/Types"

	"github.com/go-chi/chi/v5"
)

func studentSignUp(w http.ResponseWriter, r *http.Request) {
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

	newUser := Types.NewUser(req.Username, req.Password, Types.RoleStudent)
	userData.Lock()
	userData.m[req.Username] = *newUser
	userData.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func submitAssignment(w http.ResponseWriter, r *http.Request) {
	user, err := GetAuthorizedUser(w, r)
	if err != nil {
		return
	}
	// Get the Assignment id from db
	assignmentId := chi.URLParam(r, "id")
	assignmentData.RLock()
	assignment, ok := assignmentData.m[assignmentId]
	assignmentData.RUnlock()

	// Assignment does not exists
	if !ok {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	// This student cannot submit this assignment
	if !assignment.HasPermission(user.GetUsername()) {
		http.Error(w, "Permisson required", http.StatusForbidden)
		return
	}

	// Retrieve the student submission from the request
	studentSubmission, submissionHeader, err := r.FormFile("Submission")
	if err != nil {
		http.Error(w, "Invalid submission", http.StatusBadRequest)
		return
	}
	defer studentSubmission.Close()

	// Parse the name of student file to get the file extension
	submissionNameLength := len(submissionHeader.Filename)
	extension := ""
	for i := submissionNameLength - 1; i >= 0; i-- {
		if submissionHeader.Filename[i] == '.' && i != submissionNameLength-1 {
			extension = submissionHeader.Filename[i+1:]
			break
		}
	}

	// This file has no extension, invalid file
	if extension == "" {
		http.Error(w, "Invalid submission", http.StatusBadRequest)
		return
	}

	// Languages allowed for this assignment
	assignmentAllowedLanguage := assignment.GetAllowedLanguages()

	// Check if they provide some invalid file type
	_, ok = Types.SupportedLanguage[extension]
	if !ok {
		allowed := "Supported language for this assignment:\n"
		for k := range assignmentAllowedLanguage {
			allowed += fmt.Sprintf("- %s\n", k)
		}
		http.Error(w, fmt.Sprintf("Unsupported file type\n%s", allowed), http.StatusBadRequest)
		return
	}

	// If student implementation is in a language not allowed by this assignment
	runner, ok := assignmentAllowedLanguage[extension]
	if !ok {
		allowed := "Supported language for this assignment:\n"
		for k := range assignmentAllowedLanguage {
			allowed += fmt.Sprintf("- %s\n", k)
		}
		http.Error(w, fmt.Sprintf("Language not allowed for this assignment\n%s", allowed), http.StatusBadRequest)
		return
	}

	// Check the file name is to make sure the name of the file is main
	if submissionHeader.Filename != fmt.Sprintf("main.%s", extension) && extension != "java" {
		http.Error(w, fmt.Sprintf("Bad resquest: Submission file must be named main.%s instead of %s", extension, submissionHeader.Filename), http.StatusBadRequest)
		return
	}

	// Check the file name is to make sure the name of the file is Main.java for java file
	if extension == "java" && submissionHeader.Filename != fmt.Sprintf("Main.%s", extension) {
		http.Error(w, fmt.Sprintf("Bad resquest: Submission file must be named Main.%s instead of %s", extension, submissionHeader.Filename), http.StatusBadRequest)
		return
	}

	// Create a temp dir to store the submisison on the disk
	// I want to run it to make sure it is a succesful submission before saving it to the suvbmission store
	// as it will overwrite the last submission attempt
	tempDir, err := os.MkdirTemp("", "submissions_*")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	// Create a file to copy over the file from the requestg to the disk
	tmpSubmissionFilePath := filepath.Join(tempDir, submissionHeader.Filename)
	tmpSubmissionFile, err := os.Create(tmpSubmissionFilePath)
	if err != nil {
		log.Println("Failed to create a file to store student submission")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer func() { tmpSubmissionFile.Close(); os.Remove(tmpSubmissionFile.Name()) }()

	// Copy the student file from request to disk
	_, err = io.Copy(tmpSubmissionFile, studentSubmission)
	if err != nil {
		log.Println("Failed to copy student submission from request to server")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	feedback := ""
	// Get the directory where the testcases are stored
	testFolderPath := filepath.Join(assignment.GetPath(), "_input")
	err = runner(tempDir, testFolderPath)
	if err != nil && err != ExecUtil.ErrTimedOut {
		log.Println(err)
		if err == ExecUtil.ErrCompileError {
			http.Error(w, "Failed to compile your submission", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to execute your submission", http.StatusInternalServerError)
		return
	}
	if err == ExecUtil.ErrTimedOut {
		feedback = "Time limit exceeded"
	}

	// If it reaches here, the student's code probably execute successfully
	// Start storing student's submission

	// Path to store student submission
	studentSubmissionDir := filepath.Join(assignment.GetPath(), user.GetUsername())

	// Check if this student has a submission,
	// If yes, delete it.
	info, err := os.Stat(studentSubmissionDir)
	if err == nil && info.IsDir() {
		err := os.RemoveAll(studentSubmissionDir)
		if err != nil {
			log.Println("Failed to delete student existing dir inside the assignment")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// Create a folder to store the student's submission
	// Because we delete it earlier so if we reach here, treats it as it does not exist before
	err = os.MkdirAll(studentSubmissionDir, os.ModePerm)
	if err != nil {
		log.Println("Failed to create a dir to store student submission inside assignment dir")
		http.Error(w, "Internal Server Error", http.StatusBadRequest)
		return
	}

	// Create a file to store the content of the student's submission
	submissionFile, err := os.Create(filepath.Join(studentSubmissionDir, submissionHeader.Filename))
	if err != nil {
		log.Println("Failed to create a file to store student submission inside assignment dir")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer submissionFile.Close()

	// Because before we move the file pointer to the end when we copy earlier
	// Move it up to the beginning of the file to copy again
	if _, err := tmpSubmissionFile.Seek(0, 0); err != nil {
		log.Println("Failed to move the pointer of tmpFIle to the top")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Copy the file from the temp file to Submission store
	_, err = io.Copy(submissionFile, tmpSubmissionFile)
	if err != nil {
		log.Println("Failed to copy student submission from tmpDir to assignment dir")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get the path to answer
	answerFolderPath := filepath.Join(assignment.GetPath(), "_expected")

	// This function basically compared all the result from this run to the expected output
	// and return the number of cases passed
	grade := GradingUtil.CompareResult(tempDir, answerFolderPath)
	tests, err := os.ReadDir(answerFolderPath)
	if err != nil {
		log.Println("Failed to read answer folder dir")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update the assignment grades
	gradeString := fmt.Sprintf("%d/%d", grade, len(tests))
	assignment.SetGrade(user.GetUsername(), gradeString)

	var res = make(map[string]string)
	res["Assignment name"] = assignment.GetName()
	res["Grade"] = gradeString

	if feedback != "" {
		res["Reason"] = feedback
	}
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(res)
}
