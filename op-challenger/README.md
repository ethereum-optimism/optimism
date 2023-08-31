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

## Scripts

The [scripts](scripts) directory contains a collection of scripts to assist with manually creating and playing games.
This are not intended to be used in production, only to support manual testing and to aid with understanding how
dispute games work. They also serve as examples of how to use `cast` to manually interact with the dispute game
contracts.

### Dependencies

These scripts assume that the following tools are installed and available on the current `PATH`:

* `cast` (https://book.getfoundry.sh/cast/)
* `jq` (https://jqlang.github.io/jq/)
* `bash`

### [create_game.sh](scripts/create_game.sh)

```shell
./scripts/create_game.sh <RPC_URL> <GAME_FACTORY_ADDRESS> <ROOT_CLAIM> <SIGNER_ARGS>...
```

Starts a new fault dispute game that disputes the latest output proposal in the L2 output oracle.

* `RPC_URL` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_FACTORY_ADDRESS` - the address of the dispute game factory contract on L1.
* `ROOT_CLAIM` a hex encoded 32 byte hash to use as the root claim for the created game.
* `SIGNER_ARGS` the remaining args are past as arguments to `cast` when sending transactions.
  These arguments must specify a way for `cast` to sign the transactions.
  See `cast send --help` for supported options.

Creating a dispute game requires sending two transactions. The first transaction creates a
checkpoint in the `BlockOracle` that records the L1 block that will be used as the L1 head
when generating the cannon execution trace. The second transaction then creates the actual
dispute game, specifying the disputed L2 block number and previously checkpointed L1 head block.

### [move.sh](scripts/move.sh)

```shell
./scripts/move.sh <RPC_URL> <GAME_ADDRESS> (attack|defend) <PARENT_INDEX> <CLAIM> <SIGNER_ARGS>...
```

Performs a move to either attack or defend the latest claim in the specified game.

* `RPC_URL` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_ADDRESS` - the address of the dispute game to perform the move in.
* `(attack|defend)` - the type of move to make.
  * `attack` indicates that the state hash in your local cannon trace differs to the state
    hash included in the latest claim.
  * `defend` indicates that the state hash in your local cannon trace matches the state hash
    included in the latest claim.
* `PARENT_INDEX` - the index of the parent claim that will be countered by this new claim.
  The special value of `latest` will counter the latest claim added to the game.
* `CLAIM` - the state hash to include in the counter-claim you are posting.
* `SIGNER_ARGS` the remaining args are past as arguments to `cast` when sending transactions.
  These arguments must specify a way for `cast` to sign the transactions.
  See `cast send --help` for supported options.


### [list_claims.sh](scripts/list_claims.sh)

```shell
./scripts/list_claims.sh <RPC> <GAME_ADDR>
```

Prints the list of current claims in a dispute game.

* `RPC_URL` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_ADDRESS` - the address of the dispute game to list the move in.
