package GradingUtil

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func getGradeAndFeedback(root string) error {
	// Read the directory
	folders, err := os.ReadDir(root)
	if err != nil {
		log.Printf("Failed to read the directory %s to compare results", root)
		return err
	}

	// Open gradebook.csv
	f, err := os.OpenFile(filepath.Join(root, "gradebook.csv"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Println("Failed to open Gradebook.csv file.")
		return err
	}

	// Read the answer directory
	answerPath := filepath.Join(root, "_expected")
	answerDir, err := os.ReadDir(answerPath)
	if err != nil {
		log.Printf("Failed to read %s dir", answerPath)
		return err
	}

	// Write the headers for the csv
	header := []string{"Name", fmt.Sprintf("Tests passed (out of %d)", len(answerDir)), "Grade (percent)"}

	// Start the csv writer
	csvWriter := csv.NewWriter(f)
	csvWriter.Write(header)

	for _, folder := range folders {
		// These are not student dir, ignore them
		if folder.Name() == "_input" || folder.Name() == "_expected" {
			continue
		}

		// Compare result, get grade and feedback for this student
		studentPath := filepath.Join(root, folder.Name())
		grade := CompareResult(studentPath, answerPath)

		// Write the result to the csv
		csvWriter.Write([]string{folder.Name(), strconv.Itoa(grade), strconv.Itoa((grade / len(answerDir)) * 100)})
	}

	csvWriter.Flush()
	return nil
}
