//! A task for importing a block that has already been started.
use super::SealTaskError;
use crate::{
    EngineClient, EngineGetPayloadVersion, EngineState, EngineTaskExt, InsertTask,
    InsertTaskError::{self},
    task_queue::build_and_seal,
};
use alloy_rpc_types_engine::{ExecutionPayload, PayloadId};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::{OpExecutionPayload, OpExecutionPayloadEnvelope};
use std::{sync::Arc, time::Instant};
use tokio::sync::mpsc;

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
pub struct SealTask<EngineClient_: EngineClient> {
    /// The engine API client.
    pub engine: Arc<EngineClient_>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The [`PayloadId`] being sealed.
    pub payload_id: PayloadId,
    /// The [`OpAttributesWithParent`] to instruct the execution layer to build.
    pub attributes: OpAttributesWithParent,
    /// Whether or not the payload was derived, or created by the sequencer.
    pub is_attributes_derived: bool,
    /// An optional sender to convey success/failure result of the built
    /// [`OpExecutionPayloadEnvelope`] after the block has been built, imported, and canonicalized
    /// or the [`SealTaskError`] that occurred during processing.
    pub result_tx: Option<mpsc::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>>,
}

impl<EngineClient_: EngineClient> SealTask<EngineClient_> {
    /// Seals the execution payload in the EL, returning the execution envelope.
    ///
    /// ## Engine Method Selection
    /// The method used to fetch the payload from the EL is determined by the payload timestamp. The
    /// method used to import the payload into the engine is determined by the payload version.
    ///
    /// - `engine_getPayloadV2` is used for payloads with a timestamp before the Ecotone fork.
    /// - `engine_getPayloadV3` is used for payloads with a timestamp after the Ecotone fork.
    /// - `engine_getPayloadV4` is used for payloads with a timestamp after the Isthmus fork.
    async fn seal_payload(
        &self,
        cfg: &RollupConfig,
        engine: &EngineClient_,
        payload_id: PayloadId,
        payload_attrs: OpAttributesWithParent,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        let payload_timestamp = payload_attrs.attributes().payload_attributes.timestamp;

        debug!(
            target: "engine",
            payload_id = payload_id.to_string(),
            l2_time = payload_timestamp,
            "Sealing payload"
        );

        let get_payload_version = EngineGetPayloadVersion::from_cfg(cfg, payload_timestamp);
        let payload_envelope = match get_payload_version {
            EngineGetPayloadVersion::V4 => {
                let payload = engine.get_payload_v4(payload_id).await.map_err(|e| {
                    error!(target: "engine", "Payload fetch failed: {e}");
                    SealTaskError::GetPayloadFailed(e)
                })?;

                OpExecutionPayloadEnvelope {
                    parent_beacon_block_root: Some(payload.parent_beacon_block_root),
                    execution_payload: OpExecutionPayload::V4(payload.execution_payload),
                }
            }
            EngineGetPayloadVersion::V3 => {
                let payload = engine.get_payload_v3(payload_id).await.map_err(|e| {
                    error!(target: "engine", "Payload fetch failed: {e}");
                    SealTaskError::GetPayloadFailed(e)
                })?;

                OpExecutionPayloadEnvelope {
                    parent_beacon_block_root: Some(payload.parent_beacon_block_root),
                    execution_payload: OpExecutionPayload::V3(payload.execution_payload),
                }
            }
            EngineGetPayloadVersion::V2 => {
                let payload = engine.get_payload_v2(payload_id).await.map_err(|e| {
                    error!(target: "engine", "Payload fetch failed: {e}");
                    SealTaskError::GetPayloadFailed(e)
                })?;

                OpExecutionPayloadEnvelope {
                    parent_beacon_block_root: None,
                    execution_payload: match payload.execution_payload.into_payload() {
                        ExecutionPayload::V1(payload) => OpExecutionPayload::V1(payload),
                        ExecutionPayload::V2(payload) => OpExecutionPayload::V2(payload),
                        _ => unreachable!("the response should be a V1 or V2 payload"),
                    },
                }
            }
        };

        Ok(payload_envelope)
    }

    /// Inserts a payload into the engine with Holocene fallback support.
    ///
    /// This function handles:
    /// 1. Executing the InsertTask to import the payload
    /// 2. Handling deposits-only payload failures
    /// 3. Holocene fallback via build_and_seal if needed
    ///
    /// Returns Ok(()) if the payload is successfully inserted, or an error if insertion fails.
    async fn insert_payload(
        &self,
        state: &mut EngineState,
        new_payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), SealTaskError> {
        // Insert the new block into the engine.
        match InsertTask::new(
            Arc::clone(&self.engine),
            self.cfg.clone(),
            new_payload.clone(),
            self.is_attributes_derived,
        )
        .execute(state)
        .await
        {
            Err(InsertTaskError::UnexpectedPayloadStatus(e))
                if self.attributes.is_deposits_only() =>
            {
                error!(target: "engine", error = ?e, "Critical: Deposit-only payload import failed");
                return Err(SealTaskError::DepositOnlyPayloadFailed);
            }
            Err(InsertTaskError::UnexpectedPayloadStatus(e))
                if self.cfg.is_holocene_active(
                    self.attributes.attributes().payload_attributes.timestamp,
                ) =>
            {
                warn!(target: "engine", error = ?e, "Re-attempting payload import with deposits only.");

                // HOLOCENE: Re-attempt payload import with deposits only
                // First build the deposits-only payload, then seal it
                let deposits_only_attrs = self.attributes.as_deposits_only();

                return match build_and_seal(
                    state,
                    self.engine.clone(),
                    self.cfg.clone(),
                    deposits_only_attrs.clone(),
                    self.is_attributes_derived,
                )
                .await
                {
                    Ok(_) => {
                        info!(target: "engine", "Successfully imported deposits-only payload");
                        Err(SealTaskError::HoloceneInvalidFlush)
                    }
                    Err(_) => Err(SealTaskError::DepositOnlyPayloadReattemptFailed),
                }
            }
            Err(e) => {
                error!(target: "engine", "Payload import failed: {e}");
                return Err(Box::new(e).into());
            }
            Ok(_) => {
                info!(target: "engine", "Successfully imported payload")
            }
        }

        Ok(())
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
        let block_import_start_time = Instant::now();
        let new_payload = self
            .seal_payload(&self.cfg, &self.engine, self.payload_id, self.attributes.clone())
            .await?;

        let new_block_ref = L2BlockInfo::from_payload_and_genesis(
            new_payload.execution_payload.clone(),
            self.attributes.attributes().payload_attributes.parent_beacon_block_root,
            &self.cfg.genesis,
        )
        .map_err(SealTaskError::FromBlock)?;

        // Insert the payload into the engine.
        self.insert_payload(state, new_payload.clone()).await?;

        let block_import_duration = block_import_start_time.elapsed();

        info!(
            target: "engine",
            l2_number = new_block_ref.block_info.number,
            l2_time = new_block_ref.block_info.timestamp,
            block_import_duration = ?block_import_duration,
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
        } else if let Err(x) = res {
            return Err(x)
        }

        Ok(())
    }
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

        let unsafe_block_info = state.sync_state.unsafe_head().block_info;
        let parent_block_info = self.attributes.parent.block_info;

        let res = if unsafe_block_info.hash != parent_block_info.hash ||
            unsafe_block_info.number != parent_block_info.number
        {
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
