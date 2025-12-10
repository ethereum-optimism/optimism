//! A task for building a new block and importing it.
use super::BuildTaskError;
use crate::{
    EngineClient, EngineForkchoiceVersion, EngineState, EngineTaskExt,
    state::EngineSyncStateUpdate, task_queue::tasks::build::error::EngineBuildError,
};
use alloy_rpc_types_engine::{PayloadId, PayloadStatusEnum};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::OpAttributesWithParent;
use std::{sync::Arc, time::Instant};
use tokio::sync::mpsc;

/// Task for building new blocks with automatic forkchoice synchronization.
///
/// The [`BuildTask`] only performs the `engine_forkchoiceUpdated` call within the block building
/// workflow. It makes this call with the provided attributes to initiate block building on the
/// execution layer and, if successful, sends the new [`PayloadId`] via the configured sender.
///
/// ## Error Handling
///
/// The task uses [`EngineBuildError`] for build-specific failures during the forkchoice update
/// phase.
///
/// [`EngineBuildError`]: crate::EngineBuildError
#[derive(Debug, Clone, Constructor)]
pub struct BuildTask<EngineClient_: EngineClient> {
    /// The engine API client.
    pub engine: Arc<EngineClient_>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The [`OpAttributesWithParent`] to instruct the execution layer to build.
    pub attributes: OpAttributesWithParent,
    /// The optional sender through which [`PayloadId`] will be sent after the
    /// block build has been started.
    pub payload_id_tx: Option<mpsc::Sender<PayloadId>>,
}

impl<EngineClient_: EngineClient> BuildTask<EngineClient_> {
    /// Validates the provided [PayloadStatusEnum] according to the rules listed below.
    ///
    /// ## Observed [PayloadStatusEnum] Variants
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
                Err(BuildTaskError::EngineBuildError(EngineBuildError::UnexpectedPayloadStatus(
                    status,
                )))
            }
        }
    }

    /// Starts the block building process by sending an initial `engine_forkchoiceUpdate` call with
    /// the payload attributes to build.
    ///
    /// ### Success (`VALID`)
    /// If the build is successful, the [PayloadId] is returned for sealing and the successful
    /// forkchoice update identifier is relayed via the stored `payload_id_tx` sender.
    ///
    /// ### Failure (`INVALID`)
    /// If the forkchoice update fails, the [BuildTaskError].
    ///
    /// ### Syncing (`SYNCING`)
    /// If the EL is syncing, the payload attributes are buffered and the function returns early.
    /// This is a temporary state, and the function should be called again later.
    ///
    /// Note: This is `pub(super)` to allow testing via the `tests` submodule.
    pub(super) async fn start_build(
        &self,
        state: &EngineState,
        engine_client: &EngineClient_,
        attributes_envelope: OpAttributesWithParent,
    ) -> Result<PayloadId, BuildTaskError> {
        // Sanity check if the head is behind the finalized head. If it is, this is a critical
        // error.
        if state.sync_state.unsafe_head().block_info.number <
            state.sync_state.finalized_head().block_info.number
        {
            return Err(BuildTaskError::EngineBuildError(EngineBuildError::FinalizedAheadOfUnsafe(
                state.sync_state.unsafe_head().block_info.number,
                state.sync_state.finalized_head().block_info.number,
            )));
        }

        // When inserting a payload, we advertise the parent's unsafe head as the current unsafe
        // head to build on top of.
        let new_forkchoice = state
            .sync_state
            .apply_update(EngineSyncStateUpdate {
                unsafe_head: Some(attributes_envelope.parent),
                ..Default::default()
            })
            .create_forkchoice_state();

        let forkchoice_version = EngineForkchoiceVersion::from_cfg(
            &self.cfg,
            attributes_envelope.attributes.payload_attributes.timestamp,
        );
        let update = match forkchoice_version {
            EngineForkchoiceVersion::V3 => {
                engine_client
                    .fork_choice_updated_v3(new_forkchoice, Some(attributes_envelope.attributes))
                    .await
            }
            EngineForkchoiceVersion::V2 => {
                engine_client
                    .fork_choice_updated_v2(new_forkchoice, Some(attributes_envelope.attributes))
                    .await
            }
        }
        .map_err(|e| {
            error!(target: "engine_builder", "Forkchoice update failed: {}", e);
            BuildTaskError::EngineBuildError(EngineBuildError::AttributesInsertionFailed(e))
        })?;

        Self::validate_forkchoice_status(update.payload_status.status)?;

        debug!(
            target: "engine_builder",
            unsafe_hash = new_forkchoice.head_block_hash.to_string(),
            safe_hash = new_forkchoice.safe_block_hash.to_string(),
            finalized_hash = new_forkchoice.finalized_block_hash.to_string(),
            "Forkchoice update with attributes successful"
        );

        // Fetch the payload ID from the FCU. If no payload ID was returned, something went wrong -
        // the block building job on the EL should have been initiated.
        update
            .payload_id
            .ok_or(BuildTaskError::EngineBuildError(EngineBuildError::MissingPayloadId))
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for BuildTask<EngineClient_> {
    type Output = PayloadId;

    type Error = BuildTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<PayloadId, BuildTaskError> {
        debug!(
            target: "engine_builder",
            txs = self.attributes.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
            is_deposits = self.attributes.is_deposits_only(),
            "Starting new build job"
        );

        // Start the build by sending an FCU call with the current forkchoice and the input
        // payload attributes.
        let fcu_start_time = Instant::now();
        let payload_id = self.start_build(state, &self.engine, self.attributes.clone()).await?;
        let fcu_duration = fcu_start_time.elapsed();

        info!(
            target: "engine_builder",
            fcu_duration = ?fcu_duration,
            "block build started"
        );

        // If a channel was provided, send the payload ID to it.
        if let Some(tx) = &self.payload_id_tx {
            tx.send(payload_id).await.map_err(Box::new)?;
        }

        Ok(payload_id)
    }
}
