//! Models for storing block metadata in the database.
//!
//! This module defines the data structure and schema used for tracking
//! individual blocks by block number. The stored metadata includes block hash,
//! parent hash, and block timestamp.
//!
//! Unlike logs, each block is uniquely identified by its number and does not
//! require dup-sorting.

use alloy_primitives::B256;
use derive_more::Display;
use kona_protocol::BlockInfo;
use reth_codecs::Compact;
use serde::{Deserialize, Serialize};

/// Metadata reference for a single block.
///
/// This struct captures essential block information required to track canonical
/// block lineage and verify ancestry. It is stored as the value
/// in the [`crate::models::BlockRefs`] table.
#[derive(Debug, Clone, Display, PartialEq, Eq, Default, Serialize, Deserialize, Compact)]
#[display("number: {number}, hash: {hash}, parent_hash: {parent_hash}, timestamp: {timestamp}")]
pub struct BlockRef {
    /// The height of the block.
    pub number: u64,
    /// The hash of the block itself.
    pub hash: B256,
    /// The hash of the parent block (previous block in the chain).
    pub parent_hash: B256,
    /// The timestamp of the block (seconds since Unix epoch).
    pub timestamp: u64,
}

/// Converts from [`BlockInfo`] (external API format) to [`BlockRef`] (storage
/// format).
///
/// Performs a direct field mapping.
impl From<BlockInfo> for BlockRef {
    fn from(block: BlockInfo) -> Self {
        Self {
            number: block.number,
            hash: block.hash,
            parent_hash: block.parent_hash,
            timestamp: block.timestamp,
        }
    }
}

/// Converts from [`BlockRef`] (storage format) to [`BlockInfo`] (external API
/// format).
///
/// This enables decoding values stored in a compact format for use in application logic.
impl From<BlockRef> for BlockInfo {
    fn from(block: BlockRef) -> Self {
        Self {
            number: block.number,
            hash: block.hash,
            parent_hash: block.parent_hash,
            timestamp: block.timestamp,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;

    fn test_b256(val: u8) -> B256 {
        let mut val_bytes = [0u8; 32];
        val_bytes[0] = val;
        let b256_from_val = B256::from(val_bytes);
        B256::random() ^ b256_from_val
    }

    #[test]
    fn test_block_ref_compact_roundtrip() {
        let original_ref = BlockRef {
            number: 42,
            hash: test_b256(10),
            parent_hash: test_b256(11),
            timestamp: 1678886400,
        };

        let mut buffer = Vec::new();
        let bytes_written = original_ref.to_compact(&mut buffer);
        assert_eq!(bytes_written, buffer.len(), "Bytes written should match buffer length");

        let (deserialized_ref, remaining_buf) = BlockRef::from_compact(&buffer, bytes_written);
        assert_eq!(original_ref, deserialized_ref, "Original and deserialized ref should be equal");
        assert!(remaining_buf.is_empty(), "Remaining buffer should be empty after deserialization");
    }

    #[test]
    fn test_from_block_info_to_block_ref() {
        let block_info = BlockInfo {
            number: 123,
            hash: test_b256(1),
            parent_hash: test_b256(2),
            timestamp: 1600000000,
        };

        let block_ref: BlockRef = block_info.into();

        assert_eq!(block_ref.number, block_info.number, "Number should match");
        assert_eq!(block_ref.hash, block_info.hash, "Hash should match");
        assert_eq!(block_ref.parent_hash, block_info.parent_hash, "Parent hash should match");
        assert_eq!(block_ref.timestamp, block_info.timestamp, "Time (timestamp) should match");
    }

    #[test]
    fn test_from_block_ref_to_block_info() {
        let block_ref = BlockRef {
            number: 456,
            hash: test_b256(3),
            parent_hash: test_b256(4),
            timestamp: 1700000000,
        };

        let block_info: BlockInfo = block_ref.clone().into();

        assert_eq!(block_info.number, block_ref.number, "Number should match");
        assert_eq!(block_info.hash, block_ref.hash, "Hash should match");
        assert_eq!(block_info.parent_hash, block_ref.parent_hash, "Parent hash should match");
        assert_eq!(block_info.timestamp, block_ref.timestamp, "Timestamp (time) should match");
    }
}
