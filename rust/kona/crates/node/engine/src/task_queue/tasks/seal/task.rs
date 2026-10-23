//! A task for importing a block that has already been started.
use super::SealTaskError;
use crate::{
    EngineClient, EngineGetPayloadVersion, EngineState, EngineTaskExt, ImportedBlockSink,
    InsertTask,
    InsertTaskError::{self},
    task_queue::handle_rejected_attributes,
};
use alloy_rpc_types_engine::{ExecutionPayload, PayloadId};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::mpsc;

/// How a [`SealTask`] is coupled to the build that produced its payload.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum BuildSealCoupling {
    /// Build and seal run inside the same engine task, so no other task can advance the unsafe
    /// head between them. The unsafe-head staleness check is skipped: a derivation-driven reorg
    /// legitimately seals a payload whose parent differs from the current unsafe head.
    Atomic,
    /// The seal is enqueued as a task separate from its build, so another task may advance the
    /// unsafe head in between, invalidating the built payload as stale.
    Detached,
}

/// Task for block sealing and canonicalization.
///
/// The [`SealTask`] handles the following parts of the block building workflow:
///
/// 1. **Payload Construction**: Retrieves the built payload using `engine_getPayload`
/// 2. **Block Import**: Imports the payload using [`InsertTask`] for canonicalization
///
/// ## Error Handling
///
/// The task delegates to [`InsertTaskError`] for payload import failures.
///
/// [`InsertTask`]: crate::InsertTask
/// [`InsertTaskError`]: crate::InsertTaskError
#[derive(Debug, Clone, Constructor)]
#[allow(clippy::too_many_arguments)] // the derived constructor takes one field each
pub struct SealTask {
    /// The engine API client.
    pub engine: Arc<EngineClient>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The [`PayloadId`] being sealed.
    pub payload_id: PayloadId,
    /// The [`OpAttributesWithParent`] to instruct the execution layer to build.
    pub attributes: OpAttributesWithParent,
    /// Whether or not the payload was derived, or created by the sequencer.
    pub is_attributes_derived: bool,
    /// How this seal is coupled to the build that produced `payload_id`.
    pub coupling: BuildSealCoupling,
    /// An optional sender to convey success/failure result of the built
    /// [`OpExecutionPayloadEnvelope`] after the block has been built, imported, and canonicalized
    /// or the [`SealTaskError`] that occurred during processing.
    pub result_tx: Option<mpsc::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>>,
    /// Where to hand the decoded block once the engine has canonicalized it.
    pub block_sink: Arc<dyn ImportedBlockSink>,
}

