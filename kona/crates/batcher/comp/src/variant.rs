//! A variant over the different implementations of [ChannelCompressor].

use crate::{
    BrotliCompressor, ChannelCompressor, CompressionAlgo, CompressorResult, CompressorWriter,
    ZlibCompressor,
};
use kona_genesis::RollupConfig;

/// The channel compressor wraps the brotli and zlib compressor types,
/// implementing the [ChannelCompressor] trait itself.
#[derive(Debug, Clone)]
pub enum VariantCompressor {
    /// The brotli compressor.
    Brotli(BrotliCompressor),
    /// The zlib compressor.
    Zlib(ZlibCompressor),
}

impl VariantCompressor {
    /// Constructs a [VariantCompressor] using the given [RollupConfig] and timestamp.
    pub fn from_timestamp(config: &RollupConfig, timestamp: u64) -> Self {
        if config.is_fjord_active(timestamp) {
            Self::Brotli(BrotliCompressor::new(CompressionAlgo::Brotli10))
        } else {
            Self::Zlib(ZlibCompressor::new())
        }
    }
}

impl CompressorWriter for VariantCompressor {
    fn write(&mut self, data: &[u8]) -> CompressorResult<usize> {
        match self {
            Self::Brotli(compressor) => compressor.write(data),
            Self::Zlib(compressor) => compressor.write(data),
        }
    }

    fn flush(&mut self) -> CompressorResult<()> {
        match self {
            Self::Brotli(compressor) => compressor.flush(),
            Self::Zlib(compressor) => compressor.flush(),
        }
    }

    fn close(&mut self) -> CompressorResult<()> {
        match self {
            Self::Brotli(compressor) => compressor.close(),
            Self::Zlib(compressor) => compressor.close(),
        }
    }

    fn reset(&mut self) {
        match self {
            Self::Brotli(compressor) => compressor.reset(),
            Self::Zlib(compressor) => compressor.reset(),
        }
    }

    fn len(&self) -> usize {
        match self {
            Self::Brotli(compressor) => compressor.len(),
            Self::Zlib(compressor) => compressor.len(),
        }
    }

    fn read(&mut self, buf: &mut [u8]) -> CompressorResult<usize> {
        match self {
            Self::Brotli(compressor) => compressor.read(buf),
            Self::Zlib(compressor) => compressor.read(buf),
        }
    }
}

impl ChannelCompressor for VariantCompressor {
    fn get_compressed(&self) -> Vec<u8> {
        match self {
            Self::Brotli(compressor) => compressor.get_compressed(),
            Self::Zlib(compressor) => compressor.get_compressed(),
        }
    }
}

impl From<CompressionAlgo> for VariantCompressor {
    fn from(algo: CompressionAlgo) -> Self {
        match algo {
            lvl @ CompressionAlgo::Brotli9 => Self::Brotli(BrotliCompressor::new(lvl)),
            lvl @ CompressionAlgo::Brotli10 => Self::Brotli(BrotliCompressor::new(lvl)),
            lvl @ CompressionAlgo::Brotli11 => Self::Brotli(BrotliCompressor::new(lvl)),
            CompressionAlgo::Zlib => Self::Zlib(ZlibCompressor::new()),
        }
    }
}
