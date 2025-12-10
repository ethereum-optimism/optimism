//! Models for storing blockchain derivation in the database.
//!
//! This module defines the data structure and schema used for tracking
//! how blocks are derived from source. This is particularly relevant
//! in rollup contexts, such as linking an L2 block to its originating L1 block.

use super::{BlockRef, U64List};
use kona_interop::DerivedRefPair;
use reth_codecs::Compact;
use serde::{Deserialize, Serialize};

/// Represents a pair of blocks where one block [`derived`](`Self::derived`) is derived
/// from another [`source`](`Self::source`).
///
/// This structure is used to track the lineage of blocks where L2 blocks are derived from L1
/// blocks. It stores the [`BlockRef`] information for both the source and the derived blocks.
/// It is stored as value in the [`DerivedBlocks`](`crate::models::DerivedBlocks`) table.
#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct StoredDerivedBlockPair {
    /// The block that was derived from the [`source`](`Self::source`) block.
    pub derived: BlockRef,
    /// The source block from which the [`derived`](`Self::derived`) block was created.
    pub source: BlockRef,
}

impl Compact for StoredDerivedBlockPair {
    fn to_compact<B: bytes::BufMut + AsMut<[u8]>>(&self, buf: &mut B) -> usize {
        let mut bytes_written = 0;
        bytes_written += self.derived.to_compact(buf);
        bytes_written += self.source.to_compact(buf);
        bytes_written
    }

    fn from_compact(buf: &[u8], _len: usize) -> (Self, &[u8]) {
        let (derived, remaining_buf) = BlockRef::from_compact(buf, buf.len());
        let (source, final_remaining_buf) =
            BlockRef::from_compact(remaining_buf, remaining_buf.len());
        (Self { derived, source }, final_remaining_buf)
    }
}

/// Converts from [`StoredDerivedBlockPair`] (storage format) to [`DerivedRefPair`] (external API
/// format).
///
/// Performs a direct field mapping.
impl From<StoredDerivedBlockPair> for DerivedRefPair {
    fn from(pair: StoredDerivedBlockPair) -> Self {
        Self { derived: pair.derived.into(), source: pair.source.into() }
    }
}

/// Converts from [`DerivedRefPair`] (external API format) to [`StoredDerivedBlockPair`] (storage
/// format).
///
/// Performs a direct field mapping.
impl From<DerivedRefPair> for StoredDerivedBlockPair {
    fn from(pair: DerivedRefPair) -> Self {
        Self { derived: pair.derived.into(), source: pair.source.into() }
    }
}

impl StoredDerivedBlockPair {
    /// Creates a new [`StoredDerivedBlockPair`] from the given [`BlockRef`]s.
    ///
    /// # Arguments
    ///
    /// * `source` - The source block reference.
    /// * `derived` - The derived block reference.
    pub const fn new(source: BlockRef, derived: BlockRef) -> Self {
        Self { source, derived }
    }
}

/// Represents a traversal of source blocks and their derived blocks.
///
/// This structure is used to track the lineage of blocks where L2 blocks are derived from L1
/// blocks. It stores the [`BlockRef`] information for the source block and the list of derived
/// block numbers. It is stored as value in the [`BlockTraversal`](`crate::models::BlockTraversal`)
/// table.
#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct SourceBlockTraversal {
    /// The source block reference.
    pub source: BlockRef,
    /// The list of derived block numbers.
    pub derived_block_numbers: U64List,
}

impl Compact for SourceBlockTraversal {
    fn to_compact<B: bytes::BufMut + AsMut<[u8]>>(&self, buf: &mut B) -> usize {
        let mut bytes_written = 0;
        bytes_written += self.source.to_compact(buf);
        bytes_written += self.derived_block_numbers.to_compact(buf);
        bytes_written
    }

    fn from_compact(buf: &[u8], _len: usize) -> (Self, &[u8]) {
        let (source, remaining_buf) = BlockRef::from_compact(buf, buf.len());
        let (derived_block_numbers, final_remaining_buf) =
            U64List::from_compact(remaining_buf, remaining_buf.len());
        (Self { source, derived_block_numbers }, final_remaining_buf)
    }
}

