package main

import (
	"log"
	"os"
	"path/filepath"
)

var cmdSave = &Command{
	Usage: "save [packages]",
	Short: "list current dependencies to a file",
	Long: `
Save writes a list of the dependencies of the named packages along
with the exact source control revision of each dependency.

Output is to file Godeps in the first package's directory.
For more about specifying packages, see 'go help packages'.
`,
	Run: runSave,
}

func runSave(cmd *Command, args []string) {
	a, err := LoadPackages(args)
	if err != nil {
		log.Fatalln(err)
	}
	p := a[0]
	if p.Standard {
		log.Fatalln("ignoring stdlib package:", p.ImportPath)
	}

	g, err := LoadGodeps(a)
	if err != nil {
		log.Fatalln(err)
	}
	if g.Deps == nil {
		g.Deps = make([]Dependency, 0) // produce json [], not null
	}
	path := filepath.Join(p.Dir, "Godeps")
	f, err := os.Create(path)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = g.WriteTo(f)
	if err != nil {
		log.Fatalln(err)
	}
	err = f.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
