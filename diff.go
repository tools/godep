package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/tools/godep/Godeps/_workspace/src/github.com/pmezard/go-difflib/difflib"
)

var cmdDiff = &Command{
	Usage: "diff",
	Short: "shows the diff between current and previously saved set of dependencies",
	Long: `
Shows the difference, in a unified diff format, between the 
current set of dependencies and those generated on a 
previous 'go save' execution.
`,
	Run: runDiff,
}

func runDiff(cmd *Command, args []string) {
	var gold Godeps

	_, err := readOldGodeps(&gold)
	if err != nil {
		log.Fatalln(err)
	}

	pkgs := []string{"."}
	dot, err := LoadPackages(pkgs...)
	if err != nil {
		log.Fatalln(err)
	}

	ver, err := goVersion()
	if err != nil {
		log.Fatalln(err)
	}

	gnew := &Godeps{
		ImportPath: dot[0].ImportPath,
		GoVersion:  ver,
	}

	err = gnew.Load(dot)
	if err != nil {
		log.Fatalln(err)
	}

	diff, err := diffStr(&gold, gnew)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(diff)
}

// diffStr returns a unified diff string of two Godeps.
func diffStr(a, b *Godeps) (string, error) {
	var ab, bb bytes.Buffer

	_, err := a.WriteTo(&ab)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = b.WriteTo(&bb)
	if err != nil {
		log.Fatalln(err)
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(ab.String()),
		B:        difflib.SplitLines(bb.String()),
		FromFile: "Godeps",
		ToFile:   "$GOPATH",
		Context:  10,
	}
	return difflib.GetUnifiedDiffString(diff)
}
