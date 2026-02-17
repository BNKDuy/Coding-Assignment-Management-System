package ExecUtil

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func RunJava(studentDir string, testFolder string) error {

	// Compile the program
	compile := exec.Command("javac", "Main.java")
	compile.Dir = studentDir
	err := compile.Run()
	if err != nil {
		log.Println("Failed to compile ", studentDir)
		return ErrCompileError
	}

	// Read the test folder
	files, err := os.ReadDir(testFolder)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(filepath.Join(studentDir, "_output"), os.ModePerm)

	// Execute the student program
	command := ""
	for _, file := range files {
		command += fmt.Sprintf(
			"java -cp %s %s < %s > %s;",
			studentDir,
			"Main",
			filepath.Join(testFolder, file.Name()),
			filepath.Join(studentDir, "_output", file.Name()))
	}
	// Set up a timeout for 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	err = cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ErrTimedOut
		}
		fmt.Print("Failed to execute Java program")
		return ErrRuntimeError
	}
	return nil
}