impl SealTask {
    /// Seals the execution payload in the EL, returning the execution envelope.
    ///
    /// ## Engine Method Selection
    /// The method used to fetch the payload from the EL is determined by the payload timestamp. The
    /// method used to import the payload into the engine is determined by the payload version.
    ///
    /// - `engine_getPayloadV2` is used for payloads with a timestamp before the Ecotone fork.
    /// - `engine_getPayloadV3` is used for payloads with a timestamp after the Ecotone fork.
    /// - `engine_getPayloadV4` is used for payloads with a timestamp after the Isthmus fork.
    #[tracing::instrument(
        name = "get_payload",
        target = "engine",
        level = "info",
        skip_all,
        fields(
            %payload_id,
            l2_time = payload_attrs.attributes().payload_attributes.timestamp,
        )
    )]
    pub(super) async fn seal_payload(
        &self,
        cfg: &RollupConfig,
        engine: &EngineClient,
        payload_id: PayloadId,
        payload_attrs: OpAttributesWithParent,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        let payload_timestamp = payload_attrs.attributes().payload_attributes.timestamp;

        let get_payload_version = EngineGetPayloadVersion::from_cfg(cfg, payload_timestamp);
        let payload_envelope = match get_payload_version {
            EngineGetPayloadVersion::V5 => {
                // Osaka (Karst) reuses the V4-shaped envelope; only the engine method bumps to V5.
                let payload = engine
                    .get_payload_v5(payload_id)
                    .await
                    .map_err(SealTaskError::GetPayloadFailed)?;

                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V4 => {
                let payload = engine
                    .get_payload_v4(payload_id)
                    .await
                    .map_err(SealTaskError::GetPayloadFailed)?;

                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V3 => {
                let payload = engine
                    .get_payload_v3(payload_id)
                    .await
                    .map_err(SealTaskError::GetPayloadFailed)?;

                OpExecutionPayloadEnvelope::V3 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V2 => {
                let payload = engine
                    .get_payload_v2(payload_id)
                    .await
                    .map_err(SealTaskError::GetPayloadFailed)?;

                match payload.execution_payload.into_payload() {
                    ExecutionPayload::V1(payload) => OpExecutionPayloadEnvelope::V1(payload),
                    ExecutionPayload::V2(payload) => OpExecutionPayloadEnvelope::V2(payload),
                    _ => unreachable!("the response should be a V1 or V2 payload"),
                }
            }
        };

        Ok(payload_envelope)
    }

    /// Inserts a payload into the engine. If the engine rejects it, [`handle_rejected_attributes`]
    /// decides whether a deposits-only block replaces it.
    ///
    /// Returns the inserted block information, or an error if insertion fails.
    async fn insert_payload(
        &self,
        state: &mut EngineState,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<L2BlockInfo, SealTaskError> {
        // Insert the new block into the engine.
        let new_block_ref = match InsertTask::new(
            Arc::clone(&self.engine),
            self.cfg.clone(),
            payload,
            self.is_attributes_derived,
            Arc::clone(&self.block_sink),
        )
        .execute(state)
        .await
        {
            Err(err @ InsertTaskError::UnexpectedPayloadStatus(_)) => {
                warn!(target: "engine", %err, "Payload rejected at newPayload");
                return Err(handle_rejected_attributes(
                    state,
                    self.engine.clone(),
                    self.cfg.clone(),
                    &self.attributes,
                    self.is_attributes_derived,
                    self.block_sink.clone(),
                )
                .await
                .unwrap_or_else(|| Box::new(err).into()));
            }
            Err(e) => {
                error!(target: "engine", "Payload import failed: {e}");
                return Err(Box::new(e).into());
            }
            Ok(new_block_ref) => new_block_ref,
        };

        Ok(new_block_ref)
    }

    /// Seals and canonicalizes the block by fetching the payload and importing it.
    ///
    /// This function handles:
    /// 1. Fetching the execution payload from the EL
    /// 2. Importing the payload into the engine with Holocene fallback support
    /// 3. Sending the payload to the optional channel
    async fn seal_and_canonicalize_block(
        &self,
        state: &mut EngineState,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        // Fetch the payload just inserted from the EL and import it into the engine.
        let new_payload = self
            .seal_payload(&self.cfg, &self.engine, self.payload_id, self.attributes.clone())
            .await?;

        // Insert the payload into the engine and reuse its decoded block information.
        let new_block_ref = self.insert_payload(state, new_payload.clone()).await?;

        info!(
            target: "engine",
            l2_number = new_block_ref.block_info.number,
            l2_time = new_block_ref.block_info.timestamp,
            "Built and imported new {} block",
            if self.is_attributes_derived { "safe" } else { "unsafe" },
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

    #[tracing::instrument(
        name = "seal",
        target = "engine",
        level = "info",
        skip_all,
        fields(
            payload_id = %self.payload_id,
            parent_hash = %self.attributes.parent.block_info.hash,
            parent_number = self.attributes.parent.block_info.number,
            l2_time = self.attributes.attributes().payload_attributes.timestamp,
            txs = self.attributes.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
            is_deposits = self.attributes.is_deposits_only(),
            is_attributes_derived = self.is_attributes_derived,
        )
    )]
    async fn execute(&self, state: &mut EngineState) -> Result<(), SealTaskError> {
        let unsafe_block_info = state.sync_state.unsafe_head().block_info;
        let parent_block_info = self.attributes.parent.block_info;

        let build_is_stale = self.coupling == BuildSealCoupling::Detached &&
            (unsafe_block_info.hash != parent_block_info.hash ||
                unsafe_block_info.number != parent_block_info.number);

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
