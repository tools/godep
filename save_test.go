package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"text/template"
)

// node represents a file tree or a VCS repo
type node struct {
	path    string      // file name or commit type
	body    interface{} // file contents or commit tag
	entries []*node     // nil if the entry is a file
}

var (
	pkgtpl = template.Must(template.New("package").Parse(`package {{.Name}}

import (
{{range .Imports}}	{{printf "%q" .}}
{{end}})
`))
)

func pkg(name string, pkg ...string) string {
	v := struct {
		Name    string
		Imports []string
	}{name, pkg}
	var buf bytes.Buffer
	err := pkgtpl.Execute(&buf, v)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func decl(name string) string {
	return "var " + name + " int\n"
}

func godeps(importpath string, keyval ...string) *Godeps {
	g := &Godeps{
		ImportPath: importpath,
	}
	for i := 0; i < len(keyval); i += 2 {
		g.Deps = append(g.Deps, Dependency{
			ImportPath: keyval[i],
			Comment:    keyval[i+1],
		})
	}
	return g
}

func TestSave(t *testing.T) {
	var cases = []struct {
		cwd      string
		args     []string
		flagR    bool
		start    []*node
		altstart []*node
		want     []*node
		wdep     Godeps
		werr     bool
	}{
		{ // simple case, one dependency
			cwd: "C",
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
		},
		{ // strip import comment
			cwd: "C",
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", `package D // import "D"`, nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D"), nil},
				{"C/Godeps/_workspace/src/D/main.go", "package D\n", nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
		},
		{
			// dependency in same repo with existing manifest
			// see bug https://github.com/tools/godep/issues/69
			cwd:  "P",
			args: []string{"./..."},
			start: []*node{
				{
					"P",
					"",
					[]*node{
						{"main.go", pkg("P", "P/Q"), nil},
						{"Q/main.go", pkg("Q"), nil},
						{"Godeps/Godeps.json", `{}`, nil},
						{"+git", "C1", nil},
					},
				},
			},
			want: []*node{
				{"P/main.go", pkg("P", "P/Q"), nil},
				{"P/Q/main.go", pkg("Q"), nil},
			},
			wdep: Godeps{
				ImportPath: "P",
				Deps:       []Dependency{},
			},
		},
		{
			// dependency on parent directory in same repo
			// see bug https://github.com/tools/godep/issues/70
			cwd:  "P",
			args: []string{"./..."},
			start: []*node{
				{
					"P",
					"",
					[]*node{
						{"main.go", pkg("P"), nil},
						{"Q/main.go", pkg("Q", "P"), nil},
						{"+git", "C1", nil},
					},
				},
			},
			want: []*node{
				{"P/main.go", pkg("P"), nil},
				{"P/Q/main.go", pkg("Q", "P"), nil},
			},
			wdep: Godeps{
				ImportPath: "P",
				Deps:       []Dependency{},
			},
		},
		{ // transitive dependency
			cwd: "C",
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D", "T"), nil},
						{"+git", "D1", nil},
					},
				},
				{
					"T",
					"",
					[]*node{
						{"main.go", pkg("T"), nil},
						{"+git", "T1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D", "T"), nil},
				{"C/Godeps/_workspace/src/T/main.go", pkg("T"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
					{ImportPath: "T", Comment: "T1"},
				},
			},
		},
		{ // two packages, one in a subdirectory
			cwd: "C",
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D", "D/P"), nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D"), nil},
						{"P/main.go", pkg("P"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D", "D/P"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D"), nil},
				{"C/Godeps/_workspace/src/D/P/main.go", pkg("P"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
		},
		{ // repo root is not a package (no go files)
			cwd: "C",
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/P", "D/Q"), nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"P/main.go", pkg("P"), nil},
						{"Q/main.go", pkg("Q"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D/P", "D/Q"), nil},
				{"C/Godeps/_workspace/src/D/P/main.go", pkg("P"), nil},
				{"C/Godeps/_workspace/src/D/Q/main.go", pkg("Q"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/P", Comment: "D1"},
					{ImportPath: "D/Q", Comment: "D1"},
				},
			},
		},
		{ // symlink
			cwd: "C",
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.x", pkg("main", "D"), nil},
						{"main.go", "symlink:main.x", nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
		},
		{ // add one dependency; keep other dependency version
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D") + decl("D1"), nil},
						{"+git", "D1", nil},
						{"main.go", pkg("D") + decl("D2"), nil},
						{"+git", "D2", nil},
					},
				},
				{
					"E",
					"",
					[]*node{
						{"main.go", pkg("E"), nil},
						{"+git", "E1", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D", "E"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D", "E"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
				{"C/Godeps/_workspace/src/E/main.go", pkg("E"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
					{ImportPath: "E", Comment: "E1"},
				},
			},
		},
		{ // remove one dependency; keep other dependency version
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D") + decl("D1"), nil},
						{"+git", "D1", nil},
						{"main.go", pkg("D") + decl("D2"), nil},
						{"+git", "D2", nil},
					},
				},
				{
					"E",
					"",
					[]*node{
						{"main.go", pkg("E") + decl("E1"), nil},
						{"+git", "E1", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1", "E", "E1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"Godeps/_workspace/src/E/main.go", pkg("E") + decl("E1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
				{"C/Godeps/_workspace/src/E/main.go", "(absent)", nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
		},
		{ // add one dependency from same repo
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"A/main.go", pkg("A") + decl("A1"), nil},
						{"B/main.go", pkg("B") + decl("B1"), nil},
						{"+git", "D1", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/A", "D/B"), nil},
						{"Godeps/Godeps.json", godeps("C", "D/A", "D1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
				{"C/Godeps/_workspace/src/D/B/main.go", pkg("B") + decl("B1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/A", Comment: "D1"},
					{ImportPath: "D/B", Comment: "D1"},
				},
			},
		},
		{ // add one dependency from same repo, require same version
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"A/main.go", pkg("A") + decl("A1"), nil},
						{"B/main.go", pkg("B") + decl("B1"), nil},
						{"+git", "D1", nil},
						{"A/main.go", pkg("A") + decl("A2"), nil},
						{"B/main.go", pkg("B") + decl("B2"), nil},
						{"+git", "D2", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/A", "D/B"), nil},
						{"Godeps/Godeps.json", godeps("C", "D/A", "D1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/A", Comment: "D1"},
				},
			},
			werr: true,
		},
		{ // replace dependency from same repo parent dir
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D") + decl("D1"), nil},
						{"A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "D1", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"Godeps/Godeps.json", godeps("C", "D/A", "D1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
		},
		{ // replace dependency from same repo parent dir, require same version
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D") + decl("D1"), nil},
						{"A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "D1", nil},
						{"main.go", pkg("D") + decl("D2"), nil},
						{"A/main.go", pkg("A") + decl("A2"), nil},
						{"+git", "D2", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"Godeps/Godeps.json", godeps("C", "D/A", "D1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/A", Comment: "D1"},
				},
			},
			werr: true,
		},
		{ // replace dependency from same repo child dir
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D") + decl("D1"), nil},
						{"A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "D1", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/A"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", "(absent)", nil},
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/A", Comment: "D1"},
				},
			},
		},
		{ // replace dependency from same repo child dir, require same version
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D") + decl("D1"), nil},
						{"A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "D1", nil},
						{"main.go", pkg("D") + decl("D2"), nil},
						{"A/main.go", pkg("A") + decl("A2"), nil},
						{"+git", "D2", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/A"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
			werr: true,
		},
		{ // Bug https://github.com/tools/godep/issues/85
			cwd: "C",
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"A/main.go", pkg("A") + decl("A1"), nil},
						{"B/main.go", pkg("B") + decl("B1"), nil},
						{"+git", "D1", nil},
						{"A/main.go", pkg("A") + decl("A2"), nil},
						{"B/main.go", pkg("B") + decl("B2"), nil},
						{"+git", "D2", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/A", "D/B"), nil},
						{"Godeps/Godeps.json", godeps("C", "D/A", "D1", "D/B", "D1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
						{"Godeps/_workspace/src/D/B/main.go", pkg("B") + decl("B1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("A1"), nil},
				{"C/Godeps/_workspace/src/D/B/main.go", pkg("B") + decl("B1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/A", Comment: "D1"},
					{ImportPath: "D/B", Comment: "D1"},
				},
			},
		},
		{ // intermediate dependency that uses godep save -r, main -r=false
			cwd: "C",
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"+git", "", nil},
					},
				},
				{
					"T",
					"",
					[]*node{
						{"main.go", pkg("T"), nil},
						{"+git", "T1", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D", "D/Godeps/_workspace/src/T"), nil},
						{"Godeps/_workspace/src/T/main.go", pkg("T"), nil},
						{"Godeps/Godeps.json", godeps("D", "T", "T1"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D", "T"), nil},
				{"C/Godeps/_workspace/src/T/main.go", pkg("T"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
					{ImportPath: "T", Comment: "T1"},
				},
			},
		},
		{ // intermediate dependency that uses godep save -r, main -r too
			cwd:   "C",
			flagR: true,
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"+git", "", nil},
					},
				},
				{
					"T",
					"",
					[]*node{
						{"main.go", pkg("T"), nil},
						{"+git", "T1", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D", "D/Godeps/_workspace/src/T"), nil},
						{"Godeps/_workspace/src/T/main.go", pkg("T"), nil},
						{"Godeps/Godeps.json", godeps("D", "T", "T1"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "C/Godeps/_workspace/src/D"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D", "C/Godeps/_workspace/src/T"), nil},
				{"C/Godeps/_workspace/src/T/main.go", pkg("T"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
					{ImportPath: "T", Comment: "T1"},
				},
			},
		},
		{ // rewrite files under build constraints
			cwd:   "C",
			flagR: true,
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"x.go", "// +build x\n\n" + pkg("main", "D"), nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "C/Godeps/_workspace/src/D"), nil},
				{"C/x.go", "// +build x\n\n" + pkg("main", "C/Godeps/_workspace/src/D"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
		},
		{ // exclude dependency subdirectories even when obtained by a rewritten import path
			cwd: "C",
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D", "T"), nil},
						{"+git", "", nil},
					},
				},
				{
					"T",
					"",
					[]*node{
						{"main.go", pkg("T"), nil},
						{"X/main.go", pkg("X"), nil},
						{"+git", "T1", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D", "D/Godeps/_workspace/src/T/X"), nil},
						{"Godeps/_workspace/src/T/X/main.go", pkg("X"), nil},
						{"Godeps/Godeps.json", godeps("D", "T/X", "T1"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "D", "T"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D", "T/X"), nil},
				{"C/Godeps/_workspace/src/T/main.go", pkg("T"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
					{ImportPath: "T", Comment: "T1"},
				},
			},
		},
		{ // find transitive dependencies across roots
			cwd:   "C",
			flagR: true,
			altstart: []*node{
				{
					"T",
					"",
					[]*node{
						{"main.go", pkg("T"), nil},
						{"+git", "T1", nil},
					},
				},
			},
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"+git", "", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D", "D/Godeps/_workspace/src/T"), nil},
						{"Godeps/_workspace/src/T/main.go", pkg("T"), nil},
						{"Godeps/Godeps.json", godeps("D", "T", "T1"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "C/Godeps/_workspace/src/D"), nil},
				{"C/Godeps/_workspace/src/D/main.go", pkg("D", "C/Godeps/_workspace/src/T"), nil},
				{"C/Godeps/_workspace/src/T/main.go", pkg("T"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
					{ImportPath: "T", Comment: "T1"},
				},
			},
		},
		{ // pull in minimal dependencies, see https://github.com/tools/godep/issues/93
			cwd:   "C",
			flagR: true,
			start: []*node{
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/X"), nil},
						{"+git", "", nil},
					},
				},
				{
					"T",
					"",
					[]*node{
						{"main.go", pkg("T"), nil},
						{"+git", "T1", nil},
					},
				},
				{
					"D",
					"",
					[]*node{
						{"main.go", pkg("D", "D/Godeps/_workspace/src/T"), nil},
						{"X/main.go", pkg("X"), nil},
						{"Godeps/_workspace/src/T/main.go", pkg("T"), nil},
						{"Godeps/Godeps.json", godeps("D", "T", "T1"), nil},
						{"+git", "D1", nil},
					},
				},
			},
			want: []*node{
				{"C/main.go", pkg("main", "C/Godeps/_workspace/src/D/X"), nil},
				{"C/Godeps/_workspace/src/D/X/main.go", pkg("X"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/X", Comment: "D1"},
				},
			},
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	const scratch = "godeptest"
	defer os.RemoveAll(scratch)
	for _, test := range cases {
		err = os.RemoveAll(scratch)
		if err != nil {
			t.Fatal(err)
		}
		altsrc := filepath.Join(scratch, "r2", "src")
		if test.altstart != nil {
			makeTree(t, &node{altsrc, "", test.altstart}, "")
		}
		src := filepath.Join(scratch, "r1", "src")
		makeTree(t, &node{src, "", test.start}, altsrc)

		dir := filepath.Join(wd, src, test.cwd)
		err = os.Chdir(dir)
		if err != nil {
			panic(err)
		}
		root1 := filepath.Join(wd, scratch, "r1")
		root2 := filepath.Join(wd, scratch, "r2")
		err = os.Setenv("GOPATH", root1+string(os.PathListSeparator)+root2)
		if err != nil {
			panic(err)
		}
		saveR = test.flagR
		err = save(test.args)
		if g := err != nil; g != test.werr {
			if err != nil {
				t.Log(err)
			}
			t.Errorf("save err = %v want %v", g, test.werr)
		}
		err = os.Chdir(wd)
		if err != nil {
			panic(err)
		}

		checkTree(t, &node{src, "", test.want})

		f, err := os.Open(filepath.Join(dir, "Godeps/Godeps.json"))
		if err != nil {
			t.Error(err)
		}
		g := new(Godeps)
		err = json.NewDecoder(f).Decode(g)
		if err != nil {
			t.Error(err)
		}
		f.Close()

		if g.ImportPath != test.wdep.ImportPath {
			t.Errorf("ImportPath = %s want %s", g.ImportPath, test.wdep.ImportPath)
		}
		for i := range g.Deps {
			g.Deps[i].Rev = ""
		}
		if !reflect.DeepEqual(g.Deps, test.wdep.Deps) {
			t.Errorf("Deps = %v want %v", g.Deps, test.wdep.Deps)
		}
	}
}

func makeTree(t *testing.T, tree *node, altpath string) (gopath string) {
	walkTree(tree, tree.path, func(path string, n *node) {
		g, isGodeps := n.body.(*Godeps)
		body, _ := n.body.(string)
		switch {
		case isGodeps:
			for i, dep := range g.Deps {
				rel := filepath.FromSlash(dep.ImportPath)
				dir := filepath.Join(tree.path, rel)
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					dir = filepath.Join(altpath, rel)
				}
				tag := dep.Comment
				rev := strings.TrimSpace(run(t, dir, "git", "rev-parse", tag))
				g.Deps[i].Rev = rev
			}
			os.MkdirAll(filepath.Dir(path), 0770)
			f, err := os.Create(path)
			if err != nil {
				t.Errorf("makeTree: %v", err)
				return
			}
			defer f.Close()
			err = json.NewEncoder(f).Encode(g)
			if err != nil {
				t.Errorf("makeTree: %v", err)
			}
		case n.path == "+git":
			dir := filepath.Dir(path)
			run(t, dir, "git", "init") // repo might already exist, but ok
			run(t, dir, "git", "add", ".")
			run(t, dir, "git", "commit", "-m", "godep")
			if body != "" {
				run(t, dir, "git", "tag", body)
			}
		case n.entries == nil && strings.HasPrefix(body, "symlink:"):
			target := strings.TrimPrefix(body, "symlink:")
			os.Symlink(target, path)
		case n.entries == nil && body == "(absent)":
			panic("is this gonna be forever")
		case n.entries == nil:
			os.MkdirAll(filepath.Dir(path), 0770)
			err := ioutil.WriteFile(path, []byte(body), 0660)
			if err != nil {
				t.Errorf("makeTree: %v", err)
			}
		default:
			os.MkdirAll(path, 0770)
		}
	})
	return gopath
}

func checkTree(t *testing.T, want *node) {
	walkTree(want, want.path, func(path string, n *node) {
		body := n.body.(string)
		switch {
		case n.path == "+git":
			panic("is this real life")
		case n.entries == nil && strings.HasPrefix(body, "symlink:"):
			panic("why is this happening to me")
		case n.entries == nil && body == "(absent)":
			body, err := ioutil.ReadFile(path)
			if !os.IsNotExist(err) {
				t.Errorf("checkTree: %s = %s want absent", path, string(body))
				return
			}
		case n.entries == nil:
			gbody, err := ioutil.ReadFile(path)
			if err != nil {
				t.Errorf("checkTree: %v", err)
				return
			}
			if got := string(gbody); got != body {
				t.Errorf("%s = %s want %s", path, got, body)
			}
		default:
			os.MkdirAll(path, 0770)
		}
	})
}

func walkTree(n *node, path string, f func(path string, n *node)) {
	f(path, n)
	for _, e := range n.entries {
		walkTree(e, filepath.Join(path, filepath.FromSlash(e.path)), f)
	}
}

func run(t *testing.T, dir, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		panic(name + " " + strings.Join(args, " ") + ": " + err.Error())
	}
	return string(out)
}

func TestStripImportComment(t *testing.T) {
	var cases = []struct{ s, w string }{
		{`package foo`, `package foo`},
		{`anything else`, `anything else`},
		{`package foo // import "bar/foo"`, `package foo`},
		{`package foo /* import "bar/foo" */`, `package foo`},
		{`package  foo  //  import  "bar/foo" `, `package  foo`},
		{"package foo // import `bar/foo`", `package foo`},
		{`package foo /* import "bar/foo" */; var x int`, `package foo; var x int`},
		{`package foo // import "bar/foo" garbage`, `package foo // import "bar/foo" garbage`},
		{`package xpackage foo // import "bar/foo"`, `package xpackage foo // import "bar/foo"`},
	}

	for _, test := range cases {
		g := string(stripImportComment([]byte(test.s)))
		if g != test.w {
			t.Errorf("stripImportComment(%q) = %q want %q", test.s, g, test.w)
		}
	}
}
