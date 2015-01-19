package main

import (
	"fmt"
)

var cmdPath = &Command{
	Usage: "path",
	Short: "print GOPATH for dependency code",
	Long: `
Command path prints a path for use in env var GOPATH
that makes available the specified version of each dependency.

The printed path does not include any GOPATH value from
the environment.

For more about how GOPATH works, see 'go help gopath'.
`,
	Run: runPath,
}

// Print the gopath that points to
// the included dependency code.
func runPath(cmd *Command, args []string) {
	if len(args) != 0 {
		cmd.UsageExit()
	}
	gopath := prepareGopath()
	fmt.Println(gopath)
}
