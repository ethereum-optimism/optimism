//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::EngineTaskExt;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, Metrics, SyncStartError, SynchronizeTaskError,
    sync::{SafeCheckpoint, find_starting_forkchoice_with_checkpoints, sync_lookback},
    task_queue::{EngineTaskErrors, synchronize},
};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use std::{cmp::Ordering, collections::BinaryHeap, sync::Arc};
use thiserror::Error;
use tokio::sync::watch::Sender;

/// A stable ordering within each task priority. Retried work can accumulate many same-kind
/// requests, which must keep their original order (especially unsafe payload imports).
#[derive(Debug)]
struct QueuedTask {
    sequence: u64,
    task: EngineTask,
}
impl PartialEq for QueuedTask {
    fn eq(&self, other: &Self) -> bool {
        self.cmp(other) == Ordering::Equal
    }
}
impl Eq for QueuedTask {}
impl PartialOrd for QueuedTask {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}
impl Ord for QueuedTask {
    fn cmp(&self, other: &Self) -> Ordering {
        self.task.cmp(&other.task).then_with(|| other.sequence.cmp(&self.sequence))
    }
}

/// L1 dependencies are inherited across replay and incomplete spans. Only completed spans
/// become reset candidates; their source bounds every input their ancestors depend on.
#[derive(Debug, Default)]
struct SafeCheckpoints {
    entries: Vec<SafeCheckpoint>,
    source: Option<BlockInfo>,
    // Keep the retained parent's dependency even if replay crosses onto a new L1 branch.
    replay_source: Option<BlockInfo>,
}

impl SafeCheckpoints {
    fn record(
        &mut self,
        attributes: &OpAttributesWithParent,
        cfg: &RollupConfig,
        safe: L2BlockInfo,
    ) {
        let Some(mut source) = attributes.derived_from else {
            *self = Self::default();
            return;
        };
        if let Some(parent_source) = self.source {
            if parent_source.number > source.number {
                source = parent_source;
            }
        } else if attributes.parent.block_info.id() != cfg.genesis.l2 &&
            source.number <
                attributes
                    .parent
                    .l1_origin
                    .number
                    .saturating_add(sync_lookback(cfg))
                    .saturating_add(1)
        {
            // The parent's exact input is unknown after startup or a conservative reset.
            // Wait until derivation reaches an L1 block that bounds its sequencing window.
            return;
        }
        self.source = Some(source);
        if !attributes.is_last_in_span {
            return;
        }
        self.entries.retain(|entry| {
            entry.safe.block_info.number < safe.block_info.number &&
                entry.source.number < source.number
        });
        self.entries.push(SafeCheckpoint { safe, source, replay_source: self.replay_source });
        if self.entries.len() > 4096 {
            self.entries.remove(0);
        }
    }

    fn reset_to(&mut self, safe: L2BlockInfo) {
        self.entries.retain(|entry| entry.safe.block_info.number <= safe.block_info.number);
        self.source =
            self.entries.last().filter(|entry| entry.safe == safe).map(|entry| entry.source);
        self.replay_source = self.source;
        if self.source.is_none() {
            self.entries.clear();
        }
    }
}

/// The [`Engine`] task queue.
///
/// Tasks of a shared [`EngineTask`] variant are processed in FIFO order, providing synchronization
/// guarantees for the L2 execution layer and other actors. A priority queue, ordered by
/// [`EngineTask`]'s [`Ord`] implementation, is used to prioritize tasks executed by the
/// [`Engine::drain`] method.
///
///  Because tasks are executed one at a time, they are considered to be atomic operations over the
/// [`EngineState`], and are given exclusive access to the engine state during execution.
///
/// Tasks within the queue are also considered fallible. If they fail with a temporary error,
/// they are not popped from the queue, the error is returned, and they are retried on the
/// next call to [`Engine::drain`]. A task that fails with a flush error has already replaced its
/// block, so it is popped before the error is returned.
#[derive(Debug)]
pub struct Engine {
    /// The state of the engine.
    state: EngineState,
    /// A sender that can be used to notify the engine actor of state changes.
    state_sender: Sender<EngineState>,
    /// The task queue.
    tasks: BinaryHeap<QueuedTask>,
    next_sequence: u64,
    /// Preserve the task being retried ahead of newly enqueued work: it may have already
    /// performed part of an Engine API operation.
    active: Option<EngineTask>,
    /// Recent accepted completed-safe checkpoints; no unverified EL labels are recorded.
    safe_checkpoints: SafeCheckpoints,
}

