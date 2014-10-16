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
Then commit the whole Godeps directory to version control, [including _workspace](https://github.com/tools/godep/pull/123).

#### Restore

The `godep restore` command is the opposite of `godep save`.
It will install the package versions specified in
Godeps/Godeps.json to your GOPATH.

#### Edit-test Cycle

1. Edit code
2. Run `godep go test`
3. (repeat)

#### Add a Dependency

To add a new package foo/bar, do this:

1. Run `go get foo/bar`
2. Edit your code to import foo/bar.
3. Run `godep save` (or `godep save ./...`).

#### Update a Dependency

To update a package from your `$GOPATH`, do this:

1. Run `go get -u foo/bar`
2. Run `godep update foo/bar`. (You can use the `...` wildcard,
for example `godep update foo/...`).

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

#### Old Format

Old versions of godep wrote the dependency list to a file Godeps,
and didn't copy source code. This mode no longer exists, but
commands 'godep go' and 'godep path' will continue to read the old
format for some time.

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
