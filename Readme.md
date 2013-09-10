### Godep

Command godep helps build packages reproducibly by fixing their dependencies.

### Example Usage

```
$ godep save       # writes file Godeps
$ godep go install # reads file Godeps
```

The go tool is run inside a temporary sandbox containing
the specified versions of all dependencies.

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

- `restore` – install exact dependencies previously saved
- `diff` – show difference between saved and installed deps
