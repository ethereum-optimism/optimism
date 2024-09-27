# op-batcher

`op-batcher` is a service which reads unsafe L2 blocks from a sequencer, extracts the transaction into batches, compresses these batches and sends them to the data availability layer.

Batches are grouped into channels, which are themselves split into frames to be sent to the data availability layer (e.g. Ethereum L1).

## Holocene
With the Holocene hardfork, it is necessary for frames to arrive in order in the L2 consensus client for the derivation pipeline (i.e. the safe L2 chain) to make progress. "Future" batches are dropped. Already included batches are ignored.

The batcher implementation therefore maintains the following invariants, even when there is a re-org:
* Enqueue unsafe L2 blocks in order
* Keep channels in the channel queue in order
* Sends transactions in order
* When building a blob (i.e. multi frame) transaction, frames must be in in order
* re-queueing case when a channel fails to submit in time -> need to preserve order across channels (also future pending)

