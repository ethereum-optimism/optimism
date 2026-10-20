//! The forkchoice update without attributes shared by engine operations.

use crate::{EngineClient, EngineState, SynchronizeTaskError, state::EngineSyncStateUpdate};
use alloy_rpc_types_engine::{INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatusEnum};
use tokio::time::Instant;

/// Applies a heads update after the execution layer accepts the resulting forkchoice.
/// Builds send their own forkchoice update with attributes instead.
pub(in crate::task_queue) async fn synchronize(
    client: &impl EngineClient,
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
    // We shouldn't retry the synchronization there. Since the `sync_state` is only updated
    // inside the `synchronize` (except inside the ConsolidateTask, when the block is not
    // the last in the batch) - the engine will get stuck retrying the `synchronize`
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

    let fcu_time_start = Instant::now();

    // Send the forkchoice update through the input.
    let forkchoice = new_sync_state.create_forkchoice_state();

    // Handle the forkchoice update result.
    // NOTE: it doesn't matter which version we use here, because we're not sending any
    // payload attributes. The forkchoice updated call is version agnostic if no payload
    // attributes are provided.
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
        PayloadStatusEnum::Syncing => {
            debug!(target: "engine", "Attempting to update forkchoice state while EL syncing");
        }
        status => return Err(SynchronizeTaskError::UnexpectedPayloadStatus(status.clone())),
    }

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        EngineTaskError, EngineTaskErrorSeverity,
        task_queue::tasks::test_utils::{ExpectedCall, Script},
        test_utils::test_block_info,
    };
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus};
    use kona_genesis::RollupConfig;
    use rstest::rstest;
    use std::sync::Arc;

    #[rstest]
    #[case::valid(PayloadStatusEnum::Valid, true, true)]
    #[case::syncing(PayloadStatusEnum::Syncing, true, false)]
    #[case::invalid(PayloadStatusEnum::Invalid { validation_error: "invalid".into() }, false, false)]
    #[case::accepted(PayloadStatusEnum::Accepted, false, false)]
    #[tokio::test]
    async fn state_changes_only_after_an_accepted_forkchoice(
        #[case] status: PayloadStatusEnum,
        #[case] accepted: bool,
        #[case] synced: bool,
    ) {
        let script = Script::new([ExpectedCall::success(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(status)),
        )]);
        let engine = script.client(Arc::new(RollupConfig::default()));
        let mut state = EngineState::default();
        let unsafe_head = test_block_info(1);
        let update = EngineSyncStateUpdate { unsafe_head: Some(unsafe_head), ..Default::default() };
        let result = synchronize(&engine, &mut state, update).await;
        if accepted {
            result.unwrap();
            assert_eq!(state.sync_state.unsafe_head(), unsafe_head);
        } else {
            assert!(matches!(result, Err(SynchronizeTaskError::UnexpectedPayloadStatus(_))));
            assert_eq!(state.sync_state, Default::default());
        }
        assert_eq!(state.el_sync_finished, synced);
        assert!(script.calls()[0].params[1].is_null());
        script.assert_finished();
    }

    #[rstest]
    #[case::invalid_prestate(-38002, EngineTaskErrorSeverity::Reset)]
    #[case::other_rpc_error(-32000, EngineTaskErrorSeverity::Temporary)]
    #[tokio::test]
    async fn rpc_error_keeps_state_and_recovery_classification(
        #[case] code: i64,
        #[case] severity: EngineTaskErrorSeverity,
    ) {
        let script = Script::new([ExpectedCall::error("engine_forkchoiceUpdatedV3", code)]);
        let engine = script.client(Arc::new(RollupConfig::default()));
        let mut state = EngineState::default();
        let error = synchronize(&engine, &mut state, Default::default()).await.unwrap_err();
        assert_eq!(error.severity(), severity);
        assert_eq!(state, EngineState::default());
        script.assert_finished();
    }

    #[tokio::test]
    async fn first_default_forkchoice_is_sent_but_unchanged_nondefault_is_skipped() {
        let script = Script::new([ExpectedCall::success(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)),
        )]);
        let engine = script.client(Arc::new(RollupConfig::default()));
        let mut state = EngineState::default();
        synchronize(&engine, &mut state, Default::default()).await.unwrap();
        assert_eq!(script.calls().len(), 1);
        state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(test_block_info(1)),
            ..Default::default()
        });
        synchronize(&engine, &mut state, Default::default()).await.unwrap();
        assert!(script.calls().is_empty());
        script.assert_finished();
    }
}
