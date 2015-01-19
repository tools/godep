package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Godeps describes what a package needs to be rebuilt reproducibly.
// It's the same information stored in file Godeps.
type Godeps struct {
	ImportPath string
	GoVersion  string
	Packages   []string `json:",omitempty"` // Arguments to save, if any.
	Deps       []Dependency
}

// A Dependency is a specific revision of a package.
type Dependency struct {
	ImportPath string
	Comment    string `json:",omitempty"` // Description of commit, if present.
	Rev        string // VCS-specific commit ID.

	// used by command save & update
	ws   string // workspace
	root string // import path to repo root
	dir  string // full path to package

	// used by command update
	matched bool // selected for update by command line
	pkg     *Package

	// used by command go
	vcs *VCS
}

// pkgs is the list of packages to read dependencies
func (g *Godeps) Load(pkgs []*Package) error {
	var err1 error
	var path, seen []string
	for _, p := range pkgs {
		if p.Standard {
			log.Println("ignoring stdlib package:", p.ImportPath)
			continue
		}
		if p.Error.Err != "" {
			log.Println(p.Error.Err)
			err1 = errors.New("error loading packages")
			continue
		}
		_, reporoot, err := VCSFromDir(p.Dir, filepath.Join(p.Root, "src"))
		if err != nil {
			log.Println(err)
			err1 = errors.New("error loading packages")
			continue
		}
		seen = append(seen, filepath.ToSlash(reporoot))
		path = append(path, p.Deps...)
	}
	var testImports []string
	for _, p := range pkgs {
		testImports = append(testImports, p.TestImports...)
		testImports = append(testImports, p.XTestImports...)
	}
	ps, err := LoadPackages(testImports...)
	if err != nil {
		return err
	}
	for _, p := range ps {
		if p.Standard {
			continue
		}
		if p.Error.Err != "" {
			log.Println(p.Error.Err)
			err1 = errors.New("error loading packages")
			continue
		}
		path = append(path, p.ImportPath)
		path = append(path, p.Deps...)
	}
	for i, p := range path {
		path[i] = unqualify(p)
	}
	sort.Strings(path)
	path = uniq(path)
	ps, err = LoadPackages(path...)
	if err != nil {
		return err
	}
	for _, pkg := range ps {
		if pkg.Error.Err != "" {
			log.Println(pkg.Error.Err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		if pkg.Standard {
			continue
		}
		vcs, reporoot, err := VCSFromDir(pkg.Dir, filepath.Join(pkg.Root, "src"))
		if err != nil {
			log.Println(err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		if containsPathPrefix(seen, pkg.ImportPath) {
			continue
		}
		seen = append(seen, pkg.ImportPath)
		id, err := vcs.identify(pkg.Dir)
		if err != nil {
			log.Println(err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		if vcs.isDirty(pkg.Dir, id) {
			log.Println("dirty working tree:", pkg.Dir)
			err1 = errors.New("error loading dependencies")
			continue
		}
		comment := vcs.describe(pkg.Dir, id)
		g.Deps = append(g.Deps, Dependency{
			ImportPath: pkg.ImportPath,
			Rev:        id,
			Comment:    comment,
			dir:        pkg.Dir,
			ws:         pkg.Root,
			root:       filepath.ToSlash(reporoot),
			vcs:        vcs,
		})
	}
	return err1
}

func ReadGodeps(path string, g *Godeps) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return json.NewDecoder(f).Decode(g)
}

func copyGodeps(g *Godeps) *Godeps {
	h := *g
	h.Deps = make([]Dependency, len(g.Deps))
	copy(h.Deps, g.Deps)
	return &h
}

func eqDeps(a, b []Dependency) bool {
	ok := true
	for _, da := range a {
		for _, db := range b {
			if da.ImportPath == db.ImportPath && da.Rev != db.Rev {
				ok = false
			}
		}
	}
	return ok
}

func ReadAndLoadGodeps(path string) (*Godeps, error) {
	g := new(Godeps)
	err := ReadGodeps(path, g)
	if err != nil {
		return nil, err
	}

	for i := range g.Deps {
		d := &g.Deps[i]
		d.vcs, err = VCSForImportPath(d.ImportPath)
		if err != nil {
			return nil, err
		}
	}
	return g, nil
}

func (g *Godeps) WriteTo(w io.Writer) (int64, error) {
	b, err := json.MarshalIndent(g, "", "\t")
	if err != nil {
		return 0, err
	}
	n, err := w.Write(append(b, '\n'))
	return int64(n), err
}

// containsPathPrefix returns whether any string in a
// is s or a directory containing s.
// For example, pattern ["a"] matches "a" and "a/b"
// (but not "ab").
func containsPathPrefix(pats []string, s string) bool {
	for _, pat := range pats {
		if pat == s || strings.HasPrefix(s, pat+"/") {
			return true
		}
	}
	return false
}

func uniq(a []string) []string {
	i := 0
	s := ""
	for _, t := range a {
		if t != s {
			a[i] = t
			i++
			s = t
		}
	}
	return a[:i]
}

// goVersion returns the version string of the Go compiler
// currently installed, e.g. "go1.1rc3".
func goVersion() (string, error) {
	// Godep might have been compiled with a different
	// version, so we can't just use runtime.Version here.
	cmd := exec.Command("go", "version")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(out))
	s = strings.TrimSuffix(s, " "+runtime.GOOS+"/"+runtime.GOARCH)
	s = strings.TrimPrefix(s, "go version ")
	return s, nil
}
