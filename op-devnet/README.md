# op-devnet

This is a utility for running a local Bedrock devnet.
It is designed to replace the legacy Bash-based devnet runner as part of a progressive migration away from Bash automation.

The easiest way to invoke this script is to run `make devnet-up-deploy` from the root of this repository.
Otherwise, to use this script run `go run op-devnet/cmd/main.go --monorepo-dir=<path to the monorepo>`.
