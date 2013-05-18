package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Godeps describes what a package needs to be rebuilt reproducibly.
// It's the same information stored in file Godeps.
type Godeps struct {
	ImportPath string
	GoVersion  string
	Deps       []Dependency

	outerRoot string
}

// A Dependency is a specific revision of a package.
type Dependency struct {
	ImportPath string
	Rev        string // VCS-specific commit ID.
	Comment    string // Description of commit, if present.

	outerRoot string // dir, if present, in outer GOPATH
}

func LoadGodeps(p *Package) (*Godeps, error) {
	var err error
	g := new(Godeps)
	g.ImportPath = p.ImportPath
	g.GoVersion, err = goVersion()
	if err != nil {
		return nil, err
	}
	deps, err := LoadPackages(p.Deps...)
	if err != nil {
		log.Fatalln(err)
	}
	seen := []string{p.ImportPath + "/"}
	for _, dep := range deps {
		name := dep.ImportPath
		if dep.Error.Err != "" {
			log.Println(dep.Error.Err)
			err = errors.New("error loading dependencies")
			continue
		}
		if !prefixIn(seen, name) && !dep.Standard {
			seen = append(seen, name+"/")
			var id string
			id, err = vcsCurrentCheckout(dep.Dir)
			if err != nil {
				log.Println(err)
				err = errors.New("error loading dependencies")
				continue
			}
			if vcsIsDirty(dep.Dir) {
				log.Println("dirty working tree:", dep.Dir)
				err = errors.New("error loading dependencies")
				continue
			}
			comment, _ := vcsDescribe(dep.Dir, id)
			g.Deps = append(g.Deps, Dependency{
				ImportPath: name,
				Rev:        id,
				Comment:    comment,
			})
		}
	}
	if err != nil {
		return nil, err
	}
	return g, nil
}

func ReadGodeps(path string) (*Godeps, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	g := new(Godeps)
	err = json.NewDecoder(f).Decode(g)
	if err != nil {
		return nil, err
	}
	err = g.loadGoList()
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *Godeps) loadGoList() error {
	a := []string{g.ImportPath}
	for _, d := range g.Deps {
		a = append(a, d.ImportPath)
	}
	ps, err := LoadPackages(a...)
	if err != nil {
		return err
	}
	g.outerRoot = ps[0].Root
	for i, p := range ps[1:] {
		g.Deps[i].outerRoot = p.Root
	}
	return nil
}

func (g *Godeps) WriteTo(w io.Writer) (int, error) {
	b, err := json.MarshalIndent(g, "", "\t")
	if err != nil {
		return 0, err
	}
	return w.Write(append(b, '\n'))
}

func (d Dependency) ImportRoot() string {
	rr, err := repoRootForImportPathStatic(d.ImportPath, "")
	if err != nil {
		log.Fatalln(err)
	}
	return rr.root
}

// Returns a path to the local copy of d's repository.
// E.g.
//
//   ImportPath             Remote
//   github.com/kr/s3       {spool}/github.com/kr/s3
//   github.com/lib/pq/oid  {spool}/github.com/lib/pq
func (d Dependency) RepoPath() string {
	return filepath.Join(spool, "repo", d.ImportRoot())
}

// Returns a URL for the remote copy of the repository.
// E.g.
//
//   ImportPath             Remote
//   github.com/kr/s3       https://github.com/kr/s3.git
//   github.com/lib/pq/oid  https://github.com/lib/pq.git
func (d Dependency) Remote() string {
	rr, err := repoRootForImportPathStatic(d.ImportPath, "")
	if err != nil {
		log.Fatalln(err)
	}
	return rr.repo
}

// Returns a path to the checked-out copy of d's commit.
func (d Dependency) Workdir() string {
	return filepath.Join(d.Gopath(), "src", d.ImportPath)
}

// Returns a path to the checked-out copy of d's commit.
func (d Dependency) WorkdirRoot() string {
	return filepath.Join(d.Gopath(), "src", d.ImportRoot())
}

// Returns a path to a parent of Workdir such that using
// Gopath in GOPATH makes d available to the go tool.
func (d Dependency) Gopath() string {
	return filepath.Join(spool, "rev", d.Rev[:2], d.Rev[2:])
}

// Returns the url of a local disk clone of the repo, if any.
func (d Dependency) altRemote() string {
	if d.outerRoot != "" {
		return d.outerRoot + "/src/" + d.ImportRoot() + "/.git"
	}
	return ""
}

func (d Dependency) createRepo() error {
	// 1. if d's repo exists in the outer GOPATH, clone locally
	//    (but still set the origin remote to d.Remote())
	// 2. else clone the remote
	return vcsCreate(d.RepoPath(), d.Remote(), d.altRemote())
}

func (d Dependency) fetch() error {
	if vcsRevExists(d.RepoPath(), d.Rev) {
		return nil
	}
	if alt := d.altRemote(); alt != "" {
		err := vcsFetch(d.RepoPath(), alt)
		if err != nil {
			return err
		}
		if vcsRevExists(d.RepoPath(), d.Rev) {
			return nil
		}
	}
	err := vcsFetch(d.RepoPath(), "origin")
	if err != nil {
		return err
	}
	if vcsRevExists(d.RepoPath(), d.Rev) {
		return nil
	}
	return fmt.Errorf("can't find rev %s in %s", d.Rev, d.RepoPath())
}

func prefixIn(a []string, s string) bool {
	for _, p := range a {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func goVersion() (string, error) {
	var b bytes.Buffer
	cmd := exec.Command("go", "version")
	cmd.Stdout = &b
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func lookGopath(file string) string {
	pathenv := os.Getenv("GOPATH")
	if pathenv == "" {
		return ""
	}
	for _, dir := range strings.Split(pathenv, ":") {
		if dir == "" {
			continue
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := dir + "/" + file
		if err := findDir(path); err == nil {
			return path
		}
	}
	return ""
}

func findDir(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if d.Mode().IsDir() {
		return nil
	}
	return os.ErrPermission
}
