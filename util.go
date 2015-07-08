package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Runs a command in dir.
// The name and args are as in exec.Command.
// Stdout, stderr, and the environment are inherited
// from the current process.
func runIn(dir, name string, args ...string) error {
	c := exec.Command(name, args...)

	if verbose {
		output, err := c.CombinedOutput()
		fmt.Printf("execute %+v", c)
		fmt.Printf(string(output))
		if err != nil {
			return fmt.Errorf("Error is %v", err)
		}
	}

	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
