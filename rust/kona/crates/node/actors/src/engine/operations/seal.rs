//! Sealing a block that has been started, and making it canonical.
use super::Engine;
use crate::engine::{InsertTaskError, SealTaskError};
use alloy_rpc_types_engine::{ExecutionPayload, PayloadId};
use kona_engine::EngineGetPayloadVersion;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_provider::ext::engine::OpEngineApi;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

/// How a seal is coupled to the build that produced its payload.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub(in crate::engine) enum BuildSealCoupling {
    /// Build and seal run inside the same engine operation, so no other operation can advance the
    /// unsafe head between them. The unsafe-head staleness check is skipped: a derivation-driven
    /// reorg legitimately seals a payload whose parent differs from the current unsafe head.
    Atomic,
    /// The seal is a separate operation from its build, so another operation may advance the
    /// unsafe head in between, invalidating the built payload as stale.
    Detached,
}

impl Engine {
    /// Seals the payload built from `attributes` and imports it, returning the sealed payload.
    ///
    /// Retrieves the built payload using `engine_getPayload`, then imports it with
    /// `engine_newPayload` and a forkchoice update. A detached seal fails with
    /// [`SealTaskError::UnsafeHeadChangedSinceBuild`] if the unsafe head moved since the build
    /// started. `is_attributes_derived` says whether the attributes were derived or created by the
    /// sequencer.
    pub(in crate::engine) async fn seal_block(
        &mut self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
        is_attributes_derived: bool,
        coupling: BuildSealCoupling,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        debug!(
            target: "engine",
            txs = attributes.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
            is_deposits = attributes.is_deposits_only(),
            "Starting new seal job"
        );

        let unsafe_block_info = self.state.sync_state.unsafe_head().block_info;
        let parent_block_info = attributes.parent.block_info;

        let build_is_stale = coupling == BuildSealCoupling::Detached &&
            (unsafe_block_info.hash != parent_block_info.hash ||
                unsafe_block_info.number != parent_block_info.number);

        if build_is_stale {
            info!(
                target: "engine",
                unsafe_block_info = ?unsafe_block_info,
                parent_block_info = ?parent_block_info,
                "Seal attributes parent does not match unsafe head, returning rebuild error"
            );
            Err(SealTaskError::UnsafeHeadChangedSinceBuild)
        } else {
            // Seal the block and import it into the engine.
            self.seal_and_canonicalize_block(payload_id, attributes, is_attributes_derived).await
        }
    }

