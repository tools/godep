package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

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
	p := strings.Split(string(out), " ")
	if len(p) < 3 {
		return "", fmt.Errorf("Error splitting output of `go version`: Expected 3 or more elements, but there are < 3: %q", string(out))
	}
	return p[2], nil
}
