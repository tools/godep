package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Package represents a Go package.
type Package struct {
	Dir        string
	Root       string
	ImportPath string
	Deps       []string
	Standard   bool
	Processed  bool

	GoFiles        []string
	CgoFiles       []string
	IgnoredGoFiles []string

	TestGoFiles  []string
	TestImports  []string
	XTestGoFiles []string
	XTestImports []string

	Error struct {
		Err string
	}
}

type packageCache map[string]*Package

// LoadPackages loads the named packages using go list -json.
// Unlike the go tool, an empty argument list is treated as an empty list; "."
// must be given explicitly if desired.
// IgnoredGoFiles will be processed and their dependencies resolved recursively
// Files with a build tag of `ignore` are skipped. Files with other build tags
// are however processed.
func LoadPackages(names ...string) (a []*Package, err error) {
	if len(names) == 0 {
		return nil, nil
	}
	args := []string{"list", "-e", "-json"}
	cmd := exec.Command("go", append(args, names...)...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	d := json.NewDecoder(r)
	for {
		info := new(Package)
		err = d.Decode(info)
		if err == io.EOF {
			break
		}
		if err != nil {
			info.Error.Err = err.Error()
		}
		a = append(a, info)
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	return a, nil
}

// resolveIgnoredGoFiles for the given pkgs, recursively
func resolveIgnoredGoFiles(pkg *Package, pc map[string]*Package) error {
	fmt.Println("resolveIgnoredGoFiles:", pkg.ImportPath)
	var allDeps []string
	allDeps = append(allDeps, pkg.ImportPath)
	allDeps = append(allDeps, pkg.Deps...)
	allDeps = append(allDeps, pkg.TestImports...)
	allDeps = append(allDeps, pkg.XTestImports...)
	allDeps = uniq(allDeps)
	spkgs, err := LoadPackages(allDeps...)
	if err != nil {
		return err
	}
	for _, sp := range spkgs {
		if pc[sp.ImportPath] != nil {
			continue
		}
		if len(sp.IgnoredGoFiles) > 0 {
			pc[sp.ImportPath] = sp
			ni, nti, err := sp.ignoredGoFilesDeps()
			fmt.Println("sp", sp.ImportPath)
			fmt.Println("ni", ni)
			fmt.Println("nti", nti)
			if err != nil {
				panic(err)
			}
			pkg.Deps = append(pkg.Deps, ni...)
			pkg.TestImports = append(pkg.TestImports, nti...)
			if len(ni) > 0 || len(nti) > 0 {
				err := resolveIgnoredGoFiles(sp, pc)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (p *Package) ignoredGoFilesDeps() ([]string, []string, error) {
	if p.Standard {
		return nil, nil, nil
	}

	var buildMatch = "+build "
	var buildFieldSplit = func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	}
	var imports, testImports []string
	for _, fname := range p.IgnoredGoFiles {
		tgt := filepath.Join(p.Dir, fname)
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, tgt, nil, parser.ParseComments)
		if err != nil {
			return nil, nil, err
		}
		if len(f.Comments) > 0 {
			for _, c := range f.Comments {
				ct := c.Text()
				if i := strings.Index(ct, buildMatch); i != -1 {
					for _, b := range strings.FieldsFunc(ct[i+len(buildMatch):], buildFieldSplit) {
						if b == "ignore" {
							continue
						}
					}
				}
			}
		}
		for _, is := range f.Imports {
			name, err := strconv.Unquote(is.Path.Value)
			if err != nil {
				return nil, nil, err // can't happen
			}
			if strings.HasSuffix(fname, "_test.go") {
				if !hasString(p.TestImports, name) {
					testImports = append(testImports, name)
				}
			} else {
				if !hasString(p.Deps, name) {
					fmt.Println("p.Deps(", p.ImportPath, ")", p.Deps)
					imports = append(imports, name)
				}
			}
		}
	}
	p.Deps = uniq(append(p.Deps, imports...))
	p.TestImports = uniq(append(p.TestImports, testImports...))
	return imports, testImports, nil
}

func hasString(search []string, s string) bool {
	sort.Strings(search)
	i := sort.SearchStrings(search, s)
	return i < len(search) && search[i] == s
}

func (p *Package) allGoFiles() []string {
	var a []string
	a = append(a, p.GoFiles...)
	a = append(a, p.CgoFiles...)
	a = append(a, p.TestGoFiles...)
	a = append(a, p.XTestGoFiles...)
	a = append(a, p.IgnoredGoFiles...)
	return a
}
