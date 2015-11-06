package main

import (
	"fmt"
	"os/exec"
)

// Runs a command in dir.
// The name and args are as in exec.Command.
// Stdout, stderr, and the environment are inherited
// from the current process.
func runIn(dir, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	output, err := c.CombinedOutput()

	if verbose {
		fmt.Printf("execute: %+v\n", c)
		fmt.Printf(" output: %s\n", output)
	}

	return err
}
