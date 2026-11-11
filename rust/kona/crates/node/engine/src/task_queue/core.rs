//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::ExecutableEngineTask;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, Metrics, SyncStartError, SynchronizeTaskError,
    find_starting_forkchoice,
    task_queue::{EngineTaskErrors, ForkchoiceDelivery, synchronize},
};
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
/// next call to [`Engine::drain`]. A task that fails with a wait error is parked instead: until its
/// retry, a drain of [`DrainScope::AheadOfWaitingTask`] runs only tasks of other kinds, while tasks
/// of its own kind keep their place behind it. A task
/// that fails with a flush error has already replaced its block, so it is popped before the error
/// is returned.
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
    active: Option<QueuedTask>,
    /// The task that last failed with a wait error, parked until the next [`Engine::drain`].
    waiting: Option<QueuedTask>,
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
            waiting: None,
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

    /// Number of queued, active and waiting tasks, including work awaiting a retry.
    pub fn len(&self) -> usize {
        self.tasks.len() + usize::from(self.active.is_some()) + usize::from(self.waiting.is_some())
    }

    /// Whether a task is parked until its retry.
    pub const fn has_waiting_task(&self) -> bool {
        self.waiting.is_some()
    }

    /// Whether a drain of [`DrainScope::AheadOfWaitingTask`] has a task to run: a task being
    /// retried, or a queued task of another kind than the waiting task, or any queued task without
    /// one.
    pub fn has_work_ahead_of_waiting_task(&self) -> bool {
        self.active.is_some() ||
            self.waiting.as_ref().map_or(!self.tasks.is_empty(), |waiting| {
                self.tasks.iter().any(|queued| !is_same_kind(queued, waiting))
            })
    }

    /// Pops the next queued task that may run. Before the waiting task's retry, tasks of its kind
    /// stay queued behind it, in their order, while tasks of other kinds run: a finalization
    /// queued behind a consolidation that waits for the execution layer would otherwise wait as
    /// long as its sync does.
    fn pop_runnable_task(&mut self) -> Option<QueuedTask> {
        let Some(waiting) = &self.waiting else { return self.tasks.pop() };
        let mut held_behind_waiting = Vec::new();
        let mut runnable = None;
        while let Some(next) = self.tasks.pop() {
            if is_same_kind(&next, waiting) {
                held_behind_waiting.push(next);
            } else {
                runnable = Some(next);
                break;
            }
        }
        self.tasks.extend(held_behind_waiting);
        runnable
    }

    /// Parks `task` until its retry. A task parked before it goes back into the queue with its
    /// sequence.
    fn park(&mut self, task: QueuedTask) {
        self.tasks.extend(self.waiting.replace(task));
    }

    /// Puts the waiting task back into the queue with its sequence, to be retried in its turn.
    fn unpark(&mut self) {
        self.tasks.extend(self.waiting.take());
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
        synchronize(
            client.as_ref(),
            &mut self.state,
            EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            },
            ForkchoiceDelivery::Always,
        )
        .await?;

        self.state_sender.send_replace(self.state);
        metrics::counter!(Metrics::ENGINE_RESET_COUNT).increment(1);

        Ok(start.safe)
    }

    /// Clears the task queue.
    pub fn clear(&mut self) {
        self.tasks.clear();
        self.active = None;
        self.waiting = None;
    }

    /// Attempts to drain the queue by executing the [`EngineTask`]s that `scope` allows, in order.
    /// If any task returns an error along the way, the error is returned and the task is not popped
    /// from the queue (in case it must be retried), unless it is a flush or skip error. A waiting
    /// task is retried only once every queued task of another kind has run, so a failure among
    /// those leaves it parked; if it still waits, it is parked again.
    pub async fn drain(&mut self, scope: DrainScope) -> Result<(), EngineTaskErrors> {
        // Drain tasks in order of priority, halting on errors for a retry to be attempted.
        loop {
            if self.active.is_none() {
                self.active = self.next_task(scope);
            }
            let Some(active) = &self.active else { break };
            let result = active.task.execute(&mut self.state).await;

            if let Err(err) = &result {
                match err.severity() {
                    // A waiting task leaves the engine state unchanged. Parked with its sequence,
                    // it keeps its place among same-kind tasks but no longer blocks
                    // higher-priority work enqueued before its retry.
                    EngineTaskErrorSeverity::Wait => {
                        if let Some(waiting) = self.active.take() {
                            self.park(waiting);
                        }
                        return result;
                    }
                    // A flush reports that the task replaced its block with a deposits-only block,
                    // so the task is done; retrying it would replace that block again. A skip
                    // reports a task that no longer applies, so retrying it cannot succeed either.
                    EngineTaskErrorSeverity::Flush | EngineTaskErrorSeverity::Skip => {}
                    EngineTaskErrorSeverity::Critical |
                    EngineTaskErrorSeverity::Reset |
                    EngineTaskErrorSeverity::Temporary => return result,
                }
            }

            self.state_sender.send_replace(self.state);
            self.active = None;
            result?;
        }

        Ok(())
    }

    /// The next task to run: a queued task of another kind than the waiting task, and once none is
    /// left, for [`DrainScope::AllTasks`], the waiting task in its turn.
    fn next_task(&mut self, scope: DrainScope) -> Option<QueuedTask> {
        if let Some(task) = self.pop_runnable_task() {
            return Some(task);
        }
        if scope == DrainScope::AllTasks && self.waiting.is_some() {
            self.unpark();
            return self.tasks.pop();
        }
        None
    }
}

