# op-version-check

A CLI tool for determining which contract versions are deployed for
chains in a superchain. It will output a JSON file that contains a
list of each chain's versions. It is assumed that the implementations
that are being checked have already been deployed and their contract
addresses exist inside of the `superchain-registry` repository. It is
also assumed that the semantic version file in the `superchain-registry`
has been updated. The tool will output the semantic versioning to
determine which contract versions are deployed.

### Configuration

#### L1 RPC URL

The L1 RPC URL is used to determine which superchain to target. All
L2s that are not based on top of the L1 chain that corresponds to the
L1 RPC URL are filtered out from being checked. It also is used to
double check that the data in the `superchain-registry` is correct.

#### Chain IDs

A list of L2 chain IDs can be passed that will be used to filter which
L2 chains will have their versions checked. Omitting this argument will
result in all chains in the superchain being considered.

#### Deploy Config

The path to the `deploy-config` directory in the contracts package.
Since multiple L2 networks may be considered in the check, the `deploy-config`
directory must be passed and then the particular deploy config files will
be read out of the directory as needed.

#### Outfile

The file that the versions should be written to. If omitted, the file
will be written to stdout

#### Usage

It can be built and run using the [Makefile](../../Makefile) `op-version-check`
target. Run `make op-version-check` to create a binary in [../../bin/op-version-check](../../bin/op-version-check)
that can be executed, optionally providing the `--l1-rpc-url`, `--chain-ids`,
`--superchain-target`, and `--outfile` flags.

```sh
./bin/op-version-check
```
