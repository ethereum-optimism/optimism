//! Compression configuration.

use crate::{CompressionAlgo, CompressorType};

/// Config for the compressor itself.
#[derive(Debug, Clone)]
pub struct Config {
    /// TargetOutputSize is the target size that the compressed data should reach.
    /// The shadow compressor guarantees that the compressed data stays below
    /// this bound. The ratio compressor might go over.
    pub target_output_size: u64,
    /// ApproxComprRatio to assume (only ratio compressor). Should be slightly smaller
    /// than average from experiments to avoid the chances of creating a small
    /// additional leftover frame.
    pub approx_compr_ratio: f64,
    /// Kind of compressor to use. Must be one of KindKeys. If unset, NewCompressor
    /// will default to RatioKind.
    pub kind: CompressorType,

    /// Type of compression algorithm to use. Must be one of [zlib, brotli-(9|10|11)]
    pub compression_algo: CompressionAlgo,
}
