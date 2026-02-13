//! Utility functions for task execution.

use super::{BuildTask, BuildTaskError, EngineTaskExt, SealTask, SealTaskError};
use crate::{EngineClient, EngineState};
use kona_genesis::RollupConfig;
use kona_protocol::OpAttributesWithParent;
use std::sync::Arc;

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
pub(in crate::task_queue) async fn build_and_seal<EngineClient_: EngineClient>(
    state: &mut EngineState,
    engine: Arc<EngineClient_>,
    cfg: Arc<RollupConfig>,
    attributes: OpAttributesWithParent,
    is_attributes_derived: bool,
) -> Result<(), BuildAndSealError> {
    // Execute the build task
    let payload_id = BuildTask::new(
        engine.clone(),
        cfg.clone(),
        attributes.clone(),
        None, // Build task doesn't send the payload yet
    )
    .execute(state)
    .await?;

    // Execute the seal task with the payload ID from the build
    SealTask::new(engine, cfg, payload_id, attributes, is_attributes_derived, None)
        .execute(state)
        .await?;

    Ok(())
}