/// Whether two queued tasks are of the same [`EngineTask`] kind, which the queue orders alike.
fn is_same_kind(a: &QueuedTask, b: &QueuedTask) -> bool {
    a.task.cmp(&b.task) == Ordering::Equal
}

/// Which tasks an [`Engine::drain`] runs.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DrainScope {
    /// Only tasks of other kinds than the waiting task, which stays parked. Without a waiting
    /// task, every task.
    AheadOfWaitingTask,
    /// Every task, retrying the waiting task in its turn.
    AllTasks,
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
            Self::SyncStart(SyncStartError::RpcError(_)) => true,
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
        let error = tokio::time::timeout(
            std::time::Duration::from_secs(1),
            engine.drain(DrainScope::AllTasks),
        )
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
        engine.drain(DrainScope::AllTasks).await.unwrap();
        assert!(engine.is_empty());
        assert_eq!(engine.state().sync_state.unsafe_head().block_info.number, 3);
        l1.assert_finished();
        l2.assert_finished();
    }

    // A drain that may retry the waiting task runs the queued tasks of other kinds first. When one
    // of them fails, the waiting task is not retried yet and stays parked for its own retry.
    #[tokio::test]
    async fn failure_ahead_of_the_waiting_task_leaves_it_parked() {
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
        engine.enqueue(EngineTask::Consolidate(Box::new(ConsolidateTask::new(
            client.clone(),
            config.clone(),
            ConsolidateInput::Attributes(Box::new(TestAttributesBuilder::new().build())),
            Arc::new(NoopBlockSink),
            crate::DEFAULT_EL_SYNC_STALL_TIMEOUT,
        ))));
        let waiting = engine.pop_runnable_task().expect("the queued consolidation");
        engine.park(waiting);
        engine.enqueue(EngineTask::Insert(Box::new(InsertTask::new(
            client,
            config,
            payload,
            false,
            Arc::new(NoopBlockSink),
        ))));

        let error = engine.drain(DrainScope::AllTasks).await.unwrap_err();

        assert_eq!(error.severity(), EngineTaskErrorSeverity::Temporary, "got {error:?}");
        assert!(engine.has_waiting_task());
        assert_eq!(engine.len(), 2);
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
            crate::DEFAULT_EL_SYNC_STALL_TIMEOUT,
        ))));

        let error = engine.drain(DrainScope::AllTasks).await.unwrap_err();
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

        let error = engine.drain(DrainScope::AllTasks).await.unwrap_err();

        assert_eq!(error.severity(), EngineTaskErrorSeverity::Skip, "got {error:?}");
        assert!(engine.is_empty());
        assert_eq!(engine.state().sync_state, state.sync_state);
        l1.assert_finished();
        l2.assert_finished();
    }
}

#[cfg(test)]
mod scheduling_tests {
    use super::*;
    use crate::{
        ConsolidateInput, ConsolidateTask, FinalizeBlockId, FinalizeTask, InsertTask,
        NoopBlockSink,
        test_utils::{TestAttributesBuilder, test_engine_client},
    };
    use TaskKind::*;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

    #[derive(Debug, Clone, Copy, PartialEq, Eq)]
    enum TaskKind {
        Consolidate,
        Finalize,
        Insert,
    }

    /// A queued task, as its kind and sequence.
    type Scheduled = (TaskKind, u64);

    /// An engine whose queue holds tasks that are scheduled but never executed.
    struct Scheduler {
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
        engine: Engine,
    }

    impl Scheduler {
        fn new() -> Self {
            let config = Arc::new(RollupConfig::default());
            let (client, _, _) = test_engine_client(config.clone());
            let (state_tx, _) = tokio::sync::watch::channel(EngineState::default());
            Self {
                client: Arc::new(client),
                config,
                engine: Engine::new(EngineState::default(), state_tx),
            }
        }

