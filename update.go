package main

import (
	"errors"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
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
	g, err := loadDefaultGodepsFile()
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
	if _, err = g.save(); err != nil {
		return err
	}

	srcdir := relativeVendorTarget(VendorExperiment)
	copySrc(srcdir, deps)

	ok, err := needRewrite(g.Packages)
	if err != nil {
		return err
	}
	var rewritePaths []string
	if ok {
		for _, dep := range g.Deps {
			rewritePaths = append(rewritePaths, dep.ImportPath)
		}
	}
	return rewrite(nil, g.ImportPath, rewritePaths)
}

func needRewrite(importPaths []string) (bool, error) {
	if len(importPaths) == 0 {
		importPaths = []string{"."}
	}
	a, err := LoadPackages(importPaths...)
	if err != nil {
		return false, err
	}
	for _, p := range a {
		for _, name := range p.allGoFiles() {
			path := filepath.Join(p.Dir, name)
			hasSep, err := hasRewrittenImportStatement(path)
			if err != nil {
				return false, err
			}
			if hasSep {
				return true, nil
			}
		}
	}
	return false, nil
}

func hasRewrittenImportStatement(path string) (bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return false, err
	}
	for _, s := range f.Imports {
		name, _ := strconv.Unquote(s.Path.Value)
		if strings.Contains(name, sep) {
			return true, nil
		}
	}
	return false, nil
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
			log.Println("dirty working tree (please commit changes):", dep.pkg.Dir)
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
