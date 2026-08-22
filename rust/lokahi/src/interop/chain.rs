//! The observation seams, over the real thing.
//!
//! [`lokahi_interop::InteropChain`] and [`lokahi_interop::L1Canonical`] are the two read-only
//! surfaces the verification round loop sees the world through. This module implements them over
//! what a running supernode already has: each chain's controller queue, the safe-head database
//! that controller records into, and the L1 execution-layer provider every chain derives from.
//!
//! Nothing here decides anything. Every method either answers, or says *why not yet* — and the
//! one permanent "never" ([`ChainAt::HistoryUnavailable`]) is reserved for the single condition
//! that really is unrecoverable: a pairing this node did not keep.

use alloy_eips::{BlockId, BlockNumHash, BlockNumberOrTag};
use alloy_primitives::{B256, ChainId, Log};
use alloy_provider::{Provider, RootProvider};
use async_trait::async_trait;
use kona_engine::{DenyList, DenyListReadError, EngineQueries};
use kona_genesis::RollupConfig;
use kona_node_service::{ChainControllerRequest, ChainControllerRpcRequest, RewindRequest};
use kona_protocol::{BlockInfo, L2BlockInfo, OutputRoot};
use kona_safedb::{SafeDbError, SharedSafeDb};
use lokahi_interop::{
    BlockLogs, ChainAt, ChainError, InteropChain, L1Canonical, RocksOutputArchive,
};
use op_alloy_network::Optimism;
use std::{fmt::Debug, sync::Arc};
use tokio::sync::{mpsc, oneshot};
use tracing::debug;

/// One L2 chain of a running supernode, as the verifier reads it.
///
/// The three sources are deliberately distinct and each answers only what it is authoritative
/// for. The controller queue answers about the *live* chain — the local-safe head, its L1 origin,
/// and output roots — and is the same queue the chain's own JSON-RPC server asks, so the verifier
/// and an operator watching that RPC cannot be told different things. The safe-head database
/// answers about *history*, which the live engine state cannot: it holds the L1 origin of exactly
/// one L2 block, the head. The execution layer answers about *block contents*, which neither of
/// the other two carries.
pub(crate) struct NodeChain {
    /// The chain's id.
    chain_id: ChainId,
    /// The chain's rollup config, which the verifier reads the block time and the interop
    /// activation time from.
    rollup_config: RollupConfig,
    /// The chain controller's read-only query queue.
    queries: mpsc::Sender<ChainControllerRpcRequest>,
    /// The safe-head database the chain's controller records local-safe advances into.
    safe_db: SharedSafeDb,
    /// A read-only execution-layer provider, for block contents.
    el: RootProvider<Optimism>,
}

impl Debug for NodeChain {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        // The provider and the queue render as noise; which chain this is, is the useful part.
        f.debug_struct("NodeChain").field("chain_id", &self.chain_id).finish_non_exhaustive()
    }
}

impl NodeChain {
    /// Builds the observation seam over one composed chain.
    pub(crate) const fn new(
        chain_id: ChainId,
        rollup_config: RollupConfig,
        queries: mpsc::Sender<ChainControllerRpcRequest>,
        safe_db: SharedSafeDb,
        el: RootProvider<Optimism>,
    ) -> Self {
        Self { chain_id, rollup_config, queries, safe_db, el }
    }

    /// Sends one [`EngineQueries`] to the chain controller's RPC peer and awaits its answer.
    ///
    /// A closed channel or a dropped responder is [`ChainError::Unreachable`] rather than a fatal
    /// error: during startup the chain's actors may not be stepping yet, and during shutdown they
    /// have stopped. Neither is a reason to halt cross-chain verification for good.
    async fn query<T>(
        &self,
        build: impl FnOnce(oneshot::Sender<T>) -> EngineQueries,
    ) -> Result<T, ChainError> {
        let (tx, rx) = oneshot::channel();
        self.queries.send(ChainControllerRpcRequest(Box::new(build(tx)))).await.map_err(|_| {
            ChainError::Unreachable("the chain controller is not accepting queries".into())
        })?;
        rx.await.map_err(|_| {
            // The query handler logs and swallows its own failures, so a dropped responder is
            // what a failed query looks like from here.
            ChainError::Unreachable("the chain controller did not answer".into())
        })
    }

