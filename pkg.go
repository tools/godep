package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
)

type Package struct {
	Dir        string
	Root       string
	ImportPath string
	Deps       []string
	Standard   bool

	Error struct {
		Err string
	}
}

func LoadPackages(pkg []string) (a []*Package, err error) {
	args := []string{"list", "-e", "-json"}
	cmd := exec.Command("go", append(args, pkg...)...)
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	w.Close()
	d := json.NewDecoder(r)
	for {
		info := new(Package)
		err = d.Decode(info)
		if err == io.EOF {
			break
		}
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
