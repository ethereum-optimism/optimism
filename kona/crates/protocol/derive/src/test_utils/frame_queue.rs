//! Mock types for the frame queue stage.

use crate::{
    FrameQueueProvider, OriginAdvancer, OriginProvider, PipelineError, PipelineResult, Signal,
    SignalReceiver,
};
use alloc::{boxed::Box, vec::Vec};
use alloy_primitives::Bytes;
use async_trait::async_trait;
use kona_protocol::BlockInfo;

/// A mock [`FrameQueueProvider`] for testing the frame queue stage.
///
/// [`FrameQueue`]: crate::stages::FrameQueue
#[derive(Debug, Default)]
pub struct TestFrameQueueProvider {
    /// The data to return.
    pub data: Vec<PipelineResult<Bytes>>,
    /// The origin to return.
    pub origin: Option<BlockInfo>,
    /// Whether the reset method was called.
    pub reset: bool,
}

impl TestFrameQueueProvider {
    /// Creates a new [`TestFrameQueueProvider`] with the given data.
    pub const fn new(data: Vec<PipelineResult<Bytes>>) -> Self {
        Self { data, origin: None, reset: false }
    }

    /// Sets the origin for the [`TestFrameQueueProvider`].
    pub const fn set_origin(&mut self, origin: BlockInfo) {
        self.origin = Some(origin);
    }
}

impl OriginProvider for TestFrameQueueProvider {
    fn origin(&self) -> Option<BlockInfo> {
        self.origin
    }
}

#[async_trait]
impl OriginAdvancer for TestFrameQueueProvider {
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        Ok(())
    }
}

#[async_trait]
impl FrameQueueProvider for TestFrameQueueProvider {
    type Item = Bytes;

    async fn next_data(&mut self) -> PipelineResult<Self::Item> {
        self.data.pop().unwrap_or(Err(PipelineError::Eof.temp()))
    }
}

#[async_trait]
impl SignalReceiver for TestFrameQueueProvider {
    async fn signal(&mut self, _: Signal) -> PipelineResult<()> {
        self.reset = true;
        Ok(())
    }
}
