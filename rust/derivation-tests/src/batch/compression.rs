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

/// Decompress zlib-compressed data.
#[cfg(test)]
fn decompress_zlib(data: &[u8]) -> Vec<u8> {
    miniz_oxide::inflate::decompress_to_vec_zlib(data).expect("zlib decompression failed")
}

/// Decompress brotli-compressed data (without the 0x01 prefix byte).
#[cfg(test)]
fn decompress_brotli(data: &[u8]) -> Vec<u8> {
    let mut output = Vec::new();
    let mut reader = brotli::Decompressor::new(data, 4096);
    std::io::Read::read_to_end(&mut reader, &mut output).expect("brotli decompression failed");
    output
}

/// Compress data using the specified algorithm.
pub(super) fn compress(data: &[u8], algo: CompressionAlgo) -> Vec<u8> {
    match algo {
        CompressionAlgo::Zlib => miniz_oxide::deflate::compress_to_vec_zlib(data, 6),
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_zlib_compression_roundtrip() {
        let original = b"hello world, this is a test of zlib compression roundtrip!";
        let compressed = compress(original, CompressionAlgo::Zlib);
        let decompressed = decompress_zlib(&compressed);
        assert_eq!(decompressed, original, "zlib roundtrip should produce identical data");
    }

    #[test]
    fn test_brotli_compression_roundtrip() {
        let original = b"hello world, this is a test of brotli compression roundtrip!";
        let compressed = compress(original, CompressionAlgo::Brotli);

        // Brotli output must start with the 0x01 version prefix
        assert_eq!(compressed[0], 0x01, "brotli compressed data must have 0x01 prefix");

        // Decompress without the prefix byte
        let decompressed = decompress_brotli(&compressed[1..]);
        assert_eq!(decompressed, original, "brotli roundtrip should produce identical data");
    }
}
