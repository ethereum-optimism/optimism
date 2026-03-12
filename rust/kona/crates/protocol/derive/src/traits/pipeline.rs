//! Defines the interface for the core derivation pipeline.

use alloc::{boxed::Box, vec::Vec};
use async_trait::async_trait;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};

use crate::{OriginProvider, PipelineErrorKind, PipelineResult};

/// This trait defines the interface for interacting with the derivation pipeline.
#[async_trait]
pub trait Pipeline: OriginProvider {
    /// Attempts to progress the pipeline by one step.
    ///
    /// Calls `next_attributes` once. On success, returns a single-element vec.
    /// On EOF, advances the L1 origin and returns an empty vec.
    /// On any other error, returns the error for the caller to handle.
    async fn step(
        &mut self,
        cursor: L2BlockInfo,
    ) -> PipelineResult<Vec<OpAttributesWithParent>>;

    /// Returns the rollup config.
    fn rollup_config(&self) -> &RollupConfig;

    /// Returns the [`SystemConfig`] by L2 number.
    async fn system_config_by_number(
        &mut self,
        number: u64,
    ) -> Result<SystemConfig, PipelineErrorKind>;
}