    /// Returns the block at `number` as the engine describes it.
    ///
    /// A cross-safe promotion names a whole [`L2BlockInfo`], not a number, and the chain is the
    /// one authority on what that block is. Asking through the same
    /// [`EngineQueries::OutputAtBlock`] the RPC server uses reuses the engine's own decoding of
    /// it rather than growing a second one here.
    pub(crate) async fn block_at(&self, number: u64) -> Result<L2BlockInfo, ChainError> {
        self.query(|sender| EngineQueries::OutputAtBlock {
            block: BlockNumberOrTag::Number(number),
            sender,
        })
        .await
        .map(|(block, _output, _state)| block)
    }

    /// Resolves the pairing of the L2 block at `timestamp` from recorded history.
    ///
    /// Reached only when the live engine state answered
    /// [`LocalSafeAtTimestamp::BehindHead`](kona_engine::LocalSafeAtTimestamp::BehindHead) — the
    /// block *is* local-safe, and the question is only which L1 block made it so. That is the one
    /// question the live state cannot answer, and the safe-head database exists for it.
    ///
    /// The L1 answer is the earliest *recorded* block at which the safe head had reached this
    /// height. Recording granularity is one entry per engine drain, so it can be a later L1 block
    /// than the true one — never an earlier one. A consumer that reads it as "safe by this L1
    /// block", which is what cross-safety means, is therefore never told more than is true.
    async fn behind_head(&self, timestamp: u64) -> Result<ChainAt, ChainError> {
        let number = self.block_number_at_timestamp(timestamp).await?;
        let record = match self.safe_db.l1_at_safe_head(number) {
            Ok(record) => record,
            // The recorded tip has not reached this height. Transient by construction: this is a
            // timestamp behind the live local-safe head, so the record is on its way.
            Err(SafeDbError::L1AtSafeHeadNotFound | SafeDbError::NotFound) => {
                debug!(
                    target: "lokahi_interop",
                    chain_id = self.chain_id,
                    timestamp,
                    number,
                    "Safe-head history has not caught up with the local-safe head yet"
                );
                return Ok(ChainAt::NotYet);
            }
            // The records that would have answered are gone, or were never kept. The one
            // permanent verdict, and the one that halts the verifier.
            Err(SafeDbError::L1AtSafeHeadUnavailable | SafeDbError::NotEnabled) => {
                return Ok(ChainAt::HistoryUnavailable);
            }
            Err(err) => return Err(ChainError::Unreachable(format!("safe-head database: {err}"))),
        };

        // The record names the safe head as of that L1 block, which may be *above* the block the
        // timestamp asked about, so the hash has to come from the chain rather than the record.
        // Substituting the record's own head here would pair one block's number with another
        // block's hash.
        let block = self.block_info_at(number).await?;
        Ok(ChainAt::Derived { block: block.id(), l1: record.l1 })
    }

    /// Returns this chain's canonical block at `number`, as the verifier's block shape.
    pub(crate) async fn block_info_at(&self, number: u64) -> Result<BlockInfo, ChainError> {
        // Only the header is read: the round needs the block's identity and timestamp, and its
        // logs come from the receipts rather than from the transaction list.
        let block = self
            .el
            .get_block_by_number(BlockNumberOrTag::Number(number))
            .await
            .map_err(|err| ChainError::Unreachable(format!("eth_getBlockByNumber: {err}")))?
            // A height the execution layer has not imported yet, or has pruned. Transient: the
            // verifier only asks about blocks a safe head has already reached.
            .ok_or(ChainError::NotReady)?;
        let header = block.header;
        Ok(BlockInfo {
            hash: header.hash,
            number: header.number,
            parent_hash: header.parent_hash,
            timestamp: header.timestamp,
        })
    }
}

