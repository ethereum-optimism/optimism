# OP Streamer Component Audit

## General Notes

- We could simplify the name of `PollingHotShotPollingInterval` to something like `HotshotPollingInterval`.
- The value `HOTSHOT_BLOCK_STREAM_LIMIT` is never used in the code.
- Only real issues found would happen in the case of race conditions.

## File Dive: `espresso/batch_buffer.go`

### `Insert`
```go
func (b *BatchBuffer[B]) Insert(batch B, i int)
```

- Calling this function twice with the same batch will cause the batch to be inserted twice. This method does not check for duplicates; it unconditionally inserts at index `i`.

### `TryInsert`
```go
func (b *BatchBuffer[B]) TryInsert(batch B) (int, bool)
```

- This function assumes the list is already sorted but it should always be the case.

---

## File Dive: `espresso/streamer.go`

### `GetFinalizedL1`
```go
func GetFinalizedL1(header *espressoCommon.HeaderImpl) espressoCommon.L1BlockInfo
```

- Could we not create an espresso-network go sdk function for this instead if this information is present in all the headers?

### `Refresh`
```go
func (s *BatchStreamer[B]) Refresh(ctx context.Context, finalizedL1 eth.L1BlockRef, safeBatchNumber uint64, safeL1Origin eth.BlockID) error
```

- Line 173 compares `fallbackBatchPos` (Batch Index) with `hotShotPos` (Espresso Block Height). How would that be possible?

### `processEspressoTransaction`
```go
func (s *BatchStreamer[B]) processEspressoTransaction(ctx context.Context, transaction espressoCommon.Bytes)
```

- The `Debug` log in `Update` (line 304) is redundant because `fetchHotShotRange` immediately logs `Trace`. We could remove the `Debug` log in `Update` and rely on `fetchHotShotRange`'s `Trace` log for low-level debugging and its `Info` log (line 344) for successful fetches. This clarifies the logs and reduces noise.
- The log "Batch already in buffer" (line 435) is misleading. It refers to `RemainingBatches` (the pending map), NOT `BatchBuffer` (the main sorted slice).  "Batch already in remaining list" would be more accurate!
- If the batch is found in the map, it warns but then immediately overwrites it with the new copy (`s.RemainingBatches[hash] = *batch`). Since the hash is the key, the content should be identical, making this overwrite benign but redundant.

### `confirmEspressoBlockHeight`
```go
func (s *BatchStreamer[B]) confirmEspressoBlockHeight(safeL1Origin eth.BlockID) (shouldReset bool)
```

- The function returns false when FinalizedState() fails meaning "do not reset the streamer", treating a network/RPC failure the same way as "no reorg happened".

## Deeper Dive on Component Flow

### 1. The All At Once RPC Calls"
- **Background**: The function `CheckBatch` makes a synchronous L1 RPC call (`HeaderHashByNumber`) if the batch's L1 origin is said to be finalized:
- **Scenario**:
  1.  The node accumulates 500 batches in `RemainingBatches` while waiting for L1 finality.
  2.  L1 finalizes a new state.
  3.  `processRemainingBatches` runs and iterates all 500 batches.
  4.  The "Finalized" check now passes for all of them.
  5.  The node executes 500 sequential synchronous RPC calls to the L1 node inside the `Update` loop.
- **Consequence**: This leads to the streamer freezing for seconds or minutes and stops fetching new Espresso blocks.

### 2. Infinite Buffer Growth
- **Background**: The function `HasNext` only returns `true` if `BatchBuffer.Peek() == BatchPos`.
- **Scenario**:
  1. The node is expecting Batch #100.
  2. Espresso delivers Batch #101, #102, ... #50,000.
  3. Batch #100 is still missing.
- **Consequence**: `BatchBuffer` has no size limit. It will accept and store Batches #101 through #50,000 in memory, waiting forever for #100. The node will run out of memory and crash. There is no mechanism to invalidate the stream if a batch is permanently lost.
