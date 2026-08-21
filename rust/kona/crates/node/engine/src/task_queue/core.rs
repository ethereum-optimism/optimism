//! The [`Engine`] is a task queue that receives and executes [`EngineTask`]s.

use super::EngineTaskExt;
use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, EngineTask, EngineTaskError,
    EngineTaskErrorSeverity, L2ForkchoiceState, Metrics, SyncStartError, SynchronizeTask,
    SynchronizeTaskError, find_starting_forkchoice,
    state::{CrossSafePromoter, CrossSafeSource, LocalSafeHead},
    task_queue::EngineTaskErrors,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::{collections::BinaryHeap, sync::Arc};
use thiserror::Error;
use tokio::sync::watch::Sender;

/// How many times [`Engine::reset_to`] puts its forkchoice update on the wire before giving up.
///
/// [`Engine::reset`] can retry forever because every attempt re-runs the walkback and may land on
/// a different, better start point, so the retries carry new information. A targeted reset has no
/// such escape hatch: the heads are fixed by the caller, so every attempt sends the byte-identical
/// forkchoice update. A small ceiling still absorbs a transient execution-layer hiccup, and then
/// returns the error to the caller — the only party that can supply different heads or fall back
/// to the walkback.
pub(super) const RESET_TO_MAX_ATTEMPTS: usize = 3;

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
    /// A sender that can be used to notify the chain controller of state changes.
    state_sender: Sender<EngineState>,
    /// A sender that can be used to notify the chain controller of task queue length changes.
    task_queue_length: Sender<usize>,
    /// The task queue.
    tasks: BinaryHeap<EngineTask<EngineClient_>>,
}

impl<EngineClient_: EngineClient> Engine<EngineClient_> {
    /// Creates a new [`Engine`] with an empty task queue and the passed initial [`EngineState`].
    ///
    /// The cross-safe head trivially follows local-safe, which is what standalone kona-node
    /// wants: there is no cross-chain verifier, so every local-safe advance is cross-safe.
    pub fn new(
        initial_state: EngineState,
        state_sender: Sender<EngineState>,
        task_queue_length: Sender<usize>,
    ) -> Self {
        Self { state: initial_state, state_sender, task_queue_length, tasks: BinaryHeap::default() }
    }

