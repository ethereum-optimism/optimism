# sequencer-recovery

Checks an interop chain's op-conductor sequencer cluster and restarts sequencing on the canonical
chain after an interop reorg stalls it.

## The problem

When a supernode finds an invalid executing message, it replaces the block that holds the message
with a deposit-only block. The sequencers' op-nodes follow the replacement. The conductors' raft log
still holds the latest payload committed before the reorg, which is on the invalidated chain.

A conductor that becomes leader compares the raft payload with its op-node's unsafe head, finds
`unsafe head mismatch`, does not start the sequencer, and transfers leadership. Every conductor does
the same, so leadership loops through the cluster and the chain stops.

## Runbook

1. Read the state:

   ```sh
   sequencer-recovery status --config <chain>.toml --network <chain> --supernode-rpc <url>
   ```

   `status` lists sequencing problems, which `recover` fixes, and member problems, which it does
   not (for example an op-node on a fork that needs a resync). It exits 1 when either list is not
   empty.

2. Print the plan:

   ```sh
   sequencer-recovery recover --config <chain>.toml --network <chain> --supernode-rpc <url>
   ```

3. Run it:

   ```sh
   sequencer-recovery recover --execute --config <chain>.toml --network <chain> --supernode-rpc <url>
   ```

   The plan pauses every conductor, stops any extra active sequencer, and makes a sequencer whose
   chain matches the supernode the raft leader. It then starts that op-node from its own unsafe head
   and waits until it has built new blocks and caught up to wall-clock time. These blocks overwrite
   the stale raft entry. Last, it resumes the conductors and waits for sequencing to report healthy.
   If a step fails, the conductors are still resumed.

## Inputs

- `--config`: the op-conductor-ops config that netchef renders to
  `scripts/op-conductor-ops/<chain>.toml`. The sequencer IDs in it must be the raft server IDs.
- `--supernode-rpc`: the supernode's rollup RPC for this chain, for example
  `http://<supernode>:9545/<chainID>`. Use a supernode directly instead of proxyd-cl: after a
  rewind, proxyd-cl can filter out every supernode for a while and reject requests.
- `--cert`: a CA bundle for the RPC endpoints. It overrides `cert_path` from the config.

## What it does not fix

- An RPC node or standby whose execution client cannot backfill the gap after a reorg, for
  example because it has no EL peers.
- A light op-node that cannot move onto the supernode's replacement block because its execution
  client does not have the block.
- The cause. The conductor should accept its op-node's unsafe head when that head is on the chain
  the supernode verified. Until op-conductor does that, this tool is the recovery procedure.