impl Engine {
    /// Creates a new [`Engine`] with an empty task queue and the passed initial [`EngineState`].
    pub fn new(initial_state: EngineState, state_sender: Sender<EngineState>) -> Self {
        Self {
            state: initial_state,
            state_sender,
            tasks: BinaryHeap::default(),
            active: None,
            safe_checkpoints: SafeCheckpoints::default(),
            next_sequence: 0,
        }
    }

    /// Returns a reference to the inner [`EngineState`].
    pub const fn state(&self) -> &EngineState {
        &self.state
    }

    /// Returns a receiver that can be used to listen to engine state updates.
    pub fn state_subscribe(&self) -> tokio::sync::watch::Receiver<EngineState> {
        self.state_sender.subscribe()
    }

    /// Enqueues a new [`EngineTask`] for execution.
    pub fn enqueue(&mut self, task: EngineTask) {
        let sequence = self.next_sequence;
        self.next_sequence =
            self.next_sequence.checked_add(1).expect("engine task sequence exhausted");
        self.tasks.push(QueuedTask { sequence, task });
    }

    /// Number of queued and active tasks, including work awaiting a retry.
    pub fn len(&self) -> usize {
        self.tasks.len() + usize::from(self.active.is_some())
    }

    /// Whether there is no pending work.
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }

    /// Resets the engine by finding a plausible sync starting point via
    /// [`crate::find_starting_forkchoice`]. The state will be updated to the starting point, and a
    /// forkchoice update will be enqueued in order to reorg the execution layer.
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let previous_safe = self.state.sync_state.safe_head();
        let start = find_starting_forkchoice_with_checkpoints(
            &config,
            client.as_ref(),
            &self.safe_checkpoints.entries,
        )
        .await?;

        // One attempt. The actor retains the reset request and retries transient failures.
        synchronize(
            client.as_ref(),
            &mut self.state,
            EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            },
        )
        .await?;

        self.safe_checkpoints.reset_to(start.safe);
        self.state_sender.send_replace(self.state);
        if start.safe.block_info.number < previous_safe.block_info.number {
            warn!(target: "engine", old_safe = previous_safe.block_info.number,
                new_safe = start.safe.block_info.number,
                rollback_blocks = previous_safe.block_info.number - start.safe.block_info.number,
                "Reset demoted completed safe head");
        }
        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);

        Ok(start.safe)
    }

    /// Clears the task queue.
    pub fn clear(&mut self) {
        self.tasks.clear();
        self.active = None;
    }

    /// Attempts to drain the queue by executing all [`EngineTask`]s in-order. If any task returns
    /// an error along the way, the error is returned and the task is not popped from the queue (in
    /// case it must be retried), unless it is a flush error.
    pub async fn drain(&mut self) -> Result<(), EngineTaskErrors> {
        // Drain tasks in order of priority, halting on errors for a retry to be attempted.
        loop {
            if self.active.is_none() {
                self.active = self.tasks.pop().map(|queued| queued.task);
            }
            let Some(task) = &self.active else { break };
            let result = task.execute(&mut self.state).await;

            // A flush reports that the task replaced its block with a deposits-only block, so the
            // task is done; retrying it would replace that block again.
            if let Err(err) = &result &&
                err.severity() != EngineTaskErrorSeverity::Flush
            {
                return result;
            }

            if let EngineTask::Consolidate(task) = task {
                match &task.input {
                    crate::ConsolidateInput::Attributes(attributes)
                        if self.state.sync_state.local_safe_head().block_info.number ==
                            attributes.block_number() =>
                    {
                        self.safe_checkpoints.record(
                            attributes,
                            &task.cfg,
                            self.state.sync_state.local_safe_head(),
                        );
                    }
                    // Delegated derivation supplies no L1 input dependency.
                    crate::ConsolidateInput::BlockInfo(_) => {
                        self.safe_checkpoints = SafeCheckpoints::default()
                    }
                    _ => {}
                }
            }
            self.state_sender.send_replace(self.state);
            self.active = None;
            result?;
        }

        Ok(())
    }
}

