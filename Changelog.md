# v12 22/09/2015

* Extract errors into separate file.

# v11 22/09/2015

* Amend code to pass golint.

# v10 21/09/2015

* Analyse vendored package test dependencies.
* Update documentation.

# v9 17/09/2015

* Don't save test dependencies by default.

# v8 17/09/2015

* Reorganize code.

# v7 09/09/2015

* Add verbose flag.
* Skip untracked files.
* Add VCS list command.

# v6 04/09/2015

*  Revert ignoring testdata directories and instead ignore it while
processing Go files and copy the whole directory unmodified.

# v5 04/09/2015

* Fix vcs selection in restore command to work as go get does

# v4 03/09/2015

* Remove the deprecated copy option.

# v3 26/08/2015

* Ignore testdata directories

# v2 11/08/2015

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

# v1 20/07/2015

* godep version command

Output the version as well as some godep runtime information that is
useful for debugging user's issues.

The version const would be bumped each time a PR is merged into master
to ensure that we'll be able to tell which version someone got when they
did a `go get github.com/tools/godep`.