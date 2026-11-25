//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::ExecutableEngineTask;
use crate::{
    EngineClient, EngineState, EngineSyncState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, L2ForkchoiceState, Metrics, SafeLabel, SyncStartError,
    SynchronizeTaskError, find_starting_forkchoice,
    task_queue::{EngineTaskErrors, synchronize_unconditionally},
};
use alloy_eips::BlockId;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
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
}

impl Engine {
    /// Creates a new [`Engine`] with an empty task queue and the passed initial [`EngineState`].
    pub fn new(initial_state: EngineState, state_sender: Sender<EngineState>) -> Self {
        Self {
            state: initial_state,
            state_sender,
            tasks: BinaryHeap::default(),
            active: None,
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
    /// [`find_starting_forkchoice`]. Synchronizes the execution layer and publishes the
    /// recovered state before returning, even if no tasks are queued.
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let start = find_starting_forkchoice(&config, client.as_ref()).await?;

        // One attempt. The actor retains the reset request and retries transient failures. The
        // execution layer may have lost or changed its forkchoice, so the reset sends one even when
        // the recovered labels equal the current ones.
        let update = checked_reset_update(&self.state.sync_state, client.as_ref(), start).await?;
        synchronize_unconditionally(client.as_ref(), &mut self.state, update).await?;

        self.state_sender.send_replace(self.state);
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
    /// case it must be retried), unless it is a flush or skip error.
    pub async fn drain(&mut self) -> Result<(), EngineTaskErrors> {
        // Drain tasks in order of priority, halting on errors for a retry to be attempted.
        loop {
            if self.active.is_none() {
                self.active = self.tasks.pop().map(|queued| queued.task);
            }
            let Some(task) = &self.active else { break };
            let result = task.execute(&mut self.state).await;

            // A flush reports that the task replaced its block with a deposits-only block, so the
            // task is done; retrying it would replace that block again. A skip reports a task that
            // no longer applies, so retrying it cannot succeed either. A rederive reports that the
            // task moved the local-safe head back, which is done too.
            if let Err(err) = &result &&
                !matches!(
                    err.severity(),
                    EngineTaskErrorSeverity::Flush |
                        EngineTaskErrorSeverity::Skip |
                        EngineTaskErrorSeverity::Rederive
                )
            {
                return result;
            }

            self.state_sender.send_replace(self.state);
            self.active = None;
            result?;
        }

        Ok(())
    }
}

/// The labels a reset recovers from the sync `start`.
///
/// Derivation resumes from local-safe, so only local-safe is reset. The safe label is left to the
/// sync state: it follows local-safe, or, when a verifier moves it, it is held and never raised by
/// a reset, see [`crate::SafeLabel::Verified`].
const fn reset_update(start: L2ForkchoiceState) -> EngineSyncStateUpdate {
    EngineSyncStateUpdate {
        unsafe_head: Some(start.un_safe),
        local_safe_head: Some(start.safe),
        safe_head: None,
        finalized_head: Some(start.finalized),
    }
}

/// The labels a reset recovers from the sync `start`, with a verified safe label the reset would
/// keep checked against the execution layer.
///
/// The reset keeps a verified safe label at or below the recovered local-safe head, see
/// [`reset_update`]. It is the block at its height on the execution layer's chain only if the
/// execution layer did not move its head to another chain since the label was set, such as by
/// applying a forkchoice update whose safe block it then rejected. The recovered heads are on the
/// execution layer's current chain, so the label is looked up there, and falls to the recovered
/// finalized head if another block is at its height.
async fn checked_reset_update(
    sync_state: &EngineSyncState,
    client: &EngineClient,
    start: L2ForkchoiceState,
) -> Result<EngineSyncStateUpdate, EngineResetError> {
    let update = reset_update(start);
    if !matches!(sync_state.safe_label(), SafeLabel::Verified { .. }) {
        return Ok(update);
    }
    let kept = sync_state.apply_update(update).safe_head();
    if kept == start.safe || kept == start.finalized {
        return Ok(update);
    }
    let canonical = client
        .get_l2_block(BlockId::number(kept.block_info.number))
        .await
        .map_err(EngineResetError::SafeLabelLookup)?;
    if canonical.is_some_and(|block| block.header.hash == kept.block_info.hash) {
        return Ok(update);
    }
    warn!(target: "engine", safe = %kept, finalized = %start.finalized, "The safe label is not on the execution layer's chain; dropping it to the finalized head");
    Ok(EngineSyncStateUpdate { safe_head: Some(start.finalized), ..update })
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
    /// The lookup checking that the safe label is on the execution layer's chain failed.
    #[error("Failed to look up the safe label's block: {0}")]
    SafeLabelLookup(RpcError<TransportErrorKind>),
}

impl EngineResetError {
    /// Whether retrying the reset can recover without changing the node configuration.
    pub fn is_temporary(&self) -> bool {
        match self {
            // A missing L1 block below the head means a lagging backend that may catch up, and
            // any other L1 failure is an RPC failure.
            Self::SyncStart(
                SyncStartError::RpcError(_) |
                SyncStartError::L1(_) |
                SyncStartError::L1BlockMissing { .. },
            ) |
            Self::SafeLabelLookup(_) => true,
            Self::SyncStart(_) => false,
            Self::Forkchoice(err) => err.severity() != EngineTaskErrorSeverity::Critical,
        }
    }
}

#[cfg(test)]
mod recovery_tests {
    use super::*;
    use crate::{
        ConsolidateInput, ConsolidateTask, FinalizeBlockId, FinalizeTask, InsertTask,
        NoopBlockSink,
        test_utils::{
            TestAttributesBuilder, TestEngineStateBuilder, expect_sync_start, test_block_info,
            test_engine_client,
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
        l2.expect_error("engine_newPayloadV2");
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
            l2.expect("engine_newPayloadV2", PayloadStatus::from_status(PayloadStatusEnum::Valid));
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
    #[tokio::test]
    async fn flushed_task_is_dropped_not_retried() {
        let parent = test_block_info(10);
        let deposit = alloy_primitives::Bytes::from(vec![op_alloy_consensus::DEPOSIT_TX_TYPE_ID]);
        let attributes = TestAttributesBuilder::new()
            .with_parent(parent)
            .with_transactions(vec![deposit, alloy_primitives::Bytes::from(vec![0x02])])
            .build();

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
        // The task updates `engine.state()` itself; only drain publishes it.
        assert_eq!(
            state_rx.borrow().sync_state.safe_head().block_info.hash,
            imported.header.hash_slow()
        );
    }

    // An execution layer that restarted can have lost its forkchoice, so a reset sends the
    // recovered forkchoice even when the labels already equal it.
    #[tokio::test]
    async fn reset_resends_an_unchanged_forkchoice() {
        let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
            Default::default();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.inner.hash_slow();
        config.genesis.l1.hash = alloy_consensus::Header::default().hash_slow();
        let genesis =
            L2BlockInfo::from_block_and_genesis(&block.clone().into_consensus(), &config.genesis)
                .unwrap();
        let config = Arc::new(config);
        let state = TestEngineStateBuilder::new().with_unsafe_head(genesis).build();
        let forkchoice = state.sync_state.create_forkchoice_state();
        let (client, l1, l2) = test_engine_client(config.clone());
        expect_sync_start(&l1, &l2, &block, forkchoice);
        let (state_tx, _state_rx) = tokio::sync::watch::channel(state);
        let mut engine = Engine::new(state, state_tx);

        engine.reset(Arc::new(client), config).await.unwrap();

        assert_eq!(engine.state().sync_state, state.sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A reset moves local-safe to the sync start and never raises a verified safe label: one
    /// below the recovered local-safe head is held, and one the recovered local-safe head drops
    /// below falls to the recovered finalized head. A safe label following local-safe moves with
    /// it.
    #[rstest::rstest]
    #[case::verified_below_start(crate::SafeLabel::Verified { activation_timestamp: 0 }, 12, 10, 10)]
    #[case::verified_above_start(crate::SafeLabel::Verified { activation_timestamp: 0 }, 8, 10, 5)]
    #[case::local_safe(crate::SafeLabel::LocalSafe, 12, 10, 12)]
    fn reset_never_raises_a_verified_safe_label(
        #[case] safe_label: crate::SafeLabel,
        #[case] start_safe: u64,
        #[case] safe_before: u64,
        #[case] expected_safe: u64,
    ) {
        let blocks: Vec<_> = (0..=20).map(test_block_info).collect();
        let state = TestEngineStateBuilder::new()
            .with_safe_label(safe_label)
            .with_unsafe_head(blocks[20])
            .with_local_safe_head(blocks[15])
            .with_safe_head(blocks[safe_before as usize])
            .with_finalized_head(blocks[5])
            .build();
        let start = L2ForkchoiceState {
            un_safe: blocks[18],
            safe: blocks[start_safe as usize],
            finalized: blocks[5],
        };

        let reset = state.sync_state.apply_update(reset_update(start));

        assert_eq!(reset.local_safe_head(), start.safe);
        assert_eq!(reset.safe_head(), blocks[expected_safe as usize]);
    }

    /// A verified safe label the reset keeps below the recovered local-safe head is kept only if
    /// it is the execution layer's block at its height: an execution layer that moved its head
    /// to another chain leaves it off the chain the reset follows, and it falls to finalized. A
    /// label the reset sets to a recovered head is not looked up.
    #[rstest::rstest]
    #[case::canonical(10, Some(0), 10)]
    #[case::reorged_out(10, Some(1), 5)]
    #[case::missing(10, None, 5)]
    #[case::at_the_recovered_local_safe_head(12, None, 12)]
    #[tokio::test]
    async fn reset_keeps_a_verified_safe_label_only_on_the_execution_layers_chain(
        #[case] safe_before: u64,
        #[case] served: Option<u8>,
        #[case] expected_safe: u64,
    ) {
        use crate::test_utils::{test_l2_block, test_rpc_block};
        let mut blocks: Vec<op_alloy_consensus::OpBlock> = Vec::new();
        for number in 0..=20 {
            let parent = blocks
                .last()
                .map_or(alloy_primitives::B256::ZERO, |block| block.header.hash_slow());
            blocks.push(test_l2_block(number, parent, 0));
        }
        let genesis = kona_genesis::ChainGenesis {
            l2: alloy_eips::BlockNumHash { number: 0, hash: blocks[0].header.hash_slow() },
            ..Default::default()
        };
        let info = |number: u64| {
            L2BlockInfo::from_block_and_genesis(&blocks[number as usize], &genesis).unwrap()
        };
        let state = TestEngineStateBuilder::new()
            .with_safe_label(crate::SafeLabel::Verified { activation_timestamp: 0 })
            .with_unsafe_head(info(20))
            .with_local_safe_head(info(15))
            .with_safe_head(info(safe_before))
            .with_finalized_head(info(5))
            .build();
        let start = L2ForkchoiceState { un_safe: info(18), safe: info(12), finalized: info(5) };
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        if safe_before != 12 {
            let block = served.map(|fork| {
                let parent = blocks[safe_before as usize].header.parent_hash;
                test_rpc_block(&test_l2_block(safe_before, parent, fork))
            });
            l2.expect_block_header_by_number(safe_before, block);
        }

        let update = checked_reset_update(&state.sync_state, &client, start).await.unwrap();

        let reset = state.sync_state.apply_update(update);
        assert_eq!(reset.local_safe_head(), info(12));
        assert_eq!(reset.safe_head(), info(expected_safe));
        l1.assert_finished();
        l2.assert_finished();
    }

    // A skipped task is dropped instead of being kept for a retry, so it cannot hold the queue.
    #[tokio::test]
    async fn skipped_task_is_dropped_from_the_queue() {
        let config = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(config.clone());
        let state = TestEngineStateBuilder::new()
            .with_unsafe_head(test_block_info(20))
            .with_finalized_head(test_block_info(10))
            .build();
        let (state_tx, _state_rx) = tokio::sync::watch::channel(state);
        let mut engine = Engine::new(state, state_tx);
        engine.enqueue(EngineTask::Finalize(Box::new(FinalizeTask::new(
            Arc::new(client),
            config,
            FinalizeBlockId::ByNumber(5),
        ))));

        let error = engine.drain().await.unwrap_err();

        assert_eq!(error.severity(), EngineTaskErrorSeverity::Skip, "got {error:?}");
        assert!(engine.is_empty());
        assert_eq!(engine.state().sync_state, state.sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }
}
