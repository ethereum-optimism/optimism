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
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;

/// The task to insert a payload into the execution engine.
#[derive(Debug, Clone)]
pub struct InsertTask {
    /// The engine client.
    client: Arc<EngineClient>,
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

impl InsertTask {
    /// Creates a new insert task.
    pub const fn new(
        client: Arc<EngineClient>,
        rollup_config: Arc<RollupConfig>,
        payload: OpExecutionPayloadEnvelope,
        is_attributes_derived: bool,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self { client, rollup_config, payload, is_payload_safe: is_attributes_derived, block_sink }
    }
}

#[async_trait]
impl EngineTaskExt for InsertTask {
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

/// Inserts `payload` into the execution engine with `engine_newPayload`, then makes it the unsafe
/// head, and the safe head too if `is_payload_safe`, with a forkchoice update. Returns the
/// inserted block's [`L2BlockInfo`].
///
/// Once the block is canonical it is handed to `block_sink`.
pub(in crate::task_queue) async fn insert_payload(
    client: &EngineClient,
    rollup_config: &RollupConfig,
    state: &mut EngineState,
    payload: OpExecutionPayloadEnvelope,
    is_payload_safe: bool,
    block_sink: &dyn ImportedBlockSink,
) -> Result<L2BlockInfo, InsertTaskError> {
    // Insert the new payload.
    // Form the new unsafe block ref from the execution payload.
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

    info!(
        target: "engine",
        hash = %new_unsafe_ref.block_info.hash,
        number = new_unsafe_ref.block_info.number,
        "Inserted new unsafe block"
    );

    Ok(new_unsafe_ref)
}
