//! Mock testing utilities for the [`ChannelBank`](crate::stages::ChannelBank) stage.

use crate::{
    errors::PipelineError,
    stages::NextFrameProvider,
    traits::{OriginAdvancer, OriginProvider, Stage},
    types::PipelineResult,
};
use alloc::{boxed::Box, vec::Vec};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use kona_genesis::SystemConfig;
use kona_protocol::{BlockInfo, Frame};

/// A mock [`NextFrameProvider`] for testing the [`ChannelBank`] stage.
///
/// [`ChannelBank`]: crate::stages::ChannelBank
#[derive(Debug, Default)]
pub struct TestNextFrameProvider {
    /// The data to return.
    pub data: Vec<PipelineResult<Frame>>,
    /// The block info
    pub block_info: Option<BlockInfo>,
    /// Tracks if the channel bank provider has been reset.
    pub reset: bool,
}

impl TestNextFrameProvider {
    /// Creates a new [`TestNextFrameProvider`] with the given data.
    pub fn new(data: Vec<PipelineResult<Frame>>) -> Self {
        Self { data, block_info: Some(BlockInfo::default()), reset: false }
    }
}

impl OriginProvider for TestNextFrameProvider {
    fn origin(&self) -> Option<BlockInfo> {
        self.block_info
    }
}

#[async_trait]
impl OriginAdvancer for TestNextFrameProvider {
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.block_info = self.block_info.map(|mut bi| {
            bi.number += 1;
            bi
        });
        Ok(())
    }
}

#[async_trait]
impl NextFrameProvider for TestNextFrameProvider {
    async fn next_frame(&mut self) -> PipelineResult<Frame> {
        self.data.pop().unwrap_or(Err(PipelineError::Eof.temp()))
    }
}

#[async_trait]
impl Stage for TestNextFrameProvider {
    async fn reset(&mut self, _: BlockNumHash, _: SystemConfig) -> PipelineResult<()> {
        self.reset = true;
        Ok(())
    }

    async fn activate(&mut self) -> PipelineResult<()> {
        self.reset = true;
        Ok(())
    }

    async fn flush_channel(&mut self) -> PipelineResult<()> {
        self.reset = true;
        Ok(())
    }

    async fn provide_block(&mut self, _: BlockInfo) -> PipelineResult<()> {
        self.reset = true;
        Ok(())
    }
}