        fn enqueue(&mut self, kind: TaskKind) {
            let client = self.client.clone();
            let config = self.config.clone();
            self.engine.enqueue(match kind {
                Consolidate => EngineTask::Consolidate(Box::new(ConsolidateTask::new(
                    client,
                    config,
                    ConsolidateInput::Attributes(Box::new(TestAttributesBuilder::new().build())),
                    Arc::new(NoopBlockSink),
                    crate::DEFAULT_EL_SYNC_STALL_TIMEOUT,
                ))),
                Finalize => EngineTask::Finalize(Box::new(FinalizeTask::new(
                    client,
                    config,
                    FinalizeBlockId::ByNumber(1),
                ))),
                Insert => EngineTask::Insert(Box::new(InsertTask::new(
                    client,
                    config,
                    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
                        &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
                    )),
                    false,
                    Arc::new(NoopBlockSink),
                ))),
            });
        }

        /// Pops the next queued task, as a drain would, and parks it as a task that waits.
        fn pop_and_park(&mut self) -> Scheduled {
            let task = self.engine.pop_runnable_task().expect("a queued task");
            let scheduled = scheduled(&task);
            self.engine.park(task);
            scheduled
        }

        /// The order in which a drain would run the queued tasks that may run now.
        fn runnable_order(&mut self) -> Vec<Scheduled> {
            std::iter::from_fn(|| self.engine.pop_runnable_task()).map(|t| scheduled(&t)).collect()
        }

        fn waiting(&self) -> Option<Scheduled> {
            self.engine.waiting.as_ref().map(scheduled)
        }
    }

    fn scheduled(queued: &QueuedTask) -> Scheduled {
        let kind = match queued.task {
            EngineTask::Consolidate(_) => Consolidate,
            EngineTask::Finalize(_) => Finalize,
            EngineTask::Insert(_) => Insert,
            _ => unreachable!("only consolidate, finalize and insert tasks are scheduled"),
        };
        (kind, queued.sequence)
    }

    /// A consolidation parked while it waits (sequence 0) lets tasks of other kinds run before its
    /// retry, imports and finalizations alike, then runs in its turn ahead of later
    /// consolidations.
    #[rstest::rstest]
    #[case::insert_and_finalize_run_ahead(&[Finalize, Insert], &[(Insert, 2), (Finalize, 1)], &[(Consolidate, 0)])]
    #[case::finalize_runs_ahead_of_later_consolidation(&[Consolidate, Finalize], &[(Finalize, 2)], &[(Consolidate, 0), (Consolidate, 1)])]
    #[case::later_consolidation_stays_behind(&[Consolidate], &[], &[(Consolidate, 0), (Consolidate, 1)])]
    #[case::inserts_run_ahead_in_order(&[Insert, Insert], &[(Insert, 1), (Insert, 2)], &[(Consolidate, 0)])]
    fn waiting_task_keeps_its_place(
        #[case] queued_after_parking: &[TaskKind],
        #[case] expected_ahead_of_retry: &[Scheduled],
        #[case] expected_at_retry: &[Scheduled],
    ) {
        let mut scheduler = Scheduler::new();
        scheduler.enqueue(Consolidate);
        scheduler.pop_and_park();
        for kind in queued_after_parking {
            scheduler.enqueue(*kind);
        }

        let ahead_of_retry = scheduler.runnable_order();
        scheduler.engine.unpark();
        let at_retry = scheduler.runnable_order();

        assert_eq!(ahead_of_retry, expected_ahead_of_retry);
        assert_eq!(at_retry, expected_at_retry);
    }

    /// A waiting task that waits again at its retry is parked again with its sequence, still ahead
    /// of a later consolidation that does not run before it.
    #[test]
    fn task_waiting_again_is_parked_again_with_its_sequence() {
        let mut scheduler = Scheduler::new();
        scheduler.enqueue(Consolidate);
        scheduler.pop_and_park();
        scheduler.enqueue(Consolidate);

        scheduler.engine.unpark();
        let retried = scheduler.pop_and_park();

        assert_eq!(retried, (Consolidate, 0));
        assert_eq!(scheduler.waiting(), Some((Consolidate, 0)));
        assert_eq!(scheduler.runnable_order(), []);
        scheduler.engine.unpark();
        assert_eq!(scheduler.runnable_order(), [(Consolidate, 0), (Consolidate, 1)]);
    }

    /// A task that outranks the waiting one and waits too is parked in its place, and the earlier
    /// one goes back into the queue with its sequence instead of being dropped.
    #[test]
    fn parking_a_task_requeues_the_one_waiting_before_it() {
        let mut scheduler = Scheduler::new();
        scheduler.enqueue(Consolidate);
        scheduler.pop_and_park();
        scheduler.enqueue(Insert);

        let parked_second = scheduler.pop_and_park();

        assert_eq!(parked_second, (Insert, 1));
        assert_eq!(scheduler.waiting(), Some((Insert, 1)));
        assert_eq!(scheduler.engine.len(), 2);
        scheduler.engine.unpark();
        assert_eq!(scheduler.runnable_order(), [(Insert, 1), (Consolidate, 0)]);
    }
}
