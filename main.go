package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

func main() {
	if len(os.Args) < 2 {
		help(1)
	}
	switch cmd := os.Args[1]; cmd {
	case "save":
		save(os.Args[2:])
	case "help":
		help(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q.\n", cmd)
		fmt.Fprintln(os.Stderr, "Run `godep help` for usage.")
		os.Exit(1)
	}
}

func save(args []string) {
	pkg := "."
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "Usage: godep save [package]")
		fmt.Fprintln(os.Stderr, "Run `godep help` for usage.")
		os.Exit(1)
	}
	if len(args) == 1 {
		pkg = args[0]
	}
	a, err := getInfo(pkg)
	if err != nil {
		log.Fatal("godep:", err)
	}
	info := a[0]
	path := filepath.Join(info.Dir, "Godeps")
	f, err := os.Create(path)
	if err != nil {
		log.Fatal("godep:", err)
	}
	_, err = f.Write([]byte(info.ImportPath + "\n"))
	if err != nil {
		log.Fatal("godep:", err)
	}

	info.Deps = append(info.Deps, "foo")
	deps, err := getInfo(info.Deps...)
	if err != nil {
		log.Fatal("godep:", err)
	}

	cmd := exec.Command("go", "version")
	cmd.Stdout = f
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatal("godep:", err)
	}
	var prefixes []string
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

func prefixIn(a []string, s string) bool {
	for _, p := range a {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
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

var helpMessage = `
Usage: godep save
`

func help(code int) {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(helpMessage))
	os.Exit(code)
}
