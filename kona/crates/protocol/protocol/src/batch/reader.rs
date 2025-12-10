//! Contains the [`BatchReader`] which is used to iteratively consume batches from raw data.

use crate::{Batch, BrotliDecompressionError, decompress_brotli};
use alloc::vec::Vec;
use alloy_primitives::Bytes;
use alloy_rlp::Decodable;
use kona_genesis::RollupConfig;
use miniz_oxide::inflate::decompress_to_vec_zlib;

/// Error type for decompression failures.
#[derive(Debug, thiserror::Error)]
pub enum DecompressionError {
    /// The data to decompress was empty.
    #[error("the data to decompress was empty")]
    EmptyData,
    /// The compression type is not supported.
    #[error("the compression type {0} is not supported")]
    UnsupportedType(u8),
    /// A brotli decompression error.
    #[error("brotli decompression error: {0}")]
    BrotliError(#[from] BrotliDecompressionError),
    /// A zlib decompression error.
    #[error("zlib decompression error")]
    ZlibError,
    /// The RLP data is too large for the configured maximum.
    #[error("the RLP data is too large: {0} bytes, maximum allowed: {1} bytes")]
    RlpTooLarge(usize, usize),
}

/// Batch Reader provides a function that iteratively consumes batches from the reader.
/// The L1Inclusion block is also provided at creation time.
/// Warning: the batch reader can read every batch-type.
/// The caller of the batch-reader should filter the results.
#[derive(Debug)]
pub struct BatchReader {
    /// The raw data to decode.
    pub data: Option<Vec<u8>>,
    /// Decompressed data.
    pub decompressed: Vec<u8>,
    /// The current cursor in the `decompressed` data.
    pub cursor: usize,
    /// The maximum RLP bytes per channel.
    pub max_rlp_bytes_per_channel: usize,
    /// Whether brotli decompression was used.
    pub brotli_used: bool,
}

impl BatchReader {
    /// ZLIB Deflate Compression Method.
    pub const ZLIB_DEFLATE_COMPRESSION_METHOD: u8 = 8;

    /// ZLIB Reserved Compression Info.
    pub const ZLIB_RESERVED_COMPRESSION_METHOD: u8 = 15;

    /// Brotli Compression Channel Version.
    pub const CHANNEL_VERSION_BROTLI: u8 = 1;

    /// Creates a new [`BatchReader`] from the given data and max decompressed RLP bytes per
    /// channel.
    pub fn new<T>(data: T, max_rlp_bytes_per_channel: usize) -> Self
    where
        T: Into<Vec<u8>>,
    {
        Self {
            data: Some(data.into()),
            decompressed: Vec::new(),
            cursor: 0,
            max_rlp_bytes_per_channel,
            brotli_used: false,
        }
    }

    /// Helper method to decompress the data contained in the reader.
    pub fn decompress(&mut self) -> Result<(), DecompressionError> {
        if let Some(data) = self.data.take() {
            // Peek at the data to determine the compression type.
            if data.is_empty() {
                return Err(DecompressionError::EmptyData);
            }

            let compression_type = data[0];
            if (compression_type & 0x0F) == Self::ZLIB_DEFLATE_COMPRESSION_METHOD ||
                (compression_type & 0x0F) == Self::ZLIB_RESERVED_COMPRESSION_METHOD
            {
                self.decompressed =
                    decompress_to_vec_zlib(&data).map_err(|_| DecompressionError::ZlibError)?;

                // Check the size of the decompressed channel RLP.
                if self.decompressed.len() > self.max_rlp_bytes_per_channel {
                    return Err(DecompressionError::RlpTooLarge(
                        self.decompressed.len(),
                        self.max_rlp_bytes_per_channel,
                    ));
                }
            } else if compression_type == Self::CHANNEL_VERSION_BROTLI {
                self.brotli_used = true;
                self.decompressed = decompress_brotli(&data[1..], self.max_rlp_bytes_per_channel)?;
            } else {
                return Err(DecompressionError::UnsupportedType(compression_type));
            }
        }
        Ok(())
    }

    /// Pulls out the next batch from the reader.
    pub fn next_batch(&mut self, cfg: &RollupConfig) -> Option<Batch> {
        // Ensure the data is decompressed.
        self.decompress().ok()?;

        // Decompress and RLP decode the batch data, before finally decoding the batch itself.
        let decompressed_reader = &mut self.decompressed.as_slice()[self.cursor..].as_ref();
        let bytes = Bytes::decode(decompressed_reader).ok()?;
        let Ok(batch) = Batch::decode(&mut bytes.as_ref(), cfg) else {
            return None;
        };

        // Confirm that brotli decompression was performed *after* the Fjord hardfork.
        if self.brotli_used && !cfg.is_fjord_active(batch.timestamp()) {
            return None;
        }

        // Advance the cursor on the reader.
        self.cursor = self.decompressed.len() - decompressed_reader.len();
        Some(batch)
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use kona_genesis::{
        HardForkConfig, MAX_RLP_BYTES_PER_CHANNEL_BEDROCK, MAX_RLP_BYTES_PER_CHANNEL_FJORD,
    };

    fn new_compressed_batch_data() -> Bytes {
        let file_contents =
            alloc::string::String::from_utf8_lossy(include_bytes!("../../testdata/batch.hex"));
        let file_contents = &(&*file_contents)[..file_contents.len() - 1];
        let data = alloy_primitives::hex::decode(file_contents).unwrap();
        data.into()
    }

    #[test]
    fn test_batch_reader() {
        let raw = new_compressed_batch_data();
        let decompressed_len = decompress_to_vec_zlib(&raw).unwrap().len();
        let mut reader = BatchReader::new(raw, MAX_RLP_BYTES_PER_CHANNEL_BEDROCK as usize);
        reader.next_batch(&RollupConfig::default()).unwrap();
        assert_eq!(reader.cursor, decompressed_len);
    }

    #[test]
    fn test_batch_reader_fjord() {
        let raw = new_compressed_batch_data();
        let decompressed_len = decompress_to_vec_zlib(&raw).unwrap().len();
        let mut reader = BatchReader::new(raw, MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize);
        reader
            .next_batch(&RollupConfig {
                hardforks: HardForkConfig { fjord_time: Some(0), ..Default::default() },
                ..Default::default()
            })
            .unwrap();
        assert_eq!(reader.cursor, decompressed_len);
    }
}
