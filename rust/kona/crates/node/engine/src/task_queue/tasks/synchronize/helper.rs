//! The `engine_forkchoiceUpdated` call, with no attributes, that the engine tasks share.

use crate::{EngineClient, EngineState, SynchronizeTaskError, state::EngineSyncStateUpdate};
use alloy_rpc_types_engine::{INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatusEnum};
use op_alloy_provider::ext::engine::OpEngineApi;

/// Whether [`synchronize`] may skip a forkchoice update that leaves the consensus labels unchanged.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(in crate::task_queue) enum ForkchoiceDelivery {
    /// Send the update even when it leaves the consensus labels unchanged.
    ///
    /// Unchanged labels do not prove that the execution layer holds their forkchoice: a `SYNCING`
    /// answer moves the labels too, and an execution layer that restarts can lose its forkchoice.
    Always,
    /// Skip the update when it leaves the consensus labels unchanged.
    WhenChanged,
}

/// Applies `state_update` to the engine's sync state and synchronizes the execution layer's
/// forkchoice with it, using an `engine_forkchoiceUpdated` call without payload attributes.
///
/// Engine tasks call this whenever they move the unsafe, safe, or finalized head, and the engine
/// calls it during a reset to establish the initial forkchoice state. Forkchoice updates that
/// start a block build are made by `start_build` instead.
#[tracing::instrument(
    name = "synchronize",
    target = "engine",
    level = "info",
    skip_all,
    fields(?delivery, ?state_update, forkchoice = tracing::field::Empty)
)]
pub(in crate::task_queue) async fn synchronize(
    client: &EngineClient,
    state: &mut EngineState,
    state_update: EngineSyncStateUpdate,
    delivery: ForkchoiceDelivery,
) -> Result<(), SynchronizeTaskError> {
    // Apply the sync state update to the engine state.
    let new_sync_state = state.sync_state.apply_update(state_update);

    // Check if a forkchoice update is not needed, return early.
    // A forkchoice update is not needed if...
    // 1. The caller allows skipping an unchanged forkchoice,
    // 2. The engine state is not default (initial forkchoice state has been emitted), and
    // 3. The new sync state is the same as the current sync state (no changes to the sync state).
    //
    // NOTE:
    // We shouldn't retry the synchronization here. Since the `sync_state` is only updated inside
    // `synchronize` (except inside the ConsolidateTask, when the block is not the last in the
    // batch) - the engine would get stuck retrying the synchronization.
    if delivery == ForkchoiceDelivery::WhenChanged &&
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

    // Send the forkchoice update through the input.
    let forkchoice = new_sync_state.create_forkchoice_state();
    tracing::Span::current().record("forkchoice", tracing::field::debug(forkchoice));

    // Handle the forkchoice update result.
    // NOTE: it doesn't matter which version we use here, because we're not sending any payload
    // attributes. The forkchoice updated call is version agnostic if no payload attributes are
    // provided.
    let response = client.fork_choice_updated_v3(forkchoice, None).await;

    let valid_response = response.map_err(|e| {
        // Fatal forkchoice update error.
        e.as_error_resp()
            .and_then(|e| {
                (e.code == INVALID_FORK_CHOICE_STATE_ERROR as i64)
                    .then_some(SynchronizeTaskError::InvalidForkchoiceState)
            })
            .unwrap_or_else(|| SynchronizeTaskError::ForkchoiceUpdateFailed(e))
    })?;

    match &valid_response.payload_status.status {
        PayloadStatusEnum::Valid => {
            state.el_applied_last_forkchoice = true;
            if !state.el_sync_finished {
                info!(target: "engine", "Finished execution layer sync.");
                state.el_sync_finished = true;
            }
        }
        // If we're not building a new payload, we're driving EL sync.
        PayloadStatusEnum::Syncing => {
            state.el_applied_last_forkchoice = false;
            debug!(target: "engine", "Attempting to update forkchoice state while EL syncing")
        }
        // Other codes are not expected.
        status => return Err(SynchronizeTaskError::UnexpectedPayloadStatus(status.clone())),
    }

    // Apply the new sync state to the engine state.
    state.sync_state = new_sync_state;

    Ok(())
}
