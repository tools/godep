package main

import (
	"os"
	"os/exec"
)

// Returns true if path definitely exists; false if path doesn't
// exist or is unknown because of an error.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Runs a command in dir.
// The name and args are as in exec.Command.
// Stdout, stderr, and the environment are inherited
// from the current process.
func runIn(dir, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
