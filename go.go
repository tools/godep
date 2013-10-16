package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var spool = filepath.Join(os.TempDir(), "godep")

var cmdGo = &Command{
	Usage: "go command [args...]",
	Short: "run the go tool in a sandbox",
	Long: `
Go runs the go tool in a temporary GOPATH sandbox
with the dependencies listed in file Godeps.
`,
	Run: runGo,
}

// Set up a sandbox and run the go tool. The sandbox is built
// out of specific checked-out revisions of repos. We keep repos
// and revs materialized on disk under the assumption that disk
// space is cheap and plentiful, and writing files is slow.
// Everything is kept in the spool directory.
func runGo(cmd *Command, args []string) {
	gopath := prepareGopath()
	if s := os.Getenv("GOPATH"); s != "" {
		gopath += ":" + os.Getenv("GOPATH")
	}
	c := exec.Command("go", args...)
	c.Env = append(envNoGopath(), "GOPATH="+gopath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		log.Fatalln("go", err)
	}
}

// prepareGopath reads dependency information from the filesystem
// entry name, fetches any necessary code, and returns a gopath
// causing the specified dependencies to be used.
func prepareGopath() (gopath string) {
	const name = "Godeps"
	dir, isDir := findInParents(name)
	if isDir {
		return filepath.Join(dir, name, "_workspace")
	}
	g, err := ReadGodeps(filepath.Join(dir, name))
	if err != nil {
		log.Fatalln(err)
	}
	gopath, err = sandboxAll(g.Deps)
	if err != nil {
		log.Fatalln(err)
	}
	return gopath
}

// findInParents returns the path to the directory containing file name
// in the current directory or any ancestor, and whether name itself
// is a directory.
func findInParents(name string) (dir string, isDir bool) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	for {
		fi, err := os.Stat(filepath.Join(wd, name))
		if os.IsNotExist(err) && wd == "/" {
			log.Fatalln("No", name, "found (or in any parent directory)")
		}
		if os.IsNotExist(err) {
			wd = filepath.Dir(wd)
			continue
		}
		if err != nil {
			log.Fatalln(err)
		}
		return wd, fi.IsDir()
	}
}

func envNoGopath() (a []string) {
	for _, s := range os.Environ() {
		if !strings.HasPrefix(s, "GOPATH=") {
			a = append(a, s)
		}
	}
	return a
}

// sandboxAll ensures that the commits in deps are available
// on disk, and returns a GOPATH string that will cause them
// to be used.
func sandboxAll(a []Dependency) (gopath string, err error) {
	var path []string
	for _, dep := range a {
		dir, err := sandbox(dep)
		if err != nil {
			return "", err
		}
		path = append(path, dir)
	}
	return strings.Join(path, ":"), nil
}

// sandbox ensures that commit d is available on disk,
// and returns a GOPATH string that will cause it to be used.
func sandbox(d Dependency) (gopath string, err error) {
	if !exists(d.RepoPath()) {
		if err = d.CreateRepo("fast", "main"); err != nil {
			return "", fmt.Errorf("create repo: %s", err)
		}
	}
	err = d.checkout()
	if err != nil && d.FastRemotePath() != "" {
		err = d.fetchAndCheckout("fast")
	}
	if err != nil {
		err = d.fetchAndCheckout("main")
	}
	if err != nil {
		return "", err
	}
	return d.Gopath(), nil
}
