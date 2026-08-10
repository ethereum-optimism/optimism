//! A task to insert an unsafe payload into the execution engine.

use crate::{
    EngineClient, EngineState, EngineTaskExt, InsertTaskError, SynchronizeTask,
    state::EngineSyncStateUpdate,
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
}

impl<EngineClient_: EngineClient> InsertTask<EngineClient_> {
    /// Creates a new insert task.
    pub const fn new(
        client: Arc<EngineClient_>,
        rollup_config: Arc<RollupConfig>,
        payload: OpExecutionPayloadEnvelope,
        is_attributes_derived: bool,
    ) -> Self {
        Self { client, rollup_config, payload, is_payload_safe: is_attributes_derived }
    }

    /// Checks the response of the `engine_newPayload` call.
    const fn check_new_payload_status(&self, status: &PayloadStatusEnum) -> bool {
        matches!(status, PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing)
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for InsertTask<EngineClient_> {
    type Output = ();

    type Error = InsertTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), InsertTaskError> {
        let time_start = Instant::now();

        // Insert the new payload.
        // Form the new unsafe block ref from the execution payload.
        let payload = self.payload.clone();
        let insert_time_start = Instant::now();
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

        // Check the `engine_newPayload` response before decoding transactions locally. The
        // execution layer's terminal response takes precedence for externally supplied payloads.
        let response = match response {
            Ok(resp) => resp,
            Err(e) => {
                warn!(target: "engine", "Failed to insert new payload: {e}");
                return Err(InsertTaskError::InsertFailed(e));
            }
        };
        if !self.check_new_payload_status(&response.status) {
            return Err(InsertTaskError::UnexpectedPayloadStatus(response.status));
        }

        let block: OpBlock = payload.try_into_block().map_err(InsertTaskError::FromBlockError)?;
        let insert_duration = insert_time_start.elapsed();

        let new_unsafe_ref =
            L2BlockInfo::from_block_and_genesis(&block, &self.rollup_config.genesis)
                .map_err(InsertTaskError::L2BlockInfoConstruction)?;

        // Send a FCU to canonicalize the imported block.
        SynchronizeTask::new(
            Arc::clone(&self.client),
            self.rollup_config.clone(),
            EngineSyncStateUpdate {
                cross_unsafe_head: Some(new_unsafe_ref),
                unsafe_head: Some(new_unsafe_ref),
                local_safe_head: self.is_payload_safe.then_some(new_unsafe_ref),
                safe_head: self.is_payload_safe.then_some(new_unsafe_ref),
                ..Default::default()
            },
        )
        .execute(state)
        .await?;

        let total_duration = time_start.elapsed();

        info!(
            target: "engine",
            hash = %new_unsafe_ref.block_info.hash,
            number = new_unsafe_ref.block_info.number,
            total_duration = ?total_duration,
            insert_duration = ?insert_duration,
            "Inserted new unsafe block"
        );

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{TestEngineStateBuilder, test_engine_client_builder};
    use alloy_primitives::Bytes;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatus};

    #[tokio::test]
    async fn invalid_status_precedes_local_transaction_decoding() {
        let config = Arc::new(RollupConfig::default());
        let client = test_engine_client_builder()
            .with_config(config.clone())
            .with_new_payload_v1_response(PayloadStatus {
                status: PayloadStatusEnum::Invalid {
                    validation_error: "invalid transaction".to_string(),
                },
                latest_valid_hash: None,
            })
            .build();
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
            parent_hash: Default::default(),
            fee_recipient: Default::default(),
            state_root: Default::default(),
            receipts_root: Default::default(),
            logs_bloom: Default::default(),
            prev_randao: Default::default(),
            block_number: 1,
            gas_limit: 0,
            gas_used: 0,
            timestamp: 2,
            extra_data: Default::default(),
            base_fee_per_gas: Default::default(),
            block_hash: Default::default(),
            transactions: vec![Bytes::from_static(&[0xff])],
        });
        let task = InsertTask::new(Arc::new(client), config, payload, false);
        let mut state = TestEngineStateBuilder::new().build();

        assert!(matches!(
            task.execute(&mut state).await,
            Err(InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. }))
        ));
    }
}
