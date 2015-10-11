#v18 2016/10/16

* Improve error message when trying to save a conflicting revision.

#v17 2016/10/15

* Fix for v16 bug. All vcs list commands now produce paths relative to the root of the vcs.

# v16 2015/10/15

* Determine repo root using vcs commands and use that instead of dep.dir

# v15 2015/10/14

* Update .travis.yml file to do releases to github

# v14 2015/10/08

* Don't print out a workspace path when GOVENDOREXPERIMENT is active. The vendor/ directory is not a valid workspace, so can't be added to your $GOPATH.

# v13 2015/10/07

* Do restores in 2 separate steps, first download all deps and then check out the recorded revisions.
* Update Changelog date format

# v12 2015/09/22

* Extract errors into separate file.

# v11 2015/08/22

* Amend code to pass golint.

# v10 2015/09/21

* Analyse vendored package test dependencies.
* Update documentation.

# v9 2015/09/17

* Don't save test dependencies by default.

# v8 2015/09/17

* Reorganize code.

# v7 2015/09/09

* Add verbose flag.
* Skip untracked files.
* Add VCS list command.

# v6 2015/09/04

*  Revert ignoring testdata directories and instead ignore it while
processing Go files and copy the whole directory unmodified.

# v5 2015/09/04

* Fix vcs selection in restore command to work as go get does

# v4 2015/09/03

* Remove the deprecated copy option.

# v3 2015/08/26

* Ignore testdata directories

# v2 2015/08/11

* Include command line packages in the set to copy

This is a simplification to how we define the behavior
of the save command. Now it has two distinct package
parameters, the "root set" and the "destination", and
they have clearer roles. The packages listed on the
command line form the root set; they and all their
dependencies will be copied into the Godeps directory.
Additionally, the destination (always ".") will form the
initial list of "seen" import paths to exclude from
copying.

In the common case, the root set is equal to the
destination, so the effective behavior doesn't change.
This is primarily just a simpler definition. However, if
the user specifies a package on the command line that
lives outside of . then that package will be copied.

As a side effect, there's a simplification to the way we
add packages to the initial "seen" set. Formerly, to
avoid copying dependencies unnecessarily, we would try
to find the root of the VCS repo for each package in the
root set, and mark the import path of the entire repo as
seen. This meant for a repo at path C, if destination
C/S imports C/T, we would not copy C/T into C/S/Godeps.
Now we don't treat the repo root specially, and as
mentioned above, the destination alone is considered
seen.

This also means we don't require listed packages to be
in VCS unless they're outside of the destination.

# v1 2015/07/20

* godep version command

Output the version as well as some godep runtime information that is
useful for debugging user's issues.

The version const would be bumped each time a PR is merged into master
to ensure that we'll be able to tell which version someone got when they
did a `go get github.com/tools/godep`.

# Older changes

Many and more, see `git log -p`
