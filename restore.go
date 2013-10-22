package main

import (
	"log"
	"os"
	"path/filepath"
)

var cmdRestore = &Command{
	Usage: "restore",
	Short: "install package versions listed as dependencies",
	Long: `
Restore installs the version of each package specified in Godeps.
`,
	Run: runRestore,
}

func runRestore(cmd *Command, args []string) {
	g, err := ReadGodeps(findGodepsJSON())
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

func restore(dep Dependency) error {
	dir := filepath.Join(dep.outerRoot, "src", dep.repoRoot.Root)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(dir), 0777)
		if err != nil {
			return err
		}
		err = dep.vcs.vcs.Create(dir, dep.repoRoot.Repo)
		if err != nil {
			return err
		}
	}
	if !dep.vcs.exists(dir, dep.Rev) {
		dep.vcs.Download(dir)
	}
	return dep.vcs.RevSync(dir, dep.Rev)
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