/// An error occurred while attempting to reset the [`Engine`].
#[derive(Debug, Error)]
pub enum EngineResetError {
    /// An error that occurred while updating the forkchoice state.
    #[error(transparent)]
    Forkchoice(#[from] SynchronizeTaskError),
    /// An error occurred while traversing the L1 for the sync starting point.
    #[error(transparent)]
    SyncStart(#[from] SyncStartError),
}

impl EngineResetError {
    /// Whether retrying the reset can recover without changing the node configuration.
    pub fn is_temporary(&self) -> bool {
        match self {
            Self::SyncStart(
                SyncStartError::RpcError(_) |
                SyncStartError::ChainViewChanged |
                SyncStartError::CheckpointNotValidated,
            ) => true,
            Self::SyncStart(_) => false,
            Self::Forkchoice(err) => err.severity() != EngineTaskErrorSeverity::Critical,
        }
    }
}

#[cfg(test)]
mod recovery_tests {
    use super::*;
    use crate::{
        ConsolidateInput, ConsolidateTask, InsertTask, NoopBlockSink,
        test_utils::{
            TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client,
        },
    };
    use alloy_rpc_types_engine::{
        BlobsBundleV1, ExecutionPayloadV1, ExecutionPayloadV3, ForkchoiceUpdated, PayloadId,
        PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpExecutionPayloadEnvelopeV3};

    #[tokio::test]
    async fn temporary_failure_returns_with_task_retained_for_recovery() {
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
        ));
        let block: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.hash_slow();
        let config = Arc::new(config);
        let (client, l1, l2) = test_engine_client(config.clone());
        let client = Arc::new(client);
        l2.expect_error("engine_newPayloadV1");
        let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = Engine::new(EngineState::default(), state_tx);
        engine.enqueue(EngineTask::Insert(Box::new(InsertTask::new(
            client.clone(),
            config,
            payload,
            false,
            Arc::new(NoopBlockSink),
        ))));
        // The old inline retry loop never returned from this call.
        let error = tokio::time::timeout(std::time::Duration::from_secs(1), engine.drain())
            .await
            .expect("temporary errors must yield to the actor")
            .unwrap_err();
        assert_eq!(error.severity(), EngineTaskErrorSeverity::Temporary);
        assert_eq!(engine.len(), 1);
        // Accumulate additional unsafe imports during the outage. Equal-priority heap
        // entries must remain FIFO; otherwise the final unsafe head can move backwards.
        for number in 1..=3 {
            let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
            block.header.number = number;
            let payload =
                OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block));
            let imported: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
            let mut config = RollupConfig::default();
            config.genesis.l2.number = number;
            config.genesis.l2.hash = imported.header.hash_slow();
            engine.enqueue(EngineTask::Insert(Box::new(InsertTask::new(
                client.clone(),
                Arc::new(config),
                payload,
                false,
                Arc::new(NoopBlockSink),
            ))));
        }
        for _ in 0..4 {
            l2.expect("engine_newPayloadV1", PayloadStatus::from_status(PayloadStatusEnum::Valid));
            l2.expect(
                "engine_forkchoiceUpdatedV3",
                ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
            );
        }
        engine.drain().await.unwrap();
        assert!(engine.is_empty());
        assert_eq!(engine.state().sync_state.unsafe_head().block_info.number, 3);
        l1.assert_finished();
        l2.assert_finished();
    }

    // A flush means the task already replaced its block with a deposits-only block. Keeping it
    // would rebuild that replacement on every later drain.
    #[rstest::rstest]
    #[case(true)]
    #[case(false)]
    #[tokio::test]
    async fn flushed_task_is_dropped_not_retried(#[case] completed_span: bool) {
        let parent = test_block_info(10);
        let deposit = alloy_primitives::Bytes::from(vec![op_alloy_consensus::DEPOSIT_TX_TYPE_ID]);
        let mut attributes = TestAttributesBuilder::new()
            .with_parent(parent)
            .with_transactions(vec![deposit, alloy_primitives::Bytes::from(vec![0x02])])
            .build();
        let source = kona_protocol::BlockInfo { number: 20, ..Default::default() };
        attributes.derived_from = Some(source);
        attributes.is_last_in_span = completed_span;

        // The deposits-only block the engine builds. It is genesis, so its block info needs no
        // L1-info deposit.
        let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
        block.header.number = parent.block_info.number + 1;
        let replacement = OpExecutionPayloadEnvelopeV3 {
            execution_payload: ExecutionPayloadV3::from_block_slow(&block),
            block_value: Default::default(),
            blobs_bundle: BlobsBundleV1::default(),
            should_override_builder: false,
            parent_beacon_block_root: Default::default(),
        };
        let imported: op_alloy_consensus::OpBlock = OpExecutionPayloadEnvelope::V3 {
            payload: replacement.execution_payload.clone(),
            parent_beacon_block_root: replacement.parent_beacon_block_root,
        }
        .try_into_block()
        .unwrap();
        let mut config = RollupConfig::default();
        config.hardforks.ecotone_time = Some(0);
        config.hardforks.holocene_time = Some(0);
        config.genesis.l2.number = imported.header.number;
        config.genesis.l2.hash = imported.header.hash_slow();
        let config = Arc::new(config);

        let valid = PayloadStatus::from_status(PayloadStatusEnum::Valid);
        let (client, l1, l2) = test_engine_client(config.clone());
        let client = Arc::new(client);
        l2.expect_error_code("engine_forkchoiceUpdatedV3", -38003);
        let forkchoice = TestEngineStateBuilder::new()
            .with_unsafe_head(parent)
            .build()
            .sync_state
            .create_forkchoice_state();
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([forkchoice, attributes.as_deposits_only().attributes]),
            ForkchoiceUpdated {
                payload_status: valid.clone(),
                payload_id: Some(PayloadId::new([1; 8])),
            },
        );
        l2.expect("engine_getPayloadV3", replacement);
        l2.expect("engine_newPayloadV3", valid.clone());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid));
        // The safe head is at the unsafe head, so the task builds the attributes instead of
        // consolidating them against an unsafe block.
        let state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
        let (state_tx, state_rx) = tokio::sync::watch::channel(EngineState::default());
        let mut engine = Engine::new(state, state_tx);
        engine.safe_checkpoints.source = Some(source);
        engine.enqueue(EngineTask::Consolidate(Box::new(ConsolidateTask::new(
            client.clone(),
            config,
            ConsolidateInput::Attributes(Box::new(attributes)),
            Arc::new(NoopBlockSink),
        ))));

        let error = engine.drain().await.unwrap_err();
        assert_eq!(error.severity(), EngineTaskErrorSeverity::Flush, "got {error:?}");
        l1.assert_finished();
        l2.assert_finished();
        assert!(engine.is_empty());
        assert_eq!(engine.safe_checkpoints.entries.len(), usize::from(completed_span));
        if completed_span {
            assert_eq!(engine.safe_checkpoints.entries[0].source, source);
            assert_eq!(engine.safe_checkpoints.entries[0].safe.hash(), imported.header.hash_slow());
        }
        // The task updates `engine.state()` itself; only drain publishes it.
        assert_eq!(
            state_rx.borrow().sync_state.safe_head().block_info.hash,
            imported.header.hash_slow()
        );
    }
}

