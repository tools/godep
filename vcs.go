package main

import (
	"bytes"
	"code.google.com/p/go.tools/go/vcs"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type VCS struct {
	vcs *vcs.Cmd

	// run in outer GOPATH
	IdentifyCmd string
	DescribeCmd string
	IsDirtyCmd  string

	// run in sandbox repos
	CreateCmd   string
	LinkCmd     string
	ExistsCmd   string
	FetchCmd    string
	CheckoutCmd string

	// If nil, LinkCmd is used.
	LinkFunc func(dir, remote, url string) error
}

var vcsGit = &VCS{
	vcs: vcs.ByCmd("git"),

	IdentifyCmd: "rev-parse HEAD",
	DescribeCmd: "describe",
	IsDirtyCmd:  "diff --quiet HEAD",

	CreateCmd:   "init --bare",
	LinkCmd:     "remote add {remote} {url}",
	ExistsCmd:   "cat-file -e {rev}",
	FetchCmd:    "fetch --quiet {remote}",
	CheckoutCmd: "--git-dir {repo} --work-tree . checkout -q {rev}",
}

var vcsHg = &VCS{
	vcs: vcs.ByCmd("hg"),

	IdentifyCmd: "identify --id",
	DescribeCmd: "log -r . --template {latesttag}-{latesttagdistance}",
	IsDirtyCmd:  "status",

	CreateCmd:   "init",
	LinkFunc:    hgLink,
	ExistsCmd:   "cat -r {rev} .",
	FetchCmd:    "pull {remote}",
	CheckoutCmd: "clone -u {rev} {repo} .",
}

var cmd = map[*vcs.Cmd]*VCS{
	vcsGit.vcs: vcsGit,
	vcsHg.vcs:  vcsHg,
}

func VCSForImportPath(importPath string) (*VCS, *vcs.RepoRoot, error) {
	rr, err := vcs.RepoRootForImportPath(importPath, false)
	if err != nil {
		return nil, nil, err
	}
	vcs := cmd[rr.VCS]
	if vcs == nil {
		return nil, nil, fmt.Errorf("%s is unsupported: %s", rr.VCS.Name, importPath)
	}
	return vcs, rr, nil
}

func (v *VCS) identify(dir string) (string, error) {
	out, err := v.run(dir, v.IdentifyCmd)
	return string(bytes.TrimSpace(out)), err
}

func (v *VCS) describe(dir, rev string) string {
	out, _ := v.runQuiet(dir, v.DescribeCmd, "rev", rev)
	return string(bytes.TrimSpace(out))
}

func (v *VCS) isDirty(dir string) bool {
	out, err := v.run(dir, v.IsDirtyCmd)
	return err != nil || len(out) != 0
}

func (v *VCS) create(dir string) error {
	_, err := v.run(dir, v.CreateCmd)
	return err
}

func (v *VCS) link(dir, remote, url string) error {
	if v.LinkFunc != nil {
		return v.LinkFunc(dir, remote, url)
	}
	_, err := v.run(dir, v.LinkCmd, "remote", remote, "url", url)
	return err
}

func (v *VCS) exists(dir, rev string) bool {
	_, err := v.runQuiet(dir, v.ExistsCmd, "rev", rev)
	return err == nil
}

func (v *VCS) fetch(dir, remote string) error {
	_, err := v.run(dir, v.FetchCmd, "remote", remote)
	return err
}

func (v *VCS) checkout(dir, rev, repo string) error {
	_, err := v.run(dir, v.CheckoutCmd, "rev", rev, "repo", repo)
	return err
}

func (v *VCS) run(dir string, cmdline string, kv ...string) ([]byte, error) {
	return v.run1(dir, cmdline, kv, false)
}

func (v *VCS) runQuiet(dir string, cmdline string, kv ...string) ([]byte, error) {
	return v.run1(dir, cmdline, kv, true)
}

func (v *VCS) run1(dir string, cmdline string, kv []string, quiet bool) ([]byte, error) {
	m := make(map[string]string)
	for i := 0; i < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	args := strings.Fields(cmdline)
	for i, arg := range args {
		args[i] = expand(m, arg)
	}

	_, err := exec.LookPath(v.vcs.Cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "godep: missing %s command.\n", v.vcs.Name)
		return nil, err
	}

	cmd := exec.Command(v.vcs.Cmd, args...)
	cmd.Dir = dir
	if !quiet {
		cmd.Stderr = os.Stderr
	}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func expand(m map[string]string, s string) string {
	for k, v := range m {
		s = strings.Replace(s, "{"+k+"}", v, -1)
	}
	return s
}

// Mercurial has no command equivalent to git remote add.
// We handle it as a special case in process.
func hgLink(dir, remote, url string) error {
	hgdir := filepath.Join(dir, ".hg")
	if err := os.MkdirAll(hgdir, 0777); err != nil {
		return err
	}
	path := filepath.Join(hgdir, "hgrc")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	fmt.Fprintf(f, "[paths]\n%s = %s\n", remote, url)
	return f.Close()
}
