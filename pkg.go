package main

import (
	"encoding/json"
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
func LoadPackages(pc packageCache, names ...string) (a []*Package, err error) {
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
		p, ok := pc[info.ImportPath]
		if ok {
			a = append(a, p)
			continue
		} else {
			pc[info.ImportPath] = info
		}
		err = info.addIgnoredGoFilesDeps(pc)
		if err != nil && info.Error.Err == "" {
			info.Error.Err = err.Error()
		}
		pc[info.ImportPath] = info
		a = append(a, info)
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (p *Package) addIgnoredGoFilesDeps(pc packageCache) error {
	if p.Standard {
		return nil
	}

	var buildMatch = "+build "
	var buildFieldSplit = func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	}
	var imports, testImports []string
	for _, fname := range p.IgnoredGoFiles {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, filepath.Join(p.Dir, fname), nil, parser.ParseComments)
		if err != nil {
			return err
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
				return err // can't happen
			}
			if strings.HasSuffix(fname, "_test.go") {
				testImports = append(testImports, name)
			} else {
				imports = append(imports, name)
			}
		}
	}
	if len(imports) > 0 {
		pkgs, err := LoadPackages(pc, imports...)
		if err != nil {
			return err
		}
		for _, p1 := range pkgs {
			p.Deps = append(p.Deps, p1.ImportPath)
			p.Deps = append(p.Deps, p1.Deps...)
		}
		sort.Strings(p.Deps)
		uniq(p.Deps)
	}
	if len(testImports) > 0 {
		pkgs, err := LoadPackages(pc, testImports...)
		if err != nil {
			return err
		}
		for _, p1 := range pkgs {
			p.TestImports = append(p.TestImports, p1.TestImports...)
			p.XTestImports = append(p.TestImports, p1.XTestImports...)
		}
		sort.Strings(p.TestImports)
		uniq(p.TestImports)
		sort.Strings(p.XTestImports)
		uniq(p.XTestImports)
	}
	return nil
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
