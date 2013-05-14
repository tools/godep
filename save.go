package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

var cmdSave = &Command{
	Usage: "save [package]",
	Short: "list current dependencies to a file",
	Long: `
Save writes a list of the dependencies of the named package along with
the exact source control revision of each dependency.

Output is to file Godeps in the package's directory.

If package is omitted, it is taken to be ".".
`,
	Run: runSave,
}

func runSave(cmd *Command, args []string) {
	if len(args) > 1 {
		cmd.UsageExit()
	}
	pkg := "."
	if len(args) == 1 {
		pkg = args[0]
	}
	a, err := getInfo(pkg)
	if err != nil {
		log.Fatal("godep:", err)
	}
	info := a[0]
	if info.Standard {
		log.Fatalln("godep: ignoring stdlib package:", pkg)
	}
	path := filepath.Join(info.Dir, "Godeps")
	f, err := os.Create(path)
	if err != nil {
		log.Fatal("godep:", err)
	}
	_, err = f.Write([]byte(info.ImportPath + "\n"))
	if err != nil {
		log.Fatal("godep:", err)
	}

	deps, err := getInfo(info.Deps...)
	if err != nil {
		log.Fatal("godep:", err)
	}

	goVersion(f)
	prefixes := []string{info.ImportPath + "/"}
	tw := tabwriter.NewWriter(f, 0, 4, 1, ' ', 0)
	for _, dep := range deps {
		name := dep.ImportPath
		if dep.Error.Err != "" {
			log.Println("godep: error:", dep.Error.Err)
			fmt.Fprintf(tw, "err\t%s\t# %q\n", name, dep.Error.Err)
			continue
		}
		if !prefixIn(prefixes, name) && !dep.Standard {
			prefixes = append(prefixes, name+"/")
			id, comment, err := getCommit(dep)
			if err != nil {
				log.Println("godep: error:", err)
				id = "err"
				comment = err.Error()
			}
			if comment == "" {
				fmt.Fprintf(tw, "%s\t%s\n", id, name)
			} else {
				fmt.Fprintf(tw, "%s\t%s\t# %s\n", id, name, comment)
			}
		}
	}
	err = tw.Flush()
	if err != nil {
		log.Fatal("godep:", err)
	}
	err = f.Close()
	if err != nil {
		log.Fatal("godep:", err)
	}
}

func getCommit(pkg *Package) (id, comment string, err error) {
	// 1. get commit name
	// 2. make sure working dir matches last commit
	// 3. see if there's a better description
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = pkg.Dir
	cmd.Stderr = os.Stderr
	b, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	id = strings.TrimSpace(string(b))
	if len(id) > 8 {
		id = id[:8]
	}
	//cmd = exec.Command()
	return id, "", nil
}

func prefixIn(a []string, s string) bool {
	for _, p := range a {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func goVersion(w io.Writer) {
	cmd := exec.Command("go", "version")
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal("godep:", err)
	}
}
