# bedrock-devnet

This is a utility for running a local Bedrock devnet. It is designed to replace the legacy Bash-based devnet runner as part of a progressive migration away from Bash automation.

The easiest way to invoke this script is to run `make devnet-up-deploy` from the root of this repository. Otherwise, to use this script run `python3 main.py --monorepo-dir=<path to the monorepo>`. You may need to set `PYTHONPATH` to this directory if you are invoking the script from somewhere other than `bedrock-devnet`.
