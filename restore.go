package main

import (
	"log"
	"os"
	"path/filepath"
)

var cmdRestore = &Command{
	Usage: "restore",
	Short: "check out listed dependency versions in GOPATH",
	Long: `
Restore checks out the Godeps-specified version of each package in GOPATH.
`,
	Run: runRestore,
}

func runRestore(cmd *Command, args []string) {
	g, err := ReadAndLoadGodeps(findGodepsJSON())
	if err != nil {
		log.Fatalln(err)
	}
	hadError := false
	for _, dep := range g.Deps {
		err := restore(dep)
		if err != nil {
			log.Println("restore:", err)
			hadError = true
		}
	}
	if hadError {
		os.Exit(1)
	}
}

// restore downloads the given dependency and checks out
// the given revision.
func restore(dep Dependency) error {
	// make sure pkg exists somewhere in GOPATH
	err := runIn(".", "go", "get", "-d", dep.ImportPath)
	if err != nil {
		return err
	}
	ps, err := LoadPackages(dep.ImportPath)
	if err != nil {
		return err
	}
	pkg := ps[0]
	if !dep.vcs.exists(pkg.Dir, dep.Rev) {
		dep.vcs.vcs.Download(pkg.Dir)
	}
	return dep.vcs.RevSync(pkg.Dir, dep.Rev)
}

func findGodepsJSON() (path string) {
	dir, isDir := findGodeps()
	if dir == "" {
		log.Fatalln("No Godeps found (or in any parent directory)")
	}
	if isDir {
		return filepath.Join(dir, "Godeps", "Godeps.json")
	}
	return filepath.Join(dir, "Godeps")
}
