//! Stages pertaining to the reading and decoding of channels.
//!
//! Sitting after the [FrameQueue] stage, the [ChannelBank] and [ChannelAssembler] stages are
//! responsible for reading and decoding the [Frame]s into [Channel]s. The [ChannelReader] stage
//! is responsible for decoding the [Channel]s into [Batch]es, forwarding the [Batch]es to the
//! [BatchQueue] stage.
//!
//! [Frame]: kona_protocol::Frame
//! [Channel]: kona_protocol::Channel
//! [Batch]: kona_protocol::Batch
//! [FrameQueue]: crate::stages::FrameQueue
//! [BatchQueue]: crate::stages::BatchQueue

use crate::types::PipelineResult;
use alloc::boxed::Box;
use async_trait::async_trait;
use kona_protocol::Frame;

pub(crate) mod channel_provider;
pub use channel_provider::ChannelProvider;

pub(crate) mod channel_bank;
pub use channel_bank::ChannelBank;

pub(crate) mod channel_assembler;
pub use channel_assembler::ChannelAssembler;

pub(crate) mod channel_reader;
pub use channel_reader::{ChannelReader, ChannelReaderProvider};

/// Provides frames for the [`ChannelBank`] and [`ChannelAssembler`] stages.
#[async_trait]
pub trait NextFrameProvider {
    /// Retrieves the next [`Frame`] from the [`FrameQueue`] stage.
    ///
    /// [`FrameQueue`]: crate::stages::FrameQueue
    async fn next_frame(&mut self) -> PipelineResult<Frame>;
}
