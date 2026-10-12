//! Inserting a payload into the execution engine.

use crate::{Engine, InsertTaskError, state::EngineSyncStateUpdate};
use alloy_rpc_types_engine::{ExecutionPayloadInputV2, PayloadStatusEnum};
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

impl Engine {
    /// Inserts a payload with `engine_newPayload` and makes it the unsafe head.
    ///
    /// A payload derived from a safe block (`is_payload_safe`) also becomes the safe head.
    pub(in crate::engine) async fn insert(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
        is_payload_safe: bool,
    ) -> Result<L2BlockInfo, InsertTaskError> {
        // Insert the new payload.
        // Form the new unsafe block ref from the execution payload.
        let response = match payload.clone() {
            OpExecutionPayloadEnvelope::V1(payload) => self.client.new_payload_v1(payload).await,
            OpExecutionPayloadEnvelope::V2(payload) => {
                let payload_input = ExecutionPayloadInputV2 {
                    execution_payload: payload.payload_inner,
                    withdrawals: Some(payload.withdrawals),
                };
                self.client.new_payload_v2(payload_input).await
            }
            OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root } => {
                self.client.new_payload_v3(payload, parent_beacon_block_root).await
            }
            OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root } => {
                self.client.new_payload_v4(payload, parent_beacon_block_root).await
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
        if !check_new_payload_status(&response.status) {
            return Err(InsertTaskError::UnexpectedPayloadStatus(response.status));
        }

        let block: OpBlock = payload.try_into_block().map_err(InsertTaskError::FromBlockError)?;
        let new_unsafe_ref = L2BlockInfo::from_block_and_genesis(&block, &self.config.genesis)
            .map_err(InsertTaskError::L2BlockInfoConstruction)?;

        // Send a FCU to canonicalize the imported block.
        self.synchronize(EngineSyncStateUpdate {
            unsafe_head: Some(new_unsafe_ref),
            local_safe_head: is_payload_safe.then_some(new_unsafe_ref),
            safe_head: is_payload_safe.then_some(new_unsafe_ref),
            ..Default::default()
        })
        .await?;

        // The block is now canonical, so anything reading the L2 chain locally can rely on it.
        self.block_sink.block_imported(block, new_unsafe_ref);

        info!(
            target: "engine",
            hash = %new_unsafe_ref.block_info.hash,
            number = new_unsafe_ref.block_info.number,
            "Inserted new unsafe block"
        );

        Ok(new_unsafe_ref)
    }
}

/// Checks the response of the `engine_newPayload` call.
const fn check_new_payload_status(status: &PayloadStatusEnum) -> bool {
    matches!(status, PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing)
}
