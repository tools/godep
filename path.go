package main

import (
	"fmt"
)

var cmdPath = &Command{
	Usage: "path",
	Short: "print sandbox path for use in a GOPATH",
	Long: `
Path ensures a sandbox is prepared for the dependencies
in file Godeps. It prints a path for use in a GOPATH
that makes available the specified version of each dependency.

The printed path does not include any GOPATH value from
the environment.

For more about how GOPATH works, see 'go help gopath'.
`,
	Run: runPath,
}

// Set up a sandbox and print the resulting gopath.
func runPath(cmd *Command, args []string) {
	if len(args) != 0 {
		cmd.UsageExit()
	}
	gopath := prepareGopath()
	fmt.Println(gopath)
}
