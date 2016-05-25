## Why do I need to check in `vendor/`?

godep's primary concern is to allow you to repeatably build your project. Your
dependencies are part of that project. Without them it won't build. Not
committing `vendor/` adds additional external dependencies that are outside of
your control. In Go, fetching packages is tied to multiple external systems
(DNS, web servers, etc). Over time other developers or code hosting sites may
discontinue service, delete code, force push, or take any number of other
actions that may make a package unreachable.