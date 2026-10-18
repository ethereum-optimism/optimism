//! A task to insert an unsafe payload into the execution engine.

use crate::{
    EngineClient, EngineState, EngineTaskExt, ImportedBlockSink, InsertTaskError,
    state::EngineSyncStateUpdate, task_queue::synchronize,
};
use alloy_rpc_types_engine::{ExecutionPayloadInputV2, PayloadStatusEnum};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{sync::Arc, time::Instant};

/// The task to insert a payload into the execution engine.
#[derive(Debug, Clone)]
pub struct InsertTask<EngineClient_: EngineClient> {
    /// The engine client.
    client: Arc<EngineClient_>,
    /// The rollup config.
    rollup_config: Arc<RollupConfig>,
    /// The complete execution payload envelope.
    payload: OpExecutionPayloadEnvelope,
    /// If the payload is safe this is true.
    /// A payload is safe if it is derived from a safe block.
    is_payload_safe: bool,
    /// Where to hand the decoded block once the engine has canonicalized it.
    block_sink: Arc<dyn ImportedBlockSink>,
}

impl<EngineClient_: EngineClient> InsertTask<EngineClient_> {
    /// Creates a new insert task.
    pub const fn new(
        client: Arc<EngineClient_>,
        rollup_config: Arc<RollupConfig>,
        payload: OpExecutionPayloadEnvelope,
        is_attributes_derived: bool,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self { client, rollup_config, payload, is_payload_safe: is_attributes_derived, block_sink }
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for InsertTask<EngineClient_> {
    type Output = L2BlockInfo;
    type Error = InsertTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<L2BlockInfo, InsertTaskError> {
        insert_payload(
            self.client.as_ref(),
            &self.rollup_config,
            state,
            self.payload.clone(),
            self.is_payload_safe,
            self.block_sink.as_ref(),
        )
        .await
    }
}

/// Imports a payload, updates forkchoice, and hands the canonical block to the sink.
/// Derived payloads also advance the safe heads.
pub(in crate::task_queue) async fn insert_payload(
    client: &impl EngineClient,
    rollup_config: &RollupConfig,
    state: &mut EngineState,
    payload: OpExecutionPayloadEnvelope,
    is_payload_safe: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, InsertTaskError> {
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
            return Err(InsertTaskError::InsertFailed(e));
        }
    };
    if !matches!(response.status, PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing) {
        return Err(InsertTaskError::UnexpectedPayloadStatus(response.status));
    }
    let insert_duration = insert_time_start.elapsed();

    let block: OpBlock = payload.try_into_block().map_err(InsertTaskError::FromBlockError)?;
    let new_unsafe_ref = L2BlockInfo::from_block_and_genesis(&block, &rollup_config.genesis)
        .map_err(InsertTaskError::L2BlockInfoConstruction)?;

    // Send a FCU to canonicalize the imported block.
    synchronize(
        client,
        state,
        EngineSyncStateUpdate {
            unsafe_head: Some(new_unsafe_ref),
            local_safe_head: is_payload_safe.then_some(new_unsafe_ref),
            safe_head: is_payload_safe.then_some(new_unsafe_ref),
            ..Default::default()
        },
    )
    .await?;

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
