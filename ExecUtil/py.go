package ExecUtil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func RunPython(studentDir string, testFolder string) error {
	files, err := os.ReadDir(testFolder)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(filepath.Join(studentDir, "_output"), os.ModePerm)

	command := ""
	for _, file := range files {
		command += fmt.Sprintf(
			"python3 %s < %s > %s;",
			filepath.Join(studentDir, "main.py"),
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
		fmt.Print("Failed to execute Python program")
		return ErrRuntimeError
	}

	return nil
}
