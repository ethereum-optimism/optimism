//! The forkchoice update without attributes shared by engine operations.

use crate::{
    EngineClient, EnginePhase, EngineState, ForkchoiceError, state::EngineSyncStateUpdate,
};
use alloy_rpc_types_engine::PayloadStatusEnum;
use tokio::time::Instant;

/// Current acceptance of a no-attributes forkchoice update.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ForkchoiceStatus {
    /// This exact forkchoice is validated by the execution layer.
    Valid,
    /// Execution has not validated this target; the caller must retain it for retry.
    Syncing,
}

/// Records requested heads and reports whether the execution layer has validated them.
/// A `SYNCING` target requires retry while the caller remains unready. Builds send their
/// own forkchoice update with attributes instead.
pub async fn synchronize(
    client: &impl EngineClient,
    state: &mut EngineState,
    state_update: EngineSyncStateUpdate,
) -> Result<ForkchoiceStatus, ForkchoiceError> {
    // Apply the sync state update to the engine state.
    let new_sync_state = state.sync_state.apply_update(state_update);

    // An unchanged ready snapshot is already applied. Recovery/sync must confirm with the EL,
    // even for identical heads, because an earlier FCU may have been ambiguous.
    if state.phase == EnginePhase::Ready &&
        state.sync_state != Default::default() &&
        state.sync_state == new_sync_state
    {
        debug!(target: "engine", ?new_sync_state, "No forkchoice update needed");
        return Ok(ForkchoiceStatus::Valid);
    }

    // Check if the head is behind the finalized head.
    if new_sync_state.unsafe_head().block_info.number <
        new_sync_state.finalized_head().block_info.number
    {
        return Err(ForkchoiceError::FinalizedAheadOfUnsafe(
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
    let response = client.fork_choice_updated_v3(forkchoice, None).await;

    let valid_response = response.map_err(ForkchoiceError::from)?;

    let status = match &valid_response.payload_status.status {
        PayloadStatusEnum::Valid => {
            if !state.el_sync_finished {
                info!(target: "engine", "Finished execution layer sync.");
                state.el_sync_finished = true;
            }
            ForkchoiceStatus::Valid
        }
        PayloadStatusEnum::Syncing => {
            debug!(target: "engine", "Attempting to update forkchoice state while EL syncing");
            state.phase = EnginePhase::Syncing;
            ForkchoiceStatus::Syncing
        }
        status => return Err(ForkchoiceError::UnexpectedStatus(status.clone())),
    };

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

    Ok(status)
}
