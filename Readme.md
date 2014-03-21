### Godep

Command godep helps build packages reproducibly by fixing their dependencies.

This tool assumes you are working in a standard Go workspace,
as described in http://golang.org/doc/code.html. We require Go 1.1
or newer to build godep itself, but you can use it on any project
that works with Go 1 or newer.

### Install

	$ go get github.com/tools/godep

#### Getting Started

How to add godep in a new project.

Assuming you've got everything working already, so you can
build your project with `go install` and test it with `go test`,
it's one command to start using:

	$ godep save

This will save a list of dependencies to the file Godeps/Godeps.json,
and copy their source code into Godeps/_workspace.
Read over its contents and make sure it looks reasonable.
Then commit the file to version control.

You can omit the source code with the flag -copy=false.
This means fewer files to store in your local repository, but
subsequent invocations of `godep go` will need to access the
network to fetch the appropriate source code later. Using the
default behavior is faster and more reliable.

#### Edit-test Cycle

1. Edit code
2. Run `godep go test`
3. (repeat)

#### Add or Update a Dependency

To add or update package foo/bar, do this:

1. Run `godep restore`
2. Run `go get -u foo/bar`
3. Edit your code, if necessary, to import foo/bar.
4. Run `godep save`

Before committing the change, you'll probably want to inspect
the changes to Godeps, for example with `git diff`,
and make sure it looks reasonable.

#### Multiple Packages

If your repository has more than one package, you're probably
accustomed to running commands like `go test ./...`,
`go install ./...`, and `go fmt ./...`.
Similarly, you should run `godep save ./...` to capture the
dependencies of all packages.

#### Using Other Tools

The `godep path` command helps integrate with commands other
than the standard go tool. This works with any tool that reads
GOPATH from its environment, for example the recently-released
[oracle command](http://godoc.org/code.google.com/p/go.tools/cmd/oracle).

	$ GOPATH=`godep path`:$GOPATH
	$ oracle -mode=implements .

### File Format

Godeps is a json file with the following structure:

```go
type Godeps struct {
	ImportPath string
	GoVersion  string   // Abridged output of 'go version'.
	Packages   []string // Arguments to godep save, if any.
	Deps       []struct {
		ImportPath string
		Comment    string // Description of commit, if present.
		Rev        string // VCS-specific commit ID.
	}
}
```

Example Godeps:

```json
{
	"ImportPath": "github.com/kr/hk",
	"GoVersion": "go1.1.2",
	"Deps": [
		{
			"ImportPath": "code.google.com/p/go-netrc/netrc",
			"Rev": "28676070ab99"
		},
		{
			"ImportPath": "github.com/kr/binarydist",
			"Rev": "3380ade90f8b0dfa3e363fd7d7e941fa857d0d13"
		}
	]
}
```
