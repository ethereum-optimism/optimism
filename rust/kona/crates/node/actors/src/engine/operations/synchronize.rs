//! Forkchoice updates without payload attributes.

use super::Engine;
use crate::engine::SynchronizeTaskError;
use alloy_rpc_types_engine::{INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatusEnum};
use kona_engine::EngineSyncStateUpdate;
use op_alloy_provider::ext::engine::OpEngineApi;

impl Engine {
    /// Applies a sync state update and sends the resulting forkchoice to the execution layer with
    /// `engine_forkchoiceUpdated`, without payload attributes.
    ///
    /// The other operations use this to move the forkchoice: inserting, consolidating, and
    /// finalizing blocks, and resetting. Starting a build sends its own forkchoice update with
    /// payload attributes instead.
    pub(in crate::engine) async fn synchronize(
        &mut self,
        update: EngineSyncStateUpdate,
    ) -> Result<(), SynchronizeTaskError> {
        // Apply the sync state update to the engine state.
        let new_sync_state = self.state.sync_state.apply_update(update);

        // Check if a forkchoice update is not needed, return early.
        // A forkchoice update is not needed if...
        // 1. The engine state is not default (initial forkchoice state has been emitted), and
        // 2. The new sync state is the same as the current sync state (no changes to the sync
        //    state).
        //
        // NOTE:
        // We shouldn't retry the forkchoice update there. Since the `sync_state` is only updated
        // here (except during consolidation, when the block is not the last in the batch) - the
        // engine would get stuck retrying the forkchoice update.
        if self.state.sync_state != Default::default() && self.state.sync_state == new_sync_state {
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

        self.check_forkchoice_updated_status(&valid_response.payload_status.status)?;

        // Apply the new sync state to the engine state.
        self.state.sync_state = new_sync_state;

        debug!(
            target: "engine",
            forkchoice = ?forkchoice,
            response = ?valid_response,
            "Forkchoice updated"
        );

        Ok(())
    }

    /// Checks the response of the `engine_forkchoiceUpdated` call, and updates the sync status if
    /// necessary.
    fn check_forkchoice_updated_status(
        &mut self,
        status: &PayloadStatusEnum,
    ) -> Result<(), SynchronizeTaskError> {
        match status {
            PayloadStatusEnum::Valid => {
                if !self.state.el_sync_finished {
                    info!(
                        target: "engine",
                        "Finished execution layer sync."
                    );
                    self.state.el_sync_finished = true;
                }

                Ok(())
            }
            PayloadStatusEnum::Syncing => {
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
