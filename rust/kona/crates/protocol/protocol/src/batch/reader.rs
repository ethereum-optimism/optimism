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

    // -------------------------------------------------------------------------
    // Spec conformance test — CR-02
    // Review: kona-node-vs-op-node
    // Component: channel-reader
    // Reference: op-node/rollup/derive/channel.go:195 — pre-Fjord brotli rejected at
    //            BatchReader creation using the L1 origin block timestamp (isFjord flag)
    // Subject:   protocol/src/batch/reader.rs:118 — checks cfg.is_fjord_active(batch.timestamp())
    //            where batch.timestamp() is the L2 batch timestamp
    //
    // Go rejects brotli before Fjord using the L1 origin block timestamp passed into
    // BatchReader as the `isFjord` flag. The check fires BEFORE any batch bytes are read.
    //
    // Rust's BatchReader::next_batch checks `self.brotli_used && !cfg.is_fjord_active(batch.timestamp())`
    // AFTER full batch decoding. The timestamp used is the L2 batch's own timestamp, not the
    // L1 origin block timestamp.
    //
    // Near the Fjord activation boundary: a brotli-compressed channel submitted in a post-Fjord
    // L1 block (L1 origin timestamp >= fjord_time) may contain L2 batches whose timestamps are
    // pre-Fjord (L2 batch timestamp < fjord_time). Go accepts (L1 origin is post-Fjord); Rust
    // rejects (L2 batch timestamp is pre-Fjord). This is a consensus divergence.
    //
    // This test documents the CURRENT Rust behavior: brotli batch with pre-Fjord L2 timestamp
    // is rejected even when the L1 origin timestamp is post-Fjord.
    // -------------------------------------------------------------------------
    #[test]
    fn test_spec_channel_reader_brotli_pre_fjord_timestamp_check() {
        // CR-02: BatchReader rejects brotli batch because batch.timestamp() < fjord_time,
        // ignoring that the L1 origin (where the channel was submitted) is post-Fjord.
        //
        // Construct a minimal brotli-simulated reader:
        // - brotli_used = true (simulate a brotli-decompressed channel)
        // - decompressed data = RLP(Bytes([0x00, <SingleBatch with timestamp=50>]))
        // - fjord_time = 100 → is_fjord_active(50) = false → batch rejected
        //
        // If the check used L1 origin timestamp (>= fjord_time=100), the batch would be accepted.

        use alloy_rlp::Encodable;
        use crate::{SingleBatch, SINGLE_BATCH_TYPE};

        // Build a minimal SingleBatch with timestamp=50 (pre-Fjord with fjord_time=100)
        let batch = SingleBatch { timestamp: 50, ..Default::default() };
        let mut batch_payload: alloc::vec::Vec<u8> = alloc::vec![SINGLE_BATCH_TYPE];
        batch.encode(&mut batch_payload);

        // RLP-encode as `Bytes` (what Bytes::decode in next_batch expects)
        let mut decompressed: alloc::vec::Vec<u8> = alloc::vec![];
        alloy_primitives::Bytes::from(batch_payload).encode(&mut decompressed);

        // Create a pre-initialized BatchReader with brotli_used=true and pre-filled decompressed
        let mut reader = BatchReader::new(alloc::vec![], MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize);
        reader.data = None; // prevent decompress() from running
        reader.decompressed = decompressed;
        reader.brotli_used = true; // simulate brotli-decompressed channel

        // Config: fjord_time=100. The L1 origin is post-Fjord (timestamp >= 100),
        // but the batch L2 timestamp is 50 (pre-Fjord).
        let cfg = RollupConfig {
            hardforks: HardForkConfig { fjord_time: Some(100), ..Default::default() },
            ..Default::default()
        };

        // Rust rejects: brotli_used && !is_fjord_active(batch.timestamp()=50) = true → None
        let result = reader.next_batch(&cfg);
        assert!(
            result.is_none(),
            "CR-02: Rust correctly rejects brotli batch with pre-Fjord L2 timestamp \
             (timestamp=50, fjord_time=100), but this diverges from Go which uses L1 origin \
             timestamp for the same check"
        );
    }

    // -------------------------------------------------------------------------
    // Spec conformance test — CR-02 (boundary case: L1 origin post-Fjord, batch pre-Fjord)
    // Demonstrates the specific divergence scenario between Go and Rust.
    // -------------------------------------------------------------------------
    #[test]
    fn test_spec_channel_reader_brotli_fjord_gate_uses_l1_origin_timestamp() {
        // CR-02 boundary: batch.timestamp() = 50 < fjord_time = 100, but if Go used L1 origin
        // timestamp = 150 (post-Fjord), it would accept. Rust uses batch.timestamp() and rejects.
        //
        // Both Go and Rust agree when batch.timestamp() >= fjord_time. They diverge only in the
        // window [last pre-Fjord L1 block timestamp .. first post-Fjord L1 block timestamp) where
        // an honest batcher switches to brotli but early batches still have pre-Fjord L2 timestamps.

        use alloy_rlp::Encodable;
        use crate::{SingleBatch, SINGLE_BATCH_TYPE};

        // Case A: batch.timestamp() >= fjord_time → both accept (Rust accepts)
        let batch_post_fjord = SingleBatch { timestamp: 100, ..Default::default() };
        let mut payload_a: alloc::vec::Vec<u8> = alloc::vec![SINGLE_BATCH_TYPE];
        batch_post_fjord.encode(&mut payload_a);
        let mut decompressed_a: alloc::vec::Vec<u8> = alloc::vec![];
        alloy_primitives::Bytes::from(payload_a).encode(&mut decompressed_a);

        let mut reader_a =
            BatchReader::new(alloc::vec![], MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize);
        reader_a.data = None;
        reader_a.decompressed = decompressed_a;
        reader_a.brotli_used = true;

        let cfg_fjord = RollupConfig {
            hardforks: HardForkConfig { fjord_time: Some(100), ..Default::default() },
            ..Default::default()
        };
        assert!(
            reader_a.next_batch(&cfg_fjord).is_some(),
            "CR-02: brotli batch with post-Fjord L2 timestamp should be accepted"
        );

        // Case B: batch.timestamp() < fjord_time → Rust rejects (Go would accept if L1 origin
        // is post-Fjord). This is the divergence case.
        let batch_pre_fjord = SingleBatch { timestamp: 99, ..Default::default() };
        let mut payload_b: alloc::vec::Vec<u8> = alloc::vec![SINGLE_BATCH_TYPE];
        batch_pre_fjord.encode(&mut payload_b);
        let mut decompressed_b: alloc::vec::Vec<u8> = alloc::vec![];
        alloy_primitives::Bytes::from(payload_b).encode(&mut decompressed_b);

        let mut reader_b =
            BatchReader::new(alloc::vec![], MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize);
        reader_b.data = None;
        reader_b.decompressed = decompressed_b;
        reader_b.brotli_used = true;

        assert!(
            reader_b.next_batch(&cfg_fjord).is_none(),
            "CR-02: Rust rejects brotli batch with pre-Fjord L2 timestamp (timestamp=99, fjord_time=100) — \
             Go would accept if submitted in a post-Fjord L1 block"
        );
    }
}
