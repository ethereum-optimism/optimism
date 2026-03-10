//! Batch and channel encoding for the derivation pipeline.
//!
//! Encodes L2 blocks as batches, compresses into channels, and splits into frames.

mod channel_out;
mod compression;
mod singular;
mod span;

pub use channel_out::ChannelOut;
pub use compression::CompressionAlgo;
pub use singular::{L1Origin, block_to_singular_batch};
pub use span::build_span_batch;
