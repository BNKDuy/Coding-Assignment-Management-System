package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"server/ExecUtil"
	"server/GradingUtil"
	"server/ZipUtil"
)

func handleRequest(w http.ResponseWriter, r *http.Request, runner func(string, string) error) {
	// Receive and unzip the zip the input from client
	unZippedDir, _ := ZipUtil.HandleUploadZip(w, r)
	defer os.RemoveAll(unZippedDir)

	// Run the test on student's program
	GradingUtil.RunStudentsProgram(unZippedDir, runner)

	// Zip the feedback to return
	feedbackZip := ZipUtil.ZipFeedback(unZippedDir)

	// Send the ZIP back to the client
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(feedbackZip)))
	http.ServeFile(w, r, feedbackZip)
}

func handlePython(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, ExecUtil.RunPython)
}

func handleC(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, ExecUtil.RunC)
}

func handleCPP(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, ExecUtil.RunCPP)
}

func handleGo(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, ExecUtil.RunGo)
}

func handleJava(w http.ResponseWriter, r *http.Request) {
	handleRequest(w, r, ExecUtil.RunJava)
}
