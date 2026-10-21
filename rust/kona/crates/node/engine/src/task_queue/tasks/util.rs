//! Utility functions for task execution.

use super::{BuildSealCoupling, BuildTask, BuildTaskError, EngineTaskExt, SealTask, SealTaskError};
use crate::{
    EngineClient, EngineState, EngineSyncState, ImportedBlockSink, InsertTaskError,
    SynchronizeTaskError,
};
use alloy_rpc_types_engine::{
    ExecutionPayloadInputV2, ForkchoiceUpdated, INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatus,
};
use kona_genesis::RollupConfig;
use kona_protocol::OpAttributesWithParent;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{sync::Arc, time::Instant};

/// Submits an envelope using the appropriate Engine API version, without interpreting its status.
pub(super) async fn submit_payload<C: EngineClient>(
    client: &C,
    payload: OpExecutionPayloadEnvelope,
) -> Result<PayloadStatus, InsertTaskError> {
    let response = match payload {
        OpExecutionPayloadEnvelope::V1(payload) => client.new_payload_v1(payload).await,
        OpExecutionPayloadEnvelope::V2(payload) => {
            let payload_input = ExecutionPayloadInputV2 {
                execution_payload: payload.payload_inner,
                withdrawals: Some(payload.withdrawals),
            };
            client.new_payload_v2(payload_input).await
        }
        OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root } => {
            client.new_payload_v3(payload, parent_beacon_block_root).await
        }
        OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root } => {
            client.new_payload_v4(payload, parent_beacon_block_root).await
        }
    };
    response.map_err(|error| {
        warn!(target: "engine", "Failed to insert new payload: {error}");
        InsertTaskError::InsertFailed(error)
    })
}

/// Sends forkchoice without applying local state or interpreting the returned payload status.
pub(super) async fn update_forkchoice<C: EngineClient>(
    client: &C,
    sync_state: &EngineSyncState,
) -> Result<ForkchoiceUpdated, SynchronizeTaskError> {
    if sync_state.unsafe_head().block_info.number < sync_state.finalized_head().block_info.number {
        return Err(SynchronizeTaskError::FinalizedAheadOfUnsafe(
            sync_state.unsafe_head().block_info.number,
            sync_state.finalized_head().block_info.number,
        ));
    }

    let fcu_time_start = Instant::now();
    let forkchoice = sync_state.create_forkchoice_state();
    // Without payload attributes, the forkchoice update is version agnostic.
    let response = client.fork_choice_updated_v3(forkchoice, None).await.map_err(|error| {
        let error = error
            .as_error_resp()
            .and_then(|response| {
                (response.code == INVALID_FORK_CHOICE_STATE_ERROR as i64)
                    .then_some(SynchronizeTaskError::InvalidForkchoiceState)
            })
            .unwrap_or_else(|| SynchronizeTaskError::ForkchoiceUpdateFailed(error));
        debug!(target: "engine", ?error, "Unexpected forkchoice update error");
        error
    })?;
    debug!(
        target: "engine",
        fcu_duration = ?fcu_time_start.elapsed(),
        ?forkchoice,
        ?response,
        "Received forkchoice update response"
    );
    Ok(response)
}

/// Error type for build and seal operations.
#[derive(Debug, thiserror::Error)]
pub(in crate::task_queue) enum BuildAndSealError {
    /// An error occurred during the build phase.
    #[error(transparent)]
    Build(#[from] BuildTaskError),
    /// An error occurred during the seal phase.
    #[error(transparent)]
    Seal(#[from] SealTaskError),
}

/// Builds and seals a block for derived attributes in sequence.
///
/// This is a utility function that:
/// 1. Creates and executes a [`BuildTask`] to initiate block building
/// 2. Creates and executes a [`SealTask`] to seal the block, referencing the initiated payload
///
/// The consolidate task uses it for derived attributes that it cannot consolidate, and for their
/// deposits-only replacement.
///
/// # Arguments
///
/// * `state` - Mutable reference to the engine state
/// * `engine` - The engine client
/// * `cfg` - The rollup configuration
/// * `attributes` - The derived payload attributes to build
/// * `block_sink` - Where to hand the built block once the engine has canonicalized it
pub(in crate::task_queue) async fn build_and_seal<EngineClient_: EngineClient>(
    state: &mut EngineState,
    engine: Arc<EngineClient_>,
    cfg: Arc<RollupConfig>,
    attributes: OpAttributesWithParent,
    block_sink: Arc<dyn ImportedBlockSink>,
) -> Result<(), BuildAndSealError> {
    // Execute the build task
    let payload_id = BuildTask::new(
        engine.clone(),
        cfg.clone(),
        attributes.clone(),
        None, // Build task doesn't send the payload yet
    )
    .execute(state)
    .await?;

    // Execute the seal task with the payload ID from the build.
    SealTask::new(
        engine,
        cfg,
        payload_id,
        attributes,
        true,
        BuildSealCoupling::Atomic,
        None,
        block_sink,
    )
    .execute(state)
    .await?;

    Ok(())
}

/// Handles `attributes` that the engine rejected, at forkchoiceUpdated or at newPayload, by the
/// rules of op-node's `onBuildInvalid`. op-node applies these rules at forkchoiceUpdated; at
/// newPayload it applies only the derived and Holocene rule, while kona-node also keeps rejected
/// deposits-only attributes critical there.
///
/// - Rejected deposits-only attributes are critical: [`SealTaskError::DepositOnlyPayloadFailed`].
/// - Rejected derived attributes are replaced with their deposits-only block under Holocene. The
///   task then ends with [`SealTaskError::HoloceneInvalidFlush`], which flushes the rest of the
///   channel, or with [`SealTaskError::DepositOnlyPayloadReattemptFailed`] if the replacement
///   fails.
/// - Nothing replaces other attributes, so this returns `None` and the task keeps its own error. A
///   sequencer drops its payload and builds the next one.
pub(in crate::task_queue) async fn handle_rejected_attributes<EngineClient_: EngineClient>(
    state: &mut EngineState,
    engine: Arc<EngineClient_>,
    cfg: Arc<RollupConfig>,
    attributes: &OpAttributesWithParent,
    is_attributes_derived: bool,
    block_sink: Arc<dyn ImportedBlockSink>,
) -> Option<SealTaskError> {
    let parent = attributes.parent.block_info.id();
    if attributes.is_deposits_only() {
        error!(target: "engine", ?parent, "Critical: the engine rejected deposits-only attributes");
        return Some(SealTaskError::DepositOnlyPayloadFailed);
    }
    if !is_attributes_derived ||
        !cfg.is_holocene_active(attributes.attributes().payload_attributes.timestamp)
    {
        return None;
    }

    warn!(target: "engine", ?parent, "Replacing rejected derived attributes with deposits only");
    let deposits_only = attributes.as_deposits_only();
    Some(match build_and_seal(state, engine, cfg, deposits_only, block_sink).await {
        Ok(()) => {
            info!(target: "engine", "Successfully imported deposits-only payload");
            SealTaskError::HoloceneInvalidFlush
        }
        Err(err) => {
            error!(target: "engine", %err, "Critical: deposits-only replacement failed");
            SealTaskError::DepositOnlyPayloadReattemptFailed
        }
    })
}
