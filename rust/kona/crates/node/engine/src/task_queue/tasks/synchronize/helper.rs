//! The `engine_forkchoiceUpdated` call, with no attributes, that the engine tasks share.

use crate::{EngineClient, EngineState, SynchronizeTaskError, state::EngineSyncStateUpdate};
use alloy_rpc_types_engine::{INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatusEnum};
use op_alloy_provider::ext::engine::OpEngineApi;

/// Applies `state_update` to the engine's sync state and synchronizes the execution layer's
/// forkchoice with it, using an `engine_forkchoiceUpdated` call without payload attributes.
///
/// Engine tasks call this whenever they move the unsafe, safe, or finalized head, and the engine
/// calls it during a reset to establish the initial forkchoice state. Forkchoice updates that
/// start a block build are made by [`BuildTask`](crate::BuildTask) instead.
pub(in crate::task_queue) async fn synchronize(
    client: &EngineClient,
    state: &mut EngineState,
    state_update: EngineSyncStateUpdate,
) -> Result<(), SynchronizeTaskError> {
    // Apply the sync state update to the engine state.
    let new_sync_state = state.sync_state.apply_update(state_update);

    // Check if a forkchoice update is not needed, return early.
    // A forkchoice update is not needed if...
    // 1. The engine state is not default (initial forkchoice state has been emitted), and
    // 2. The new sync state is the same as the current sync state (no changes to the sync state).
    //
    // NOTE:
    // We shouldn't retry the synchronization here. Since the `sync_state` is only updated inside
    // `synchronize` (except inside the ConsolidateTask, when the block is not the last in the
    // batch) - the engine would get stuck retrying the synchronization.
    if state.sync_state != Default::default() && state.sync_state == new_sync_state {
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
    // NOTE: it doesn't matter which version we use here, because we're not sending any payload
    // attributes. The forkchoice updated call is version agnostic if no payload attributes are
    // provided.
    let response = client.fork_choice_updated_v3(forkchoice, None).await;

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

    match &valid_response.payload_status.status {
        PayloadStatusEnum::Valid => {
            if !state.el_sync_finished {
                info!(target: "engine", "Finished execution layer sync.");
                state.el_sync_finished = true;
            }
        }
        // If we're not building a new payload, we're driving EL sync.
        PayloadStatusEnum::Syncing => {
            debug!(target: "engine", "Attempting to update forkchoice state while EL syncing")
        }
        // Other codes are not expected.
        status => return Err(SynchronizeTaskError::UnexpectedPayloadStatus(status.clone())),
    }

    // Apply the new sync state to the engine state.
    state.sync_state = new_sync_state;

    debug!(
        target: "engine",
        forkchoice = ?forkchoice,
        response = ?valid_response,
        "Forkchoice updated"
    );

    Ok(())
}
