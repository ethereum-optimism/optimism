# op-batcher

`op-batcher` is a service which reads unsafe L2 blocks from a sequencer, extracts the transaction into batches, compresses these batches and sends them to the data availability layer.

Batches are grouped into channels, which are themselves split into frames to be sent to the data availability layer (e.g. Ethereum L1).

## Holocene
With the Holocene hardfork, it is necessary for frames to arrive in order in the L2 consensus client for the derivation pipeline (i.e. the safe L2 chain) to make progress. "Future" batches are dropped. Already included batches are ignored.

The batcher implementation therefore maintains the following invariants, even when there is a re-org:

### Enqueue unsafe L2 blocks in order

Blocks are checked to form a chain when being added.

What about when there is an unsafe reorg? Now, we keep the old blocks in the batcher and just start again afterwards. It is a bit indirect, but we do close the channel manager, submit everything which is pending, and then clear out the state and exit the process. So we will end up starting fresh and the blocks will be in order.

If a channel times out, (i.e. a tx is confirmed, but too late) it gets requeued asynchronously from the add+publish goroutine. This means that the next channel's blocks could have been dequeued already, meaning that blocks are put back out of order.

blocks: [00,01,02,03,04,05] (start)
blocks: [02,03,04,05] (first channel dequeued)
blocks: [04,05] (second channel dequeued)
blocks: [00,01,04,05] (first channel requeued, blocks now out of order)


Possible solutions:
* wait until the entire channel has been confirmed and definitely not timed out before dequeuing the next blocks (this could slow things down though)
* requeue _all_ channels when one times out, and i) even include channels which began submitting and ii) make sure we reorder blocks when requeueing. Since we would not flip the DA type at this point, it would be safe to requeue channels which are partially submitted.

This is less of a problem for DA switching / requeueing because this happens synchronously with the add+publish fn in the loop (no dequeueing can happen before the blocks are requeued)

We want to move blocks atomically between the blocks queue and the channel queue. This looks fairly safe at the moment. If the process crashes at an unfortunate moment, it rebuilds its state from scratch anyway.



### Keep channels in the channel queue in order
### Sends transactions in order
### When building a blob (i.e. multi frame) transaction, frames must be in in order
### re-queueing case when a channel fails to submit in time -> need to preserve order across channels (also future pending)

