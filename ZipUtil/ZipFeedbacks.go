package ZipUtil

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ZipFeedback(source string) string {
	// Create the feedback zip to store the feedback
	outZip := filepath.Join(source, "feedback.zip")
	f, _ := os.Create(outZip)
	defer f.Close()

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new zip archive.
	w := zip.NewWriter(buf)

	// Add a compressor to compress the zip
	w.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})

	// Open the gradebook from the server directory
	gradebookPath := filepath.Join(source, "gradebook.csv")
	grade, err := os.ReadFile(gradebookPath)
	if err != nil {
		fmt.Print("Failed to open gradebook.csv")
	}
	// Create a gradebook in the zip folder to return
	gradebookFile, err := w.Create("gradebook.csv")
	if err != nil {
		fmt.Println(err)
	}

	// Copy the data from the gradebook from the server to the zip
	_, err = gradebookFile.Write(grade)
	if err != nil {
		fmt.Println("Failed to write the gradebook to zip")
	}

	// Iterate over the directory and get the feedback for each students
	files, err := os.ReadDir(source)
	if err != nil {
		fmt.Print(err)
	}

	for _, file := range files {
		// Ignore these, they are not student folder
		if file.Name() == "_answer" ||
			file.Name() == "feedback.zip" ||
			file.Name() == "gradebook.csv" ||
			file.Name() == "_input" ||
			file.Name() == "_expected" {
			continue
		}

		// Create a .txt file with student's name and put the feedback in there
		f, err := w.Create(file.Name() + ".txt")
		if err != nil {
			fmt.Print(err)
		}
		feedbackPath := filepath.Join(source, file.Name(), "results.txt")
		data, err := os.ReadFile(feedbackPath)
		if err != nil {
			fmt.Printf("Failed to open feedback file for %s", file.Name())
			data = []byte("No feedback found!\n")
		}
		_, err = f.Write(data)
		if err != nil {
			fmt.Printf("Zip write err for %s: %s", file, err)
		}
	}

	// Close to flush central directory to the buffer
	if err := w.Close(); err != nil {
		fmt.Printf("zip close: %s", err)
	}

	// Write the buffer to disk
	if err := os.WriteFile(outZip, buf.Bytes(), 0644); err != nil {
		fmt.Printf("write zip: %s", err)
	}

	// Return the path of the zip
	return outZip
}
