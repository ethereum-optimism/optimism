//! One hosted chain, as the supernode's query API reads it.
//!
//! Every answer comes out of the chain's own queues rather than out of a second view assembled
//! here. The sync status is produced by [`kona_rpc::RollupRpc`] — the very object serving that
//! chain's `optimism_syncStatus` — so an operator watching one chain and a proposer reading the
//! aggregate cannot be told different things. The optimistic pairing comes from
//! [`kona_engine::EngineQueries::LocalSafeSnapshotAt`], one borrow of the engine state, and its L1
//! half from the safe-head database that same chain's controller writes.
//!
//! op-supernode's `ChainContainer` does the same job with two independent reads —
//! `LocalSafeBlockAtTimestamp` then `safeDBAtL2`, each sampling the sync status again — and
//! documents the window between them. There is no window here.

use crate::query::error::QueryError;
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, ChainId};
use kona_engine::LocalSafeSnapshot;
use kona_genesis::RollupConfig;
use kona_node_service::QueuedEngineRpcClient;
use kona_protocol::{OutputRoot, SyncStatus};
use kona_rpc::{EngineRpcClient, L1WatcherQuerySender, RollupNodeApiServer, RollupRpc};
use kona_safedb::{SafeDbError, SharedSafeDb};
use lokahi_interop::ChainAt;
use std::sync::Arc;
use tracing::debug;

/// One chain's optimistic output at a timestamp: the block, its output-root preimage, and the
/// lowest L1 block the pair can be derived from.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct OptimisticOutput {
    /// The L2 block carrying the timestamp.
    pub(crate) block: BlockNumHash,
    /// That block's output-root preimage.
    pub(crate) output: OutputRoot,
    /// The lowest L1 block from which the block can be derived.
    pub(crate) required_l1: BlockNumHash,
}

/// One hosted chain's read-only surface for the supernode query API.
#[derive(Debug)]
pub(crate) struct QueryChain {
    /// The chain's id.
    chain_id: ChainId,
    /// The chain's rollup config, which the timestamp-to-block arithmetic comes from.
    rollup_config: Arc<RollupConfig>,
    /// The chain controller's query queue, which the engine answers through.
    engine: QueuedEngineRpcClient,
    /// kona's rollup RPC over this chain's queues.
    rollup: RollupRpc<QueuedEngineRpcClient>,
    /// The safe-head database the chain's controller records local-safe advances into.
    safe_db: SharedSafeDb,
}

impl QueryChain {
    /// Builds the query surface over one composed chain's channels.
    pub(crate) fn new(
        chain_id: ChainId,
        rollup_config: Arc<RollupConfig>,
        engine: QueuedEngineRpcClient,
        l1_queries: L1WatcherQuerySender,
        safe_db: SharedSafeDb,
    ) -> Self {
        Self {
            chain_id,
            rollup_config,
            rollup: RollupRpc::new(engine.clone(), l1_queries, chain_id),
            engine,
            safe_db,
        }
    }

    /// Returns which chain this is.
    pub(crate) const fn chain_id(&self) -> ChainId {
        self.chain_id
    }

    /// Returns this chain's sync status, as its own `optimism_syncStatus` reports it.
    ///
    /// Served through [`RollupRpc`] rather than reassembled here, so the aggregate cannot drift
    /// from the per-chain answer. The call is counted in that chain's `rollup_rpc` metric, which
    /// is what it is: an `op_syncStatus` served for that chain.
    pub(crate) async fn sync_status(&self) -> Result<SyncStatus, QueryError> {
        self.rollup
            .op_sync_status()
            .await
            .map_err(|err| QueryError::Chain { chain_id: self.chain_id, source: err.to_string() })
    }

    /// Returns this chain's optimistic output at `timestamp`, or [`None`] when the chain has not
    /// derived that far yet.
    ///
    /// This is op-supernode's `OptimisticOutputAtTimestamp` and `OptimisticAt` in one pass, and
    /// the pass matters: those two resolve the same block number independently, and this resolves
    /// it once and checks the block it reads back against the block it was promised. A response
    /// that pairs one block's output root with another block's L1 requirement is well-formed and
    /// wrong, which is worse than absent.
    ///
    /// [`None`] is op-supernode's `ethereum.NotFound`: the chain is left out of the optimistic
    /// map, and the caller reports no super root rather than a partial one. An error is one of the
    /// two conditions op-supernode also fails the whole call on — a timestamp before this chain's
    /// genesis, which `TargetBlockNumber` rejects, and safe-head history that is gone for good.
    ///
    /// One difference from op-supernode, and it is a gap rather than a choice: op-supernode
    /// consults its deny list first (`LastDeniedOutputV0`), so for a block that was invalidated
    /// and replaced it reports the *original* block's output — which is what "optimistically, had
    /// verification succeeded" means once a replacement exists. lokahi has no invalidated-output
    /// archive yet, so this reports the chain's current canonical output at that height. The two
    /// agree everywhere no block has been invalidated, and diverge at exactly the heights where
    /// one has.
    pub(crate) async fn optimistic_at(
        &self,
        timestamp: u64,
    ) -> Result<Option<OptimisticOutput>, QueryError> {
        let snapshot = self.local_safe_snapshot_at(timestamp).await?;

        // `from_snapshot` answers from the live state, or returns `None` to say the pairing is
        // behind the head and lives in recorded history. The same mapping the interop verifier
        // reads this state through, so the two cannot disagree about what the chain has derived.
        let (number, required_l1) = match ChainAt::from_snapshot(&snapshot) {
            Some(ChainAt::Derived { block, l1 }) => (block.number, l1),
            Some(ChainAt::NotYet) => return Ok(None),
            Some(ChainAt::BeforeGenesis) => {
                return Err(QueryError::BeforeGenesis { chain_id: self.chain_id, timestamp });
            }
            // Not reachable: only the history lookup below can find history missing.
            Some(ChainAt::HistoryUnavailable) => {
                return Err(QueryError::HistoryUnavailable { chain_id: self.chain_id, timestamp });
            }
            None => match self.required_l1_from_history(timestamp)? {
                Some(pairing) => pairing,
                None => return Ok(None),
            },
        };

        let (block, output, _state) = self
            .rollup
            .engine_client
            .output_at_block(number.into())
            .await
            .map_err(|err| QueryError::Chain {
                chain_id: self.chain_id,
                source: format!("output at block {number}: {err}"),
            })?;

        Ok(Some(OptimisticOutput { block: block.block_info.id(), output, required_l1 }))
    }

