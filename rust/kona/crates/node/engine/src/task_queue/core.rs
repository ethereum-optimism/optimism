//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::EngineTaskExt;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, Metrics, SyncStartError, SynchronizeTask, SynchronizeTaskError,
    find_starting_forkchoice, task_queue::EngineTaskErrors,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::{cmp::Ordering, collections::BinaryHeap, sync::Arc};
use thiserror::Error;
use tokio::sync::watch::Sender;

/// A stable ordering within each task priority. Retried work can accumulate many same-kind
/// requests, which must keep their original order (especially unsafe payload imports).
#[derive(Debug)]
struct QueuedTask<C: EngineClient> {
    sequence: u64,
    task: EngineTask<C>,
}
impl<C: EngineClient> PartialEq for QueuedTask<C> {
    fn eq(&self, other: &Self) -> bool {
        self.cmp(other) == Ordering::Equal
    }
}
impl<C: EngineClient> Eq for QueuedTask<C> {}
impl<C: EngineClient> PartialOrd for QueuedTask<C> {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}
impl<C: EngineClient> Ord for QueuedTask<C> {
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
/// next call to [`Engine::drain`].
#[derive(Debug)]
pub struct Engine<EngineClient_: EngineClient> {
    /// The state of the engine.
    state: EngineState,
    /// A sender that can be used to notify the engine actor of state changes.
    state_sender: Sender<EngineState>,
    /// A sender that can be used to notify the engine actor of task queue length changes.
    task_queue_length: Sender<usize>,
    /// The task queue.
    tasks: BinaryHeap<QueuedTask<EngineClient_>>,
    next_sequence: u64,
    /// Preserve the task being retried ahead of newly enqueued work: it may have already
    /// performed part of an Engine API operation.
    active: Option<EngineTask<EngineClient_>>,
}

impl<EngineClient_: EngineClient> Engine<EngineClient_> {
    /// Creates a new [`Engine`] with an empty task queue and the passed initial [`EngineState`].
    pub fn new(
        initial_state: EngineState,
        state_sender: Sender<EngineState>,
        task_queue_length: Sender<usize>,
    ) -> Self {
        Self {
            state: initial_state,
            state_sender,
            task_queue_length,
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

    /// Returns a receiver that can be used to listen to engine queue length updates.
    pub fn queue_length_subscribe(&self) -> tokio::sync::watch::Receiver<usize> {
        self.task_queue_length.subscribe()
    }

    /// Enqueues a new [`EngineTask`] for execution.
    /// Updates the queue length and notifies listeners of the change.
    pub fn enqueue(&mut self, task: EngineTask<EngineClient_>) {
        let sequence = self.next_sequence;
        self.next_sequence =
            self.next_sequence.checked_add(1).expect("engine task sequence exhausted");
        self.tasks.push(QueuedTask { sequence, task });
        self.task_queue_length.send_replace(self.len());
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
    /// [`find_starting_forkchoice`]. The state will be updated to the starting point, and a
    /// forkchoice update will be enqueued in order to reorg the execution layer.
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient_>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        let start = find_starting_forkchoice(&config, client.as_ref()).await?;

        // One attempt. The actor retains the reset request and retries transient failures.
        let synchronize = SynchronizeTask::new(
            client,
            config,
            EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            },
        );
        synchronize.execute(&mut self.state).await?;

        kona_macros::inc!(counter, Metrics::ENGINE_RESET_COUNT);

        Ok(start.safe)
    }

    /// Clears the task queue.
    pub fn clear(&mut self) {
        self.tasks.clear();
        self.active = None;
        self.task_queue_length.send_replace(0);
    }

    /// Attempts to drain the queue by executing all [`EngineTask`]s in-order. If any task returns
    /// an error along the way, it is not popped from the queue (in case it must be retried) and
    /// the error is returned.
    pub async fn drain(&mut self) -> Result<(), EngineTaskErrors> {
        // Drain tasks in order of priority, halting on errors for a retry to be attempted.
        loop {
            if self.active.is_none() {
                self.active = self.tasks.pop().map(|queued| queued.task);
            }
            let Some(task) = &self.active else { break };
            task.execute(&mut self.state).await?;
            self.state_sender.send_replace(self.state);
            self.active = None;
            self.task_queue_length.send_replace(self.len());
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
            Self::SyncStart(SyncStartError::RpcError(_)) => true,
            Self::SyncStart(_) => false,
            Self::Forkchoice(err) => err.severity() != EngineTaskErrorSeverity::Critical,
        }
    }
}

#[cfg(test)]
mod recovery_tests {
    use super::*;
    use crate::{InsertTask, NoopBlockSink, test_utils::MockEngineClient};
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
    };
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

    #[tokio::test]
    async fn temporary_failure_returns_with_task_retained_for_recovery() {
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
        ));
        let block: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.hash_slow();
        let config = Arc::new(config);
        let client = Arc::new(MockEngineClient::builder().with_config(config.clone()).build());
        let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
        let (queue_tx, queue_rx) = tokio::sync::watch::channel(0);
        let mut engine = Engine::new(EngineState::default(), state_tx, queue_tx);
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
        assert_eq!(*queue_rx.borrow(), 1);
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
        client
            .set_new_payload_v1_response(PayloadStatus::from_status(PayloadStatusEnum::Valid))
            .await;
        client
            .set_fork_choice_updated_v3_response(ForkchoiceUpdated::new(
                PayloadStatus::from_status(PayloadStatusEnum::Valid),
            ))
            .await;
        engine.drain().await.unwrap();
        assert!(engine.is_empty());
        assert_eq!(*queue_rx.borrow(), 0);
        assert_eq!(engine.state().sync_state.unsafe_head().block_info.number, 3);
    }
}
