//! Error types for derivation pipeline stages.

use kona_protocol::MAX_SPAN_BATCH_ELEMENTS;
use thiserror::Error;

/// A frame decompression error.
#[derive(Error, Debug, PartialEq, Eq)]
pub enum BatchDecompressionError {
    /// The buffer exceeds the [`MAX_SPAN_BATCH_ELEMENTS`] protocol parameter.
    #[error("The batch exceeds the maximum number of elements: {max_size}", max_size = MAX_SPAN_BATCH_ELEMENTS)]
    BatchTooLarge,
}
