//! A task for sealing and importing a sequenced block.

use super::SealTaskError;
use crate::{
    EngineClient, EngineGetPayloadVersion, EngineState, EngineTaskExt, ImportedBlockSink,
    task_queue::insert_payload_with_holocene_fallback,
};
use alloy_rpc_types_engine::{ExecutionPayload, PayloadId};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::OpAttributesWithParent;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{sync::Arc, time::Instant};
use tokio::sync::mpsc;

/// Seals a sequenced build whose parent must still be the unsafe head.
/// Derived builds and imports run together through their shared operation instead.
#[derive(Debug, Clone, Constructor)]
pub struct SealTask<EngineClient_: EngineClient> {
    /// The engine client.
    pub engine: Arc<EngineClient_>,
    /// The chain rules.
    pub cfg: Arc<RollupConfig>,
    /// The build to seal.
    pub payload_id: PayloadId,
    /// The attributes from which the build started.
    pub attributes: OpAttributesWithParent,
    /// Receives the payload or failure; absent callers leave recovery to the queue.
    pub result_tx: Option<mpsc::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>>,
    /// Receives blocks after they become canonical.
    pub block_sink: Arc<dyn ImportedBlockSink>,
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for SealTask<EngineClient_> {
    type Output = ();
    type Error = SealTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), SealTaskError> {
        debug!(
            target: "engine",
            txs = self.attributes.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
            is_deposits = self.attributes.is_deposits_only(),
            "Starting new seal job"
        );
        let result = async {
            let unsafe_block_info = state.sync_state.unsafe_head().block_info;
            let parent_block_info = self.attributes.parent.block_info;
            if unsafe_block_info.id() != parent_block_info.id() {
                info!(target: "engine", ?unsafe_block_info, ?parent_block_info,
                    "Seal attributes parent does not match unsafe head, returning rebuild error");
                return Err(SealTaskError::UnsafeHeadChangedSinceBuild);
            }
            let block_import_start = Instant::now();
            let payload = get_payload(
                self.engine.as_ref(),
                &self.cfg,
                self.payload_id,
                self.attributes.attributes().payload_attributes.timestamp,
            )
            .await?;
            let block = insert_payload_with_holocene_fallback(
                self.engine.as_ref(),
                &self.cfg,
                state,
                &self.attributes,
                payload.clone(),
                false,
                self.block_sink.as_ref(),
            )
            .await?;
            info!(target: "engine", l2_number = block.block_info.number,
                l2_time = block.block_info.timestamp,
                block_import_duration = ?block_import_start.elapsed(),
                "Built and imported new unsafe block");
            Ok(payload)
        }
        .await;
        // Sending the result preserves the existing caller-owned seal error policy.
        if let Some(tx) = &self.result_tx {
            tx.send(result).await.map_err(|error| SealTaskError::MpscSend(Box::new(error)))?;
        } else {
            result?;
        }
        Ok(())
    }
}

/// Fetches the built payload using the Engine API version active at its timestamp.
pub(in crate::task_queue) async fn get_payload(
    engine: &impl EngineClient,
    cfg: &RollupConfig,
    payload_id: PayloadId,
    payload_timestamp: u64,
) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
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
            let payload = engine.get_payload_v5(payload_id).await.map_err(|e| {
                error!(target: "engine", "Payload fetch failed: {e}");
                SealTaskError::GetPayloadFailed(e)
            })?;

            OpExecutionPayloadEnvelope::V4 {
                parent_beacon_block_root: payload.parent_beacon_block_root,
                payload: payload.execution_payload,
            }
        }
        EngineGetPayloadVersion::V4 => {
            let payload = engine.get_payload_v4(payload_id).await.map_err(|e| {
                error!(target: "engine", "Payload fetch failed: {e}");
                SealTaskError::GetPayloadFailed(e)
            })?;

            OpExecutionPayloadEnvelope::V4 {
                parent_beacon_block_root: payload.parent_beacon_block_root,
                payload: payload.execution_payload,
            }
        }
        EngineGetPayloadVersion::V3 => {
            let payload = engine.get_payload_v3(payload_id).await.map_err(|e| {
                error!(target: "engine", "Payload fetch failed: {e}");
                SealTaskError::GetPayloadFailed(e)
            })?;

            OpExecutionPayloadEnvelope::V3 {
                parent_beacon_block_root: payload.parent_beacon_block_root,
                payload: payload.execution_payload,
            }
        }
        EngineGetPayloadVersion::V2 => {
            let payload = engine.get_payload_v2(payload_id).await.map_err(|e| {
                error!(target: "engine", "Payload fetch failed: {e}");
                SealTaskError::GetPayloadFailed(e)
            })?;

            match payload.execution_payload.into_payload() {
                ExecutionPayload::V1(payload) => OpExecutionPayloadEnvelope::V1(payload),
                ExecutionPayload::V2(payload) => OpExecutionPayloadEnvelope::V2(payload),
                _ => unreachable!("the response should be a V1 or V2 payload"),
            }
        }
    };

    Ok(payload_envelope)
}
