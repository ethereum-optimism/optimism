# op-dispute-mon

The `op-dispute-mon` is an off-chain service to monitor dispute games.

## Quickstart

Clone this repo. Then run:

```shell
make op-dispute-mon
```

This will build the `op-dispute-mon` binary which can be run with
`./op-dispute-mon/bin/op-dispute-mon`.

## Usage

`op-dispute-mon` is configurable via command line flags and environment variables. The help menu
shows the available config options and can be accessed by running `./bin/op-dispute-mon --help`.

```shell

# Start the op-dispute-mon with predefined network and RPC endpoints
./bin/op-dispute-mon \
  --network <Predefined-Network> \
  --l1-eth-rpc <L1-Ethereum-RPC-URL> \
  --rollup-rpc <Optimism-Rollup-RPC-URL>

```
