# op-challenger

The `op-challenger` is a modular **op-stack** challenge agent written in
golang for dispute games including, but not limited to,attestation games,
fault games, and validity games. To learn more about dispute games, visit
the [fault proof specs][proof-specs].

[proof-specs]: https://specs.optimism.io/experimental/fault-proof/index.html

## Quickstart

To build the `op-challenger`, run `make` (which executes the `make build`
[Makefile](./Makefile) target). To view a list of available commands and
options, run `./bin/op-challenger --help`.

## Usage

`op-challenger` is configurable via command line flags and environment
variables. The help menu shows the available config options and can be
accessed by running `./op-challenger --help`.

### Running with Cannon on Local Devnet

To run `op-challenger` against the local devnet, first clean and run
the devnet from the root of the repository.
```shell
make devnet-clean
make devnet-up
```

Then build the `op-challenger` with `make op-challenger`.

Run the `op-challenger` with:
```shell
DISPUTE_GAME_FACTORY=$(jq -r .DisputeGameFactoryProxy .devnet/addresses.json)
./op-challenger/bin/op-challenger \
  --trace-type cannon \
  --l1-eth-rpc http://localhost:8545 \
  --rollup-rpc http://localhost:9546 \
  --game-factory-address $DISPUTE_GAME_FACTORY \
  --datadir temp/challenger-data \
  --cannon-rollup-config .devnet/rollup.json  \
  --cannon-l2-genesis .devnet/genesis-l2.json \
  --cannon-bin ./cannon/bin/cannon \
  --cannon-server ./op-program/bin/op-program \
  --cannon-prestate ./op-program/bin/prestate.json \
  --l2-eth-rpc http://localhost:9545 \
  --mnemonic "test test test test test test test test test test test junk" \
  --hd-path "m/44'/60'/0'/0/8" \
  --num-confirmations 1
```

The mnemonic and hd-path above is a prefunded address on the devnet.
The challenger will monitor dispute games and respond to any invalid
claims by posting the correct trace as the counter-claim. The commands
below can then be used to create and interact with games.

## Subcommands

The `op-challenger` has a few subcommands to interact with on-chain
fault dispute games. The subcommands support game creation, performing
game moves, and viewing fault dispute game data. They should not be
used in production and are intended to provide convenient manual testing.

### create-game

```shell
./bin/op-challenger create-game \
  --l1-eth-rpc <L1_ETH_RPC> \
  --game-address <GAME_FACTORY_ADDRESS> \
  --output-root <OUTPUT_ROOT> \
  --l2-block-num <L2_BLOCK_NUM> \
  <SIGNER_ARGS>
```

Starts a new fault dispute game that disputes the latest output proposal
in the L2 output oracle.

* `L1_ETH_RPC` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_FACTORY_ADDRESS` - the address of the dispute game factory contract on L1.
* `OUTPUT_ROOT` a hex encoded 32 byte hash that is used as the proposed output root.
* `L2_BLOCK_NUM` the L2 block number the proposed output root is from.
* `SIGNER_ARGS` arguments to specify the key to sign transactions with (e.g `--private-key`)

Optionally, you may specify the game type (aka "trace type") using the `--trace-type`
flag, which is set to the cannon trace type by default.

### move

The `move` subcommand can be run with either the `--attack` or `--defend` flag,
but not both.

```shell
./bin/op-challenger move \
  --l1-eth-rpc <L1_ETH_RPC> \
  --game-address <GAME_ADDRESS> \
  --attack \
  --parent-index <PARENT_INDEX> \
  --claim <CLAIM> \
  <SIGNER_ARGS>
```

Performs a move to either attack or defend the latest claim in the specified game.

* `L1_ETH_RPC` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_ADDRESS` - the address of the dispute game to perform the move in.
* `(attack|defend)` - the type of move to make.
  * `attack` indicates that the state hash in your local cannon trace differs to the state
    hash included in the latest claim.
  * `defend` indicates that the state hash in your local cannon trace matches the state hash
    included in the latest claim.
* `PARENT_INDEX` - the index of the parent claim that will be countered by this new claim.
  The special value of `latest` will counter the latest claim added to the game.
* `CLAIM` - the state hash to include in the counter-claim you are posting.
* `SIGNER_ARGS` arguments to specify the key to sign transactions with (e.g `--private-key`)

### resolve-claim

```shell
./bin/op-challenger resolve-claim \
  --l1-eth-rpc <L1_ETH_RPC> \
  --game-address <GAME_ADDRESS> \
  --claim <CLAIM_INDEX> \
  <SIGNER_ARGS>
```

Resolves a claim in a dispute game. Note that this will fail if the claim has already been resolved or if the claim is
not yet resolvable. If the claim is resolved successfully, the result is printed.

* `L1_ETH_RPC` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_ADDRESS` - the address of the dispute game to resolve.
* `CLAIM_INDEX` - the index of the claim to resolve.
* `SIGNER_ARGS` arguments to specify the key to sign transactions with (e.g `--private-key`).

### resolve

```shell
./bin/op-challenger resolve \
  --l1-eth-rpc <L1_ETH_RPC> \
  --game-address <GAME_ADDRESS> \
  <SIGNER_ARGS>
```

Resolves a dispute game. Note that this will fail if the dispute game has already
been resolved or if the clocks have not yet expired and further moves are possible.
If the game is resolved successfully, the result is printed.

* `L1_ETH_RPC` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_ADDRESS` - the address of the dispute game to resolve.
* `SIGNER_ARGS` arguments to specify the key to sign transactions with (e.g `--private-key`).

### list-games

```shell
./bin/op-challenger list-games \
  --l1-eth-rpc <L1_ETH_RPC> \
  --game-factory-address <GAME_FACTORY_ADDRESS>
```

Prints the games created by the game factory along with their current status.

* `L1_ETH_RPC` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_FACTORY_ADDRESS` - the address of the dispute game factory contract on L1.

### list-claims

```shell
./bin/op-challenger list-games \
  --l1-eth-rpc <L1_ETH_RPC> \
  --game-address <GAME_ADDRESS>
```

Prints the list of current claims in a dispute game.

* `L1_ETH_RPC` - the RPC endpoint of the L1 endpoint to use (e.g. `http://localhost:8545`).
* `GAME_ADDRESS` - the address of the dispute game to list the move in.