#[cfg(test)]
mod reset_tests {
    use super::*;
    use crate::test_utils::{TestEngineStateBuilder, test_engine_client, test_l2_block};
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
    use kona_protocol::BlockInfo;

    #[rstest::rstest]
    #[case::canonical_origin(false)]
    #[case::replaced_origin(true)]
    #[tokio::test]
    async fn reset_retains_recent_completed_safe_instead_of_walking_seventy_thousand_blocks(
        #[case] reorged_unsafe: bool,
    ) {
        let config = Arc::new(RollupConfig { seq_window_size: 3600, ..Default::default() });
        let safe = test_l2_block(117029, 10635, 0, B256::ZERO);
        let unsafe_block = test_l2_block(
            117030,
            if reorged_unsafe { 10636 } else { 10635 },
            if reorged_unsafe { 0 } else { 1 },
            safe.header.hash,
        );
        let finalized = test_l2_block(44302, 7084, 0, B256::ZERO);
        let safe_info =
            L2BlockInfo::from_block_and_genesis(&safe.clone().into_consensus(), &config.genesis)
                .unwrap();
        let unsafe_info = L2BlockInfo::from_block_and_genesis(
            &unsafe_block.clone().into_consensus(),
            &config.genesis,
        )
        .unwrap();
        let mut head: <alloy_network::Ethereum as alloy_provider::Network>::BlockResponse =
            Default::default();
        head.header.inner.number = 10690;
        head.header.hash = B256::repeat_byte(2); // A replacement head, not the old branch's hash.
        let mut source: <alloy_network::Ethereum as alloy_provider::Network>::BlockResponse =
            Default::default();
        source.header.inner.number = 10679;
        source.header.hash = B256::repeat_byte(3);
        let checkpoint = SafeCheckpoint {
            safe: safe_info,
            source: BlockInfo {
                number: source.header.number,
                hash: source.header.hash,
                ..Default::default()
            },
            replay_source: None,
        };
        let (client, l1, l2) = test_engine_client(config.clone());
        for (tag, block) in [("finalized", &finalized), ("safe", &safe), ("latest", &unsafe_block)]
        {
            l2.expect_params("eth_getBlockByNumber", serde_json::json!([tag, true]), block);
        }
        let mut origin = head.clone();
        origin.header.hash = if reorged_unsafe { B256::ZERO } else { unsafe_info.l1_origin.hash };
        l1.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!([format!("{:#x}", unsafe_info.l1_origin.number), false]),
            origin,
        );
        if reorged_unsafe {
            l2.expect_params(
                "eth_getBlockByHash",
                serde_json::json!([safe_info.hash(), true]),
                &safe,
            );
            let mut origin = source.clone();
            origin.header.inner.number = safe_info.l1_origin.number;
            origin.header.hash = safe_info.l1_origin.hash;
            l1.expect_params(
                "eth_getBlockByNumber",
                serde_json::json!([format!("{:#x}", safe_info.l1_origin.number), false]),
                origin,
            );
        }
        let kept_unsafe = if reorged_unsafe { safe_info } else { unsafe_info };
        l1.expect_params("eth_getBlockByNumber", serde_json::json!(["latest", false]), &head);
        let mut origin = head.clone();
        origin.header.inner.number = kept_unsafe.l1_origin.number;
        origin.header.hash = kept_unsafe.l1_origin.hash;
        l1.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!([format!("{:#x}", kept_unsafe.l1_origin.number), false]),
            origin,
        );
        l1.expect_params("eth_getBlockByNumber", serde_json::json!(["0x29b7", false]), source);
        l2.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!([format!("{:#x}", safe_info.block_info.number), false]),
            &safe,
        );

        l2.expect_params(
            "eth_getBlockByNumber",
            serde_json::json!([format!("{:#x}", kept_unsafe.block_info.number), false]),
            if reorged_unsafe { &safe } else { &unsafe_block },
        );
        if !reorged_unsafe {
            l2.expect_params(
                "eth_getBlockByHash",
                serde_json::json!([safe_info.hash(), true]),
                &safe,
            );
        }
        let forkchoice = alloy_rpc_types_engine::ForkchoiceState {
            head_block_hash: kept_unsafe.hash(),
            safe_block_hash: safe_info.hash(),
            finalized_block_hash: finalized.header.hash,
        };
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([forkchoice, null]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        l1.expect_params("eth_getBlockByNumber", serde_json::json!(["0x29c2", false]), head);
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([forkchoice, null]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        let state = TestEngineStateBuilder::new()
            .with_safe_head(safe_info)
            .with_unsafe_head(unsafe_info)
            .build();
        let (sender, receiver) = tokio::sync::watch::channel(state);
        let mut engine = Engine::new(state, sender);
        engine.safe_checkpoints.entries.push(checkpoint);
        // No old L2 ancestors are supplied: the historical full-window walk fails this fixture.
        assert_eq!(engine.reset(Arc::new(client), config).await.unwrap(), safe_info);
        assert_eq!(engine.state().sync_state.safe_head(), safe_info);
        assert_eq!(receiver.borrow().sync_state.safe_head(), safe_info);
        assert_eq!(engine.state().sync_state.unsafe_head(), kept_unsafe);
        l1.assert_finished();
        l2.assert_finished();
    }

    #[rstest::rstest]
    #[case::walk_stops_at_finality(22, false)]
    #[case::already_finalized(20, false)]
    #[case::conflicting_finalized(20, true)]
    #[case::below_finalized(18, false)]
    #[tokio::test]
    async fn unsafe_walk_never_passes_finality(#[case] number: u64, #[case] conflict: bool) {
        let config = Arc::new(RollupConfig { seq_window_size: 3600, ..Default::default() });
        let (client, l1, l2) = test_engine_client(config.clone());
        let finalized = test_l2_block(20, 99, 0, B256::ZERO);
        let parent = test_l2_block(21, 100, 0, finalized.header.hash);
        let mut head = if number == 20 {
            finalized.clone()
        } else {
            test_l2_block(number, 100, 1, parent.header.hash)
        };
        if number == 22 {
            head.header.inner.parent_hash = parent.header.hash;
        }
        if conflict {
            head.header.inner.timestamp += 1;
        }
        head.header.hash = head.header.inner.hash_slow();
        for (tag, block) in [("finalized", &finalized), ("safe", &finalized), ("latest", &head)] {
            l2.expect_params("eth_getBlockByNumber", serde_json::json!([tag, true]), block);
        }
        if number == 22 {
            let mut replaced =
                alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default();
            replaced.header.hash = B256::ZERO;
            for _ in 0..2 {
                l1.expect_params(
                    "eth_getBlockByNumber",
                    serde_json::json!(["0x64", false]),
                    &replaced,
                );
            }
            for block in [&parent, &finalized] {
                l2.expect_params(
                    "eth_getBlockByHash",
                    serde_json::json!([block.header.hash, true]),
                    block,
                );
            }
        }
        if number >= 20 && !conflict {
            let mut origin =
                alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default();
            origin.header.hash = B256::with_last_byte(99);
            origin.header.inner.number = 99;
            l1.expect_params("eth_getBlockByNumber", serde_json::json!(["0x63", false]), origin);
        }
        let result = crate::find_starting_forkchoice(&config, &client).await;
        if number < 20 || conflict {
            let err = result.unwrap_err();
            assert!(matches!(err, SyncStartError::MismatchedFinalizedBlock(..)));
            assert!(!EngineResetError::SyncStart(err).is_temporary());
        } else {
            let heads = result.unwrap();
            assert_eq!(heads.un_safe, heads.finalized);
            assert_eq!(heads.safe, heads.finalized);
        }
        l1.assert_finished();
        l2.assert_finished();
    }
}