#[async_trait]
impl InteropChain for NodeChain {
    fn chain_id(&self) -> ChainId {
        self.chain_id
    }

    fn rollup_config(&self) -> &RollupConfig {
        &self.rollup_config
    }

    async fn local_safe_at(&self, timestamp: u64) -> Result<ChainAt, ChainError> {
        let snapshot =
            self.query(|sender| EngineQueries::LocalSafeSnapshotAt { timestamp, sender }).await?;

        // The live state answers for the head's own timestamp and rules out the impossible ones;
        // `None` is its redirection to history, and the only case that reads anything else.
        match ChainAt::from_snapshot(&snapshot) {
            Some(at) => Ok(at),
            None => self.behind_head(timestamp).await,
        }
    }

    async fn block_logs(&self, block: BlockNumHash) -> Result<BlockLogs, ChainError> {
        let info = self.block_info_at(block.number).await?;
        // Asked by number above and checked against the hash here. The caller treats a block
        // other than the one it asked for as a broken invariant, so a chain that reorged under
        // the round has to come back as "not yet" and be re-observed, not as a different block.
        if info.hash != block.hash {
            debug!(
                target: "lokahi_interop",
                chain_id = self.chain_id,
                number = block.number,
                asked = %block.hash,
                found = %info.hash,
                "Canonical block at that height is no longer the one the round observed"
            );
            return Err(ChainError::NotReady);
        }

        let receipts = self
            .el
            .get_block_receipts(BlockId::Hash(block.hash.into()))
            .await
            .map_err(|err| ChainError::Unreachable(format!("eth_getBlockReceipts: {err}")))?
            .ok_or(ChainError::NotReady)?;

        // Receipts come in transaction order and each one's logs in emission order, so flattening
        // reproduces the block's global log index sequence — which is what an initiating
        // message's index is meaningful against.
        let logs = receipts
            .iter()
            .flat_map(|receipt| receipt.inner.logs())
            .map(|log| log.inner.clone())
            .collect::<Vec<Log>>();

        Ok(BlockLogs { block: info, logs })
    }

    async fn output_at(&self, number: u64) -> Result<OutputRoot, ChainError> {
        // The same query as `Self::block_at`, asked for its other half. Kept separate rather than
        // returned as a pair, because no caller ever wants both: a promotion needs the block, and
        // only an invalidation needs the commitment.
        self.query(|sender| EngineQueries::OutputAtBlock {
            block: BlockNumberOrTag::Number(number),
            sender,
        })
        .await
        .map(|(_block, output, _state)| output)
    }

    async fn first_safe_head_timestamp(&self) -> Result<u64, ChainError> {
        let record = match self.safe_db.first_entry() {
            Ok(record) => record,
            // A cold-starting node has recorded nothing yet. This is the normal answer while its
            // chains catch up, and cold start waits it out.
            Err(SafeDbError::NotFound) => return Err(ChainError::NotReady),
            Err(err) => return Err(ChainError::Unreachable(format!("safe-head database: {err}"))),
        };
        Ok(self.timestamp_at_block_number(record.safe_head.number))
    }

    async fn block_number_at_timestamp(&self, timestamp: u64) -> Result<u64, ChainError> {
        let genesis = &self.rollup_config.genesis;
        let Some(elapsed) = timestamp.checked_sub(genesis.l2_time) else {
            // Before genesis there is no earlier block to floor onto, so genesis is the answer.
            return Ok(genesis.l2.number);
        };
        // Blocks are spaced exactly `block_time` apart from genesis onwards, so the arithmetic is
        // the whole answer and asking the execution layer would only be a slower way to compute
        // it. Integer division is the flooring the trait asks for.
        Ok(genesis.l2.number + elapsed / self.rollup_config.block_time.max(1))
    }
}

