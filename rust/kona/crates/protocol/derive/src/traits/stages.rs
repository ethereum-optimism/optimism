//! This module contains common traits for stages within the derivation pipeline.

use alloc::boxed::Box;
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use kona_genesis::SystemConfig;
use kona_protocol::BlockInfo;

use crate::{PipelineResult, Signal};

/// Provides a way for the pipeline to accept a signal from the driver.
#[async_trait]
pub trait SignalReceiver {
    /// Receives a signal from the driver.
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()>;
}

/// Trait for pipeline stages.
///
/// The [`DerivationPipeline`] receives external [`Signal`]s, computes
/// the correct L1 origin and system config, then dispatches to stages
/// via these methods.
///
/// [`DerivationPipeline`]: crate::DerivationPipeline
#[async_trait]
pub trait Stage {
    /// Reset the stage to derive from the given L1 origin with the given system config.
    async fn reset(
        &mut self,
        l1_origin: BlockNumHash,
        system_config: SystemConfig,
    ) -> PipelineResult<()>;

    /// Soft-reset for hardfork activation. Clears buffered channels and batches
    /// but preserves derivation state (origin, epoch tracking, system config).
    async fn activate(&mut self) -> PipelineResult<()>;

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
