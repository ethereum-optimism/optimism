//! A task for the `engine_forkchoiceUpdated` method, with no attributes.

use crate::{
    EngineClient, EngineState, EngineTaskExt, SynchronizeTaskError, state::EngineSyncStateUpdate,
};
use alloy_rpc_types_engine::{INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatusEnum};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::time::Instant;

/// Internal task for execution layer forkchoice synchronization.
///
/// The [`SynchronizeTask`] performs `engine_forkchoiceUpdated` calls to synchronize
/// the execution layer's forkchoice state with the rollup node's view. This task
/// operates without payload attributes and is primarily used internally by other
/// engine tasks rather than being directly enqueued by users.
///
/// ## Usage Patterns
///
/// - **Internal Synchronization**: Called by [`InsertTask`], [`ConsolidateTask`], and
///   [`FinalizeTask`]
/// - **Engine Reset**: Used during engine resets to establish initial forkchoice state
/// - **Safe Head Updates**: Synchronizes safe and finalized head changes
///
/// ## Automatic Integration
///
/// Unlike the legacy `ForkchoiceTask`, forkchoice updates during block building are now
/// explicitly handled within [`BuildTask`], eliminating the need for explicit
/// forkchoice management in most user scenarios.
///
/// [`InsertTask`]: crate::InsertTask
/// [`ConsolidateTask`]: crate::ConsolidateTask  
/// [`FinalizeTask`]: crate::FinalizeTask
/// [`BuildTask`]: crate::BuildTask
#[derive(Debug, Clone)]
pub struct SynchronizeTask<EngineClient_: EngineClient> {
    /// The engine client.
    pub client: Arc<EngineClient_>,
    /// The rollup config.
    pub rollup: Arc<RollupConfig>,
    /// The sync state update to apply to the engine state.
    pub state_update: EngineSyncStateUpdate,
    /// Whether a `VALID` response is required instead of accepting `SYNCING`.
    require_valid_payload_status: bool,
}

impl<EngineClient_: EngineClient> SynchronizeTask<EngineClient_> {
    /// Creates a forkchoice synchronization task.
    pub const fn new(
        client: Arc<EngineClient_>,
        rollup: Arc<RollupConfig>,
        state_update: EngineSyncStateUpdate,
    ) -> Self {
        Self { client, rollup, state_update, require_valid_payload_status: false }
    }

    /// Requires the execution layer to return `VALID` before applying the state update.
    pub const fn require_valid_payload_status(mut self) -> Self {
        self.require_valid_payload_status = true;
        self
    }

    /// Checks the response of the `engine_forkchoiceUpdated` call, and updates the sync status if
    /// necessary.
    fn check_forkchoice_updated_status(
        &self,
        state: &mut EngineState,
        status: &PayloadStatusEnum,
    ) -> Result<(), SynchronizeTaskError> {
        match status {
            PayloadStatusEnum::Valid => {
                if !state.el_sync_finished {
                    info!(
                        target: "engine",
                        "Finished execution layer sync."
                    );
                    state.el_sync_finished = true;
                }

                Ok(())
            }
            PayloadStatusEnum::Syncing if !self.require_valid_payload_status => {
                // If we're not building a new payload, we're driving EL sync.
                debug!(target: "engine", "Attempting to update forkchoice state while EL syncing");
                Ok(())
            }
            s => {
                // Other codes are not expected.
                Err(SynchronizeTaskError::UnexpectedPayloadStatus(s.clone()))
            }
        }
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for SynchronizeTask<EngineClient_> {
    type Output = ();
    type Error = SynchronizeTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<Self::Output, SynchronizeTaskError> {
        // Apply the sync state update to the engine state.
        let new_sync_state = state.sync_state.apply_update(self.state_update);

        // Check if a forkchoice update is not needed, return early.
        // A forkchoice update is not needed if...
        // 1. The engine state is not default (initial forkchoice state has been emitted), and
        // 2. The new sync state is the same as the current sync state (no changes to the sync
        //    state).
        //
        // NOTE:
        // We shouldn't retry the synchronize task there. Since the `sync_state` is only updated
        // inside the `SynchronizeTask` (except inside the ConsolidateTask, when the block is not
        // the last in the batch) - the engine will get stuck retrying the `SynchronizeTask`
        if !self.require_valid_payload_status &&
            state.sync_state != Default::default() &&
            state.sync_state == new_sync_state
        {
            debug!(target: "engine", ?new_sync_state, "No forkchoice update needed");
            return Ok(());
        }

        // Check if the head is behind the finalized head.
        if new_sync_state.unsafe_head().block_info.number <
            new_sync_state.finalized_head().block_info.number
        {
            return Err(SynchronizeTaskError::FinalizedAheadOfUnsafe(
                new_sync_state.unsafe_head().block_info.number,
                new_sync_state.finalized_head().block_info.number,
            ));
        }

        let fcu_time_start = Instant::now();

        // Send the forkchoice update through the input.
        let forkchoice = new_sync_state.create_forkchoice_state();

        // Handle the forkchoice update result.
        // NOTE: it doesn't matter which version we use here, because we're not sending any
        // payload attributes. The forkchoice updated call is version agnostic if no payload
        // attributes are provided.
        let response = self.client.fork_choice_updated_v3(forkchoice, None).await;

        let valid_response = response.map_err(|e| {
            // Fatal forkchoice update error.
            let error = e
                .as_error_resp()
                .and_then(|e| {
                    (e.code == INVALID_FORK_CHOICE_STATE_ERROR as i64)
                        .then_some(SynchronizeTaskError::InvalidForkchoiceState)
                })
                .unwrap_or_else(|| SynchronizeTaskError::ForkchoiceUpdateFailed(e));

            debug!(target: "engine", error = ?error, "Unexpected forkchoice update error");

            error
        })?;

        self.check_forkchoice_updated_status(state, &valid_response.payload_status.status)?;

        // Apply the new sync state to the engine state.
        state.sync_state = new_sync_state;

        let fcu_duration = fcu_time_start.elapsed();
        debug!(
            target: "engine",
            fcu_duration = ?fcu_duration,
            forkchoice = ?forkchoice,
            response = ?valid_response,
            "Forkchoice updated"
        );

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{TestEngineStateBuilder, test_block_info, test_engine_client_builder};
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus};

    #[tokio::test]
    async fn strict_synchronization_revalidates_unchanged_syncing_head() {
        let unsafe_head = test_block_info(1);
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();
        let client = Arc::new(
            test_engine_client_builder()
                .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(
                    PayloadStatus::from_status(PayloadStatusEnum::Syncing),
                ))
                .build(),
        );
        let task = SynchronizeTask::new(
            client,
            Arc::new(RollupConfig::default()),
            EngineSyncStateUpdate { unsafe_head: Some(unsafe_head), ..Default::default() },
        )
        .require_valid_payload_status();

        let err = task.execute(&mut state).await.unwrap_err();
        assert!(matches!(
            err,
            SynchronizeTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Syncing)
        ));
        assert_eq!(state.sync_state.unsafe_head(), unsafe_head);
    }
}
