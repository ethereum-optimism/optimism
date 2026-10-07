//! A task for sealing a sequenced block and importing it.
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
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::mpsc;

/// Task for sealing a sequenced block and canonicalizing it.
///
/// The [`SealTask`] handles the following parts of the sequencer's block building workflow:
///
/// 1. **Staleness Check**: The seal is enqueued separately from its build, so another task may have
///    moved the unsafe head in between. If the build's parent is no longer the unsafe head, the
///    seal is aborted with [`SealTaskError::UnsafeHeadChangedSinceBuild`].
/// 2. **Payload Construction**: Retrieves the built payload using `engine_getPayload`
/// 3. **Block Import**: Inserts the payload into the engine to canonicalize it
#[derive(Debug, Clone, Constructor)]
pub struct SealTask {
    /// The engine API client.
    pub engine: Arc<EngineClient>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The [`PayloadId`] being sealed.
    pub payload_id: PayloadId,
    /// The [`OpAttributesWithParent`] to instruct the execution layer to build.
    pub attributes: OpAttributesWithParent,
    /// An optional sender to convey success/failure result of the built
    /// [`OpExecutionPayloadEnvelope`] after the block has been built, imported, and canonicalized
    /// or the [`SealTaskError`] that occurred during processing.
    pub result_tx: Option<mpsc::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>>,
    /// Where to hand the decoded block once the engine has canonicalized it.
    pub block_sink: Arc<dyn ImportedBlockSink>,
}

impl SealTask {
    /// Seals and canonicalizes the block by fetching the payload and importing it.
    ///
    /// This function handles:
    /// 1. Fetching the execution payload from the EL
    /// 2. Importing the payload into the engine with Holocene fallback support
    async fn seal_and_canonicalize_block(
        &self,
        state: &mut EngineState,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        // Fetch the payload just inserted from the EL and import it into the engine.
        let new_payload = get_payload(
            self.engine.as_ref(),
            &self.cfg,
            self.payload_id,
            self.attributes.attributes().payload_attributes.timestamp,
        )
        .await?;

        // Insert the payload into the engine and reuse its decoded block information.
        let new_block_ref = insert_payload_with_holocene_fallback(
            self.engine.as_ref(),
            &self.cfg,
            state,
            &self.attributes,
            new_payload.clone(),
            // The payload is sequenced, not derived.
            false,
            self.block_sink.as_ref(),
        )
        .await?;

        info!(
            target: "engine",
            l2_number = new_block_ref.block_info.number,
            l2_time = new_block_ref.block_info.timestamp,
            "Built and imported new unsafe block",
        );

        Ok(new_payload)
    }

    /// Sends the provided result via the `result_tx` sender if one exists, returning the
    /// appropriate error if it does not.
    ///
    /// This allows the original caller to handle errors, removing that burden from the engine,
    /// which may not know the caller's intent or retry preferences. If the original caller did not
    /// provide a mechanism to get notified of updates, handle the error in the default manner in
    /// the task queue logic.
    async fn send_channel_result_or_get_error(
        &self,
        res: Result<OpExecutionPayloadEnvelope, SealTaskError>,
    ) -> Result<(), SealTaskError> {
        // NB: If a response channel was provided, that channel will receive success/failure info,
        // and this task will always succeed. If not, task failure will be relayed to the caller.
        if let Some(tx) = &self.result_tx {
            tx.send(res).await.map_err(|e| SealTaskError::MpscSend(Box::new(e)))?;
        } else {
            res?;
        }

        Ok(())
    }
}

#[async_trait]
impl EngineTaskExt for SealTask {
    type Output = ();

    type Error = SealTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), SealTaskError> {
        debug!(
            target: "engine",
            txs = self.attributes.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
            is_deposits = self.attributes.is_deposits_only(),
            "Starting new seal job"
        );

        let unsafe_block_info = state.sync_state.unsafe_head().block_info;
        let parent_block_info = self.attributes.parent.block_info;

        let build_is_stale = unsafe_block_info.hash != parent_block_info.hash ||
            unsafe_block_info.number != parent_block_info.number;

        let res = if build_is_stale {
            info!(
                target: "engine",
                unsafe_block_info = ?unsafe_block_info,
                parent_block_info = ?parent_block_info,
                "Seal attributes parent does not match unsafe head, returning rebuild error"
            );
            Err(SealTaskError::UnsafeHeadChangedSinceBuild)
        } else {
            // Seal the block and import it into the engine.
            self.seal_and_canonicalize_block(state).await
        };

        self.send_channel_result_or_get_error(res).await?;

        Ok(())
    }
}

/// Seals the execution payload in the EL, returning the execution envelope.
///
/// ## Engine Method Selection
/// The method used to fetch the payload from the EL is determined by the payload timestamp.
///
/// - `engine_getPayloadV2` is used for payloads with a timestamp before the Ecotone fork.
/// - `engine_getPayloadV3` is used for payloads with a timestamp after the Ecotone fork.
/// - `engine_getPayloadV4` is used for payloads with a timestamp after the Isthmus fork.
/// - `engine_getPayloadV5` is used for payloads with a timestamp after the Karst fork.
pub(in crate::task_queue) async fn get_payload(
    engine: &EngineClient,
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