impl NodeChain {
    /// The timestamp of this chain's block at `number`.
    ///
    /// The inverse of [`InteropChain::block_number_at_timestamp`], and exact for the same reason:
    /// block spacing is fixed by the rollup config.
    fn timestamp_at_block_number(&self, number: u64) -> u64 {
        let genesis = &self.rollup_config.genesis;
        genesis.l2_time +
            number.saturating_sub(genesis.l2.number) * self.rollup_config.block_time.max(1)
    }
}

/// The rewind seam over one composed chain: the write half of applying an invalidation.
///
/// Reads through the same [`NodeChain`] the verifier observes with, and writes through the chain
/// controller's request channel — the only writer of the chain's heads — so the rewind is ordered
/// with everything else that moves them.
pub(crate) struct ChainRewindRoute {
    /// The chain, for reading what it currently carries at the invalidated height.
    chain: Arc<NodeChain>,
    /// The chain controller's request channel, which applies the rewind.
    requests: mpsc::Sender<ChainControllerRequest>,
}

impl Debug for ChainRewindRoute {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("ChainRewindRoute").field("chain", &self.chain).finish_non_exhaustive()
    }
}

impl ChainRewindRoute {
    /// Builds the rewind seam over one composed chain.
    pub(crate) const fn new(
        chain: Arc<NodeChain>,
        requests: mpsc::Sender<ChainControllerRequest>,
    ) -> Self {
        Self { chain, requests }
    }
}

#[async_trait]
impl lokahi_interop::RewindableChain for ChainRewindRoute {
    /// The decision structure is op-supernode's `InvalidateBlock`
    /// (`op-supernode/supernode/chain_container/invalidation.go:438-465`): a canonical block at
    /// the height that is *not* the invalidated one means the replacement already stands, and the
    /// rewind is skipped; a missing block is a partially applied rewind from a previous crash and
    /// is driven to completion; only the invalidated block itself still standing starts a fresh
    /// rewind.
    async fn rewind_off(&self, invalidated: BlockNumHash) -> Result<bool, ChainError> {
        match self.chain.block_info_at(invalidated.number).await {
            Ok(info) if info.hash != invalidated.hash => {
                debug!(
                    target: "lokahi_interop",
                    number = invalidated.number,
                    invalidated = %invalidated.hash,
                    canonical = %info.hash,
                    "The canonical block differs from the invalidated one; no rewind needed"
                );
                return Ok(false);
            }
            // Still canonical: rewind. Or no canonical block at the height at all — a prior
            // crashed attempt may have left it that way, so the rewind is driven to completion
            // rather than skipped.
            Ok(_) | Err(ChainError::NotReady) => {}
            Err(err) => return Err(err),
        }

        // The forkchoice target needs the parent as a full `L2BlockInfo`, and the chain is the
        // authority on what that block is. Read at apply time rather than captured at decision
        // time: canonicality below the invalidated height is untouched by the rewind, so the
        // canonical block at `number - 1` is the invalidated block's parent on every retry.
        let parent = self.chain.block_at(invalidated.number - 1).await?;
        if parent.block_info.number != invalidated.number - 1 {
            return Err(ChainError::Unreachable(format!(
                "asked for block {} and was answered with block {}",
                invalidated.number - 1,
                parent.block_info.number
            )));
        }

        let (result_tx, mut result_rx) = mpsc::channel(1);
        self.requests
            .send(ChainControllerRequest::Rewind(Box::new(RewindRequest { parent, result_tx })))
            .await
            .map_err(|_| {
                ChainError::Unreachable("the chain controller is not accepting requests".into())
            })?;
        result_rx
            .recv()
            .await
            .ok_or_else(|| {
                ChainError::Unreachable("the chain controller dropped the rewind request".into())
            })?
            .map_err(|err| ChainError::Unreachable(format!("rewind failed: {err}")))?;

        Ok(true)
    }
}

/// The invalidated-output archive, read as the deny list the chain's engine consults.
///
/// One record serves both: the archive keeps the invalidated block's output preimage for the
/// optimistic superroot branch, and its existence at a `(height, hash)` is the denial — the same
/// doubling op-supernode's deny list does, whose records carry the output preimage fields too.
#[derive(Debug)]
pub(crate) struct ArchiveDenyList {
    /// The archive backing the answers.
    archive: Arc<RocksOutputArchive>,
}

