//! Starting to build a new block.

use super::Engine;
use crate::engine::{BuildTaskError, EngineBuildError};
use alloy_rpc_types_engine::{PayloadId, PayloadStatusEnum};
use kona_engine::{EngineForkchoiceVersion, EngineSyncStateUpdate};
use kona_protocol::OpAttributesWithParent;
use op_alloy_provider::ext::engine::OpEngineApi;

impl Engine {
    /// Starts building a block by sending `engine_forkchoiceUpdated` with the payload attributes,
    /// and returns the [`PayloadId`] to seal it with.
    ///
    /// The forkchoice advertises the attributes' parent as the unsafe head to build on.
    ///
    /// ### Success (`VALID`)
    /// If the build is successful, the [`PayloadId`] is returned for sealing.
    ///
    /// ### Failure (`INVALID`)
    /// If the forkchoice update fails, the [`BuildTaskError`].
    ///
    /// ### Syncing (`SYNCING`)
    /// If the EL is syncing, the build cannot start. This is a temporary state, and the build
    /// may be retried later.
    pub(in crate::engine) async fn start_build(
        &self,
        attributes_envelope: OpAttributesWithParent,
    ) -> Result<PayloadId, BuildTaskError> {
        debug!(
            target: "engine_builder",
            txs = attributes_envelope.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
            is_deposits = attributes_envelope.is_deposits_only(),
            "Starting new build job"
        );

        // Sanity check if the head is behind the finalized head. If it is, this is a critical
        // error.
        if self.state.sync_state.unsafe_head().block_info.number <
            self.state.sync_state.finalized_head().block_info.number
        {
            return Err(BuildTaskError::EngineBuildError(
                EngineBuildError::FinalizedAheadOfUnsafe(
                    self.state.sync_state.unsafe_head().block_info.number,
                    self.state.sync_state.finalized_head().block_info.number,
                ),
            ));
        }

        // When inserting a payload, we advertise the parent's unsafe head as the current unsafe
        // head to build on top of.
        let new_forkchoice = self
            .state
            .sync_state
            .apply_update(EngineSyncStateUpdate {
                unsafe_head: Some(attributes_envelope.parent),
                ..Default::default()
            })
            .create_forkchoice_state();

        let forkchoice_version = EngineForkchoiceVersion::from_cfg(
            &self.config,
            attributes_envelope.attributes.payload_attributes.timestamp,
        );
        let update = match forkchoice_version {
            EngineForkchoiceVersion::V3 => {
                self.client
                    .fork_choice_updated_v3(new_forkchoice, Some(attributes_envelope.attributes))
                    .await
            }
            EngineForkchoiceVersion::V2 => {
                self.client
                    .fork_choice_updated_v2(new_forkchoice, Some(attributes_envelope.attributes))
                    .await
            }
        }
        .map_err(|e| {
            error!(target: "engine_builder", "Forkchoice update failed: {}", e);
            BuildTaskError::EngineBuildError(EngineBuildError::AttributesInsertionFailed(e))
        })?;

        validate_forkchoice_status(update.payload_status.status)?;

        debug!(
            target: "engine_builder",
            unsafe_hash = new_forkchoice.head_block_hash.to_string(),
            safe_hash = new_forkchoice.safe_block_hash.to_string(),
            finalized_hash = new_forkchoice.finalized_block_hash.to_string(),
            "Forkchoice update with attributes successful"
        );

        // Fetch the payload ID from the FCU. If no payload ID was returned, something went wrong -
        // the block building job on the EL should have been initiated.
        let payload_id = update
            .payload_id
            .ok_or(BuildTaskError::EngineBuildError(EngineBuildError::MissingPayloadId))?;

        info!(
            target: "engine_builder",
            "block build started"
        );

        Ok(payload_id)
    }
}

/// Validates the provided [`PayloadStatusEnum`] according to the rules listed below.
///
/// ## Observed [`PayloadStatusEnum`] Variants
/// - `VALID`: Returns Ok(()) - forkchoice update was successful
/// - `INVALID`: Returns error with validation details
/// - `SYNCING`: Returns temporary error - EL is syncing
/// - Other: Returns error for unexpected status codes
fn validate_forkchoice_status(status: PayloadStatusEnum) -> Result<(), BuildTaskError> {
    match status {
        PayloadStatusEnum::Valid => Ok(()),
        PayloadStatusEnum::Invalid { validation_error } => {
            error!(target: "engine_builder", "Forkchoice update failed: {}", validation_error);
            Err(BuildTaskError::EngineBuildError(EngineBuildError::InvalidPayload(
                validation_error,
            )))
        }
        PayloadStatusEnum::Syncing => {
            warn!(target: "engine_builder", "Forkchoice update failed temporarily: EL is syncing");
            Err(BuildTaskError::EngineBuildError(EngineBuildError::EngineSyncing))
        }
        PayloadStatusEnum::Accepted => {
            // Other codes are never returned by `engine_forkchoiceUpdate`
            Err(BuildTaskError::EngineBuildError(EngineBuildError::UnexpectedPayloadStatus(status)))
        }
    }
}

#[cfg(test)]
mod tests;
