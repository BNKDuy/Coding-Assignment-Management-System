package GradingUtil

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func RunStudentsProgram(root string, runner func(string, string) error) {
	// Get all the folder from the root dir
	files, err := os.ReadDir(root)
	if err != nil {
		log.Fatal(err)
	}

	// Semaphor to avoid too many goroutines that will overload the machine
	// Max numbers of goroutines is set to the number of CPU cores
	sem := make(chan struct{}, runtime.NumCPU())

	// Waitgroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	inputPath := filepath.Join(root, "_input")
	// Iterate over students folder
	for _, file := range files {
		if file.Name() == "_input" || file.Name() == "_expected" {
			continue
		}
		studentPath := filepath.Join(root, file.Name())
		wg.Add(1)
		sem <- struct{}{} // acquire a slot in the semaphor
		go func(sp string, ip string) {
			defer wg.Done()
			defer func() { <-sem }() // release the slot in the semaphor
			_ = runner(sp, ip)
		}(studentPath, inputPath)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Generate the grade and feedback for students
	err = getGradeAndFeedback(root)
	if err != nil {
		log.Println("Failed to calculate grade and generate feedback")
	}
}
