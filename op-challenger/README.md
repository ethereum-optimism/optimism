# op-challenger

The `op-challenger` is a modular [op-stack](https://stack.optimism.io/) challenge agent written in golang for dispute games including, but not limited to, attestation games, fault games, and validity games.

## Quickstart

First, clone the [Optimism Monorepo](https://github.com/ethereum-optimism/optimism) and set the `MONOREPO_DIR` environment variable to the path of that directory in a `.env` file as shown in `.env.example`.

Then, you can simply run `make`, which will compile all solidity + golang sources, bring up the Optimism [devnet](https://github.com/ethereum-optimism/optimism/blob/develop/ops-bedrock/devnet-up.sh) while also deploying the [mock dispute game contracts](./contracts), and then run the `op-challenger`.

Alternatively, you can build the `op-challenger` binary locally using the pre-configured makefile target by running `make build`, and then running `./op-challenger --help` to see the available options.

## Usage

`op-challenger` is configurable via command line flags and environment variables. The help menu shows the available config options and can be accessed by running `./op-challenger --help`.

Note that there are many global options, but the most important ones are:

- `OP_CHALLENGER_L1_ETH_RPC`: An L1 Ethereum RPC URL
- `OP_CHALLENGER_ROLLUP_RPC`: A Rollup Node RPC URL
- `OP_CHALLENGER_L2OO_ADDRESS`: The L2OutputOracle Contract Address
- `OP_CHALLENGER_DGF_ADDRESS`: Dispute Game Factory Contract Address
- `OP_CHALLENGER_PRIVATE_KEY`: The Private Key of the account that will be used to send challenge transactions
- `OP_CHALLENGER_L2_CHAIN_ID`: The chain id of the L2 network

Here is a reduced output from running `./op-challenger --help`:

```bash
NAME:
   op-challenger - Modular Challenger Agent
USAGE:
   main [global options] command [command options] [arguments...]
VERSION:
   1.0.0
DESCRIPTION:
   A modular op-stack challenge agent for output dispute games written in golang.
COMMANDS:
   help, h  Shows a list of commands or help for one command
GLOBAL OPTIONS:
   --l1-eth-rpc value                      HTTP provider URL for L1. [$OP_CHALLENGER_L1_ETH_RPC]
   --rollup-rpc value                      HTTP provider URL for the rollup node. [$OP_CHALLENGER_ROLLUP_RPC]
   --l2oo-address value                    Address of the L2OutputOracle contract. [$OP_CHALLENGER_L2OO_ADDRESS]
   --dgf-address value                     Address of the DisputeGameFactory contract. [$OP_CHALLENGER_DGF_ADDRESS]
   --private-key value                     The private key to use with the service. Must not be used with mnemonic. [$OP_CHALLENGER_PRIVATE_KEY]
   ...
   --help, -h                              show help
   --version, -v                           print the version
```
## Acknowledgements
- [op-challenger (golang)](https://github.com/refcell/op-challenger): an inital golang challenge agent 🚀
- [op-challenger (rust)](https://github.com/clabby/op-challenger): a rust challenge agent 🦀