impl ArchiveDenyList {
    /// Builds the deny-list view over a chain's archive.
    pub(crate) const fn new(archive: Arc<RocksOutputArchive>) -> Self {
        Self { archive }
    }
}

impl DenyList for ArchiveDenyList {
    fn is_denied(&self, number: u64, hash: B256) -> Result<bool, DenyListReadError> {
        self.archive
            .get(number, hash)
            .map(|output| output.is_some())
            .map_err(|err| DenyListReadError(err.to_string()))
    }

    fn max_denied_height(&self) -> Result<Option<u64>, DenyListReadError> {
        self.archive.max_height().map_err(|err| DenyListReadError(err.to_string()))
    }
}

/// The L1 every chain of the cluster derives from, as the verifier reads it.
///
/// One provider for the whole process, matching the single L1 watcher the supernode runs: the
/// chains of an interop cluster share an L1 by definition, and asking two providers could answer
/// two different canonical chains.
#[derive(Debug, Clone)]
pub(crate) struct L1Provider {
    /// The L1 execution-layer provider.
    el: RootProvider,
}

impl L1Provider {
    /// Builds the L1 seam over an execution-layer provider.
    pub(crate) const fn new(el: RootProvider) -> Self {
        Self { el }
    }
}

#[async_trait]
impl L1Canonical for L1Provider {
    async fn canonical_hash_at(&self, number: u64) -> Result<B256, ChainError> {
        // A height the L1 has not reached, or has pruned, is a "not yet" rather than a verdict:
        // the verifier's question is whether a block it relied on is *still* canonical, and an
        // unanswerable height must not be reported as "no".
        self.el
            .get_block_by_number(BlockNumberOrTag::Number(number))
            .await
            .map_err(|err| ChainError::Unreachable(format!("eth_getBlockByNumber: {err}")))?
            .map(|block| block.header.hash)
            .ok_or(ChainError::NotReady)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_genesis::ChainGenesis;
    use kona_safedb::{SafeDb, SafeHeadRecord};
    use std::sync::Arc;
    use url::Url;

    /// A safe-head database that answers every read with one chosen error.
    ///
    /// The mapping from those errors onto [`ChainAt`] is the whole decision this seam makes on the
    /// verifier's behalf: one of them halts verification for good and the others cost a retry, so
    /// getting it wrong is either a healthy node that stops or a broken one that spins.
    #[derive(Debug)]
    struct FailingSafeDb(fn() -> SafeDbError);

    impl SafeDb for FailingSafeDb {
        fn enabled(&self) -> bool {
            true
        }

        fn safe_head_updated(
            &self,
            _safe_head: kona_protocol::L2BlockInfo,
            _l1_head: BlockNumHash,
        ) -> Result<(), SafeDbError> {
            Ok(())
        }

        fn safe_head_reset(
            &self,
            _safe_head: kona_protocol::L2BlockInfo,
        ) -> Result<(), SafeDbError> {
            Ok(())
        }

        fn safe_head_at_l1(&self, _l1_block_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
            Err((self.0)())
        }

        fn first_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
            Err((self.0)())
        }

        fn last_entry(&self) -> Result<SafeHeadRecord, SafeDbError> {
            Err((self.0)())
        }

        fn l1_at_safe_head(&self, _target_l2_num: u64) -> Result<SafeHeadRecord, SafeDbError> {
            Err((self.0)())
        }

        fn close(&self) -> Result<(), SafeDbError> {
            Ok(())
        }
    }

    /// A chain whose history reads all fail with `err`.
    ///
    /// The execution layer is never reached: every case below is decided before the block hash is
    /// looked up, which is itself part of what is being asserted — a chain whose history is gone
    /// must not be waiting on an RPC to find that out.
    fn chain_with(err: fn() -> SafeDbError) -> NodeChain {
        let (queries, _rx) = mpsc::channel(1);
        NodeChain::new(
            901,
            RollupConfig {
                block_time: 2,
                genesis: ChainGenesis { l2_time: 1_000, ..Default::default() },
                ..Default::default()
            },
            queries,
            Arc::new(FailingSafeDb(err)),
            RootProvider::<Optimism>::new_http(Url::parse("http://127.0.0.1:1/").unwrap()),
        )
    }

