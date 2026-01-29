//! Contains stages pertaining to the processing of [Batch]es.
//!
//! Sitting after the [ChannelReader] stage, the [`BatchStream`] and [`BatchProvider`] stages are
//! responsible for validating and ordering the [Batch]es. The [`BatchStream`] stage is
//! responsible for streaming [SingleBatch]es from [SpanBatch]es, while the [`BatchProvider`]
//! stage is responsible for ordering and validating the [Batch]es for the [AttributesQueue]
//! stage.
//!
//! [Batch]: kona_protocol::Batch
//! [SingleBatch]: kona_protocol::SingleBatch
//! [SpanBatch]: kona_protocol::SpanBatch
//! [ChannelReader]: crate::stages::channel::ChannelReader
//! [AttributesQueue]: crate::stages::attributes_queue::AttributesQueue

use crate::types::PipelineResult;
use alloc::boxed::Box;
use async_trait::async_trait;
use kona_protocol::{Batch, BlockInfo, L2BlockInfo};

mod batch_stream;
pub use batch_stream::{BatchStream, BatchStreamProvider};

mod batch_queue;
pub use batch_queue::BatchQueue;

mod batch_validator;
pub use batch_validator::BatchValidator;

mod batch_provider;
pub use batch_provider::BatchProvider;

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
    ) -> PipelineResult<Batch>;

    /// Returns the number of [`SingleBatch`]es that are currently buffered in the [`BatchStream`]
    /// from a [`SpanBatch`].
    ///
    /// [`SpanBatch`]: kona_protocol::SpanBatch
    /// [`SingleBatch`]: kona_protocol::SingleBatch
    fn span_buffer_size(&self) -> usize;

    /// Allows the stage to flush the buffer in the [crate::stages::BatchStream]
    /// if an invalid single batch is found. Pre-holocene hardfork, this will be a no-op.
    fn flush(&mut self);
}
