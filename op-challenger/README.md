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
