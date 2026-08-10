//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::EngineTaskExt;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, Metrics, SyncStartError, SynchronizeTask, SynchronizeTaskError,
    find_starting_forkchoice, task_queue::EngineTaskErrors,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::{collections::BinaryHeap, sync::Arc};
use thiserror::Error;
use tokio::sync::watch::Sender;

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
/// Temporary failures remain queued for retry. Permanently invalid tasks are removed.
#[derive(Debug)]
pub struct Engine<EngineClient_: EngineClient> {
    /// The state of the engine.
    state: EngineState,
    /// A sender that can be used to notify the engine actor of state changes.
    state_sender: Sender<EngineState>,
    /// A sender that can be used to notify the engine actor of task queue length changes.
    task_queue_length: Sender<usize>,
    /// The task queue.
    tasks: BinaryHeap<EngineTask<EngineClient_>>,
}

impl<EngineClient_: EngineClient> Engine<EngineClient_> {
    /// Creates a new [`Engine`] with an empty task queue and the passed initial [`EngineState`].
    pub fn new(
        initial_state: EngineState,
        state_sender: Sender<EngineState>,
        task_queue_length: Sender<usize>,
    ) -> Self {
        Self { state: initial_state, state_sender, task_queue_length, tasks: BinaryHeap::default() }
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
        self.tasks.push(task);
        self.task_queue_length.send_replace(self.tasks.len());
    }

    /// Resets the engine by finding a plausible sync starting point via
    /// [`find_starting_forkchoice`]. The state will be updated to the starting point, and a
    /// forkchoice update will be enqueued in order to reorg the execution layer.
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient_>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        // Clear any outstanding tasks to prepare for the reset.
        self.clear();

        let mut start = find_starting_forkchoice(&config, client.as_ref()).await?;

        // Retry to synchronize the engine until we succeeds or a critical error occurs.
        while let Err(err) = SynchronizeTask::new(
            client.clone(),
            config.clone(),
            EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                cross_unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            },
        )
        .execute(&mut self.state)
        .await
        {
            match err.severity() {
                EngineTaskErrorSeverity::Temporary |
                EngineTaskErrorSeverity::Flush |
                EngineTaskErrorSeverity::Reset => {
                    warn!(target: "engine", ?err, "Forkchoice update failed during reset. Trying again...");
                    start = find_starting_forkchoice(&config, client.as_ref()).await?;
                }
                EngineTaskErrorSeverity::Drop | EngineTaskErrorSeverity::Critical => {
                    return Err(EngineResetError::Forkchoice(err));
                }
            }
        }

        kona_macros::inc!(counter, Metrics::ENGINE_RESET_COUNT);

        Ok(start.safe)
    }

    /// Clears the task queue.
    pub fn clear(&mut self) {
        self.tasks.clear();
    }

    /// Executes queued tasks in priority order, retaining failures unless marked `Drop`.
    pub async fn drain(&mut self) -> Result<(), EngineTaskErrors> {
        while let Some(task) = self.tasks.peek() {
            match task.execute(&mut self.state).await {
                Ok(()) => {
                    // Update the state and notify the engine actor.
                    self.state_sender.send_replace(self.state);
                }
                Err(err) if err.severity() == EngineTaskErrorSeverity::Drop => {}
                Err(err) => return Err(err),
            }

            self.tasks.pop();
            self.task_queue_length.send_replace(self.tasks.len());
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        InsertTask,
        test_utils::{TestEngineStateBuilder, test_engine_client_builder},
    };
    use alloy_primitives::Bytes;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatus, PayloadStatusEnum};
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
    use std::time::Duration;
    use tokio::{sync::watch, time::timeout};

    fn v1_payload(transactions: Vec<Bytes>) -> OpExecutionPayloadEnvelope {
        OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
            parent_hash: Default::default(),
            fee_recipient: Default::default(),
            state_root: Default::default(),
            receipts_root: Default::default(),
            logs_bloom: Default::default(),
            prev_randao: Default::default(),
            block_number: 1,
            gas_limit: 0,
            gas_used: 0,
            timestamp: 2,
            extra_data: Default::default(),
            base_fee_per_gas: Default::default(),
            block_hash: Default::default(),
            transactions,
        })
    }

    #[tokio::test]
    async fn drops_permanently_invalid_unsafe_payload() {
        let config = Arc::new(RollupConfig::default());
        let client = test_engine_client_builder()
            .with_config(config.clone())
            .with_new_payload_v1_response(PayloadStatus {
                status: PayloadStatusEnum::Invalid {
                    validation_error: "invalid state root".to_string(),
                },
                latest_valid_hash: None,
            })
            .build();
        let state = TestEngineStateBuilder::new().build();
        let (state_sender, _) = watch::channel(state);
        let (queue_length_sender, queue_length) = watch::channel(0);
        let mut engine = Engine::new(state, state_sender, queue_length_sender);

        engine.enqueue(EngineTask::Insert(Box::new(InsertTask::new(
            Arc::new(client),
            config,
            v1_payload(Vec::new()),
            false,
        ))));
        assert_eq!(*queue_length.borrow(), 1);

        timeout(Duration::from_secs(1), engine.drain())
            .await
            .expect("drain timed out")
            .expect("drain failed");

        assert_eq!(*queue_length.borrow(), 0);
    }

    #[tokio::test]
    async fn drops_unsafe_payload_with_malformed_transaction() {
        let config = Arc::new(RollupConfig::default());
        let client = test_engine_client_builder()
            .with_config(config.clone())
            .with_new_payload_v1_response(PayloadStatus {
                status: PayloadStatusEnum::Valid,
                latest_valid_hash: None,
            })
            .build();
        let state = TestEngineStateBuilder::new().build();
        let (state_sender, _) = watch::channel(state);
        let (queue_length_sender, queue_length) = watch::channel(0);
        let mut engine = Engine::new(state, state_sender, queue_length_sender);

        engine.enqueue(EngineTask::Insert(Box::new(InsertTask::new(
            Arc::new(client),
            config,
            v1_payload(vec![Bytes::from_static(&[0xff])]),
            false,
        ))));

        timeout(Duration::from_secs(1), engine.drain())
            .await
            .expect("drain timed out")
            .expect("drain failed");

        assert_eq!(*queue_length.borrow(), 0);
    }
}