#[cfg(test)]
mod checkpoint_history_tests {
    use super::*;
    use crate::test_utils::{TestAttributesBuilder, test_block_info};

    #[rstest::rstest]
    #[case::completed_parent(true)]
    #[case::incomplete_span(false)]
    fn replay_inherits_parent_inputs_even_when_child_source_is_earlier(#[case] completed: bool) {
        let cfg = RollupConfig { seq_window_size: 10, ..Default::default() };
        let parent = test_block_info(20);
        let source = BlockInfo { number: 1000, ..Default::default() };
        let mut history = SafeCheckpoints { source: Some(source), ..Default::default() };
        let mut attrs = TestAttributesBuilder::new().with_parent(parent).build();
        attrs.derived_from = Some(source);
        attrs.is_last_in_span = completed;
        let child = test_block_info(21);
        history.record(&attrs, &cfg, child);
        assert_eq!(history.entries.len(), usize::from(completed));
        if completed {
            history.reset_to(child);
        }
        attrs.parent = child;
        attrs.derived_from.as_mut().unwrap().number = 900;
        attrs.is_last_in_span = true;
        let grandchild = test_block_info(22);
        history.record(&attrs, &cfg, grandchild);
        assert_eq!(history.entries.len(), 1);
        assert_eq!(history.entries[0].source, source);
        assert_eq!(history.entries[0].replay_source, completed.then_some(source));
        assert_eq!(history.entries[0].safe, grandchild);
    }

