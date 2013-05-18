package main

import (
	"log"
	"os"
	"path/filepath"
)

var cmdSave = &Command{
	Usage: "save [package]",
	Short: "list current dependencies to a file",
	Long: `
Save writes a list of the dependencies of the named package along with
the exact source control revision of each dependency.

Output is to file Godeps in the package's directory.

If package is omitted, it is taken to be ".".
`,
	Run: runSave,
}

func runSave(cmd *Command, args []string) {
	if len(args) > 1 {
		cmd.UsageExit()
	}
	pkg := "."
	if len(args) == 1 {
		pkg = args[0]
	}
	a, err := LoadPackages(pkg)
	if err != nil {
		log.Fatalln(err)
	}
	p := a[0]
	if p.Standard {
		log.Fatalln("ignoring stdlib package:", p.ImportPath)
	}
	g, err := LoadGodeps(p)
	if err != nil {
		log.Fatalln(err)
	}
	path := filepath.Join(p.Dir, "Godeps")
	f, err := os.Create(path)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = g.WriteTo(f)
	if err != nil {
		log.Fatalln(err)
	}
	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
