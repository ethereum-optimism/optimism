//! Models for the backfill trie-state snapshot (see
//! [`crate::backfill`] for the design rationale).

use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use bytes::BufMut;
use reth_codecs::DecompressError;
use reth_db::{
    DatabaseError,
    table::{Compress, Decode, Decompress, Encode},
};
use serde::{Deserialize, Serialize};

/// Single-row key for the snapshot metadata table.
///
/// There is only ever one snapshot per proofs store, so the table has a
/// fixed singleton key.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
#[repr(u8)]
pub enum SnapshotMetaKey {
    /// The singleton key — there is only ever one snapshot meta row.
    Singleton = 0,
}

impl Encode for SnapshotMetaKey {
    type Encoded = [u8; 1];

    fn encode(self) -> Self::Encoded {
        [self as u8]
    }
}

impl Decode for SnapshotMetaKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        match value.first() {
            Some(&0) => Ok(Self::Singleton),
            _ => Err(DatabaseError::Decode),
        }
    }
}

/// Lifecycle status of the trie snapshot.
///
/// A snapshot's invariant is that it reflects trie state at
/// [`SnapshotMeta::anchor`]. Either it's being built and not yet
/// trustworthy ([`Self::Building`]), or it's done and reflects that block
/// exactly ([`Self::Ready`]). If a snapshot ever falls out of sync it is
/// dropped via
/// [`OpProofsBackfillProvider::clear_snapshot`](crate::api::OpProofsBackfillProvider::clear_snapshot),
/// not left around in a third state.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[repr(u8)]
pub enum SnapshotStatus {
    /// Snapshot is being constructed by [`crate::snapshot::SnapshotInitJob`].
    /// Reads must be refused until status transitions to [`Self::Ready`].
    Building = 0,
    /// Snapshot is consistent and reflects trie state at [`SnapshotMeta::anchor`].
    Ready = 1,
}

/// Metadata for the trie-state snapshot.
///
/// Encoding: `[status: 1B] ‖ [block_number: 8B BE] ‖ [block_hash: 32B]` (= 41 B).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct SnapshotMeta {
    /// The block (number + hash) the snapshot's trie state corresponds to.
    pub anchor: BlockNumHash,
    /// Current lifecycle state.
    pub status: SnapshotStatus,
}

impl SnapshotMeta {
    /// Encoded byte length.
    pub const ENCODED_LEN: usize = 1 + 8 + 32;

    /// Convenience constructor.
    pub const fn new(anchor: BlockNumHash, status: SnapshotStatus) -> Self {
        Self { anchor, status }
    }
}

impl Compress for SnapshotMeta {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        buf.put_u8(self.status as u8);
        buf.put_u64(self.anchor.number);
        buf.put_slice(self.anchor.hash.as_slice());
    }
}

impl Decompress for SnapshotMeta {
    fn decompress(value: &[u8]) -> Result<Self, DecompressError> {
        if value.len() != Self::ENCODED_LEN {
            return Err(DecompressError::new(DatabaseError::Decode));
        }
        let status = match value[0] {
            0 => SnapshotStatus::Building,
            1 => SnapshotStatus::Ready,
            _ => return Err(DecompressError::new(DatabaseError::Decode)),
        };
        let number = u64::from_be_bytes(
            value[1..9].try_into().map_err(|_| DecompressError::new(DatabaseError::Decode))?,
        );
        let hash = B256::from_slice(&value[9..41]);
        Ok(Self { anchor: BlockNumHash::new(number, hash), status })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn snapshot_meta_key_roundtrip() {
        let encoded = SnapshotMetaKey::Singleton.encode();
        let decoded = SnapshotMetaKey::decode(&encoded).unwrap();
        assert_eq!(decoded, SnapshotMetaKey::Singleton);
    }

    #[test]
    fn snapshot_meta_roundtrip_all_statuses() {
        let earliest = BlockNumHash::new(12_345_678, B256::repeat_byte(0xab));
        for status in [SnapshotStatus::Building, SnapshotStatus::Ready] {
            let original = SnapshotMeta::new(earliest, status);
            let compressed = original.compress();
            assert_eq!(compressed.len(), SnapshotMeta::ENCODED_LEN);
            let decompressed = SnapshotMeta::decompress(&compressed).unwrap();
            assert_eq!(original, decompressed);
        }
    }

    #[test]
    fn snapshot_meta_decompress_rejects_wrong_length() {
        assert!(SnapshotMeta::decompress(&[0u8; 10]).is_err());
        assert!(SnapshotMeta::decompress(&[0u8; 41 + 1]).is_err());
    }

    #[test]
    fn snapshot_meta_decompress_rejects_invalid_status() {
        let mut buf = vec![0xff_u8; SnapshotMeta::ENCODED_LEN];
        // status byte at position 0; 0xff is not a valid SnapshotStatus.
        buf[0] = 0xff;
        assert!(SnapshotMeta::decompress(&buf).is_err());
    }
}