    #[rstest::rstest]
    #[case::sequencing_window(false)]
    #[case::alt_da_window(true)]
    fn unknown_parent_waits_for_its_dependency_bound_and_resets_forget_unrelated_inputs(
        #[case] alt_da: bool,
    ) {
        let mut cfg = RollupConfig { seq_window_size: 10, ..Default::default() };
        if alt_da {
            cfg.alt_da_config = Some(kona_genesis::AltDAConfig {
                da_challenge_window: Some(7),
                da_resolve_window: Some(8),
                ..Default::default()
            });
        }
        let parent = test_block_info(20);
        let mut attrs = TestAttributesBuilder::new().with_parent(parent).build();
        let bound = parent.l1_origin.number + if alt_da { 15 } else { 10 } + 1;
        attrs.derived_from = Some(BlockInfo { number: bound - 1, ..Default::default() });
        attrs.is_last_in_span = true;
        let child = test_block_info(21);
        let mut history = SafeCheckpoints::default();
        history.record(&attrs, &cfg, child);
        assert!(history.entries.is_empty());
        attrs.derived_from.as_mut().unwrap().number = bound;
        history.record(&attrs, &cfg, child);
        assert_eq!(history.entries.len(), 1);
        history.reset_to(parent);
        assert!(history.entries.is_empty());
        assert!(history.source.is_none());
    }
}