    /// Creates a new [`Engine`] whose cross-safe head is fed exclusively by externally minted
    /// [`crate::CrossSafePromotion`]s, returning the unique [`CrossSafePromoter`] that mints them.
    ///
    /// Local-safe advances no longer move the cross-safe head, so absence of promotion holds the
    /// previous value — including across engine resets.
    pub fn with_external_cross_safe(
        initial_state: EngineState,
        state_sender: Sender<EngineState>,
        task_queue_length: Sender<usize>,
    ) -> (Self, CrossSafePromoter) {
        let mut initial_state = initial_state;
        initial_state.sync_state =
            initial_state.sync_state.with_cross_safe_source(CrossSafeSource::Promoted);

        (
            Self {
                state: initial_state,
                state_sender,
                task_queue_length,
                tasks: BinaryHeap::default(),
            },
            CrossSafePromoter::new(),
        )
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
    ///
    /// This is the standalone default: the caller has no opinion about where the engine should
    /// land, so the walkback discovers it. A caller that *does* know the target heads should use
    /// [`Engine::reset_to`] instead and skip the discovery round trips entirely.
    ///
    /// Returns the local-safe head the engine reset to.
    pub async fn reset(
        &mut self,
        client: Arc<EngineClient_>,
        config: Arc<RollupConfig>,
    ) -> Result<L2BlockInfo, EngineResetError> {
        // Clear any outstanding tasks to prepare for the reset.
        self.clear();

        let mut start = find_starting_forkchoice(&config, client.as_ref()).await?;

        // Retry to synchronize the engine until we succeed or a critical error occurs. Each retry
        // re-runs the walkback rather than reusing `start`: the previous start point is the thing
        // that just failed, so re-discovery is what makes trying again worthwhile.
        while let Err(err) = self.synchronize_to(&client, &config, start).await {
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

        kona_macros::inc!(
            counter,
            Metrics::ENGINE_RESET_COUNT,
            Metrics::CHAIN_ID_LABEL => self.state.chain_id.to_string()
        );

        Ok(start.local_safe)
    }

    /// Resets the engine to `target`, skipping the [`find_starting_forkchoice`] walkback.
    ///
    /// [`Engine::reset`] spends RPC round trips discovering a start point. A caller that already
    /// knows which heads the engine must land on — because it derived them, or because an
    /// authority handed them over — has nothing to gain from that search, and re-deriving could
    /// pick a *different* block than the one the caller means. `reset_to` therefore applies
    /// `target` verbatim.
    ///
    /// The cross-safe head is not part of [`L2ForkchoiceState`] and this method mints no
    /// promotion, so under [`CrossSafeSource::Promoted`] — the interop engine, where cross-safe is
    /// a head in its own right — a reset cannot move it forward. Under
    /// [`CrossSafeSource::LocalSafe`] cross-safe *is* local-safe, so it follows the reset through
    /// the trivial promotion [`EngineSyncState::apply_update`] mints; there is no separate head to
    /// hold back.
    ///
    /// A *rewinding* target is held in order by that same state transition, which holds the
    /// cross-safe head down to a rewound local-safe head, so there is deliberately no clamp here;
    /// a second one would just be another opinion about the same invariant.
    ///
    /// [`EngineSyncState::apply_update`]: crate::EngineSyncState::apply_update
    ///
    /// Returns the local-safe head the engine reset to.
    pub async fn reset_to(
        &mut self,
        client: Arc<EngineClient_>,
        config: Arc<RollupConfig>,
        target: L2ForkchoiceState,
    ) -> Result<L2BlockInfo, EngineResetError> {
        // Clear any outstanding tasks to prepare for the reset.
        self.clear();

        let mut attempts_remaining = RESET_TO_MAX_ATTEMPTS;
        while let Err(err) = self.synchronize_to(&client, &config, target).await {
            attempts_remaining -= 1;

            match err.severity() {
                EngineTaskErrorSeverity::Temporary |
                EngineTaskErrorSeverity::Flush |
                EngineTaskErrorSeverity::Reset
                    if attempts_remaining > 0 =>
                {
                    warn!(
                        target: "engine",
                        ?err,
                        attempts_remaining,
                        "Forkchoice update failed during targeted reset. Trying again..."
                    );
                }
                _ => return Err(EngineResetError::Forkchoice(err)),
            }
        }

        kona_macros::inc!(counter, Metrics::ENGINE_RESET_COUNT);

        Ok(target.local_safe)
    }

    /// Runs the single [`SynchronizeTask`] that moves the engine's heads to `target`.
    async fn synchronize_to(
        &mut self,
        client: &Arc<EngineClient_>,
        config: &Arc<RollupConfig>,
        target: L2ForkchoiceState,
    ) -> Result<(), SynchronizeTaskError> {
        SynchronizeTask::new(
            client.clone(),
            config.clone(),
            EngineSyncStateUpdate {
                unsafe_head: Some(target.un_safe),
                // A reset installs a walkback point found by traversing the L2 chain, not one
                // produced by derivation, so there is no L1 key to pair with it. Writing it
                // unpaired is also what invalidates the pairing recorded before the reset, which
                // no longer describes the head the engine is on.
                local_safe_head: Some(LocalSafeHead::unpaired(target.local_safe)),
                finalized_head: Some(target.finalized),
            },
        )
        .execute(&mut self.state)
        .await
    }

    /// Clears the task queue.
    pub fn clear(&mut self) {
        self.tasks.clear();
    }

    /// Attempts to drain the queue by executing all [`EngineTask`]s in-order. If any task returns
    /// an error along the way, it is not popped from the queue (in case it must be retried) and
    /// the error is returned.
    pub async fn drain(&mut self) -> Result<(), EngineTaskErrors> {
        // Drain tasks in order of priority, halting on errors for a retry to be attempted.
        while let Some(task) = self.tasks.peek() {
            // Execute the task
            task.execute(&mut self.state).await?;

            // Update the state and notify the chain controller.
            self.state_sender.send_replace(self.state);

            // Pop the task from the queue now that it's been executed.
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
