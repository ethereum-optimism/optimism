//! A mock implementation of the [`NextBatchProvider`] stage for testing.

use crate::{
    errors::PipelineError,
    stages::NextBatchProvider,
    traits::{OriginAdvancer, OriginProvider, SignalReceiver},
    types::{PipelineResult, Signal},
};
use alloc::{boxed::Box, vec::Vec};
use async_trait::async_trait;
use kona_protocol::{Batch, BlockInfo, L2BlockInfo};

/// A mock provider for the [`NextBatchProvider`] stage.
#[derive(Debug, Default)]
pub struct TestNextBatchProvider {
    /// The origin of the L1 block.
    pub origin: Option<BlockInfo>,
    /// A list of batches to return.
    pub batches: Vec<PipelineResult<Batch>>,
    /// Tracks if the provider has been flushed.
    pub flushed: bool,
    /// Tracks if the reset method was called.
    pub reset: bool,
}

impl TestNextBatchProvider {
    /// Creates a new [`TestNextBatchProvider`] with the given origin and batches.
    pub fn new(batches: Vec<PipelineResult<Batch>>) -> Self {
        Self { origin: Some(BlockInfo::default()), batches, flushed: false, reset: false }
    }
}

impl OriginProvider for TestNextBatchProvider {
    fn origin(&self) -> Option<BlockInfo> {
        self.origin
    }
}

#[async_trait]
impl NextBatchProvider for TestNextBatchProvider {
    fn flush(&mut self) {
        self.flushed = true;
    }

    fn span_buffer_size(&self) -> usize {
        self.batches.len()
    }

    async fn next_batch(&mut self, _: L2BlockInfo, _: &[BlockInfo]) -> PipelineResult<Batch> {
        self.batches.pop().ok_or(PipelineError::Eof.temp())?
    }
}

#[async_trait]
impl OriginAdvancer for TestNextBatchProvider {
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.origin = self.origin.map(|mut origin| {
            origin.number += 1;
            origin
        });
        Ok(())
    }
}

#[async_trait]
impl SignalReceiver for TestNextBatchProvider {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            Signal::Reset { .. } => self.reset = true,
            Signal::FlushChannel => self.flushed = true,
            _ => {}
        }
        Ok(())
    }
}
