package main

import (
	"go/build"
	"log"
	"os"
	"strings"
)

var cmdRestore = &Command{
	Name:  "restore",
	Short: "check out listed dependency versions in GOPATH",
	Long: `
Restore checks out the Godeps-specified version of each package in GOPATH.

If -v is given, verbose output is enabled.

If -d is given, debug output is enabled (you probably don't want this, see -v above).
`,
	Run: runRestore,
}

// Three phases:
// 1. Download all deps
// 2. Restore all deps (checkout the recorded rev)
// 3. Attempt to load all deps as a simple consistency check
func runRestore(cmd *Command, args []string) {
	var hadError bool
	checkErr := func() {
		if hadError {
			os.Exit(1)
		}
	}

	g, err := loadDefaultGodepsFile()
	if err != nil {
		log.Fatalln(err)
	}
	for i, dep := range g.Deps {
		verboseln("Downloading dependency (if needed):", dep.ImportPath)
		err := download(&dep)
		if err != nil {
			log.Printf("error downloading dep (%s): %s\n", dep.ImportPath, err)
			hadError = true
		}
		g.Deps[i] = dep
	}
	checkErr()
	for _, dep := range g.Deps {
		verboseln("Restoring dependency (if needed):", dep.ImportPath)
		err := restore(dep)
		if err != nil {
			log.Printf("error restoring dep (%s): %s\n", dep.ImportPath, err)
			hadError = true
		}
	}
	checkErr()
	for _, dep := range g.Deps {
		verboseln("Checking dependency:", dep.ImportPath)
		_, err := LoadPackages(dep.ImportPath)
		if err != nil {
			log.Printf("Dep (%s) restored, but was unable to load it with error:\n\t%s\n", dep.ImportPath, err)
			if me, ok := err.(errorMissingDep); ok {
				log.Println("\tThis may be because the dependencies were saved with an older version of godep (< v33).")
				log.Printf("\tTry `go get %s`. Then `godep save` to update deps.\n", me.i)
			}
			hadError = true
		}
	}
	checkErr()
}

// download downloads the given dependency.
// 2 Passes: 1) go get -d <pkg>, 2) git pull (if necessary)
func download(dep *Dependency) error {
	// make sure pkg exists somewhere in GOPATH

	args := []string{"get", "-d"}
	if debug {
		args = append(args, "-v")
	}

	o, err := runInWithOutput(".", "go", append(args, dep.ImportPath)...)
	if strings.Contains(o, "no buildable Go source files") {
		// We were able to fetch the repo, but didn't find any code to build
		// this can happen when a repo has changed structure or if the dep won't normally
		// be built on the current architecture until we implement our own fetcher this
		// may be the "best"" we can do.
		// TODO: replace go get
		err = nil
	}

	pkg, err := build.Import(dep.ImportPath, ".", build.FindOnly)
	if err != nil {
		debugln("Error finding package "+dep.ImportPath+" after go get:", err)
		return err
	}

	dep.vcs, err = VCSForImportPath(dep.ImportPath)
	if err != nil {
		dep.vcs, _, err = VCSFromDir(pkg.Dir, pkg.Root)
		if err != nil {
			return err
		}
	}

	if !dep.vcs.exists(pkg.Dir, dep.Rev) {
		dep.vcs.vcs.Download(pkg.Dir)
	}

	return nil
}

// restore checks out the given revision.
func restore(dep Dependency) error {
	debugln("Restoring:", dep.ImportPath, dep.Rev)

	pkg, err := build.Import(dep.ImportPath, ".", build.FindOnly)
	if err != nil {
		// THi should never happen
		debugln("Error finding package "+dep.ImportPath+" on restore:", err)
		return err
	}

	return dep.vcs.RevSync(pkg.Dir, dep.Rev)
}
