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

    // `BuildTask` advertises the attributes parent as the forkchoice head. Keep the in-memory
    // unsafe head in sync with that FCU before `SealTask` verifies the build parent. This matters
    // when derivation builds a replacement block behind the current unsafe tip.
    state.sync_state = state.sync_state.apply_update(crate::state::EngineSyncStateUpdate {
        unsafe_head: Some(attributes.parent),
        cross_unsafe_head: Some(attributes.parent),
        ..Default::default()
    });

    // Execute the seal task with the payload ID from the build
    SealTask::new(engine, cfg, payload_id, attributes, is_attributes_derived, None)
        .execute(state)
        .await?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{
        MockEngineClient, TestAttributesBuilder, TestEngineStateBuilder, test_block_info,
    };
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum};

    #[tokio::test]
    async fn reorgs_in_memory_unsafe_head_before_sealing() {
        let cfg = Arc::new(RollupConfig::default());
        let parent = test_block_info(1);
        let current_unsafe = test_block_info(2);
        let attributes = TestAttributesBuilder::new().with_parent(parent).build();
        let client = Arc::new(
            MockEngineClient::builder()
                .with_config(cfg.clone())
                .with_fork_choice_updated_v2_response(ForkchoiceUpdated {
                    payload_status: PayloadStatus::from_status(PayloadStatusEnum::Valid),
                    payload_id: Some(PayloadId::new([1; 8])),
                })
                .build(),
        );
        let mut state = TestEngineStateBuilder::new()
            .with_unsafe_head(current_unsafe)
            .with_safe_head(parent)
            .with_finalized_head(parent)
            .build();

        let err = build_and_seal(&mut state, client, cfg, attributes, true).await.unwrap_err();

        assert!(!matches!(
            err,
            BuildAndSealError::Seal(SealTaskError::UnsafeHeadChangedSinceBuild)
        ));
        assert_eq!(state.sync_state.unsafe_head(), parent);
        assert_eq!(state.sync_state.cross_unsafe_head(), parent);
    }
}
