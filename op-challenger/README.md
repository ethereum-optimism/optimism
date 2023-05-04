# op-challenger

The `op-challenger` is a modular **op-stack** challenge agent
written in golang for dispute games including, but not limited to, attestation games, fault
games, and validity games. To learn more about dispute games, visit the
[dispute game specs](../specs/dispute-game.md).

## Quickstart

First, clone this repo. Then, run `make`, which will build all required targets.
Alternatively, run `make devnet` to bring up the [devnet](../ops-bedrock/devnet-up.sh)
which deploys the [mock dispute game contracts](./contracts) as well as an
`op-challenger` instance.

Alternatively, you can build the `op-challenger` binary locally using the pre-configured
[Makefile](./Makefile) target by running `make build`, and then running `./op-challenger --help`
to see a list of available options.

## Usage

`op-challenger` is configurable via command line flags and environment variables. The help menu
shows the available config options and can be accessed by running `./op-challenger --help`.

Note that there are many global options, but the most important ones are:

- `OP_CHALLENGER_L1_ETH_RPC`: An L1 Ethereum RPC URL
- `OP_CHALLENGER_ROLLUP_RPC`: A Rollup Node RPC URL
- `OP_CHALLENGER_L2OO_ADDRESS`: The L2OutputOracle Contract Address
- `OP_CHALLENGER_DGF_ADDRESS`: Dispute Game Factory Contract Address

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
   ...
   --help, -h                              show help
   --version, -v                           print the version
```


