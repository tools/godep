package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"

	"github.com/kr/fs"
)

// rewrite visits the go files in pkgs, plus all go files
// in the directory tree Godeps, rewriting import statments
// according to the rules for func qualify.
func rewrite(pkgs []*Package, qual string, paths []string) error {
	for _, path := range pkgFiles(pkgs) {
		err := rewriteTree(path, qual, paths)
		if err != nil {
			return err
		}
	}
	return rewriteTree("Godeps", qual, paths)
}

// pkgFiles returns the full filesystem path to all go files in pkgs.
func pkgFiles(pkgs []*Package) []string {
	var a []string
	for _, pkg := range pkgs {
		for _, s := range pkg.allGoFiles() {
			a = append(a, filepath.Join(pkg.Dir, s))
		}
	}
	return a
}

// rewriteTree recursively visits the go files in path, rewriting
// import statments according to the rules for func qualify.
func rewriteTree(path, qual string, paths []string) error {
	w := fs.Walk(path)
	for w.Step() {
		if w.Err() != nil {
			log.Println("rewrite:", w.Err())
			continue
		}
		if !w.Stat().IsDir() && strings.HasSuffix(w.Path(), ".go") {
			err := rewriteGoFile(w.Path(), qual, paths)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// rewriteGoFile rewrites import statments in the named file
// according to the rules for func qualify.
func rewriteGoFile(name, qual string, paths []string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	var changed bool
	for _, s := range f.Imports {
		name, err := strconv.Unquote(s.Path.Value)
		if err != nil {
			return err // can't happen
		}
		q := qualify(unqualify(name), qual, paths)
		if q != name {
			s.Path.Value = strconv.Quote(q)
			changed = true
		}
	}
	if !changed {
		return nil
	}

	ast.SortImports(fset, f)
	wpath := name + ".temp"
	w, err := os.Create(wpath)
	if err != nil {
		return err
	}
	err = (&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(w, fset, f)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return os.Rename(wpath, name)
}

// sep is the signature set of path elements that
// precede the original path of an imported package
// in a rewritten import path.
const sep = "/Godeps/_workspace/src/"

// unqualify returns the part of importPath after the last
// occurrence of the signature path elements
// (Godeps/_workspace/src) that always precede imported
// packages in rewritten import paths.
//
// For example,
//   unqualify(C)                         = C
//   unqualify(D/Godeps/_workspace/src/C) = C
func unqualify(importPath string) string {
	if i := strings.LastIndex(importPath, sep); i != -1 {
		importPath = importPath[i+len(sep):]
	}
	return importPath
}

// qualify qualifies importPath with its corresponding import
// path in the Godeps src copy of package pkg. If importPath
// is a directory lexically contained in a path in paths,
// it will be qualified with package pkg; otherwise, it will
// be returned unchanged.
//
// For example, given paths {D, T} and pkg C,
//   importPath  returns
//   C           C
//   fmt         fmt
//   D           C/Godeps/_workspace/src/D
//   D/P         C/Godeps/_workspace/src/D/P
//   T           C/Godeps/_workspace/src/T
func qualify(importPath, pkg string, paths []string) string {
	if containsPathPrefix(paths, importPath) {
		return pkg + sep + importPath
	}
	return importPath
}
