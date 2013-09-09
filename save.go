package main

import (
	"log"
	"os"
)

var cmdSave = &Command{
	Usage: "save [packages]",
	Short: "list current dependencies to a file",
	Long: `
Save writes a list of the dependencies of the named packages along
with the exact source control revision of each dependency.

Output goes to file Godeps.

For more about specifying packages, see 'go help packages'.
`,
	Run: runSave,
}

func runSave(cmd *Command, args []string) {
	dot, err := LoadPackages([]string{"."})
	if err != nil {
		log.Fatalln(err)
	}
	vers, err := goVersion()
	if err != nil {
		log.Fatalln(err)
	}
	a, err := LoadPackages(args)
	if err != nil {
		log.Fatalln(err)
	}
	p := a[0]
	if p.Standard {
		log.Fatalln("ignoring stdlib package:", p.ImportPath)
	}

	g := &Godeps{
		ImportPath: dot[0].ImportPath,
		GoVersion:  vers,
	}
	err = g.Load(a)
	if err != nil {
		log.Fatalln(err)
	}
	if g.Deps == nil {
		g.Deps = make([]Dependency, 0) // produce json [], not null
	}
	f, err := os.Create("Godeps")
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
