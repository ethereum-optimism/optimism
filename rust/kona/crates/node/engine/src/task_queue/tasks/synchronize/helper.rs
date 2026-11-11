//! The `engine_forkchoiceUpdated` call, with no attributes, that the engine tasks share.

use crate::{
    EngineClient, EngineState, EngineSyncState, SafeLabel, SynchronizeTaskError,
    state::EngineSyncStateUpdate,
};
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

/// How the execution layer answered a forkchoice update it accepted.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(in crate::task_queue) enum ForkchoiceReply {
    /// The execution layer does not have the head block yet and is syncing toward it.
    Syncing,
    /// The execution layer applied the forkchoice.
    Valid,
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

    ensure_safe_labels_not_behind_finality(&new_sync_state)?;

    // Send the forkchoice update through the input.
    let forkchoice = new_sync_state.create_forkchoice_state();
    tracing::Span::current().record("forkchoice", tracing::field::debug(forkchoice));
    let reply = send_forkchoice(client, &new_sync_state).await?;
    record_forkchoice_reply(state, reply);

    // Apply the new sync state to the engine state.
    state.sync_state = new_sync_state;

    Ok(())
}

/// Refuses a sync state whose safe or local-safe head sits below its finalized head.
fn ensure_safe_labels_not_behind_finality(
    sync_state: &EngineSyncState,
) -> Result<(), SynchronizeTaskError> {
    let labels = [
        (SafeLabel::LocalSafe, sync_state.local_safe_head()),
        (SafeLabel::Safe, sync_state.safe_head()),
    ];
    for (label, head) in labels {
        if let Err(violation) = sync_state.ensure_not_behind_finality(head.block_info.number) {
            debug!(target: "engine", %violation, %label, "Safe label would move below finality");
            return Err(SynchronizeTaskError::FinalizedAheadOfSafe {
                finalized_number: sync_state.finalized_head().block_info.number,
                label,
                label_number: head.block_info.number,
            });
        }
    }
    Ok(())
}

/// Sends the engine's own `sync_state` forkchoice again, to undo a forkchoice update that the
/// execution layer made canonical before rejecting it. Neither the engine state nor its record of
/// the execution layer's sync changes.
pub(in crate::task_queue) async fn restore_forkchoice(
    client: &EngineClient,
    sync_state: &EngineSyncState,
) -> Result<ForkchoiceReply, SynchronizeTaskError> {
    warn!(
        target: "engine",
        forkchoice = ?sync_state.create_forkchoice_state(),
        "Restoring the engine's forkchoice on the execution layer"
    );
    let reply = send_forkchoice(client, sync_state).await.inspect_err(|err| {
        warn!(target: "engine", ?err, "Restoring the engine's forkchoice failed");
    })?;
    info!(target: "engine", ?reply, "Restored the engine's forkchoice on the execution layer");
    Ok(reply)
}

/// Records how the execution layer answered the engine's forkchoice: whether it applied it, and
/// whether its sync has finished, which only a `VALID` answer marks and nothing clears.
pub(in crate::task_queue) fn record_forkchoice_reply(
    state: &mut EngineState,
    reply: ForkchoiceReply,
) {
    state.el_applied_last_forkchoice = reply == ForkchoiceReply::Valid;
    if reply == ForkchoiceReply::Valid && !state.el_sync_finished {
        info!(target: "engine", "Finished execution layer sync.");
        state.el_sync_finished = true;
    }
}

/// Sends `sync_state`'s forkchoice to the execution layer, with no payload attributes, and returns
/// whether it applied the forkchoice or is still syncing toward its head. Neither the engine state
/// nor its record of the execution layer's sync changes.
pub(in crate::task_queue) async fn send_forkchoice(
    client: &EngineClient,
    sync_state: &EngineSyncState,
) -> Result<ForkchoiceReply, SynchronizeTaskError> {
    // NOTE: it doesn't matter which version we use here, because we're not sending any payload
    // attributes. The forkchoice updated call is version agnostic if no payload attributes are
    // provided.
    let response = client.fork_choice_updated_v3(sync_state.create_forkchoice_state(), None).await;

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
        PayloadStatusEnum::Valid => Ok(ForkchoiceReply::Valid),
        // If we're not building a new payload, we're driving EL sync.
        PayloadStatusEnum::Syncing => {
            debug!(target: "engine", "Attempting to update forkchoice state while EL syncing");
            Ok(ForkchoiceReply::Syncing)
        }
        // Other codes are not expected.
        status => Err(SynchronizeTaskError::UnexpectedPayloadStatus(status.clone())),
    }
}
