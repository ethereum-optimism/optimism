//! Test utilities for the [ChannelReader] stage.
//!
//! [ChannelReader]: crate::stages::ChannelReader

use crate::{
    ChannelReaderProvider, OriginAdvancer, OriginProvider, PipelineError, PipelineResult, Signal,
    SignalReceiver,
};
use alloc::{boxed::Box, vec::Vec};
use alloy_primitives::Bytes;
use async_trait::async_trait;
use kona_protocol::BlockInfo;

/// A mock [`ChannelReaderProvider`] for testing the [`ChannelReader`] stage.
///
/// [`ChannelReader`]: crate::stages::ChannelReader
#[derive(Debug, Default)]
pub struct TestChannelReaderProvider {
    /// The data to return.
    pub data: Vec<PipelineResult<Option<Bytes>>>,
    /// The origin block info
    pub block_info: Option<BlockInfo>,
    /// Tracks if the channel reader provider has been reset.
    pub reset: bool,
}

impl TestChannelReaderProvider {
    /// Creates a new [`TestChannelReaderProvider`] with the given data.
    pub fn new(data: Vec<PipelineResult<Option<Bytes>>>) -> Self {
        Self { data, block_info: Some(BlockInfo::default()), reset: false }
    }
}

impl OriginProvider for TestChannelReaderProvider {
    fn origin(&self) -> Option<BlockInfo> {
        self.block_info
    }
}

#[async_trait]
impl OriginAdvancer for TestChannelReaderProvider {
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        Ok(())
    }
}

#[async_trait]
impl ChannelReaderProvider for TestChannelReaderProvider {
    async fn next_data(&mut self) -> PipelineResult<Option<Bytes>> {
        self.data.pop().unwrap_or(Err(PipelineError::Eof.temp()))
    }
}

#[async_trait]
impl SignalReceiver for TestChannelReaderProvider {
    async fn signal(&mut self, _: Signal) -> PipelineResult<()> {
        self.reset = true;
        Ok(())
    }
}
