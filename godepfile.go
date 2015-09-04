package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
)

var godepsFile = filepath.Join("Godeps", "Godeps.json")

// Godeps describes what a package needs to be rebuilt reproducibly.
// It's the same information stored in file Godeps.
type Godeps struct {
	ImportPath string
	GoVersion  string
	Packages   []string `json:",omitempty"` // Arguments to save, if any.
	Deps       []Dependency
	isOldFile  bool
}

func readGodeps() (Godeps, error) {
	var godeps Godeps
	f, err := os.Open(godepsFile)
	if err != nil {
		f, err = os.Open("Godeps")
		if os.IsNotExist(err) {
			return godeps, nil
		}
		if err == nil {
			godeps.isOldFile = true
		}
	}
	if err != nil {
		return godeps, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&godeps)
	return godeps, err
}

// pkgs is the list of packages to read dependencies
func (g *Godeps) Load(pkgs []*Package, destImportPath string) error {
	var err1 error
	var path []string
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
		path = append(path, p.ImportPath)
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
	seen := []string{destImportPath}
	for _, pkg := range ps {
		if pkg.Error.Err != "" {
			log.Println(pkg.Error.Err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		if pkg.Standard || containsPathPrefix(seen, pkg.ImportPath) {
			continue
		}
		seen = append(seen, pkg.ImportPath)
		vcs, reporoot, err := VCSFromDir(pkg.Dir, filepath.Join(pkg.Root, "src"))
		if err != nil {
			log.Println(err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		id, err := vcs.identify(pkg.Dir)
		if err != nil {
			log.Println(err)
			err1 = errors.New("error loading dependencies")
			continue
		}
		if vcs.isDirty(pkg.Dir, id) {
			log.Println("dirty working tree (please commit changes):", pkg.Dir)
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
