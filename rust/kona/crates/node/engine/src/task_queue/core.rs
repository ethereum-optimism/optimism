//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::ExecutableEngineTask;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, L2ForkchoiceState, Metrics, SyncStartError, SynchronizeTaskError,
    find_starting_forkchoice,
    task_queue::{
        EngineTaskErrors, ForkchoiceReply, restore_forkchoice, synchronize_unconditionally,
    },
};
use alloy_primitives::B256;
use alloy_rpc_types_engine::ForkchoiceState;
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
        let start = self.find_starting_forkchoice_restoring_finality(&client, &config).await?;

        // One attempt. The actor retains the reset request and retries transient failures. The
        // execution layer may have lost or changed its forkchoice, so the reset sends one even when
        // the recovered labels equal the current ones.
        synchronize_unconditionally(
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

        self.state_sender.send_replace(self.state);
        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);

        Ok(start.safe)
    }

    /// Finds the sync start. A walk that misses the execution layer's finalized block follows a
    /// forkchoice update that the execution layer made canonical before rejecting it, which
    /// leaves its finalized label where it was. The engine restores a forkchoice on that block's
    /// chain and walks once more.
    async fn find_starting_forkchoice_restoring_finality(
        &self,
        client: &EngineClient,
        config: &RollupConfig,
    ) -> Result<L2ForkchoiceState, EngineResetError> {
        let (mismatch, el_finalized) = match find_starting_forkchoice(config, client).await {
            Err(mismatch @ SyncStartError::MismatchedFinalizedBlock { finalized, .. }) => {
                (mismatch, finalized)
            }
            result => return Ok(result?),
        };

        warn!(target: "engine", %mismatch, "Restoring a forkchoice on the finalized chain before walking again");
        self.restore_finalized_chain(client, el_finalized.hash).await?;
        match find_starting_forkchoice(config, client).await {
            Err(mismatch @ SyncStartError::MismatchedFinalizedBlock { .. }) => {
                Err(EngineResetError::FinalityNotRestored(mismatch))
            }
            result => Ok(result?),
        }
    }

    /// Puts the execution layer back on the chain of its finalized block, `el_finalized`. An
    /// engine that has established finality sends its own forkchoice again, which keeps its
    /// unsafe head. Its labels may include a delegated safe head that the execution layer never
    /// accepted, or an unsafe head it does not have; when it rejects them or answers `SYNCING`,
    /// and after a restart, a forkchoice of the finalized block alone restores its chain instead.
    async fn restore_finalized_chain(
        &self,
        client: &EngineClient,
        el_finalized: B256,
    ) -> Result<(), SynchronizeTaskError> {
        if self.state.sync_state.established_finalized_head().is_some() {
            let engine_forkchoice = self.state.sync_state.create_forkchoice_state();
            match restore_forkchoice(client, engine_forkchoice).await {
                Ok(ForkchoiceReply::Valid) => return Ok(()),
                Ok(ForkchoiceReply::Syncing) |
                Err(SynchronizeTaskError::InvalidForkchoiceState) => {}
                Err(err) => return Err(err),
            }
        }
        let finalized_only = ForkchoiceState {
            head_block_hash: el_finalized,
            safe_block_hash: el_finalized,
            finalized_block_hash: el_finalized,
        };
        restore_forkchoice(client, finalized_only).await.map(|_| ())
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
            // no longer applies, so retrying it cannot succeed either.
            if let Err(err) = &result &&
                !matches!(
                    err.severity(),
                    EngineTaskErrorSeverity::Flush | EngineTaskErrorSeverity::Skip
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

/// An error occurred while attempting to reset the [`Engine`].
#[derive(Debug, Error)]
pub enum EngineResetError {
    /// The walk still missed the finalized block after a forkchoice on that block's chain was
    /// restored, as while the execution layer has not yet made it canonical.
    #[error("Sync start still misses the finalized block after a restore: {0}")]
    FinalityNotRestored(SyncStartError),
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
            // After a restore the execution layer holds the engine's chain, and retrying gives it
            // time to make that chain canonical. A missing block at the finalized height means
            // the execution layer answered inconsistently. A missing L1 block below the head means
            // a lagging backend that may catch up, and any other L1 failure is an RPC failure.
            Self::FinalityNotRestored(_) |
            Self::SyncStart(
                SyncStartError::FinalizedHeightUnavailable(_) |
                SyncStartError::RpcError(_) |
                SyncStartError::L1(_) |
                SyncStartError::L1BlockMissing { .. },
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
        ConsolidateInput, ConsolidateTask, FinalizeBlockId, FinalizeTask, InsertTask,
        NoopBlockSink,
        test_utils::{
            RpcMock, TestAttributesBuilder, TestEngineStateBuilder, expect_sync_start,
            expect_sync_start_walk, test_block_info, test_engine_client, test_l2_chain,
        },
    };
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{
        BlobsBundleV1, ExecutionPayloadV1, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated,
        PayloadId, PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_consensus::OpBlock;
    use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpExecutionPayloadEnvelopeV3};

    /// A canonical chain `1..=4` and a fork from block 1 at `2..=4`, as an execution layer holds
    /// them after a rejected forkchoice update made the fork canonical before checking block 2,
    /// the finalized block.
    struct ForkedExecutionLayer {
        canonical: Vec<OpBlock>,
        fork: Vec<OpBlock>,
    }

    impl ForkedExecutionLayer {
        fn new() -> Self {
            let canonical = test_l2_chain(B256::ZERO, 1, 4, 0);
            let fork = test_l2_chain(canonical[0].header.hash_slow(), 2, 4, 1);
            Self { canonical, fork }
        }

        fn info(block: &OpBlock) -> L2BlockInfo {
            L2BlockInfo::from_block_and_genesis(block, &RollupConfig::default().genesis).unwrap()
        }

        fn finalized(&self) -> L2BlockInfo {
            Self::info(&self.canonical[1])
        }

        /// The engine's labels: unsafe at canonical block 4, safe and finalized at block 2.
        fn engine_state(&self) -> EngineState {
            TestEngineStateBuilder::new()
                .with_unsafe_head(Self::info(&self.canonical[3]))
                .with_safe_head(self.finalized())
                .with_finalized_head(self.finalized())
                .build()
        }

        fn expect_walk_on_fork(&self, l1: &RpcMock, l2: &RpcMock) {
            expect_sync_start_walk(
                l1,
                l2,
                &self.canonical[1],
                &self.fork[2],
                true,
                &[&self.fork[1], &self.fork[0]],
            );
        }

        fn expect_walk_on_canonical_chain(&self, l1: &RpcMock, l2: &RpcMock) {
            expect_sync_start_walk(
                l1,
                l2,
                &self.canonical[1],
                &self.canonical[3],
                true,
                &[&self.canonical[2], &self.canonical[1]],
            );
        }

        /// The forkchoice of the finalized block alone.
        fn finalized_only_forkchoice(&self) -> ForkchoiceState {
            let finalized = self.finalized().block_info.hash;
            ForkchoiceState {
                finalized_block_hash: finalized,
                head_block_hash: finalized,
                safe_block_hash: finalized,
            }
        }
    }

    async fn reset_from(
        initial: EngineState,
        client: EngineClient,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let (state_tx, _state_rx) = tokio::sync::watch::channel(initial);
        let mut engine = Engine::new(initial, state_tx);
        engine.reset(Arc::new(client), Arc::new(RollupConfig::default())).await
    }

    /// A reset whose walk misses the finalized block, which the execution layer still holds,
    /// sends the engine's own labels again, which makes its chain canonical, and then starts from
    /// the walk that meets the finalized block.
    #[tokio::test]
    async fn reset_restores_the_engine_forkchoice_when_the_walk_misses_finality() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let forkchoice = initial.sync_state.create_forkchoice_state();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        l2.expect_forkchoice_updated(forkchoice, PayloadStatusEnum::Valid);
        el.expect_walk_on_canonical_chain(&l1, &l2);
        l2.expect_forkchoice_updated(forkchoice, PayloadStatusEnum::Valid);

        let safe = reset_from(initial, client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// The engine's own labels may carry a safe head the execution layer never accepted; when it
    /// rejects them, a forkchoice of the finalized block alone restores its chain instead.
    #[tokio::test]
    async fn rejected_restore_falls_back_to_the_finalized_block() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let finalized_only = el.finalized_only_forkchoice();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        l2.expect_error_code(
            "engine_forkchoiceUpdatedV3",
            i64::from(alloy_rpc_types_engine::INVALID_FORK_CHOICE_STATE_ERROR),
        );
        l2.expect_forkchoice_updated(finalized_only, PayloadStatusEnum::Valid);
        el.expect_walk_on_canonical_chain(&l1, &l2);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );

        let safe = reset_from(initial, client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A walk that still misses the finalized block after the engine's labels and then the
    /// finalized block alone were sent, as while the execution layer syncs back to that chain,
    /// fails the reset with a retryable error.
    #[tokio::test]
    async fn reset_retries_while_the_restored_forkchoice_is_not_yet_canonical() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Syncing,
        );
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Syncing);
        el.expect_walk_on_fork(&l1, &l2);

        let error = reset_from(initial, client).await.unwrap_err();

        assert!(matches!(error, EngineResetError::FinalityNotRestored(_)), "got {error:?}");
        assert!(error.is_temporary());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// After a restart the engine has no finality of its own, so a rejected forkchoice update
    /// that left the execution layer on a fork from below its finalized block is undone with the
    /// execution layer's own finalized label: a forkchoice of that block alone, from which the
    /// walk then starts.
    #[tokio::test]
    async fn reset_after_a_restart_restores_the_el_finalized_block() {
        let el = ForkedExecutionLayer::new();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Valid);
        expect_sync_start_walk(&l1, &l2, &el.canonical[1], &el.canonical[1], true, &[]);
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Valid);

        let safe = reset_from(EngineState::default(), client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// An engine whose labels the execution layer answers `SYNCING`, as when it lacks the
    /// engine's unsafe head, restores the finalized block alone instead, and the walk then meets
    /// it.
    #[tokio::test]
    async fn syncing_restore_falls_back_to_the_finalized_block() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Syncing,
        );
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Valid);
        el.expect_walk_on_canonical_chain(&l1, &l2);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );

        let safe = reset_from(initial, client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// The fallback restores the execution layer's own finalized label, which it just served, and
    /// not the engine's finalized head, which is ahead of that label when the execution layer
    /// answered `SYNCING` to the forkchoice update that carried it.
    #[tokio::test]
    async fn fallback_restores_the_el_finalized_label_below_the_engine_finalized_head() {
        let el = ForkedExecutionLayer::new();
        let engine_finalized = ForkedExecutionLayer::info(&el.canonical[2]);
        let initial = TestEngineStateBuilder::new()
            .with_unsafe_head(ForkedExecutionLayer::info(&el.canonical[3]))
            .with_safe_head(engine_finalized)
            .with_finalized_head(engine_finalized)
            .build();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Syncing,
        );
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Valid);
        expect_sync_start_walk(&l1, &l2, &el.canonical[1], &el.canonical[1], true, &[]);
        l2.expect_forkchoice_updated(el.finalized_only_forkchoice(), PayloadStatusEnum::Valid);

        let safe = reset_from(initial, client).await.unwrap();

        assert_eq!(safe, el.finalized());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A permanent sync-start failure on the walk after a restore stays fatal instead of being
    /// retried as a finality that is not restored yet.
    #[tokio::test]
    async fn permanent_sync_start_error_after_a_restore_stays_fatal() {
        let el = ForkedExecutionLayer::new();
        let initial = el.engine_state();
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        el.expect_walk_on_fork(&l1, &l2);
        l2.expect_forkchoice_updated(
            initial.sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );
        expect_sync_start_walk(&l1, &l2, &el.canonical[1], &el.canonical[3], true, &[]);
        l2.expect("eth_getBlockByHash", None::<crate::test_utils::TestRpcBlock>);

        let error = reset_from(initial, client).await.unwrap_err();

        assert!(
            matches!(error, EngineResetError::SyncStart(SyncStartError::BlockNotFound(_))),
            "got {error:?}"
        );
        assert!(!error.is_temporary());
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A reset retries the sync-start errors that a lagging or briefly inconsistent provider
    /// causes, and stops the node on the others.
    #[rstest::rstest]
    #[case::finalized_height_unavailable(SyncStartError::FinalizedHeightUnavailable(2), true)]
    #[case::l1_block_missing(SyncStartError::L1BlockMissing { number: 2, l1_head: 3 }, true)]
    #[case::block_not_found(SyncStartError::BlockNotFound(2u64.into()), false)]
    fn sync_start_errors_a_reset_retries(
        #[case] error: SyncStartError,
        #[case] is_temporary: bool,
    ) {
        assert_eq!(EngineResetError::SyncStart(error).is_temporary(), is_temporary);
    }

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
