//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::EngineTaskExt;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, Metrics, SyncStartError, SynchronizeTask, SynchronizeTaskError,
    find_starting_forkchoice, task_queue::EngineTaskErrors,
};
use alloy_rpc_types_eth::Transaction;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpBlockConversionError, to_system_config};
use op_alloy_consensus::OpTxEnvelope;
use std::{collections::BinaryHeap, sync::Arc};
use thiserror::Error;
use tokio::sync::watch::Sender;

/// Maximum number of times a task with a temporary error can be retried
/// before being escalated to a reset error.
const MAX_TASK_RETRIES: usize = 10;

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
/// next call to [`Engine::drain`]. After [`MAX_TASK_RETRIES`] attempts, temporary errors
/// are escalated to reset errors to prevent infinite stalls.
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
    /// Tracks the number of consecutive retries for the current task at the front of the queue.
    /// Reset to 0 when a task succeeds or when a different task moves to the front.
    task_retry_count: usize,
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
            task_retry_count: 0,
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
    ) -> Result<(L2BlockInfo, BlockInfo, SystemConfig), EngineResetError> {
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
                EngineTaskErrorSeverity::Critical => {
                    return Err(EngineResetError::Forkchoice(err));
                }
            }
        }

        // Find the new safe head's L1 origin and SystemConfig.
        let origin_block = start
            .safe
            .l1_origin
            .number
            .saturating_sub(config.channel_timeout(start.safe.block_info.timestamp));
        let l1_origin_info: BlockInfo = client
            .get_l1_block(origin_block.into())
            .await
            .map_err(SyncStartError::RpcError)?
            .ok_or(SyncStartError::BlockNotFound(origin_block.into()))?
            .into_consensus()
            .into();
        let l2_safe_block = client
            .get_l2_block(start.safe.block_info.hash.into())
            .full()
            .await
            .map_err(SyncStartError::RpcError)?
            .ok_or(SyncStartError::BlockNotFound(origin_block.into()))?
            .into_consensus()
            .map_transactions(|t| <Transaction<OpTxEnvelope> as Clone>::clone(&t).into_inner());
        let system_config = to_system_config(&l2_safe_block, &config)?;

        kona_macros::inc!(counter, Metrics::ENGINE_RESET_COUNT);

        Ok((start.safe, l1_origin_info, system_config))
    }

    /// Clears the task queue and resets the retry counter.
    pub fn clear(&mut self) {
        self.tasks.clear();
        self.task_retry_count = 0;
    }

    /// Attempts to drain the queue by executing all [`EngineTask`]s in-order. If any task returns
    /// an error along the way, it is not popped from the queue (in case it must be retried) and
    /// the error is returned.
    ///
    /// To prevent infinite stalls, temporary errors are escalated to reset errors after
    /// [`MAX_TASK_RETRIES`] consecutive failures of the same task.
    pub async fn drain(&mut self) -> Result<(), EngineTaskErrors> {
        // Drain tasks in order of priority, halting on errors for a retry to be attempted.
        while let Some(task) = self.tasks.peek() {
            // Execute the task
            match task.execute(&mut self.state).await {
                Ok(_) => {
                    // Task succeeded - reset retry counter and pop the task
                    self.task_retry_count = 0;

                    // Update the state and notify the engine actor.
                    self.state_sender.send_replace(self.state);

                    // Pop the task from the queue now that it's been executed.
                    self.tasks.pop();
                    self.task_queue_length.send_replace(self.tasks.len());
                }
                Err(e) => {
                    // Task failed - check if we should retry or escalate
                    match e.severity() {
                        EngineTaskErrorSeverity::Temporary => {
                            self.task_retry_count += 1;

                            if self.task_retry_count >= MAX_TASK_RETRIES {
                                // Too many retries - pop task and trigger reset to prevent infinite stall
                                warn!(
                                    target: "engine",
                                    retry_count = self.task_retry_count,
                                    error = %e,
                                    "Task exceeded max retries, dropping task and triggering reset"
                                );

                                // Reset retry counter and pop the failed task
                                self.task_retry_count = 0;
                                self.tasks.pop();
                                self.task_queue_length.send_replace(self.tasks.len());

                                // Trigger engine reset by returning a reset error
                                return Err(EngineTaskErrors::Consolidate(
                                    crate::ConsolidateTaskError::ForkchoiceUpdateFailed(
                                        crate::SynchronizeTaskError::MaxRetriesExceeded {
                                            original_error: e.to_string(),
                                            retry_count: MAX_TASK_RETRIES,
                                        },
                                    ),
                                ));
                            }

                            // Still under retry limit - return error without popping
                            debug!(
                                target: "engine",
                                retry_count = self.task_retry_count,
                                max_retries = MAX_TASK_RETRIES,
                                "Task failed with temporary error, will retry"
                            );
                            return Err(e);
                        }
                        _ => {
                            // Critical, Reset, or Flush errors - reset counter and return immediately
                            self.task_retry_count = 0;
                            return Err(e);
                        }
                    }
                }
            }
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
    /// An error occurred while constructing the `SystemConfig` for the new safe head.
    #[error(transparent)]
    SystemConfigConversion(#[from] OpBlockConversionError),
}
