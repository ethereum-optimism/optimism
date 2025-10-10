# op-dispute-mon

The `op-dispute-mon` is an off-chain service to monitor dispute games.

## Output root agreement behavior

When validating proposed output roots across multiple rollup nodes, `op-dispute-mon` accounts for node sync status to avoid false disagreements:

- If a node returns "not found" but is behind (its reported UnsafeL2 is below the requested L2 block, with a small tolerance), that response is ignored for agreement purposes.
- If all nodes report "not found" and the proposal is in the future (above every node's reported UnsafeL2), the proposal is treated as invalid and we disagree without error.
- If at least one node returns an output, agreement/disagreement is determined among the nodes that found an output, and the claim must also be considered safe by at least one node.

This logic reduces noise from lagging nodes while maintaining strictness for future proposals.

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
  --rollup-rpc <Optimism-Rollup-RPC-URL>,<Secondary-RPC-URL>,<Tertiary-RPC-URL>

# For networks using op-supervisor:
./bin/op-dispute-mon \
  --network <Predefined-Network> \
  --l1-eth-rpc <L1-Ethereum-RPC-URL> \
  --supervisor-rpc <Supervisor-RPC-URL>,<Secondary-RPC-URL>,<Tertiary-RPC-URL>

```
