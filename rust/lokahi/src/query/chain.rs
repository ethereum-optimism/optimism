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
use lokahi_interop::{ChainAt, RocksOutputArchive};
use std::sync::Arc;
use tracing::debug;

/// One chain's optimistic output at a timestamp: the output-root preimage of the block carrying
/// it, and the lowest L1 block that block can be derived from.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct OptimisticOutput {
    /// The output-root preimage of the block carrying the timestamp.
    ///
    /// The block's own id is not kept: `eth.OutputWithRequiredL1` does not carry one, and the
    /// preimage already names the block hash it commits to.
    pub(crate) output: OutputRoot,
    /// The lowest L1 block from which the block can be derived.
    pub(crate) required_l1: BlockNumHash,
}

/// One chain's block at a timestamp, and the L1 block that made it safe.
///
/// Named rather than a pair, because the two halves are only meaningful together: a response that
/// puts one block's output root next to another block's L1 requirement is well-formed and wrong,
/// and a `(u64, BlockNumHash)` is exactly the shape that lets that happen unnoticed.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct Pairing {
    /// The number of the L2 block carrying the timestamp.
    number: u64,
    /// The lowest L1 block the block can be derived from.
    required_l1: BlockNumHash,
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
    /// The chain's invalidated-output archive, consulted before the canonical output.
    ///
    /// op-supernode's deny list, in its optimistic role: `OptimisticOutputAtTimestamp` asks
    /// `LastDeniedOutputV0` first, so a height whose block was invalidated and replaced reports
    /// the *original* block's output — "optimistically, had verification succeeded" — rather
    /// than the replacement's. [`None`] where the chain has no archive (interop off, so nothing
    /// is ever invalidated), which is also the shape of op-node's own single-chain `superroot`
    /// route: an op-node consults no deny list, so a chain route built without an archive
    /// answers exactly as op-node does.
    archive: Option<Arc<RocksOutputArchive>>,
}

