# op-challenger

The `op-challenger` is a modular **op-stack** challenge agent
written in golang for dispute games including, but not limited to, attestation games, fault
games, and validity games. To learn more about dispute games, visit the
[fault proof specs](../specs/fault-proof.md).

## Quickstart

First, clone this repo. Then run:

```shell
cd op-challenger
make alphabet
```

This creates a local devnet, starts a dispute game using the simple alphabet trace type and runs two op-challenger
instances with differing views of the correct alphabet to play the game.

Alternatively, you can build the `op-challenger` binary directly using the pre-configured
[Makefile](./Makefile) target by running `make build`, and then running `./bin/op-challenger --help`
to see a list of available options.

## Usage

`op-challenger` is configurable via command line flags and environment variables. The help menu
shows the available config options and can be accessed by running `./op-challenger --help`.

### Running with Cannon on Local Devnet

To run `op-challenger` against the local devnet, first ensure the required components are built and the devnet is running.
From the top level of the repository run:

```shell
make devnet-clean
make cannon-prestate op-challenger
make devnet-up
```

Then start `op-challenger` with:
```shell
DISPUTE_GAME_FACTORY=$(jq -r .DisputeGameFactoryProxy .devnet/addresses.json)
./op-challenger/bin/op-challenger \
  --trace-type cannon \
  --l1-eth-rpc http://localhost:8545 \
  --game-factory-address $DISPUTE_GAME_FACTORY \
  --agree-with-proposed-output=true \
  --datadir temp/challenger-data \
  --cannon-rollup-config .devnet/rollup.json  \
  --cannon-l2-genesis .devnet/genesis-l2.json \
  --cannon-bin ./cannon/bin/cannon \
  --cannon-server ./op-program/bin/op-program \
  --cannon-prestate ./op-program/bin/prestate.json \
  --cannon-l2 http://localhost:9545 \
  --mnemonic "test test test test test test test test test test test junk" \
  --hd-path "m/44'/60'/0'/0/8" \
  --num-confirmations 1
```

The mnemonic and hd-path above is a prefunded address on the devnet. The challenger respond to any created games by
posting the correct trace as the counter-claim. The scripts below can then be used to create and interact with games.

## Scripts

The [scripts](scripts) directory contains a collection of scripts to assist with manually creating and playing games.
This are not intended to be used in production, only to support manual testing and to aid with understanding how
dispute games work. They also serve as examples of how to use `cast` to manually interact with the dispute game
contracts.

### Understanding Revert Reasons

When actions performed by these scripts fails, they typically print a message that includes the
abi encoded revert reason provided by the contract. e.g.

```
Error:
(code: 3, message: execution reverted, data: Some(String("0x67fe1950")))
```

The `cast 4byte` command can be used to decode these revert reasons. e.g.

```shell
$ cast 4byte 0x67fe1950
GameNotInProgress()
```

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
    * The root claim must have the high-order byte set to the
      invalid [VM status](../specs/cannon-fault-proof-vm.md#state-hash) (`0x01`) to indicate that the trace concludes
      that the disputed output root is invalid.
      e.g. `0x0146381068b59d2098495baa72ed2f773c1e09458610a7a208984859dff73add`
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

### [resolve.sh](scripts/resolve.sh)

```shell
./scripts/resolve.sh <RPC_URL> <GAME_ADDRESS> <SIGNER_ARGS>...
```

Resolves a dispute game. Note that this will fail if the dispute game has already been resolved
or if the clocks have not yet expired and further moves are possible.
If the game is resolved successfully, the result is printed.

* `RPC_URL` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_ADDRESS` - the address of the dispute game to resolve.
* `SIGNER_ARGS` the remaining args are past as arguments to `cast` when sending transactions.
  These arguments must specify a way for `cast` to sign the transactions.
  See `cast send --help` for supported options.

### [list_games.sh](scripts/list_games.sh)

```shell
./scripts/list_games.sh <RPC> <GAME_FACTORY_ADDRESS>
```

Prints the games created by the game factory along with their current status.

* `RPC_URL` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_FACTORY_ADDRESS` - the address of the dispute game factory contract on L1.

### [list_claims.sh](scripts/list_claims.sh)

```shell
./scripts/list_claims.sh <RPC> <GAME_ADDR>
```

Prints the list of current claims in a dispute game.

* `RPC_URL` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_ADDRESS` - the address of the dispute game to list the move in.
