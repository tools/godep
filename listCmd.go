package main

var cmdList = &Command{
	Name:  "list",
	Args:  "",
	Short: "list dependencies and their state",
	Long: `

-goroot: Include dependencies located in $GOROOT

Package names surrounded by "()" are main packages.

/Users/emuller/go/src/github.com/heroku/log-shuttle
    Local
        github.com/heroku/log-shuttle
        (github.com/heroku/log-shuttle/cmd/main/log-shuttle)
    Godeps/_workspace/src
        X
        Y
        Z
    vendor/
        github.com/heroku/rollrus
        (github.com/mattes/migrate)
    $GOPATH
        /home/foo/go/src
            github.com/heroku/slog
        /home/foo/go2/src
            D
    $GOROOT
        /usr/local/go/src
            net
            foo
            bar
    MISSING
        github.com/foozle/bazzle

-json: Output in JSON

{
    "Location": "/Users/emuller/go/src/github.com/heroku/log-shuttle",
    "Local" : [
        "github.com/heroku/log-shuttle",
        "(github.com/heroku/log-shuttle/cmd/main/log-shuttle)"
    ],
    "Godeps/_workspace/src":[
        "X", "Y", "Z"
        ],
    "vendor": [
        "github.com/heroku/rollrus",
        "(github.com/mattes/migrate)"
    ],
    "$GOPATH": {
        "/home/foo/go/src": [
          "github.com/heroku/slog"
        ],
        "/home/foo/go2/src": [
            "D"
        ]
    },
    "$GOROOT": {
        "/usr/local/go/src": [
            "net",
            "foo",
            "bar"
        ]
    },
    "MISSING": [
        "github.com/foozle/bazzle"
    ]
}



For more about specifying packages, see 'go help packages'.
`,
	Run:          runList,
	OnlyInGOPATH: true,
}

func runList(cmd *Command, args []string) {

}