impl QueryChain {
    /// Builds the query surface over one composed chain's channels.
    pub(crate) fn new(
        chain_id: ChainId,
        rollup_config: Arc<RollupConfig>,
        engine: QueuedEngineRpcClient,
        l1_queries: L1WatcherQuerySender,
        safe_db: SharedSafeDb,
        archive: Option<Arc<RocksOutputArchive>>,
    ) -> Self {
        Self {
            chain_id,
            rollup_config,
            rollup: RollupRpc::new(engine.clone(), l1_queries, chain_id),
            engine,
            safe_db,
            archive,
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
            .map_err(|err| QueryError::Chain { chain_id: self.chain_id, reason: err.to_string() })
    }

    /// Returns this chain's optimistic output at `timestamp`, or [`None`] when the chain has not
    /// derived that far yet.
    ///
    /// This is op-supernode's `OptimisticOutputAtTimestamp` and `OptimisticAt` in one pass, and
    /// the pass matters. Those two each sample the chain's sync status again and resolve the block
    /// number independently, and op-supernode documents the window between them; here the block
    /// number and its L1 pairing come out of one borrow of the engine state, so there is no window
    /// in which they can come to describe different blocks. The output root is then read at that
    /// number, which is what op-supernode does too — an optimistic output is pre-verification and
    /// reorg-able by construction, and this branch says so.
    ///
    /// [`None`] is op-supernode's `ethereum.NotFound`: the chain is left out of the optimistic
    /// map, and the caller reports no super root rather than a partial one. An error is one of the
    /// two conditions op-supernode also fails the whole call on — a timestamp before this chain's
    /// genesis, which `TargetBlockNumber` rejects, and safe-head history that is gone for good.
    ///
    /// The archive is consulted before the canonical read, exactly where op-supernode consults
    /// its deny list (`OptimisticOutputAtTimestamp` asks `LastDeniedOutputV0` first): a height
    /// whose block was invalidated and replaced reports the *original* block's output — which is
    /// what "optimistically, had verification succeeded" means once a replacement exists. The L1
    /// pairing is not archived, and op-supernode's `OptimisticAt` resolves it from the live chain
    /// there too. An archive read error fails the call, as op-supernode's does.
    pub(crate) async fn optimistic_at(
        &self,
        timestamp: u64,
    ) -> Result<Option<OptimisticOutput>, QueryError> {
        let snapshot = self.local_safe_snapshot_at(timestamp).await?;

        // `from_snapshot` answers from the live state, or returns `None` to say the pairing is
        // behind the head and lives in recorded history. The same mapping the interop verifier
        // reads this state through, so the two cannot disagree about what the chain has derived.
        let Pairing { number, required_l1 } = match ChainAt::from_snapshot(&snapshot) {
            Some(ChainAt::Derived { block, l1 }) => {
                Pairing { number: block.number, required_l1: l1 }
            }
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

        if let Some(archived) = self.last_archived_at(number)? {
            return Ok(Some(OptimisticOutput { output: archived, required_l1 }));
        }

        let (_block, output, _state) =
            self.rollup.engine_client.output_at_block(number.into()).await.map_err(|err| {
                QueryError::Chain {
                    chain_id: self.chain_id,
                    reason: format!("output at block {number}: {err}"),
                }
            })?;

        Ok(Some(OptimisticOutput { output, required_l1 }))
    }

    /// Returns the most recently archived (invalidated) output at `number`, if any.
    ///
    /// op-supernode: `LastDeniedOutputV0`, and the same error posture — a deny-list read failure
    /// fails the whole call rather than silently answering with the replacement's output.
    fn last_archived_at(&self, number: u64) -> Result<Option<OutputRoot>, QueryError> {
        let Some(archive) = &self.archive else { return Ok(None) };
        archive
            .last_at(number)
            .map(|archived| archived.map(|archived| archived.output_root))
            .map_err(|err| QueryError::Chain {
                chain_id: self.chain_id,
                reason: format!("invalidated-output archive at height {number}: {err}"),
            })
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
        let (block, output, _state) =
            self.rollup.engine_client.output_at_block(verified.number.into()).await.map_err(
                |err| QueryError::Chain {
                    chain_id: self.chain_id,
                    reason: format!("output at verified block {}: {err}", verified.number),
                },
            )?;

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
        self.engine.local_safe_snapshot_at(timestamp).await.map_err(|err| QueryError::Chain {
            chain_id: self.chain_id,
            reason: format!("local-safe snapshot at {timestamp}: {err}"),
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
    fn required_l1_from_history(&self, timestamp: u64) -> Result<Option<Pairing>, QueryError> {
        let number = self.block_number_at_timestamp(timestamp);

        // Genesis L2 is trivially safe at L1 block 0, answered before the database is consulted —
        // op-supernode's own guard (`op-supernode/supernode/chain_container/virtual_node/
        // virtual_node.go:263-269`). It uses block 0 rather than the L2 genesis's L1 origin
        // because the dispute contracts may predate that origin, allowing games anchored to
        // earlier L1 heads; the zero hash is what its `eth.BlockID{Number: 0}` carries.
        if number == self.rollup_config.genesis.l2.number {
            return Ok(Some(Pairing {
                number,
                required_l1: BlockNumHash { number: 0, hash: B256::ZERO },
            }));
        }

        match self.safe_db.l1_at_safe_head(number) {
            Ok(record) => Ok(Some(Pairing { number, required_l1: record.l1 })),
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
                reason: format!("safe-head database: {err}"),
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

#[cfg(test)]
mod tests {
    use super::*;
    use kona_engine::{
        EngineQueries, EngineState, EngineSyncState, LocalSafeAtTimestamp, LocalSafeHead,
    };
    use kona_genesis::ChainGenesis;
    use kona_node_service::ChainControllerRpcRequest;
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use kona_safedb::{SafeDb, SafeHeadRecord};
    use lokahi_interop::{ArchivedOutput, open_output_archive};
    use tokio::sync::mpsc;

    /// The genesis timestamp of the fixture chain: block 0 at `1_787_416_188`, two-second blocks.
    ///
    /// Chosen so block 1 carries `1_787_416_190` — the exact query the fault-proof suites open
    /// with, whose failure text is pinned verbatim below.
    const GENESIS_TIME: u64 = 1_787_416_188;

    /// A safe-head database that answers every history read with one chosen error.
    #[derive(Debug)]
    struct FailingSafeDb(fn() -> SafeDbError);

    impl SafeDb for FailingSafeDb {
        fn enabled(&self) -> bool {
            true
        }

        fn safe_head_updated(
            &self,
            _safe_head: L2BlockInfo,
            _l1_head: BlockNumHash,
        ) -> Result<(), SafeDbError> {
            Ok(())
        }

        fn safe_head_reset(&self, _safe_head: L2BlockInfo) -> Result<(), SafeDbError> {
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

    /// The output-root preimage the stub engine serves for every `OutputAtBlock`.
    fn canonical_output() -> OutputRoot {
        OutputRoot::from_parts(
            B256::repeat_byte(0x11),
            B256::repeat_byte(0x12),
            B256::repeat_byte(0x13),
        )
    }

    /// A stub engine that answers every snapshot query with `local_safe_at` and every output
    /// query with [`canonical_output`].
    fn engine_answering(local_safe_at: LocalSafeAtTimestamp) -> QueuedEngineRpcClient {
        let (tx, mut rx) = mpsc::channel::<ChainControllerRpcRequest>(8);
        tokio::spawn(async move {
            while let Some(ChainControllerRpcRequest(query)) = rx.recv().await {
                match *query {
                    EngineQueries::LocalSafeSnapshotAt { timestamp, sender } => {
                        let _ = sender.send(LocalSafeSnapshot {
                            timestamp,
                            local_safe_at,
                            sync_state: EngineSyncState::default(),
                            el_sync_finished: true,
                        });
                    }
                    EngineQueries::OutputAtBlock { sender, .. } => {
                        let _ = sender.send((
                            L2BlockInfo::default(),
                            canonical_output(),
                            EngineState::default(),
                        ));
                    }
                    _ => {}
                }
            }
        });
        QueuedEngineRpcClient::new(tx)
    }

    /// A [`QueryChain`] for chain 901 over the stub engine, the given database and archive.
    fn chain(
        local_safe_at: LocalSafeAtTimestamp,
        safe_db: SharedSafeDb,
        archive: Option<Arc<lokahi_interop::RocksOutputArchive>>,
    ) -> QueryChain {
        let (l1_queries, _l1_rx) = mpsc::channel(1);
        QueryChain::new(
            901,
            Arc::new(RollupConfig {
                block_time: 2,
                genesis: ChainGenesis { l2_time: GENESIS_TIME, ..Default::default() },
                ..Default::default()
            }),
            engine_answering(local_safe_at),
            l1_queries,
            safe_db,
            archive,
        )
    }

    /// A history gap below the recorded floor fails the call with op-supernode's error — pinned
    /// verbatim, because this exact text is what the fault-proof acceptance suites surfaced
    /// (`TestFPP`: "Failed to get absolute prestate") when the floor sat above block 1.
    #[tokio::test]
    async fn a_history_gap_below_the_floor_fails_with_op_supernodes_error() {
        let chain = chain(
            LocalSafeAtTimestamp::BehindHead,
            Arc::new(FailingSafeDb(|| SafeDbError::L1AtSafeHeadUnavailable)),
            None,
        );

        let err = chain.optimistic_at(GENESIS_TIME + 2).await.expect_err("history is gone");
        assert_eq!(
            err.to_string(),
            "chain 901 no longer records which L1 block made its block at timestamp \
             1787416190 safe"
        );
    }

    /// Genesis is safe at L1 block 0 by definition, answered before the database is consulted —
    /// op-supernode's guard (`virtual_node.go:263-269`). The database here answers every read
    /// with the permanent error, so before the guard existed this call failed with the error
    /// pinned above, for the very timestamp a dispute game anchored at genesis asks about.
    #[tokio::test]
    async fn a_genesis_timestamp_answers_at_l1_block_zero_without_history() {
        let chain = chain(
            LocalSafeAtTimestamp::BehindHead,
            Arc::new(FailingSafeDb(|| SafeDbError::L1AtSafeHeadUnavailable)),
            None,
        );

        let output = chain
            .optimistic_at(GENESIS_TIME)
            .await
            .expect("genesis answers")
            .expect("genesis is never omitted");

        assert_eq!(output.required_l1, BlockNumHash { number: 0, hash: B256::ZERO });
        assert_eq!(output.output, canonical_output());
    }

    /// A height whose block was invalidated and replaced answers with the *original* block's
    /// archived output, not the replacement's — op-supernode's `LastDeniedOutputV0` consult in
    /// `OptimisticOutputAtTimestamp`. Serving the replacement instead makes the challenger
    /// derive transition states the proof program refutes, on heights that were the whole point
    /// of the dispute.
    #[tokio::test]
    async fn an_invalidated_height_answers_with_the_archived_output() {
        let dir = tempfile::tempdir().expect("tempdir");
        let archive = Arc::new(open_output_archive(dir.path()).expect("archive opens"));
        let invalidated = OutputRoot::from_parts(
            B256::repeat_byte(0x21),
            B256::repeat_byte(0x22),
            B256::repeat_byte(0x23),
        );
        archive
            .record(5, ArchivedOutput { output_root: invalidated, decision_timestamp: 1 })
            .expect("the invalidated output archives");

        let head = LocalSafeHead::derived_from(
            L2BlockInfo {
                block_info: BlockInfo {
                    number: 5,
                    timestamp: GENESIS_TIME + 10,
                    ..Default::default()
                },
                ..Default::default()
            },
            BlockInfo { number: 40, ..Default::default() },
        );
        let chain = chain(
            LocalSafeAtTimestamp::Head(head),
            Arc::new(FailingSafeDb(|| SafeDbError::L1AtSafeHeadUnavailable)),
            Some(archive),
        );

        let output = chain
            .optimistic_at(GENESIS_TIME + 10)
            .await
            .expect("the archive answers")
            .expect("the height is derived");

        assert_eq!(output.output, invalidated, "the original output, not the replacement's");
    }

    /// A height nothing invalidated answers the canonical output, archive or no archive: the
    /// consult is a lookup, never a gate.
    #[tokio::test]
    async fn a_height_never_invalidated_answers_the_canonical_output() {
        let dir = tempfile::tempdir().expect("tempdir");
        let archive = Arc::new(open_output_archive(dir.path()).expect("archive opens"));

        let head = LocalSafeHead::derived_from(
            L2BlockInfo {
                block_info: BlockInfo {
                    number: 5,
                    timestamp: GENESIS_TIME + 10,
                    ..Default::default()
                },
                ..Default::default()
            },
            BlockInfo { number: 40, ..Default::default() },
        );
        for archive in [Some(archive), None] {
            let chain = chain(
                LocalSafeAtTimestamp::Head(head),
                Arc::new(FailingSafeDb(|| SafeDbError::L1AtSafeHeadUnavailable)),
                archive,
            );

            let output = chain
                .optimistic_at(GENESIS_TIME + 10)
                .await
                .expect("the canonical read answers")
                .expect("the height is derived");
            assert_eq!(output.output, canonical_output());
        }
    }
}
