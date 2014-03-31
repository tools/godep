package main

import (
	"errors"
	"github.com/kr/fs"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var cmdSave = &Command{
	Usage: "save [-copy=false] [packages]",
	Short: "list and copy dependencies into Godeps",
	Long: `
Save writes a list of the dependencies of the named packages along
with the exact source control revision of each dependency, and copies
their source code into a subdirectory.

The dependency list is a JSON document with the following structure:

	type Godeps struct {
		ImportPath string
		GoVersion  string   // Abridged output of 'go version'.
		Packages   []string // Arguments to godep save, if any.
		Deps       []struct {
			ImportPath string
			Comment    string // Tag or description of commit.
			Rev        string // VCS-specific commit ID.
		}
	}

If -copy=false is given, the list alone is written to file Godeps.

Otherwise, the list is written to Godeps/Godeps.json, and source
code for all dependencies is copied into Godeps/_workspace.

For more about specifying packages, see 'go help packages'.
`,
	Run: runSave,
}

var saveCopy = true

func init() {
	cmdSave.Flag.BoolVar(&saveCopy, "copy", true, "copy source code")
}

func runSave(cmd *Command, args []string) {
	err := save(args)
	if err != nil {
		log.Fatalln(err)
	}
}

func save(pkgs []string) error {
	dot, err := LoadPackages(".")
	if err != nil {
		return err
	}
	ver, err := goVersion()
	if err != nil {
		return err
	}
	g := &Godeps{
		ImportPath: dot[0].ImportPath,
		GoVersion:  ver,
	}
	if len(pkgs) > 0 {
		g.Packages = pkgs
	} else {
		pkgs = []string{"."}
	}
	a, err := LoadPackages(pkgs...)
	if err != nil {
		return err
	}
	err = g.Load(a)
	if err != nil {
		return err
	}
	if a := badSandboxVCS(g.Deps); a != nil && !saveCopy {
		log.Println("Unsupported sandbox VCS:", strings.Join(a, ", "))
		log.Printf("Instead, run: godep save -copy %s", strings.Join(pkgs, " "))
		return errors.New("error")
	}
	if g.Deps == nil {
		g.Deps = make([]Dependency, 0) // produce json [], not null
	}
	manifest := "Godeps"
	if saveCopy {
		manifest = filepath.Join("Godeps", "Godeps.json")
		os.Remove("Godeps") // remove regular file if present; ignore error
		path := filepath.Join("Godeps", "Readme")
		err = writeFile(path, strings.TrimSpace(Readme)+"\n")
		if err != nil {
			log.Println(err)
		}
	}
	f, err := os.Create(manifest)
	if err != nil {
		return err
	}
	_, err = g.WriteTo(f)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	if saveCopy {
		// We use a name starting with "_" so the go tool
		// ignores this directory when traversing packages
		// starting at the project's root. For example,
		//   godep go list ./...
		workspace := filepath.Join("Godeps", "_workspace")
		err = os.RemoveAll(workspace)
		if err != nil {
			return err
		}
		err = copySrc(filepath.Join(workspace, "src"), g)
		if err != nil {
			return err
		}
		writeVCSIgnore(workspace)
	}
	return nil
}

// badSandboxVCS returns a list of VCSes that don't work
// with the `godep go` sandbox code.
func badSandboxVCS(deps []Dependency) (a []string) {
	for _, d := range deps {
		if d.vcs.CreateCmd == "" {
			a = append(a, d.vcs.vcs.Name)
		}
	}
	sort.Strings(a)
	return uniq(a)
}

func copySrc(dir string, g *Godeps) error {
	ok := true
	for _, dep := range g.Deps {
		srcdir := filepath.Join(dep.ws, "src")
		w := fs.Walk(dep.dir)
		for w.Step() {
			err := copyPkgFile(dir, srcdir, w)
			if err != nil {
				log.Println(err)
				ok = false
			}
		}
	}
	if !ok {
		return errors.New("error copying source code")
	}
	return nil
}

func copyPkgFile(dstroot, srcroot string, w *fs.Walker) error {
	if w.Err() != nil {
		return w.Err()
	}
	if c := w.Stat().Name()[0]; c == '.' || c == '_' {
		// Skip directories using a rule similar to how
		// the go tool enumerates packages.
		// See $GOROOT/src/cmd/go/main.go:/matchPackagesInFs
		w.SkipDir()
	}
	if w.Stat().IsDir() {
		return nil
	}
	rel, err := filepath.Rel(srcroot, w.Path())
	if err != nil { // this should never happen
		return err
	}
	return copyFile(filepath.Join(dstroot, rel), w.Path())
}

// copyFile copies a regular file from src to dst.
// dst is opened with os.Create.
func copyFile(dst, src string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0777)
	if err != nil {
		return err
	}

	linkDst, err := os.Readlink(src)
	if err == nil {
		return os.Symlink(linkDst, dst)
	}

	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)
	err1 := w.Close()
	if err == nil {
		err = err1
	}

	return err
}

// Func writeVCSIgnore writes "ignore" files inside dir for known VCSs,
// so that dir/pkg and dir/bin don't accidentally get committed.
// It logs any errors it encounters.
func writeVCSIgnore(dir string) {
	// Currently git is the only VCS for which we know how to do this.
	// Mercurial and Bazaar have similar mechasims, but they apparently
	// require writing files outside of dir.
	const ignore = "/pkg\n/bin\n"
	name := filepath.Join(dir, ".gitignore")
	err := writeFile(name, ignore)
	if err != nil {
		log.Println(err)
	}
}

// writeFile is like ioutil.WriteFile but it creates
// intermediate directories with os.MkdirAll.
func writeFile(name, body string) error {
	err := os.MkdirAll(filepath.Dir(name), 0777)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(name, []byte(body), 0666)
}

const Readme = `
This directory tree is generated automatically by godep.

Please do not edit.

See https://github.com/tools/godep for more information.
`