    /// The recorded tip has not reached the height yet. The block *is* local-safe — that is why
    /// this path was taken — so the record is on its way and waiting is right.
    #[tokio::test]
    async fn a_history_tip_behind_the_query_is_a_wait() {
        let chain = chain_with(|| SafeDbError::L1AtSafeHeadNotFound);
        assert_eq!(chain.behind_head(1_100).await.unwrap(), ChainAt::NotYet);
    }

    /// An empty database is the same wait: nothing has been recorded yet.
    #[tokio::test]
    async fn an_empty_history_is_a_wait() {
        let chain = chain_with(|| SafeDbError::NotFound);
        assert_eq!(chain.behind_head(1_100).await.unwrap(), ChainAt::NotYet);
    }

    /// The records that would have answered are gone. This is the one condition no retry fixes,
    /// and the only one that may halt the verifier.
    #[tokio::test]
    async fn a_pruned_history_is_permanent() {
        let chain = chain_with(|| SafeDbError::L1AtSafeHeadUnavailable);
        assert_eq!(chain.behind_head(1_100).await.unwrap(), ChainAt::HistoryUnavailable);
    }

    /// A host running the verifier against a database that records nothing can never answer this
    /// question, so it is permanent rather than a retry that would spin forever.
    #[tokio::test]
    async fn a_disabled_history_is_permanent() {
        let chain = chain_with(|| SafeDbError::NotEnabled);
        assert_eq!(chain.behind_head(1_100).await.unwrap(), ChainAt::HistoryUnavailable);
    }

    /// A damaged or closed store is neither of the above: it says nothing about the chain, so it
    /// is reported as a failed read and retried.
    #[tokio::test]
    async fn a_broken_store_is_a_failed_read() {
        let chain = chain_with(|| SafeDbError::Closed);
        assert!(matches!(chain.behind_head(1_100).await, Err(ChainError::Unreachable(_))));
    }

    /// Timestamps between two blocks floor onto the earlier one, which is the block that carries
    /// them as far as a round is concerned.
    #[tokio::test]
    async fn a_timestamp_between_blocks_floors_onto_the_earlier_one() {
        let chain = chain_with(|| SafeDbError::NotFound);
        // Genesis is block 0 at t=1000 with a two-second block time.
        assert_eq!(chain.block_number_at_timestamp(1_000).await.unwrap(), 0);
        assert_eq!(chain.block_number_at_timestamp(1_001).await.unwrap(), 0);
        assert_eq!(chain.block_number_at_timestamp(1_002).await.unwrap(), 1);
        assert_eq!(chain.block_number_at_timestamp(1_003).await.unwrap(), 1);
    }

    /// A timestamp before genesis has no earlier block to floor onto, so genesis is the answer;
    /// whether that timestamp is verifiable at all is the caller's question, not this one's.
    #[tokio::test]
    async fn a_timestamp_before_genesis_answers_with_genesis() {
        let chain = chain_with(|| SafeDbError::NotFound);
        assert_eq!(chain.block_number_at_timestamp(1).await.unwrap(), 0);
    }

    /// The two directions have to agree, or the timestamp cold start picks from a chain's first
    /// recorded safe head would name a different block than the round starting there asks for.
    #[tokio::test]
    async fn the_block_and_timestamp_conversions_are_inverses() {
        let chain = chain_with(|| SafeDbError::NotFound);
        for number in 0..64u64 {
            let timestamp = chain.timestamp_at_block_number(number);
            assert_eq!(
                chain.block_number_at_timestamp(timestamp).await.unwrap(),
                number,
                "block {number} at timestamp {timestamp}"
            );
        }
    }
}
