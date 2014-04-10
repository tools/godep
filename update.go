package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var cmdUpdate = &Command{
	Usage: "update [packages]",
	Short: "use different revision of selected packages",
	Long: `
Update changes the named dependency packages to use the
revision of each currently installed in GOPATH. New code will
be copied into Godeps and the new revision will be written to
the manifest.

For more about specifying packages, see 'go help packages'.
`,
	Run: runUpdate,
}

func runUpdate(cmd *Command, args []string) {
	err := update(args)
	if err != nil {
		log.Fatalln(err)
	}
}

func update(args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}
	var g Godeps
	manifest := filepath.Join("Godeps", "Godeps.json")
	err := ReadGodeps(manifest, &g)
	if os.IsNotExist(err) {
		manifest = "Godeps"
		err = ReadGodeps(manifest, &g)
	}
	if err != nil {
		return err
	}
	for _, arg := range args {
		any := markMatches(arg, g.Deps)
		if !any {
			log.Println("not in manifest:", arg)
		}
	}
	deps, err := LoadVCSAndUpdate(g.Deps)
	if err != nil {
		return err
	}
	if len(deps) == 0 {
		return errors.New("no packages can be updated")
	}
	f, err := os.Create(manifest)
	if err != nil {
		return err
	}
	_, err = g.WriteTo(f)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	if manifest != "Godeps" {
		srcdir := filepath.FromSlash("Godeps/_workspace/src")
		copySrc(srcdir, deps)
	}
	return nil
}

// markMatches marks each entry in deps with an import path that
// matches pat. It returns whether any matches occurred.
func markMatches(pat string, deps []Dependency) (matched bool) {
	f := matchPattern(pat)
	for i, dep := range deps {
		if f(dep.ImportPath) {
			deps[i].matched = true
			matched = true
		}
	}
	return matched
}

// matchPattern(pattern)(name) reports whether
// name matches pattern.  Pattern is a limited glob
// pattern in which '...' means 'any string' and there
// is no other special syntax.
// Taken from $GOROOT/src/cmd/go/main.go.
func matchPattern(pattern string) func(name string) bool {
	re := regexp.QuoteMeta(pattern)
	re = strings.Replace(re, `\.\.\.`, `.*`, -1)
	// Special case: foo/... matches foo too.
	if strings.HasSuffix(re, `/.*`) {
		re = re[:len(re)-len(`/.*`)] + `(/.*)?`
	}
	reg := regexp.MustCompile(`^` + re + `$`)
	return func(name string) bool {
		return reg.MatchString(name)
	}
}

func LoadVCSAndUpdate(deps []Dependency) ([]Dependency, error) {
	var err1 error
	var paths []string
	for _, dep := range deps {
		paths = append(paths, dep.ImportPath)
	}
	ps, err := LoadPackages(paths...)
	if err != nil {
		return nil, err
	}
	noupdate := make(map[string]bool) // repo roots
	var candidates []*Dependency
	var tocopy []Dependency
	for i := range deps {
		dep := &deps[i]
		for _, pkg := range ps {
			if dep.ImportPath == pkg.ImportPath {
				dep.pkg = pkg
				break
			}
		}
		if dep.pkg == nil {
			log.Println(dep.ImportPath + ": error listing package")
			err1 = errors.New("error loading dependencies")
			continue
		}
		if dep.pkg.Error.Err != "" {
			log.Println(dep.pkg.Error.Err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		vcs, reporoot, err := VCSFromDir(dep.pkg.Dir, filepath.Join(dep.pkg.Root, "src"))
		if err != nil {
			log.Println(err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		dep.dir = dep.pkg.Dir
		dep.ws = dep.pkg.Root
		dep.root = filepath.ToSlash(reporoot)
		dep.vcs = vcs
		if dep.matched {
			candidates = append(candidates, dep)
		} else {
			noupdate[dep.root] = true
		}
	}
	if err1 != nil {
		return nil, err1
	}

	for _, dep := range candidates {
		dep.dir = dep.pkg.Dir
		dep.ws = dep.pkg.Root
		if noupdate[dep.root] {
			continue
		}
		id, err := dep.vcs.identify(dep.pkg.Dir)
		if err != nil {
			log.Println(err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		if dep.vcs.isDirty(dep.pkg.Dir, id) {
			log.Println("dirty working tree:", dep.pkg.Dir)
			err1 = errors.New("error loading dependencies")
			break
		}
		dep.Rev = id
		dep.Comment = dep.vcs.describe(dep.pkg.Dir, id)
		tocopy = append(tocopy, *dep)
	}
	if err1 != nil {
		return nil, err1
	}
	return tocopy, nil
}
