package ZipUtil

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func Unzip(source string) string {
	// Open a zip reader
	r, err := zip.OpenReader(source)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Put it in a _tmp folder to process
	dir := "_tmp"
	// Ensure the directory exists first
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Println(err)
	}

	// Make a folder in that temp dir to unzip this zip file to
	des, err := os.MkdirTemp(dir, "temp_*")
	if err != nil {
		fmt.Print(err)
	}

	// Iterate through the files in the archive,
	for _, f := range r.File {
		// Handle folder, if not exist, create 1
		outpath := filepath.Join(des, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(outpath, os.ModePerm)
			continue
		}

		// If it reach here, it is a file
		outfile, err := os.Create(outpath)
		if err != nil {
			fmt.Print(err)
		}
		defer outfile.Close()

		// Source file
		sourceFile, err := f.Open()
		if err != nil {
			fmt.Print(err)
		}
		defer sourceFile.Close()

		// Copy the source file from the upload zip to the file in the directory
		_, err = io.Copy(outfile, sourceFile)
		if err != nil {
			fmt.Print(err)
		}
	}

	// Return the path to the folder
	return des
}

func UnzipTests(source string, dest string) error {
	// Open a zip reader
	r, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer r.Close()

	// Iterate through the files in the archive,
	for _, f := range r.File {
		// Handle folder, if not exist, create 1
		outpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(outpath, os.ModePerm)
			continue
		}

		// If it reach here, it is a file
		outfile, err := os.Create(outpath)
		if err != nil {
			fmt.Print(err)
		}
		defer outfile.Close()

		// Source file
		sourceFile, err := f.Open()
		if err != nil {
			fmt.Print(err)
		}
		defer sourceFile.Close()

		// Copy the source file from the upload zip to the file in the directory
		_, err = io.Copy(outfile, sourceFile)
		if err != nil {
			fmt.Print(err)
		}
	}

	return nil
}
