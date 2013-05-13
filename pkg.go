package main

import (
	"encoding/json"
	_ "github.com/kr/s3"
	"io"
	"os"
	"os/exec"
)

type Package struct {
	Dir        string
	ImportPath string
	Deps       []string
	Standard   bool

	Error struct {
		Err string
	}
}

func getInfo(pkg ...string) (a []*Package, err error) {
	args := []string{"list", "-e", "-json"}
	cmd := exec.Command("go", append(args, pkg...)...)
	r, w := io.Pipe()
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	d := json.NewDecoder(r)
	for _ = range pkg {
		info := new(Package)
		err = d.Decode(info)
		if err != nil {
			info.Error.Err = err.Error()
		}
		a = append(a, info)
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	return a, nil
}
