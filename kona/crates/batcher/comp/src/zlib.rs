//! Contains ZLIB compression and decompression primitives for Optimism.

use crate::{ChannelCompressor, CompressorResult, CompressorWriter};
use alloc::vec::Vec;
use miniz_oxide::inflate::DecompressError;

/// The best compression.
const BEST_ZLIB_COMPRESSION: u8 = 9;

/// Method to compress data using ZLIB.
pub fn compress_zlib(data: &[u8]) -> Vec<u8> {
    miniz_oxide::deflate::compress_to_vec(data, BEST_ZLIB_COMPRESSION)
}

/// Method to decompress data using ZLIB.
pub fn decompress_zlib(data: &[u8]) -> Result<Vec<u8>, DecompressError> {
    miniz_oxide::inflate::decompress_to_vec(data)
}

/// The ZLIB compressor.
#[derive(Debug, Clone, Default)]
#[non_exhaustive]
pub struct ZlibCompressor {
    /// Holds a non-compressed buffer.
    buffer: Vec<u8>,
    /// The compressed buffer.
    compressed: Vec<u8>,
}

impl ZlibCompressor {
    /// Create a new ZLIB compressor.
    pub const fn new() -> Self {
        Self { buffer: Vec::new(), compressed: Vec::new() }
    }
}

impl CompressorWriter for ZlibCompressor {
    fn write(&mut self, data: &[u8]) -> CompressorResult<usize> {
        self.buffer.extend_from_slice(data);
        self.compressed.clear();
        self.compressed.extend_from_slice(&compress_zlib(&self.buffer));
        Ok(data.len())
    }

    fn flush(&mut self) -> CompressorResult<()> {
        Ok(())
    }

    fn close(&mut self) -> CompressorResult<()> {
        Ok(())
    }

    fn reset(&mut self) {
        self.buffer.clear();
        self.compressed.clear();
    }

    fn len(&self) -> usize {
        self.compressed.len()
    }

    fn read(&mut self, buf: &mut [u8]) -> CompressorResult<usize> {
        let len = self.compressed.len().min(buf.len());
        buf[..len].copy_from_slice(&self.compressed[..len]);
        Ok(len)
    }
}

impl ChannelCompressor for ZlibCompressor {
    fn get_compressed(&self) -> Vec<u8> {
        self.compressed.clone()
    }
}
