//! Utility functions for task execution.

use super::{BuildSealCoupling, BuildTask, BuildTaskError, EngineTaskExt, SealTask, SealTaskError};
use crate::{EngineClient, EngineState, ImportedBlockSink};
use alloy_rpc_types_engine::INVALID_FORK_CHOICE_STATE_ERROR;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_genesis::RollupConfig;
use kona_protocol::OpAttributesWithParent;
use std::sync::Arc;

/// Whether the engine rejected a forkchoice update because the state itself is inconsistent, i.e.
/// the safe or finalized block is not an ancestor of the head. Re-sending such an update can never
/// succeed; the node has to reset.
pub(in crate::task_queue) fn is_invalid_forkchoice_state(
    err: &RpcError<TransportErrorKind>,
) -> bool {
    err.as_error_resp().is_some_and(|e| e.code == INVALID_FORK_CHOICE_STATE_ERROR as i64)
}

/// Error type for build and seal operations.
#[derive(Debug, thiserror::Error)]
pub(in crate::task_queue) enum BuildAndSealError {
    /// An error occurred during the build phase.
    #[error(transparent)]
    Build(#[from] BuildTaskError),
    /// An error occurred during the seal phase.
    #[error(transparent)]
    Seal(#[from] SealTaskError),
}

/// Builds and seals a payload in sequence.
///
/// This is a utility function that:
/// 1. Creates and executes a [`BuildTask`] to initiate block building
/// 2. Creates and executes a [`SealTask`] to seal the block, referencing the initiated payload
///
/// This pattern is commonly used for Holocene deposits-only fallback and other scenarios
/// where a build-then-seal workflow is needed.
///
/// # Arguments
///
/// * `state` - Mutable reference to the engine state
/// * `engine` - The engine client
/// * `cfg` - The rollup configuration
/// * `attributes` - The payload attributes to build
/// * `is_attributes_derived` - Whether the attributes were derived or created by the sequencer
/// * `block_sink` - Where to hand the built block once the engine has canonicalized it
pub(in crate::task_queue) async fn build_and_seal<EngineClient_: EngineClient>(
    state: &mut EngineState,
    engine: Arc<EngineClient_>,
    cfg: Arc<RollupConfig>,
    attributes: OpAttributesWithParent,
    is_attributes_derived: bool,
    block_sink: Arc<dyn ImportedBlockSink>,
) -> Result<(), BuildAndSealError> {
    // Execute the build task
    let payload_id = BuildTask::new(
        engine.clone(),
        cfg.clone(),
        attributes.clone(),
        BuildSealCoupling::Atomic,
        None, // Build task doesn't send the payload yet
    )
    .execute(state)
    .await?;

    // Execute the seal task with the payload ID from the build.
    SealTask::new(
        engine,
        cfg,
        payload_id,
        attributes,
        is_attributes_derived,
        BuildSealCoupling::Atomic,
        None,
        block_sink,
    )
    .execute(state)
    .await?;

    Ok(())
}
