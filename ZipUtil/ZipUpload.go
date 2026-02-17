package ZipUtil

import (
	"io"
	"net/http"
	"os"
)

func HandleUploadZip(w http.ResponseWriter, r *http.Request) (string, error) {
	// Max byte reader of 200MB
	r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		http.Error(w, "bad multipart form: "+err.Error(), http.StatusBadRequest)
		return "", err
	}

	// Get the Zip from the field file
	file, _, err := r.FormFile("zipfile")
	if err != nil {
		http.Error(w, "missing file: "+err.Error(), http.StatusBadRequest)
		file.Close()
		return "", err
	}
	defer file.Close()

	// Create e temporary zip file on the directory to copy over from the request
	tmpZip, err := os.CreateTemp("", "upload_*.zip")
	if err != nil {
		http.Error(w, "tempfile: "+err.Error(), http.StatusInternalServerError)
		tmpZip.Close()
		os.Remove(tmpZip.Name())
		return "", err
	}
	defer tmpZip.Close()
	defer os.Remove(tmpZip.Name())

	// Copy the zip from the request to the a temporary zip in my server
	if _, err := io.Copy(tmpZip, file); err != nil {
		http.Error(w, "save zip: "+err.Error(), http.StatusInternalServerError)
		return "", err
	}

	// Unzip the file and return the path
	unZippedDir := Unzip(tmpZip.Name())

	// From here down, handle how the expected output

	// Get the test cases from zip
	testCases, _, err := r.FormFile("Testcases")
	if err != nil {
		http.Error(w, "Test cases from the request", http.StatusBadRequest)
		testCases.Close()
		return "", err
	}
	defer testCases.Close()

	// Create e temporary zip file on the directory to copy over from the request
	tmpTestCaseZip, err := os.CreateTemp("", "test_case_*.zip")
	if err != nil {
		http.Error(w, "tempfile: "+err.Error(), http.StatusInternalServerError)
		tmpTestCaseZip.Close()
		os.Remove(tmpTestCaseZip.Name())
		return "", err
	}
	defer tmpTestCaseZip.Close()
	defer os.Remove(tmpTestCaseZip.Name())

	// Copy the zip from the request to the a temporary zip in my server
	if _, err := io.Copy(tmpTestCaseZip, testCases); err != nil {
		http.Error(w, "save zip: "+err.Error(), http.StatusInternalServerError)
		return "", err
	}

	// Unzip the file and return the path
	err = UnzipTests(tmpTestCaseZip.Name(), unZippedDir)
	if err != nil {
		http.Error(w, "Invalid testcases format", http.StatusBadRequest)
		return "", err
	}

	return unZippedDir, nil
}
