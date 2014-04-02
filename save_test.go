package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"text/template"
)

// node represents a file tree or a VCS repo
type node struct {
	path    string  // file name or commit type
	body    string  // file contents or commit tag
	entries []*node // nil if the entry is a file
}

var (
	pkgtpl = template.Must(template.New("package").Parse(`
package {{.Name}}
import ({{range .Imports}}{{printf "%q" .}}; {{end}})
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

func TestSave(t *testing.T) {
	var cases = []struct {
		cwd   string
		args  []string
		start []*node
		want  []*node
		wdep  Godeps
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
		makeTree(t, &node{src, "", test.start})

		dir := filepath.Join(wd, src, test.cwd)
		err = os.Chdir(dir)
		if err != nil {
			panic(err)
		}
		err = os.Setenv("GOPATH", filepath.Join(wd, gopath))
		if err != nil {
			panic(err)
		}
		err = save(test.args)
		if err != nil {
			t.Error("save:", err)
			continue
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

func makeTree(t *testing.T, tree *node) (gopath string) {
	walkTree(tree, tree.path, func(path string, n *node) {
		switch {
		case n.path == "+git":
			dir := filepath.Dir(path)
			run(t, dir, "git", "init") // repo might already exist, but ok
			run(t, dir, "git", "add", ".")
			run(t, dir, "git", "commit", "-m", "godep")
			if n.body != "" {
				run(t, dir, "git", "tag", n.body)
			}
		case n.entries == nil:
			os.MkdirAll(filepath.Dir(path), 0770)
			err := ioutil.WriteFile(path, []byte(n.body), 0660)
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
		switch {
		case n.path == "+git":
			panic("is this real life")
		case n.entries == nil:
			body, err := ioutil.ReadFile(path)
			if err != nil {
				t.Errorf("checkTree: %v", err)
				return
			}
			got := string(body)
			if got != n.body {
				t.Errorf("%s = %s want %s", path, got, n.body)
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

func run(t *testing.T, dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}
