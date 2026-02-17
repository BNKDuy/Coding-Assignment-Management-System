package GradingUtil

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func CompareResult(studentPath string, answerPath string) int {
	sum := 0
	// Open the directory that stores the answers
	answerDir, err := os.ReadDir(answerPath)
	if err != nil {
		log.Printf("Failed to read directory for %s when trying to compare results.", answerPath)
		return sum
	}
	// The number of tests in the answer folder
	numTest := len(answerDir)

	// This is the feedback
	testResult := ""

	// Iterate through every expected test case and compare that to a student test case
	for _, test := range answerDir {
		// Open the student output
		output := filepath.Join(studentPath, "_output", test.Name())
		_, err = os.Stat(output)
		if os.IsNotExist(err) {
			log.Printf("%s: Missing\n", strings.TrimSuffix(test.Name(), ".txt\n"))
			testResult += fmt.Sprintf("%s: fail\n", strings.TrimSuffix(test.Name(), ".txt"))
			continue
		}
		// Read the content of the student output
		outputContent, err := os.ReadFile(output)
		if err != nil {
			continue
		}

		// Read the content of the expected output
		expected := filepath.Join(answerPath, test.Name())
		expectedContent, err := os.ReadFile(expected)
		if err != nil {
			continue
		}

		// If the result is the same, the students output is correct
		// Give them mark
		// And give them feedback too
		if bytes.Equal(bytes.TrimSpace(outputContent), bytes.TrimSpace(expectedContent)) {
			sum += 1
			testResult += fmt.Sprintf("%s: pass\n", strings.TrimSuffix(test.Name(), ".txt"))
		} else {
			testResult += fmt.Sprintf("%s: fail\n", strings.TrimSuffix(test.Name(), ".txt"))
		}
	}

	// If this student passed all the test cases
	if sum == numTest {
		testResult += "All test passed!"
	}

	// Write the feedback to the student directory
	_ = os.WriteFile(filepath.Join(studentPath, "results.txt"), []byte(testResult), 0666)

	return sum
}
