package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

var godepsFile = filepath.Join("Godeps", "Godeps.json")

// Godeps describes what a package needs to be rebuilt reproducibly.
// It's the same information stored in file Godeps.
type Godeps struct {
	ImportPath string
	GoVersion  string
	Packages   []string `json:",omitempty"` // Arguments to save, if any.
	Deps       []Dependency
	isFile     bool
}

func readGodeps(g *Godeps) (isFile bool, err error) {
	f, err := os.Open(godepsFile)
	if err != nil {
		isFile = true
		f, err = os.Open("Godeps")
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	err = json.NewDecoder(f).Decode(g)
	f.Close()
	return isFile, err
}
