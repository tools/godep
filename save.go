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
with the exact source control revision of each dependency. It writes
output to file Godeps in the current directory, in JSON format with
the following structure:

    type Godeps struct {
    	ImportPath string
    	GoVersion  string   // Output of "go version".
    	Packages   []string // Arguments to godep save, if any.
    	Deps       []struct {
    		ImportPath string
    		Comment    string // Tag or description of commit, if present.
    		Rev        string // VCS-specific commit ID.
    	}
    }

For more about specifying packages, see 'go help packages'.
`,
	Run: runSave,
}

func runSave(cmd *Command, args []string) {
	g := &Godeps{
		ImportPath: MustLoadPackages(".")[0].ImportPath,
		GoVersion:  mustGoVersion(),
	}
	if len(args) > 0 {
		g.Packages = args
	} else {
		args = []string{"."}
	}
	a := MustLoadPackages(args...)
	err := g.Load(a)
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
