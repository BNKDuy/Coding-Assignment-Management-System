package ExecUtil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func RunC(studentDir string, testFolder string) error {
	// Set up a timeout for 5 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Compile the program
	compile := exec.Command("gcc", "main.c", "-O2", "-o", "main")
	compile.Dir = studentDir
	err := compile.Run()
	if err != nil {
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
			"%s < %s > %s;",
			filepath.Join(studentDir, "main"),
			filepath.Join(testFolder, file.Name()),
			filepath.Join(studentDir, "_output", file.Name()))
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	err = cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ErrTimedOut
		}
		fmt.Print("Failed to execute C program")
		return ErrRuntimeError
	}
	return nil
}
