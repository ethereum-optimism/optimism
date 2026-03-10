//! Compression utilities for batch data.

use std::io::Write;

/// Compression algorithm for channel data.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CompressionAlgo {
    /// Zlib compression (pre-Fjord).
    Zlib,
    /// Brotli compression with 0x01 prefix byte (Fjord+).
    Brotli,
}

/// Compress data using the specified algorithm.
pub(super) fn compress(data: &[u8], algo: CompressionAlgo) -> Vec<u8> {
    match algo {
        CompressionAlgo::Zlib => {
            
            miniz_oxide::deflate::compress_to_vec_zlib(data, 6)
        }
        CompressionAlgo::Brotli => {
            // Brotli channel data has a 0x01 prefix byte (channel_version_brotli)
            let mut output = vec![0x01];
            let mut compressor = brotli::CompressorWriter::new(&mut output, 4096, 6, 22);
            compressor.write_all(data).expect("brotli compression failed");
            drop(compressor);
            output
        }
    }
}
