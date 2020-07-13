# Block Production

![Overview of Optimistic Rollup block production contracts on L1 Ethereum.](https://github.com/ethereum-optimism/optimism-monorepo/tree/f9f7f32f11c35acdf3b1b46ca8d076da09172516/packages/docs/.gitbook/assets/rollup-contracts-overview%20%281%29.png)

## Canonical Transaction Chain

This is a monotonically increasing list of transactions which is maintained in an Ethereum smart contract. It can never change & is only reverted if L1 Ethereum blocks are reverted. It is the final word on what transactions are applied to the rollup chain, and in what order. Transactions from this chain come from one of two "queues": the OVM Transaction queue, and the L1-&gt;L2 Transaction Queue.

### OVM Transaction Queue

This is where the sequencer is allowed to post transactions which they received off chain to be applied to the rollup chain. Transactions can only be moved from the OVM Transaction Queue to the Canonical Transaction Chain if the transactions in the L1-&gt;L2 transaction queue are not older than some number of L1 blocks.

### L1-&gt;L2 Transaction queue

This is where users who are being censored, as well as L1 contracts like deposit contracts, enqueue transactions to be added to the rollup chain. After some number of L1 blocks, the L1-&gt;L2 transactions _must be included_ next in the canonical transaction chain. This enforces censorship resistance.

## State Commitment Chain

The state commitment chain is a rollup list of OVM “outputs” \(namely, state roots and outgoing messages\) which must correspond to the canonical chain’s inputs. When a transaction is appended to the canonical transaction chain, there is only 1 valid state which should result. These outputs are committed by the sequencer, and rolled back in the case of fraud--without touching the canonical transactions.

