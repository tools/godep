### Godep

Command godep helps build packages reproducibly by fixing their dependencies.

### Install

    $ go get github.com/kr/godep

### Workflow

There are two commands: `save` and `go`.

- Command `save` inspects the workspace (GOPATH) for the currently-used
set of dependencies, and saves them in file `Godeps`.
- Command `go` reads the list of dependencies from `Godeps`,
sets up a temporary GOPATH, and runs the go tool.

Reference documentation is available from `godep help`.

#### Getting Started

How to add godep in a new project.

1. Get your project building properly with `go install`.
2. Run `godep save`.
3. Read over the contents of file `Godeps`, make sure it looks reasonable.
4. Commit `Godeps` to version control.

#### Edit-test Cycle

1. Edit code
2. Run `godep go test`
3. (repeat)

#### Add or Update a Dependency

(Note: this flow is currently more difficult than it could
be, because the workspace doesn't necessarily have the same
versions of existing dependencies. See [issue #2](https://github.com/kr/godep/issues/2) for more.)

1. Edit code; add a new import
2. Run `godep save`
3. Inspect the changes to `Godeps`, for example with `git diff`,
and make sure it looks reasonable.
There should be a single new entry, or a single changed entry,
or whatever change you were trying to make. If you see unexpected
things (such as versions of other dependencies having changed),
either edit `Godeps` to restore the desired versions or check out
the desired versions in your workspace and re-run `godep save`.
4. Commit the change.

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
	GoVersion  string   // Output of "go version".
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
	"GoVersion": "go version go1.1.2 darwin/amd64",
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

### Possible Future Commands

- [`restore`](https://github.com/kr/godep/issues/2) – install exact dependencies previously saved
- [`diff`](https://github.com/kr/godep/issues/1) – show difference between saved and installed deps
