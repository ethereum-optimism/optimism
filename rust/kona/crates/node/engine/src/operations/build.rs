//! Starting and retrieving execution payloads.

use crate::{
    BuildError, EngineClient, EngineForkchoiceVersion, EngineGetPayloadVersion, EnginePhase,
    EngineState, EngineSyncStateUpdate, ForkchoiceError,
};
use alloy_rpc_types_engine::{
    ExecutionPayload, INVALID_PAYLOAD_ATTRIBUTES_ERROR, PayloadId, PayloadStatusEnum,
};
use alloy_transport::TransportResult;
use kona_genesis::RollupConfig;
use kona_protocol::OpAttributesWithParent;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::time::Instant;

/// Starts a build and records the parent forkchoice accepted by the execution layer.
/// The owner validates its epoch and parent context before calling this operation.
pub async fn start_build(
    engine_client: &impl EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes_envelope: &OpAttributesWithParent,
) -> Result<PayloadId, BuildError> {
    debug!(
        target: "engine_builder",
        txs = attributes_envelope.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
        is_deposits = attributes_envelope.is_deposits_only(),
        "Starting new build job"
    );

    // Sanity check if the head is behind the finalized head. If it is, this is a critical error.
    if attributes_envelope.parent.block_info.number <
        state.sync_state.finalized_head().block_info.number
    {
        return Err(BuildError::Forkchoice(ForkchoiceError::FinalizedAheadOfUnsafe(
            attributes_envelope.parent.block_info.number,
            state.sync_state.finalized_head().block_info.number,
        )));
    }

    // When inserting a payload, we advertise the parent's unsafe head as the current unsafe head
    // to build on top of.
    let new_state = state.sync_state.apply_update(EngineSyncStateUpdate {
        unsafe_head: Some(attributes_envelope.parent),
        ..Default::default()
    });
    let new_forkchoice = new_state.create_forkchoice_state();

    let forkchoice_version = EngineForkchoiceVersion::from_cfg(
        cfg,
        attributes_envelope.attributes.payload_attributes.timestamp,
    );
    let fcu_start = Instant::now();
    let update = match forkchoice_version {
        EngineForkchoiceVersion::V3 => {
            engine_client
                .fork_choice_updated_v3(
                    new_forkchoice,
                    Some(attributes_envelope.attributes.clone()),
                )
                .await
        }
        EngineForkchoiceVersion::V2 => {
            engine_client
                .fork_choice_updated_v2(
                    new_forkchoice,
                    Some(attributes_envelope.attributes.clone()),
                )
                .await
        }
    }
    .map_err(|error| {
        if error.as_error_resp().is_some_and(|e| e.code == INVALID_PAYLOAD_ATTRIBUTES_ERROR as i64)
        {
            return BuildError::InvalidAttributes(error);
        }
        BuildError::Forkchoice(error.into())
    })?;

    match update.payload_status.status {
        PayloadStatusEnum::Valid => {
            // A successful FCU already moved the EL, even if its payload ID is missing or the
            // later retrieval/import fails. The owner publishes these partial mutations.
            state.sync_state = new_state;
            state.el_sync_finished = true;
        }
        PayloadStatusEnum::Invalid { validation_error } => {
            error!(target: "engine_builder", "Forkchoice update failed: {}", validation_error);
            return Err(BuildError::Rejected(validation_error));
        }
        PayloadStatusEnum::Syncing => {
            state.phase = EnginePhase::Syncing;
            warn!(target: "engine_builder", "Forkchoice update failed temporarily: EL is syncing");
            return Err(BuildError::Syncing);
        }
        // Other codes are never returned by `engine_forkchoiceUpdate`.
        status @ PayloadStatusEnum::Accepted => {
            return Err(BuildError::UnexpectedStatus(status));
        }
    }

    debug!(
        target: "engine_builder",
        unsafe_hash = new_forkchoice.head_block_hash.to_string(),
        safe_hash = new_forkchoice.safe_block_hash.to_string(),
        finalized_hash = new_forkchoice.finalized_block_hash.to_string(),
        "Forkchoice update with attributes successful"
    );

    // Fetch the payload ID from the FCU. If no payload ID was returned, something went wrong - the
    // block building job on the EL should have been initiated.
    let payload_id = update.payload_id.ok_or(BuildError::MissingPayloadId)?;

    info!(
        target: "engine_builder",
        fcu_duration = ?fcu_start.elapsed(),
        "block build started"
    );

    Ok(payload_id)
}

/// Fetches the built payload using the Engine API version active at its timestamp.
pub async fn get_payload(
    engine: &impl EngineClient,
    cfg: &RollupConfig,
    payload_id: PayloadId,
    payload_timestamp: u64,
) -> TransportResult<OpExecutionPayloadEnvelope> {
    debug!(
        target: "engine",
        payload_id = payload_id.to_string(),
        l2_time = payload_timestamp,
        "Sealing payload"
    );

    let get_payload_version = EngineGetPayloadVersion::from_cfg(cfg, payload_timestamp);
    let payload_envelope = match get_payload_version {
        EngineGetPayloadVersion::V5 => {
            // Osaka (Karst) reuses the V4-shaped envelope; only the engine method bumps to V5.
            let payload = engine.get_payload_v5(payload_id).await?;

            OpExecutionPayloadEnvelope::V4 {
                parent_beacon_block_root: payload.parent_beacon_block_root,
                payload: payload.execution_payload,
            }
        }
        EngineGetPayloadVersion::V4 => {
            let payload = engine.get_payload_v4(payload_id).await?;

            OpExecutionPayloadEnvelope::V4 {
                parent_beacon_block_root: payload.parent_beacon_block_root,
                payload: payload.execution_payload,
            }
        }
        EngineGetPayloadVersion::V3 => {
            let payload = engine.get_payload_v3(payload_id).await?;

            OpExecutionPayloadEnvelope::V3 {
                parent_beacon_block_root: payload.parent_beacon_block_root,
                payload: payload.execution_payload,
            }
        }
        EngineGetPayloadVersion::V2 => {
            let payload = engine.get_payload_v2(payload_id).await?;

            match payload.execution_payload.into_payload() {
                ExecutionPayload::V1(payload) => OpExecutionPayloadEnvelope::V1(payload),
                ExecutionPayload::V2(payload) => OpExecutionPayloadEnvelope::V2(payload),
                _ => unreachable!("the response should be a V1 or V2 payload"),
            }
        }
    };

    Ok(payload_envelope)
}
