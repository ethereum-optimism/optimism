//! Importing and canonicalizing execution payloads.

use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, ImportedBlockSink, InsertError, synchronize,
};
use alloy_rpc_types_engine::{ExecutionPayloadInputV2, PayloadStatusEnum};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::time::Instant;

/// Whether an imported block is unsafe or completes locally derived input.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ImportMode {
    /// Unsafe gossip or sequencing may initiate execution-layer synchronization.
    Unsafe,
    /// Derived blocks require validation; only a concluding block advances completed safe.
    Derived {
        /// Whether this block completes the batch under its active derivation rules.
        concluding: bool,
    },
}

/// Imports a payload, updates forkchoice, and hands the canonical block to the sink.
/// Derived prefixes advance pending safe; concluding blocks also advance completed safe.
pub async fn insert_payload(
    client: &impl EngineClient,
    rollup_config: &RollupConfig,
    state: &mut EngineState,
    payload: &OpExecutionPayloadEnvelope,
    mode: ImportMode,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, InsertError> {
    let time_start = Instant::now();

    // Insert the new payload.
    // Form the new unsafe block ref from the execution payload.
    let insert_time_start = Instant::now();
    let response = match payload.clone() {
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

    // Check the `engine_newPayload` response.
    let response = match response {
        Ok(resp) => resp,
        Err(e) => {
            warn!(target: "engine", "Failed to insert new payload: {e}");
            return Err(InsertError::Rpc(e));
        }
    };
    if response.status != PayloadStatusEnum::Valid &&
        (mode != ImportMode::Unsafe || response.status != PayloadStatusEnum::Syncing)
    {
        return Err(InsertError::Rejected(response.status));
    }
    let insert_duration = insert_time_start.elapsed();

    let block: OpBlock = payload.clone().try_into_block().map_err(InsertError::Payload)?;
    let new_unsafe_ref = L2BlockInfo::from_block_and_genesis(&block, &rollup_config.genesis)
        .map_err(InsertError::BlockInfo)?;

    let concluding = matches!(mode, ImportMode::Derived { concluding: true });

    // Send a FCU to canonicalize the imported block.
    synchronize(
        client,
        state,
        EngineSyncStateUpdate {
            unsafe_head: Some(new_unsafe_ref),
            local_safe_head: concluding.then_some(new_unsafe_ref),
            safe_head: concluding.then_some(new_unsafe_ref),
            ..Default::default()
        },
    )
    .await?;

    if matches!(mode, ImportMode::Derived { .. }) {
        state.pending_safe_head = new_unsafe_ref;
    }

    // The block is now canonical, so anything reading the L2 chain locally can rely on it.
    block_sink.block_imported(block, new_unsafe_ref);

    let total_duration = time_start.elapsed();

    info!(
        target: "engine",
        hash = %new_unsafe_ref.block_info.hash,
        number = new_unsafe_ref.block_info.number,
        total_duration = ?total_duration,
        insert_duration = ?insert_duration,
        "Inserted new unsafe block"
    );

    Ok(new_unsafe_ref)
}
