//! This module contains common traits for stages within the derivation pipeline.

use alloc::boxed::Box;
use async_trait::async_trait;
use kona_protocol::BlockInfo;

use crate::{PipelineResult, Signal};

/// Providers a way for the pipeline to accept a signal from the driver.
#[async_trait]
pub trait SignalReceiver {
    /// Receives a signal from the driver.
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()>;
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