impl SourceBlockTraversal {
    /// Creates a new [`SourceBlockTraversal`] from the given [`BlockRef`] and [`U64List`].
    ///
    /// # Arguments
    ///
    /// * `source` - The source block reference.
    /// * `derived_block_numbers` - The list of derived block numbers.
    pub const fn new(source: BlockRef, derived_block_numbers: U64List) -> Self {
        Self { source, derived_block_numbers }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::BlockRef;
    use alloy_primitives::B256;
    use kona_interop::DerivedRefPair;
    use kona_protocol::BlockInfo;
    use reth_codecs::Compact;

    fn test_b256(val: u8) -> B256 {
        let mut val_bytes = [0u8; 32];
        val_bytes[0] = val;
        let b256_from_val = B256::from(val_bytes);
        B256::random() ^ b256_from_val
    }

    #[test]
    fn test_derived_block_pair_compact_roundtrip() {
        let source_ref = BlockRef {
            number: 100,
            hash: test_b256(1),
            parent_hash: test_b256(2),
            timestamp: 1000,
        };
        let derived_ref = BlockRef {
            number: 200,
            hash: test_b256(3),
            parent_hash: test_b256(4),
            timestamp: 1010,
        };

        let original_pair = StoredDerivedBlockPair { source: source_ref, derived: derived_ref };

        let mut buffer = Vec::new();
        let bytes_written = original_pair.to_compact(&mut buffer);

        assert_eq!(bytes_written, buffer.len(), "Bytes written should match buffer length");
        let (deserialized_pair, remaining_buf) =
            StoredDerivedBlockPair::from_compact(&buffer, bytes_written);

        assert_eq!(
            original_pair, deserialized_pair,
            "Original and deserialized pairs should be equal"
        );
        assert!(remaining_buf.is_empty(), "Remaining buffer should be empty after deserialization");
    }

    #[test]
    fn test_from_stored_to_derived_ref_pair() {
        let source_ref =
            BlockRef { number: 1, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 100 };
        let derived_ref =
            BlockRef { number: 2, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 200 };

        let stored =
            StoredDerivedBlockPair { source: source_ref.clone(), derived: derived_ref.clone() };

        // Convert to DerivedRefPair
        let derived: DerivedRefPair = stored.into();

        // The conversion should map fields directly (BlockRef -> BlockInfo)
        assert_eq!(BlockInfo::from(source_ref), derived.source);
        assert_eq!(BlockInfo::from(derived_ref), derived.derived);
    }

    #[test]
    fn test_from_derived_ref_pair_to_stored() {
        let source_info =
            BlockInfo { number: 10, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 100 };
        let derived_info =
            BlockInfo { number: 20, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 200 };

        let pair = DerivedRefPair { source: source_info, derived: derived_info };

        // Convert to StoredDerivedBlockPair
        let stored: StoredDerivedBlockPair = pair.into();

        // The conversion should map fields directly (BlockInfo -> BlockRef)
        assert_eq!(BlockRef::from(source_info), stored.source);
        assert_eq!(BlockRef::from(derived_info), stored.derived);
    }

    #[test]
    fn test_source_block_traversal_compact_roundtrip() {
        let source_ref = BlockRef {
            number: 123,
            hash: test_b256(10),
            parent_hash: test_b256(11),
            timestamp: 1111,
        };
        let derived_block_numbers = U64List(vec![1, 2, 3, 4, 5]);
        let original = SourceBlockTraversal { source: source_ref, derived_block_numbers };

        let mut buffer = Vec::new();
        let bytes_written = original.to_compact(&mut buffer);
        assert_eq!(bytes_written, buffer.len(), "Bytes written should match buffer length");
        let (deserialized, remaining_buf) =
            SourceBlockTraversal::from_compact(&buffer, bytes_written);
        assert_eq!(
            original, deserialized,
            "Original and deserialized SourceBlockTraversal should be equal"
        );
        assert!(remaining_buf.is_empty(), "Remaining buffer should be empty after deserialization");
    }
}
