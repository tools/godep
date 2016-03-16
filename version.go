package main

import (
	"fmt"
	"runtime"
)

const version = 58

var cmdVersion = &Command{
	Name:  "version",
	Short: "show version info",
	Long: `

Displays the version of godep as well as the target OS, architecture and go runtime version.
`,
	Run: runVersion,
}

func versionString() string {
	return fmt.Sprintf("godep v%d (%s/%s/%s)", version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}

func runVersion(cmd *Command, args []string) {
	fmt.Printf("%s\n", versionString())
}
