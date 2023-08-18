# op-erigon

This directory is used to build a local erigon binary for the integration
tests in op-e2e (and possibly for future use elsewhere).

## Why?

The top level go mod has replace directives to use op-geth instead of vanilla
go-ethereum, among other dependency replacements.  These replacements are
incompatible with erigon.  Unfortunately, replace directives are global, and
are irrespective of build tags or other hacks the only way to replace them is
with a new go.mod.  This directory simply contains a go.mod with replace
directives pointing to the desired op-erigon version.

## Why not a git submodule?

Go generally doesn't play very nicely with git submodules.  In order to
utilize a submodule, generally speaking, you need to either treat it exactly
like you would an independent repository (e.g. executing commands only locally
inside it), or you must utilize replace directives pointing to local paths.

As already pointed out, replace directives are often problematic, and they are
ignored unless they belong to the top level go.mod.  That means that if
someone tries to build an op-erigon binary with a simple `go build` command
pointing to a repository with sub-modules, this command is likely to fail in
confusing ways.

## How to keep this version synced?

Keeping this version aligned with the upstream version being tested is as
simple as specifying the desired upstream tag or branch in the go.mod and
running `go mod tidy` or, executing a `go get` with the appropriate version
info.  Checks to ensure the docker image and e2e versions are aligned should
be added to CI and the make process if and when such enhancements are added.
