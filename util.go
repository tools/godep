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
	_, err := runInWithOutput(dir, name, args...)
	return err
}

func runInWithOutput(dir, name string, args ...string) (string, error) {
	c := exec.Command(name, args...)
	c.Dir = dir
	o, err := c.CombinedOutput()

	if debug {
		fmt.Printf("execute: %+v\n", c)
		fmt.Printf(" output: %s\n", string(o))
	}

	return string(o), err
}
