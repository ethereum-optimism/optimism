//! This module contains common traits for stages within the derivation pipeline.

use alloc::boxed::Box;
use async_trait::async_trait;
use kona_genesis::SystemConfig;
use alloy_eips::BlockNumHash;
use kona_protocol::BlockInfo;

use crate::{PipelineResult, Signal};

/// Provides a way for the pipeline to accept a signal from the driver.
#[async_trait]
pub trait SignalReceiver {
    /// Receives a signal from the driver.
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()>;
}

/// Trait for resetting and signaling pipeline stages.
///
/// Replaces the internal use of [`SignalReceiver`] for stage-to-stage
/// communication. The [`DerivationPipeline`] receives external [`Signal`]s,
/// performs the L2 walkback to compute the correct L1 origin and system config,
/// then dispatches to stages via these methods.
///
/// Matches op-node's `ResettableStage` pattern where stages receive reset
/// parameters as direct method arguments.
///
/// [`DerivationPipeline`]: crate::DerivationPipeline
#[async_trait]
pub trait StageReset {
    /// Reset the stage to derive from the given L1 origin with the given system config.
    async fn reset(
        &mut self,
        l1_origin: BlockNumHash,
        system_config: SystemConfig,
    ) -> PipelineResult<()>;

    /// Activate a hardfork at the given L1 origin with the given system config.
    /// Default: same as reset. Override only if activation differs (e.g., `BatchQueue`).
    async fn activate(
        &mut self,
        l1_origin: BlockNumHash,
        system_config: SystemConfig,
    ) -> PipelineResult<()> {
        self.reset(l1_origin, system_config).await
    }

    /// Flush the currently active channel.
    async fn flush_channel(&mut self) -> PipelineResult<()>;

    /// Provide a new L1 block to the traversal stage.
    async fn provide_block(&mut self, block: BlockInfo) -> PipelineResult<()>;
}

/// Provides a method for accessing the pipeline's current L1 origin.
pub trait OriginProvider {
    /// Returns the optional L1 [`BlockInfo`] origin.
    fn origin(&self) -> Option<BlockInfo>;
}

/// Defines a trait for advancing the L1 origin of the pipeline.
#[async_trait]
pub trait OriginAdvancer {
    /// Advances the internal state of the lowest stage to the next l1 origin.
    /// This method is the equivalent of the reference implementation `advance_l1_block`.
    async fn advance_origin(&mut self) -> PipelineResult<()>;
}
