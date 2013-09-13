package main

import (
	"fmt"
	"log"
)

var cmdPath = &Command{
	Usage: "path",
	Short: "print sandbox GOPATH",
	Long: `
Path ensures a sandbox is prepared for the dependencies
in file Godeps. It prints a GOPATH that makes available
the specified version of each dependency.

The printed GOPATH does not include any GOPATH value from
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
	g, err := ReadGodeps("Godeps")
	if err != nil {
		log.Fatalln(err)
	}
	gopath, err := sandboxAll(g.Deps)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(gopath)
}