    /// Returns the output root of the block the verifier committed to at `timestamp`.
    ///
    /// op-supernode reads this by block hash (`OutputRootAtL2BlockHash`), and an execution layer
    /// that no longer has that hash on its canonical chain answers `NotFound`, which fails the
    /// call. The read here is by number and the hash is verified against what came back, which
    /// rejects the same case: a chain that reorged below a block the verifier declared cross-safe
    /// would otherwise contribute the output root of a block nothing verified, and the super root
    /// over it would be well-formed and wrong. The verifier reports the reorg on its own next
    /// round; until then this call fails rather than answering.
    pub(crate) async fn output_root_at(
        &self,
        timestamp: u64,
        verified: BlockNumHash,
    ) -> Result<B256, QueryError> {
        let (block, output, _state) = self
            .rollup
            .engine_client
            .output_at_block(verified.number.into())
            .await
            .map_err(|err| QueryError::Chain {
                chain_id: self.chain_id,
                source: format!("output at verified block {}: {err}", verified.number),
            })?;

        let canonical = block.block_info.hash;
        if canonical != verified.hash {
            return Err(QueryError::ReorgedBelowVerified {
                timestamp,
                chain_id: self.chain_id,
                number: verified.number,
                verified: verified.hash,
                canonical,
            });
        }

        Ok(output.hash())
    }

    /// Reads the engine's local-safe snapshot at `timestamp`.
    async fn local_safe_snapshot_at(
        &self,
        timestamp: u64,
    ) -> Result<LocalSafeSnapshot, QueryError> {
        self.engine.local_safe_snapshot_at(timestamp).await.map_err(|err| {
            QueryError::Chain {
                chain_id: self.chain_id,
                source: format!("local-safe snapshot at {timestamp}: {err}"),
            }
        })
    }

    /// Resolves the block at `timestamp` and the L1 block it became safe at, from recorded
    /// history.
    ///
    /// Reached only when the live state said the timestamp is *behind* the local-safe head, so the
    /// block is local-safe and the only open question is which L1 block made it so. The recorded
    /// L1 is the earliest entry whose safe head had reached this height, which the recording
    /// granularity can round up but never down — so a consumer reading it as "safe by this L1
    /// block" is never told more than is true.
    fn required_l1_from_history(
        &self,
        timestamp: u64,
    ) -> Result<Option<(u64, BlockNumHash)>, QueryError> {
        let number = self.block_number_at_timestamp(timestamp);
        match self.safe_db.l1_at_safe_head(number) {
            Ok(record) => Ok(Some((number, record.l1))),
            // The recorded tip has not reached this height yet. Transient by construction, since
            // the block is already local-safe.
            Err(SafeDbError::L1AtSafeHeadNotFound | SafeDbError::NotFound) => {
                debug!(
                    target: "lokahi",
                    chain_id = self.chain_id,
                    timestamp,
                    number,
                    "Safe-head history has not caught up with the local-safe head yet"
                );
                Ok(None)
            }
            // The records that would have answered are gone, or were never kept. op-supernode
            // fails the call here too, through `ErrHistoryUnavailable`.
            Err(SafeDbError::L1AtSafeHeadUnavailable | SafeDbError::NotEnabled) => {
                Err(QueryError::HistoryUnavailable { chain_id: self.chain_id, timestamp })
            }
            Err(err) => Err(QueryError::Chain {
                chain_id: self.chain_id,
                source: format!("safe-head database: {err}"),
            }),
        }
    }

    /// The number of the block carrying `timestamp`, flooring onto the preceding block.
    ///
    /// Block spacing is fixed by the rollup config from genesis onwards, so this is the whole
    /// answer — op-supernode's `TargetBlockNumber` computes it the same way. A timestamp before
    /// genesis never reaches here: the snapshot reports it as `BeforeGenesis` first.
    fn block_number_at_timestamp(&self, timestamp: u64) -> u64 {
        let genesis = &self.rollup_config.genesis;
        let elapsed = timestamp.saturating_sub(genesis.l2_time);
        genesis.l2.number + elapsed / self.rollup_config.block_time.max(1)
    }
}
