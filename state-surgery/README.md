# state-surgery

This package performs state surgery. It takes the following input:

1. A v0 database
2. A partial `genesis.json`
3. A list of addresses that transacted on the network prior to this past regenesis.
4. A list of addresses that performed approvals on prior versions of the OVM ETH contract.

It creates an initialized Bedrock Geth database as output. It does this by performing the following steps:

1. Iterates over the old state.
2. For each account in the old state, add that account and its storage to the new state after copying its balance from the OVM_ETH contract.
3. Iterates over the pre-allocated accounts in the genesis file and adds them to the new state.
4. Imports any accounts that have OVM ETH balances but aren't in state.
5. Configures a genesis block in the new state using `genesis.json`.

It performs the following integrity checks:

1. OVM ETH storage slots must be completely accounted for.
2. The total supply of OVM ETH migrated must match the total supply of the OVM ETH contract.

This process takes about two hours on mainnet.

Unlike previous iterations of our state surgery scripts, this one does not write results to a `genesis.json` file. This is for the following reasons:

1. **Performance**. It's much faster to write binary to LevelDB than it is to write strings to a JSON file.
2. **State Size**. There are nearly 1MM accounts on mainnet, which would create a genesis file several gigabytes in size. This is impossible for Geth to import without a large amount of memory, since the entire JSON gets buffered into memory. Importing the entire state database will be much faster, and can be done with fewer resources.

## Data Files

The following data files are used for mainnet:

1. `mainnet-ovm-4-addresses.csv`: Contains all addresses that used OVM ETH during regenesis 4. Calculated by parsing Mint, Burn, and Transfer events from that network's OVM ETH contract.
2. `mainnet-ovm-4-allowances.csv`: Contains all addresses that performed an approval on OVM ETH during regenesis 4 and who they approved. Calculated by parsing Approve events on that network's OVM ETH contract.

These files are used to build the list of OVM ETH storage slots.

## Compilation

Run `make surgery`.

## Usage

```
NAME:
   surgery - migrates data from v0 to Bedrock

USAGE:
   surgery [global options] command [command options] [arguments...]

COMMANDS:
   dump-addresses  dumps addresses from OVM ETH
   migrate         migrates state in OVM ETH
   help, h         Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --data-dir value, -d value  data directory to read
   --help, -h                  show help (default: false)
```
