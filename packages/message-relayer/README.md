# Batch Submitter

Contains an executable batch submitter service which watches L1 and a local L2 node and submits batches to the
`CanonicalTransactionChain` & `StateCommitmentChain` based on its local information.

## Configuration
All configuration is done via environment variables.

## Building & Running
1. Make sure dependencies are installed just run `yarn` in the base directory
2. Build `yarn build`
3. Run `yarn start`

## Controlling log output verbosity
Before running, set the `DEBUG` environment variable to specify the verbosity level. It must be made up of comma-separated values of patterns to match in debug logs. Here's a few common options:
* `debug*` - Will match all debug statements -- very verbose
* `info*` - Will match all info statements -- less verbose, useful in most cases
* `warn*` - Will match all warnings -- recommended at a minimum
* `error*` - Will match all errors -- would not omit this

Examples:
* Everything but debug: `export DEBUG=info*,error*,warn*`
* Most verbose: `export DEBUG=info*,error*,warn*,debug*`
