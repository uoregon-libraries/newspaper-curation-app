// sudo.go is a way to provide a fake sudo to the docker container so various
// scripts will run without issue, since elevated permissions aren't needed in
// the container.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	// Check if there are command line arguments
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s command [arguments...]", os.Args[0])
		os.Exit(1)
	}

	// Prepare the command to execute
	var cmd = exec.Command(os.Args[1], os.Args[2:]...)

	// Set the command to use the current user's environment variables
	cmd.Env = os.Environ()

	// Redirect standard input, output, and error
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute the command
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		os.Exit(1)
	}
}
