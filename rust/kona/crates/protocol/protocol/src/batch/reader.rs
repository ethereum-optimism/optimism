//! Contains the [`BatchReader`] which is used to iteratively consume batches from raw data.

use crate::{Batch, BrotliDecompressionError, decompress_brotli};
use alloc::vec::Vec;
use alloy_primitives::Bytes;
use alloy_rlp::Decodable;
use kona_genesis::RollupConfig;
use miniz_oxide::inflate::{TINFLStatus, decompress_to_vec_zlib_with_limit};

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
}

/// Batch Reader provides a function that iteratively consumes batches from the reader.
/// The `L1Inclusion` block is also provided at creation time.
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
    /// No-op if the data has already been decompressed.
    pub fn decompress(&mut self) -> Result<(), DecompressionError> {
        if !self.decompressed.is_empty() {
            return Ok(());
        }
        match self.data.take() {
            None => Err(DecompressionError::EmptyData),
            Some(data) if data.is_empty() => Err(DecompressionError::EmptyData),
            Some(data) => {
                // Peek at the data to determine the compression type.
                let compression_type = data[0];
                if (compression_type & 0x0F) == Self::ZLIB_DEFLATE_COMPRESSION_METHOD ||
                    (compression_type & 0x0F) == Self::ZLIB_RESERVED_COMPRESSION_METHOD
                {
                    self.decompress_zlib(data)
                } else if compression_type == Self::CHANNEL_VERSION_BROTLI {
                    self.decompress_brotli(data)
                } else {
                    Err(DecompressionError::UnsupportedType(compression_type))
                }
            }
        }
    }

    fn decompress_zlib(&mut self, data: Vec<u8>) -> Result<(), DecompressionError> {
        // Decompress with a limit to prevent zip-bomb attacks.
        // Per spec, if decompressed data exceeds the limit, the output is
        // truncated to max_rlp_bytes_per_channel bytes (not rejected).
        match decompress_to_vec_zlib_with_limit(&data, self.max_rlp_bytes_per_channel) {
            Ok(decompressed) => {
                self.decompressed = decompressed;
            }
            Err(e) if (e.status == TINFLStatus::HasMoreOutput || !e.output.is_empty()) => {
                // Either: limit reached — truncate per spec and keep partial output.
                // Or: decompression error with partial output — keep it so
                // batches decoded before the error point are accepted.
                self.decompressed = e.output;
            }
            Err(_) => {
                return Err(DecompressionError::ZlibError);
            }
        }
        Ok(())
    }

    fn decompress_brotli(&mut self, data: Vec<u8>) -> Result<(), DecompressionError> {
        self.brotli_used = true;
        // Note: the first byte of the channel data is the Brotli channel version but not part of
        // the compressed data, so it's skipped here but not for zlib.
        self.decompressed = decompress_brotli(&data[1..], self.max_rlp_bytes_per_channel)?;
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
    use miniz_oxide::{
        deflate::{CompressionLevel, compress_to_vec_zlib},
        inflate::decompress_to_vec_zlib,
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

    /// Builds zlib-compressed channel data containing `n` copies of the same
    /// batch by duplicating the decompressed RLP content from the test fixture.
    fn new_multi_batch_compressed_data(n: usize) -> (Bytes, usize) {
        let raw = new_compressed_batch_data();
        let single = decompress_to_vec_zlib(&raw).unwrap();

        let mut multi = Vec::with_capacity(single.len() * n);
        for _ in 0..n {
            multi.extend_from_slice(&single);
        }
        let decompressed_len = multi.len();
        (compress_to_vec_zlib(&multi, CompressionLevel::BestSpeed.into()).into(), decompressed_len)
    }

    #[test]
    fn test_zlib_truncation_instead_of_rejection() {
        let raw = new_compressed_batch_data();
        let decompressed_len = decompress_to_vec_zlib(&raw).unwrap().len();
        assert!(decompressed_len > 1, "test data must decompress to >1 byte");

        // Set limit below decompressed size — should truncate, not error.
        let limit = decompressed_len / 2;
        let mut reader = BatchReader::new(raw, limit);
        assert!(reader.decompress().is_ok());
        assert_eq!(reader.decompressed.len(), limit);
    }

    #[test]
    fn test_zlib_truncation_yields_decodable_batches() {
        let n = 3;
        let (compressed, full_len) = new_multi_batch_compressed_data(n);
        let single_batch_len = full_len / n;

        // Full decompression should yield all n batches.
        let mut reader = BatchReader::new(compressed.clone(), full_len);
        let mut count = 0;
        while reader.next_batch(&RollupConfig::default()).is_some() {
            count += 1;
        }
        assert_eq!(count, n, "should decode {n} batches from full channel");

        // Truncate to just under the last batch — should yield n-1 batches.
        let limit = full_len - 1;
        let mut reader = BatchReader::new(compressed, limit);
        let mut count = 0;
        while reader.next_batch(&RollupConfig::default()).is_some() {
            count += 1;
        }
        assert_eq!(
            count,
            n - 1,
            "truncated channel should yield {exp} batches (cursor at {cursor}, \
             single batch is {single_batch_len} bytes, limit {limit})",
            exp = n - 1,
            cursor = reader.cursor,
        );
        // First n-1 batches should have been fully consumed.
        assert_eq!(reader.cursor, single_batch_len * (n - 1));
    }
}
