//! A task to insert an unsafe payload into the execution engine.

use crate::{
    EngineClient, EngineState, EngineTaskExt, InsertTaskError, InsertTaskErrorKind,
    PayloadEnvelopeOrigin, SynchronizeTask, state::EngineSyncStateUpdate,
    validate_execution_payload_envelope_version,
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
    /// Whether the payload was derived from L1, locally sequenced, or remotely received.
    origin: PayloadEnvelopeOrigin,
}

impl<EngineClient_: EngineClient> InsertTask<EngineClient_> {
    /// Creates a task for a payload with an explicit origin.
    pub const fn new(
        client: Arc<EngineClient_>,
        rollup_config: Arc<RollupConfig>,
        payload: OpExecutionPayloadEnvelope,
        origin: PayloadEnvelopeOrigin,
    ) -> Self {
        Self { client, rollup_config, payload, origin }
    }

    /// Wraps an underlying failure with this payload's origin.
    fn error(&self, kind: impl Into<InsertTaskErrorKind>) -> InsertTaskError {
        InsertTaskError::new(self.origin, kind.into())
    }

    /// Checks the response of the `engine_newPayload` call.
    const fn check_new_payload_status(&self, status: &PayloadStatusEnum) -> bool {
        matches!(status, PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing)
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for InsertTask<EngineClient_> {
    type Output = L2BlockInfo;

    type Error = InsertTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<L2BlockInfo, InsertTaskError> {
        validate_execution_payload_envelope_version(&self.rollup_config, &self.payload)
            .map_err(|error| self.error(error))?;

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
                return Err(self.error(InsertTaskErrorKind::InsertFailed(e)));
            }
        };
        if !self.check_new_payload_status(&response.status) {
            return Err(self.error(InsertTaskErrorKind::UnexpectedPayloadStatus(response.status)));
        }

        let insert_duration = insert_time_start.elapsed();

        let block: OpBlock = payload.try_into_block().map_err(|error| self.error(error))?;
        let new_unsafe_ref =
            L2BlockInfo::from_block_and_genesis(&block, &self.rollup_config.genesis)
                .map_err(|error| self.error(InsertTaskErrorKind::L2BlockInfoConstruction(error)))?;

        // Send a FCU to canonicalize the imported block.
        SynchronizeTask::new(
            Arc::clone(&self.client),
            self.rollup_config.clone(),
            EngineSyncStateUpdate {
                cross_unsafe_head: Some(new_unsafe_ref),
                unsafe_head: Some(new_unsafe_ref),
                local_safe_head: self.origin.is_safe().then_some(new_unsafe_ref),
                safe_head: self.origin.is_safe().then_some(new_unsafe_ref),
                ..Default::default()
            },
        )
        .execute(state)
        .await
        .map_err(|error| self.error(error))?;

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
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        ExecutionPayloadEnvelopeVersion,
        test_utils::{TestEngineStateBuilder, test_engine_client_builder},
    };
    use alloy_primitives::Bytes;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatus};
    use kona_genesis::HardForkConfig;

    #[test]
    fn rejects_payload_version_mismatch_before_engine_call() {
        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { canyon_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let client = test_engine_client_builder().with_config(config.clone()).build();
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
            transactions: Vec::new(),
        });
        let task = InsertTask::new(
            Arc::new(client),
            config.clone(),
            payload,
            PayloadEnvelopeOrigin::RemoteSequencer,
        );

        assert!(matches!(
            validate_execution_payload_envelope_version(&config, &task.payload),
            Err(crate::ExecutionPayloadEnvelopeVersionError {
                actual: ExecutionPayloadEnvelopeVersion::V1,
                expected: ExecutionPayloadEnvelopeVersion::V2,
                timestamp: 2,
            })
        ));
    }

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
        let task = InsertTask::new(
            Arc::new(client),
            config,
            payload,
            PayloadEnvelopeOrigin::RemoteSequencer,
        );
        let mut state = TestEngineStateBuilder::new().build();

        assert!(matches!(
            task.execute(&mut state).await,
            Err(error)
                if matches!(
                    error.kind(),
                    InsertTaskErrorKind::UnexpectedPayloadStatus(
                        PayloadStatusEnum::Invalid { .. }
                    )
                )
        ));
    }
}
