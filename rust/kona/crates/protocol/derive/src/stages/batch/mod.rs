//! Contains stages pertaining to the processing of [`Batch`]es.
//!
//! Sitting after the [`ChannelReader`](crate::stages::channel::ChannelReader) stage, the
//! [`BatchStream`] and [`BatchProvider`] stages are responsible for validating and ordering the
//! [`Batch`]es. The [`BatchStream`] stage is responsible for streaming
//! [`SingleBatch`](kona_protocol::SingleBatch)es from [`SpanBatch`](kona_protocol::SpanBatch)es,
//! while the [`BatchProvider`] stage is responsible for ordering and validating the [`Batch`]es
//! for the [`AttributesQueue`](crate::stages::attributes_queue::AttributesQueue) stage.

use crate::types::PipelineResult;
use alloc::boxed::Box;
use async_trait::async_trait;
use kona_protocol::{Batch, BlockInfo, L2BlockInfo};

mod batch_stream;
pub use batch_stream::{BatchStream, BatchStreamProvider, StagedSpan};

mod batch_queue;
pub use batch_queue::BatchQueue;

mod batch_validator;
pub use batch_validator::BatchValidator;

mod batch_provider;
pub use batch_provider::BatchProvider;

/// A batch on its way to the next stage, together with the claim that only a span batch can make
/// about it.
///
/// The claim cannot ride on the batch itself: a [`SingleBatch`](kona_protocol::SingleBatch) is
/// RLP-encoded as the `0x00` wire format, which has no way to express a sibling.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct StagedBatch<B> {
    /// The batch.
    pub batch: B,
    /// Whether the batch is a span batch element that shares the timestamp of the block it
    /// builds on.
    pub is_sibling: bool,
}

impl<B> StagedBatch<B> {
    /// Stages a batch that no span batch vouched for as a sibling.
    pub const fn new(batch: B) -> Self {
        Self { batch, is_sibling: false }
    }
}

/// Provides [`Batch`]es for the [`BatchQueue`] and [`BatchValidator`] stages.
#[async_trait]
pub trait NextBatchProvider {
    /// Returns the next [`Batch`] in the [`ChannelReader`] stage, if the stage is not complete.
    /// This function can only be called once while the stage is in progress, and will return
    /// [`None`] on subsequent calls unless the stage is reset or complete. If the stage is
    /// complete and the batch has been consumed, an [PipelineError::Eof] error is returned.
    ///
    /// [`ChannelReader`]: crate::stages::ChannelReader
    /// [PipelineError::Eof]: crate::errors::PipelineError::Eof
    async fn next_batch(
        &mut self,
        parent: L2BlockInfo,
        l1_origins: &[BlockInfo],
    ) -> PipelineResult<StagedBatch<Batch>>;

    /// Returns the number of [`SingleBatch`]es that are currently buffered in the [`BatchStream`]
    /// from a [`SpanBatch`].
    ///
    /// [`SpanBatch`]: kona_protocol::SpanBatch
    /// [`SingleBatch`]: kona_protocol::SingleBatch
    fn span_buffer_size(&self) -> usize;

    /// Allows the stage to flush the buffer in the [`crate::stages::BatchStream`]
    /// if an invalid single batch is found. Pre-holocene hardfork, this will be a no-op.
    fn flush(&mut self);
}
