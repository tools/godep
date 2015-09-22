package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
)

var (
	godepsFile    = filepath.Join("Godeps", "Godeps.json")
	oldGodepsFile = filepath.Join("Godeps")
)

// Godeps describes what a package needs to be rebuilt reproducibly.
// It's the same information stored in file Godeps.
type Godeps struct {
	ImportPath string
	GoVersion  string
	Packages   []string `json:",omitempty"` // Arguments to save, if any.
	Deps       []Dependency
	isOldFile  bool
}

func createGodepsFile() (*os.File, error) {
	return os.Create(godepsFile)
}

func loadGodepsFile(path string) (Godeps, error) {
	var g Godeps
	f, err := os.Open(path)
	if err != nil {
		return g, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&g)
	return g, err
}

func loadDefaultGodepsFile() (Godeps, error) {
	var err error
	g, err1 := loadGodepsFile(godepsFile)
	if err1 != nil {
		if os.IsNotExist(err1) {
			g, err = loadGodepsFile(oldGodepsFile)
			if err == nil {
				g.isOldFile = true
			}
		}
	}
	return g, err1
}

// pkgs is the list of packages to read dependencies for
func (g *Godeps) fill(pkgs []*Package, destImportPath string) error {
	var err1 error
	var path, testImports []string
	for _, p := range pkgs {
		if p.Standard {
			log.Println("ignoring stdlib package:", p.ImportPath)
			continue
		}
		if p.Error.Err != "" {
			log.Println(p.Error.Err)
			err1 = errorLoadingPackages
			continue
		}
		path = append(path, p.ImportPath)
		path = append(path, p.Deps...)
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
			err1 = errorLoadingPackages
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
			err1 = errorLoadingDeps
			continue
		}
		if pkg.Standard || containsPathPrefix(seen, pkg.ImportPath) {
			continue
		}
		seen = append(seen, pkg.ImportPath)
		vcs, reporoot, err := VCSFromDir(pkg.Dir, filepath.Join(pkg.Root, "src"))
		if err != nil {
			log.Println(err)
			err1 = errorLoadingDeps
			continue
		}
		id, err := vcs.identify(pkg.Dir)
		if err != nil {
			log.Println(err)
			err1 = errorLoadingDeps
			continue
		}
		if vcs.isDirty(pkg.Dir, id) {
			log.Println("dirty working tree (please commit changes):", pkg.Dir)
			err1 = errorLoadingDeps
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

func (g *Godeps) copy() *Godeps {
	h := *g
	h.Deps = make([]Dependency, len(g.Deps))
	copy(h.Deps, g.Deps)
	return &h
}

func (g *Godeps) file() string {
	if g.isOldFile {
		return oldGodepsFile
	}
	return godepsFile
}

func (g *Godeps) save() (int64, error) {
	f, err := os.Create(g.file())
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return g.writeTo(f)
}

func (g *Godeps) writeTo(w io.Writer) (int64, error) {
	b, err := json.MarshalIndent(g, "", "\t")
	if err != nil {
		return 0, err
	}
	n, err := w.Write(append(b, '\n'))
	return int64(n), err
}
