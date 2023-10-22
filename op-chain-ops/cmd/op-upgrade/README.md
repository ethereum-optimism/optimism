# op-upgrade

A CLI tool for building Safe bundles that can upgrade many chains
at the same time. It will output a JSON file that is compatible with
the Safe UI. It is assumed that the implementations that are being
upgraded to have already been deployed and their contract addresses
exist inside of the `superchain-registry` repository. It is also
assumed that the semantic version file in the `superchain-registry`
has been updated. The tool will use semantic versioning to determine
which contract versions should be upgraded to and then build all of
the calldata.

### Configuration

#### L1 RPC URL

The L1 RPC URL is used to determine which superchain to target. All
L2s that are not based on top of the L1 chain that corresponds to the
L1 RPC URL are filtered out from being included. It also is used to
double check that the data in the `superchain-registry` is correct.

#### Chain IDs

A list of L2 chain IDs can be passed that will be used to filter which
L2 chains will have upgrades included in the bundle transaction. Omitting
this argument will result in all chains in the superchain being considered.

#### Deploy Config

The path to the `deploy-config` directory in the contracts package.
Since multiple L2 networks may be included in the bundle, the `deploy-config`
directory must be passed and then the particular deploy config files will
be read out of the directory as needed.

#### Outfile

The file that the bundle should be written to. If omitted, the file
will be written to stdout.