    /// Fetches the built payload from the EL, returning the execution envelope.
    ///
    /// ## Engine Method Selection
    /// The method used to fetch the payload from the EL is determined by the payload timestamp. The
    /// method used to import the payload into the engine is determined by the payload version.
    ///
    /// - `engine_getPayloadV2` is used for payloads with a timestamp before the Ecotone fork.
    /// - `engine_getPayloadV3` is used for payloads with a timestamp after the Ecotone fork.
    /// - `engine_getPayloadV4` is used for payloads with a timestamp after the Isthmus fork.
    /// - `engine_getPayloadV5` is used for payloads with a timestamp after the Karst fork.
    pub(super) async fn fetch_payload(
        &self,
        payload_id: PayloadId,
        payload_attrs: &OpAttributesWithParent,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        let payload_timestamp = payload_attrs.attributes().payload_attributes.timestamp;

        debug!(
            target: "engine",
            payload_id = payload_id.to_string(),
            l2_time = payload_timestamp,
            "Sealing payload"
        );

        let get_payload_version =
            EngineGetPayloadVersion::from_cfg(&self.config, payload_timestamp);
        let payload_envelope = match get_payload_version {
            EngineGetPayloadVersion::V5 => {
                // Osaka (Karst) reuses the V4-shaped envelope; only the engine method bumps to V5.
                let payload = self.client.get_payload_v5(payload_id).await.map_err(|e| {
                    error!(target: "engine", "Payload fetch failed: {e}");
                    SealTaskError::GetPayloadFailed(e)
                })?;

                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V4 => {
                let payload = self.client.get_payload_v4(payload_id).await.map_err(|e| {
                    error!(target: "engine", "Payload fetch failed: {e}");
                    SealTaskError::GetPayloadFailed(e)
                })?;

                OpExecutionPayloadEnvelope::V4 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V3 => {
                let payload = self.client.get_payload_v3(payload_id).await.map_err(|e| {
                    error!(target: "engine", "Payload fetch failed: {e}");
                    SealTaskError::GetPayloadFailed(e)
                })?;

                OpExecutionPayloadEnvelope::V3 {
                    parent_beacon_block_root: payload.parent_beacon_block_root,
                    payload: payload.execution_payload,
                }
            }
            EngineGetPayloadVersion::V2 => {
                let payload = self.client.get_payload_v2(payload_id).await.map_err(|e| {
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

    /// Inserts a sealed payload into the engine with Holocene fallback support.
    ///
    /// This function handles:
    /// 1. Inserting the payload
    /// 2. Handling deposits-only payload failures
    /// 3. Holocene fallback via `build_and_seal` if needed
    ///
    /// Returns the inserted block information, or an error if insertion fails.
    async fn import_sealed_payload(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
        attributes: &OpAttributesWithParent,
        is_attributes_derived: bool,
    ) -> Result<L2BlockInfo, SealTaskError> {
        // Insert the new block into the engine.
        let new_block_ref = match self.insert(payload, is_attributes_derived).await {
            Err(InsertTaskError::UnexpectedPayloadStatus(e)) if attributes.is_deposits_only() => {
                error!(target: "engine", error = ?e, "Critical: Deposit-only payload import failed");
                return Err(SealTaskError::DepositOnlyPayloadFailed);
            }
            Err(InsertTaskError::UnexpectedPayloadStatus(e))
                if self
                    .config
                    .is_holocene_active(attributes.attributes().payload_attributes.timestamp) =>
            {
                warn!(target: "engine", error = ?e, "Re-attempting payload import with deposits only.");

                // HOLOCENE: Re-attempt payload import with deposits only
                // First build the deposits-only payload, then seal it
                let deposits_only_attrs = attributes.as_deposits_only();

                return match self.build_and_seal(deposits_only_attrs, is_attributes_derived).await {
                    Ok(_) => {
                        info!(target: "engine", "Successfully imported deposits-only payload");
                        Err(SealTaskError::HoloceneInvalidFlush)
                    }
                    Err(_) => Err(SealTaskError::DepositOnlyPayloadReattemptFailed),
                };
            }
            Err(e) => {
                error!(target: "engine", "Payload import failed: {e}");
                return Err(Box::new(e).into());
            }
            Ok(new_block_ref) => {
                info!(target: "engine", "Successfully imported payload");
                new_block_ref
            }
        };

        Ok(new_block_ref)
    }

    /// Seals and canonicalizes the block by fetching the payload and importing it.
    ///
    /// This function handles:
    /// 1. Fetching the execution payload from the EL
    /// 2. Importing the payload into the engine with Holocene fallback support
    async fn seal_and_canonicalize_block(
        &mut self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
        is_attributes_derived: bool,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        // Fetch the payload just inserted from the EL and import it into the engine.
        let new_payload = self.fetch_payload(payload_id, &attributes).await?;

        // Insert the payload into the engine and reuse its decoded block information.
        let new_block_ref = self
            .import_sealed_payload(new_payload.clone(), &attributes, is_attributes_derived)
            .await?;

        info!(
            target: "engine",
            l2_number = new_block_ref.block_info.number,
            l2_time = new_block_ref.block_info.timestamp,
            "Built and imported new {} block",
            if is_attributes_derived { "safe" } else { "unsafe" },
        );

        Ok(new_payload)
    }
}

#[cfg(test)]
mod tests;
