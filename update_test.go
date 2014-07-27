package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpdate(t *testing.T) {
	var cases = []struct {
		cwd   string
		args  []string
		start []*node
		want  []*node
		wdep  Godeps
		werr  bool
	}{
		{ // simple case, update one dependency
			cwd:  "C",
			args: []string{"D"},
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
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D2"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D2"},
				},
			},
		},
		{ // update one dependency, keep other one
			cwd:  "C",
			args: []string{"D"},
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
						{"main.go", pkg("E") + decl("E2"), nil},
						{"+git", "E2", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D", "E"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1", "E", "E1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"Godeps/_workspace/src/E/main.go", pkg("E") + decl("E1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D2"), nil},
				{"C/Godeps/_workspace/src/E/main.go", pkg("E") + decl("E1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D2"},
					{ImportPath: "E", Comment: "E1"},
				},
			},
		},
		{ // update all dependencies
			cwd:  "C",
			args: []string{"..."},
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
						{"main.go", pkg("E") + decl("E2"), nil},
						{"+git", "E2", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D", "E"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1", "E", "E1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"Godeps/_workspace/src/E/main.go", pkg("E") + decl("E1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D2"), nil},
				{"C/Godeps/_workspace/src/E/main.go", pkg("E") + decl("E2"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D2"},
					{ImportPath: "E", Comment: "E2"},
				},
			},
		},
		{ // one match of two patterns
			cwd:  "C",
			args: []string{"D", "X"},
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
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D2"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D2"},
				},
			},
		},
		{ // no matches
			cwd:  "C",
			args: []string{"X"},
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
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D"), nil},
						{"Godeps/Godeps.json", godeps("C", "D", "D1"), nil},
						{"Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/main.go", pkg("D") + decl("D1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D", Comment: "D1"},
				},
			},
			werr: true,
		},
		{ // update just one package of two in a repo skips it
			cwd:  "C",
			args: []string{"D/A", "E"},
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"A/main.go", pkg("A") + decl("D1"), nil},
						{"B/main.go", pkg("B") + decl("D1"), nil},
						{"+git", "D1", nil},
						{"A/main.go", pkg("A") + decl("D2"), nil},
						{"B/main.go", pkg("B") + decl("D2"), nil},
						{"+git", "D2", nil},
					},
				},
				{
					"E",
					"",
					[]*node{
						{"main.go", pkg("E") + decl("E1"), nil},
						{"+git", "E1", nil},
						{"main.go", pkg("E") + decl("E2"), nil},
						{"+git", "E2", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/A", "D/B", "E"), nil},
						{"Godeps/Godeps.json", godeps("C", "D/A", "D1", "D/B", "D1", "E", "E1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("D1"), nil},
						{"Godeps/_workspace/src/D/B/main.go", pkg("B") + decl("D1"), nil},
						{"Godeps/_workspace/src/E/main.go", pkg("E") + decl("E1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("D1"), nil},
				{"C/Godeps/_workspace/src/D/B/main.go", pkg("B") + decl("D1"), nil},
				{"C/Godeps/_workspace/src/E/main.go", pkg("E") + decl("E2"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/A", Comment: "D1"},
					{ImportPath: "D/B", Comment: "D1"},
					{ImportPath: "E", Comment: "E2"},
				},
			},
		},
		{ // update just one package of two in a repo, none left
			cwd:  "C",
			args: []string{"D/A"},
			start: []*node{
				{
					"D",
					"",
					[]*node{
						{"A/main.go", pkg("A") + decl("D1"), nil},
						{"B/main.go", pkg("B") + decl("D1"), nil},
						{"+git", "D1", nil},
						{"A/main.go", pkg("A") + decl("D2"), nil},
						{"B/main.go", pkg("B") + decl("D2"), nil},
						{"+git", "D2", nil},
					},
				},
				{
					"C",
					"",
					[]*node{
						{"main.go", pkg("main", "D/A", "D/B"), nil},
						{"Godeps/Godeps.json", godeps("C", "D/A", "D1", "D/B", "D1"), nil},
						{"Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("D1"), nil},
						{"Godeps/_workspace/src/D/B/main.go", pkg("B") + decl("D1"), nil},
						{"+git", "", nil},
					},
				},
			},
			want: []*node{
				{"C/Godeps/_workspace/src/D/A/main.go", pkg("A") + decl("D1"), nil},
				{"C/Godeps/_workspace/src/D/B/main.go", pkg("B") + decl("D1"), nil},
			},
			wdep: Godeps{
				ImportPath: "C",
				Deps: []Dependency{
					{ImportPath: "D/A", Comment: "D1"},
					{ImportPath: "D/B", Comment: "D1"},
				},
			},
			werr: true,
		},
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	const gopath = "godeptest"
	defer os.RemoveAll(gopath)
	for _, test := range cases {
		err = os.RemoveAll(gopath)
		if err != nil {
			t.Fatal(err)
		}
		src := filepath.Join(gopath, "src")
		makeTree(t, &node{src, "", test.start}, "")

		dir := filepath.Join(wd, src, test.cwd)
		err = os.Chdir(dir)
		if err != nil {
			panic(err)
		}
		err = os.Setenv("GOPATH", filepath.Join(wd, gopath))
		if err != nil {
			panic(err)
		}
		log.SetOutput(ioutil.Discard)
		err = update(test.args)
		log.SetOutput(os.Stderr)
		if g := err != nil; g != test.werr {
			t.Errorf("update err = %v (%v) want %v", g, err, test.werr)
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
