package db

// 3 types of databases:
// --------------------
// 1) The canonical events data, available as soon as seen in unsafe context (block building, p2p blocks, etc.).
//    -> Of the Events dependency-index type.
//    -> Primary key is block seal of parent-block that we apply txs and thus log events on top of.
// 2) The best-effort local safe data, optimistic: we store what the op-node is able to derive locally, without cross-L2 checks.
//    -> when local safe is not cross-valid: truncate to when this was derived, then mark it as bad (use an entry flag byte).
//    -> Primary key is L1, we register each L2 block when we know it is derived (in local view terms) from a L1 block.
//    -> Of the "FromDA" dependency index type.
// 3) The absolute cross-safe, synchronous:
//    -> Check if next local-safe after cross-safe meets all dependencies (incl intra block),
//       conditional on knowing the local-safe data (check L1 hash), and logs (check L2 block seal).
//       If yes, then add a cross-safe entry. If not available, wait.
//       If conflicting, enter local-safe cross-invalid path (see above). It'll be remembered, and we move on with an empty block.
//    -> Only roll back cross-safe when L1 reorgs.
//    -> Primary key is L1, we register each L2 block derived from the L1 block, as soon as it becomes cross-safe.
//       Might take until a later L1 block than the local derived-from, since dependencies may be batch-submitted late.
//    -> Of the "FromDA" dependency index type.

// TODO: can we fully reuse the "FromDA" type for both local and cross safe?
// We do need that marker to remember and move on after invalid local-safe input data.
// Primary key nuance is also different. Where "cross" version really implies "derived" transitively for all L2 chains.

// "FromDA" dependency DB type.
// --------------------
// Notes:
// - Serves both "Local-safe" and "Cross-safe" DBs.
// - Derived L2 blocks follow *after* a seal of the *current* derived-from L1 block.
// - Checkpoints should contain both blocks since last derived-from change, and absolute L2 block num
//
// Interface:
// - AddDerived(derivedFrom eth.BlockID, derived eth.BlockRef) -> whenever a L2 block is derived from a L1 block
// - SealDerivedFrom(derivedFrom eth.BlockRef)    -> we register every L1 block
// - Rewind(newHeadBlockNum uint64) error
//
// - LatestDerivedFrom() eth.BlockRef          -> return last known primary key (the L1 block)
// - LatestDerived() eth.BlockRef              -> return last known value (the L2 block that was derived)
// - LastDerivedAt(derivedFrom eth.BlockID) eth.BlockRef         -> historical lookup func, used for finality
// - IteratorStartingAt(derivedFrom uint64, blocksSince uint32) (Iterator, error)      -> for tools, debugging, etc.
// - DerivedFrom(derived eth.BlockID) (eth.BlockRef, error)   -> to support sync-start / tools; determine where a L2 block was derived from

// Events (after cleanup):     -> note: logs follow *after* a seal of the *parent* block.
// --------------------
// Notes:
// - Cleanup is minor API change (drop unused return values and unused public functions)
//
// Interface:
// - SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error
// - AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *types.ExecutingMessage) error
//
// - Rewind(newHeadBlockNum uint64) error
//
// - Contains(blockNum uint64, logIdx uint32, logHash common.Hash) error // don't return entry index
// - HasSealedBlock(block eth.BlockID) error     // don't return entry index
// - LatestSealedBlockNum() (n uint64, ok bool)
// - IteratorStartingAt(sealedNum uint64, logsSince uint32) (Iterator, error)  // maybe simplify iterator public funcs, single conditional iterator func is nice

// Events (legacy, pre-cleanup):
// --------------------
// Notes:
// - Currently implemented
// - Used to capture canonical block info of the unsafe chain
//
// Interface:
// - SealBlock, AddLog, Rewind
// - Contains(blockNum uint64, logIdx uint32, logHash common.Hash) (entrydb.EntryIdx, error)
// - LatestSealedBlockNum() (n uint64, ok bool)
// - FindSealedBlock(block eth.BlockID) (nextEntry entrydb.EntryIdx, err error)
// - IteratorStartingAt(sealedNum uint64, logsSince uint32) (Iterator, error)

// Op-node data inputs:
// --------------------
// - UpdateLocalUnsafe(chainID types.ChainID, ref eth.BlockRef)
//    -> fetch events, append them
//    -> seal block
// - UpdateLocalSafe(chainID types.ChainID, at eth.BlockRef, ref eth.BlockRef) error
//    -> add local-verified
// - UpdateFinalizeL1(ref eth.BlockRef) error
//    -> warn if older than last known finalized block. If newer, store as finalized L1 block.

// Block safety queries:
// --------------------
// - UnsafeL2(chainID types.ChainID) (heads.HeadPointer, error)
//    -> return tip of events DB (can be in-progress block)
// - CrossUnsafeL2(chainID types.ChainID) (heads.HeadPointer, error)
//    -> return verification progress of events DB (can be in-progress block)
// - LocalSafeL2(chainID types.ChainID) (eth.BlockID, error)
//    -> return tip of the local safe derived-from DB
// - CrossSafeL2(chainID types.ChainID) (eth.BlockID, error)
//    -> return tip of the cross-safe derived-from DB
// - FinalizedL2(chainId types.ChainID) (eth.BlockID, error)
//    -> return last cross-safe verified block at the point of L1 finality
//        -> If the finality signal is newer than what we know of,
//        then take the latest info we do know, and check if it's present in the finalized L1 chain.
