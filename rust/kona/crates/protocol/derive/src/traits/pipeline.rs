//! Defines the interface for the core derivation pipeline.

use alloc::boxed::Box;
use alloy_primitives::B256;
use async_trait::async_trait;
use core::iter::Iterator;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};

use crate::{OriginProvider, PipelineErrorKind, StepResult};

/// This trait defines the interface for interacting with the derivation pipeline.
#[async_trait]
pub trait Pipeline: OriginProvider + Iterator<Item = OpAttributesWithParent> {
    /// Peeks at the next [`OpAttributesWithParent`] from the pipeline.
    fn peek(&self) -> Option<&OpAttributesWithParent>;

    /// Attempts to progress the pipeline.
    async fn step(&mut self, cursor: L2BlockInfo) -> StepResult;

    /// Returns the rollup config.
    fn rollup_config(&self) -> &RollupConfig;

    /// Returns the [`SystemConfig`] for the L2 block with the given hash.
    async fn system_config_by_l2_hash(
        &mut self,
        hash: B256,
    ) -> Result<SystemConfig, PipelineErrorKind>;
}
