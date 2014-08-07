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
	Usage: "go command [arguments]",
	Short: "run the go tool in a sandbox",
	Long: `
Go runs the go tool in a temporary GOPATH sandbox
with the dependencies listed in file Godeps.

Any go tool command can run this way, but "godep go get"
is unnecessary and has been disabled. Instead, use
"godep go install".
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
		gopath += string(os.PathListSeparator) + os.Getenv("GOPATH")
	}
	if len(args) > 0 && args[0] == "get" {
		log.Printf("invalid subcommand: %q", "go get")
		fmt.Fprintln(os.Stderr, "Use 'godep go install' instead.")
		fmt.Fprintln(os.Stderr, "Run 'godep help go' for usage.")
		os.Exit(2)
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
	dir, isDir := findGodeps()
	if dir == "" {
		log.Fatalln("No Godeps found (or in any parent directory)")
	}
	if isDir {
		return filepath.Join(dir, "Godeps", "_workspace")
	}
	log.Println(strings.TrimSpace(noSourceCodeWarning))
	g, err := ReadAndLoadGodeps(filepath.Join(dir, "Godeps"))
	if err != nil {
		log.Fatalln(err)
	}
	gopath, err = sandboxAll(g.Deps)
	if err != nil {
		log.Fatalln(err)
	}
	return gopath
}

// findGodeps looks for a directory entry "Godeps" in the
// current directory or any parent, and returns the containing
// directory and whether the entry itself is a directory.
// If Godeps can't be found, findGodeps returns "".
// For any other error, it exits the program.
func findGodeps() (dir string, isDir bool) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	return findInParents(wd, "Godeps")
}

// isRoot returns true iff a path is a root.
// On Unix: "/".
// On Windows: "C:\", "D:\", ...
func isRoot(p string) bool {
	p = filepath.Clean(p)
	volume := filepath.VolumeName(p)

	p = strings.TrimPrefix(p, volume)
	p = filepath.ToSlash(p)

	return p == "/"
}

// findInParents returns the path to the directory containing name
// in dir or any ancestor, and whether name itself is a directory.
// If name cannot be found, findInParents returns the empty string.
func findInParents(dir, name string) (container string, isDir bool) {
	for {
		fi, err := os.Stat(filepath.Join(dir, name))
		if os.IsNotExist(err) && isRoot(dir) {
			return "", false
		}
		if os.IsNotExist(err) {
			dir = filepath.Dir(dir)
			continue
		}
		if err != nil {
			log.Fatalln(err)
		}
		return dir, fi.IsDir()
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

const noSourceCodeWarning = `
warning: outdated Godeps missing source code

The ability to read this format will be removed in the future.
See http://goo.gl/RpYs8e for a discussion of the upcoming removal.

To avoid this warning, ask the maintainer of this package to run
'godep save' without flag -copy.
`
