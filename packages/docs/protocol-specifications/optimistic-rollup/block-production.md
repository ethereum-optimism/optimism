# Block Production

![Overview of Optimistic Rollup block production contracts on L1 Ethereum.](../../.gitbook/assets/rollup-contracts-overview.png)

## Canonical Transaction Chain

This is a monotonically increasing list of transactions which is maintained in an Ethereum smart contract. It can never change & is only reverted when Ethereum blocks are reverted.

#### OVM Transaction Queue

Anyone may post to the OVM transaction queue. However, only the sequencer may immediately move their transactions into the canonical transaction chain. Everyone else must wait for the `slow_queue_timeout` period.

## State Commitment Chain



