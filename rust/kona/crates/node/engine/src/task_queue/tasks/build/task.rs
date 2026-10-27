//! A task for building a new block and importing it.
use super::BuildTaskError;
use crate::{
    EngineClient, EngineForkchoiceVersion, EngineState, EngineTaskExt, ForkchoiceStatus,
    state::EngineSyncStateUpdate, task_queue::tasks::build::error::EngineBuildError,
};
use alloy_rpc_types_engine::{PayloadId, PayloadStatusEnum};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::OpAttributesWithParent;
use op_alloy_provider::ext::engine::OpEngineApi;
use std::sync::Arc;
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
pub struct BuildTask {
    /// The engine API client.
    pub engine: Arc<EngineClient>,
    /// The [`RollupConfig`].
    pub cfg: Arc<RollupConfig>,
    /// The [`OpAttributesWithParent`] to instruct the execution layer to build.
    pub attributes: OpAttributesWithParent,
    /// The optional sender through which [`PayloadId`] will be sent after the
    /// block build has been started.
    pub payload_id_tx: Option<mpsc::Sender<PayloadId>>,
}

#[async_trait]
impl EngineTaskExt for BuildTask {
    type Output = PayloadId;

    type Error = BuildTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<PayloadId, BuildTaskError> {
        let payload_id =
            start_build(self.engine.as_ref(), &self.cfg, state, self.attributes.clone()).await?;

        // If a channel was provided, send the payload ID to it.
        if let Some(tx) = &self.payload_id_tx {
            tx.send(payload_id).await.map_err(Box::new)?;
        }

        Ok(payload_id)
    }
}

/// Starts building a block on the execution layer by sending an `engine_forkchoiceUpdated` call
/// with the payload attributes to build, returning the [`PayloadId`] of the build job.
///
/// ### Success (`VALID`)
/// If the build is successful, the [`PayloadId`] is returned for sealing.
///
/// ### Failure (`INVALID`)
/// If the forkchoice update fails, an [`EngineBuildError`] is returned.
///
/// ### Syncing (`SYNCING`)
/// If the EL is syncing, [`EngineBuildError::EngineSyncing`] is returned, and the build should be
/// attempted again later.
///
/// Any other status is unexpected and returned as an [`EngineBuildError`].
#[tracing::instrument(
    name = "build",
    target = "engine_builder",
    level = "info",
    skip_all,
    fields(
        parent = ?attributes_envelope.parent.block_info.id(),
        l2_time = attributes_envelope.attributes().payload_attributes.timestamp,
        txs = attributes_envelope.attributes().transactions.as_ref().map_or(0, |txs| txs.len()),
        is_deposits = attributes_envelope.is_deposits_only(),
        unsafe_hash = %attributes_envelope.parent.block_info.hash,
        safe_hash = %state.sync_state.safe_head().block_info.hash,
        finalized_hash = %state.sync_state.finalized_head().block_info.hash,
    )
)]
pub(in crate::task_queue) async fn start_build(
    engine_client: &EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    attributes_envelope: OpAttributesWithParent,
) -> Result<PayloadId, EngineBuildError> {
    // Sanity check if the head is behind the finalized head. If it is, this is a critical error.
    if state.sync_state.unsafe_head().block_info.number <
        state.sync_state.finalized_head().block_info.number
    {
        return Err(EngineBuildError::FinalizedAheadOfUnsafe(
            state.sync_state.unsafe_head().block_info.number,
            state.sync_state.finalized_head().block_info.number,
        ));
    }

    // When inserting a payload, we advertise the parent's unsafe head as the current unsafe head
    // to build on top of.
    let new_forkchoice = state
        .sync_state
        .apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(attributes_envelope.parent),
            ..Default::default()
        })
        .create_forkchoice_state();

    let forkchoice_version = EngineForkchoiceVersion::from_cfg(
        cfg,
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
    .map_err(EngineBuildError::AttributesInsertionFailed)?;

    match update.payload_status.status {
        PayloadStatusEnum::Valid => {
            // This FCU can satisfy a pending label update too. Keep sync-completion notification
            // at synchronize/import time, so starting a build does not trigger a reset mid-build.
            state.forkchoice_status = ForkchoiceStatus::Applied(new_forkchoice);
        }
        PayloadStatusEnum::Invalid { validation_error } => {
            error!(target: "engine_builder", "Forkchoice update failed: {}", validation_error);
            return Err(EngineBuildError::InvalidPayload(validation_error));
        }
        PayloadStatusEnum::Syncing => {
            warn!(target: "engine_builder", "Forkchoice update failed temporarily: EL is syncing");
            return Err(EngineBuildError::EngineSyncing);
        }
        // Other codes are never returned by `engine_forkchoiceUpdate`.
        status @ PayloadStatusEnum::Accepted => {
            return Err(EngineBuildError::UnexpectedPayloadStatus(status));
        }
    }

    // Fetch the payload ID from the FCU. If no payload ID was returned, something went wrong - the
    // block building job on the EL should have been initiated.
    let payload_id = update.payload_id.ok_or(EngineBuildError::MissingPayloadId)?;

    Ok(payload_id)
}
