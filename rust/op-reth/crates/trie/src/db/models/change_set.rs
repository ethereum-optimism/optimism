use crate::db::{HashedStorageKey, StorageTrieKey};
use alloy_primitives::B256;
use reth_db::{
    DatabaseError,
    table::{self, Decode, Encode},
};
use reth_trie_common::StoredNibbles;
use serde::{Deserialize, Serialize};

/// The keys of the entries in the history tables.
#[derive(Debug, Default, Clone, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
pub struct ChangeSet {
    /// Keys changed in [`AccountTrieHistory`](super::AccountTrieHistory) table.
    pub account_trie_keys: Vec<StoredNibbles>,
    /// Keys changed in [`StorageTrieHistory`](super::StorageTrieHistory) table.
    pub storage_trie_keys: Vec<StorageTrieKey>,
    /// Keys changed in [`HashedAccountHistory`](super::HashedAccountHistory) table.
    pub hashed_account_keys: Vec<B256>,
    /// Keys changed in [`HashedStorageHistory`](super::HashedStorageHistory) table.
    pub hashed_storage_keys: Vec<HashedStorageKey>,
}

impl table::Encode for ChangeSet {
    type Encoded = Vec<u8>;

    fn encode(self) -> Self::Encoded {
        bincode::serde::encode_to_vec(&self, bincode::config::standard())
            .expect("ChangeSet serialization should not fail")
    }
}

impl table::Decode for ChangeSet {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        bincode::serde::decode_from_slice(value, bincode::config::standard())
            .map(|(v, _)| v)
            .map_err(|_| DatabaseError::Decode)
    }
}

impl table::Compress for ChangeSet {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: bytes::BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        let encoded = self.clone().encode();
        buf.put_slice(&encoded);
    }
}

impl table::Decompress for ChangeSet {
    fn decompress(value: &[u8]) -> Result<Self, DatabaseError> {
        Self::decode(value)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use reth_db::table::{Compress, Decompress};

    #[test]
    fn test_encode_decode_empty_change_set() {
        let change_set = ChangeSet {
            account_trie_keys: vec![],
            storage_trie_keys: vec![],
            hashed_account_keys: vec![],
            hashed_storage_keys: vec![],
        };

        let encoded = change_set.clone().encode();
        let decoded = ChangeSet::decode(&encoded).expect("Failed to decode");
        assert_eq!(change_set, decoded);
    }

    #[test]
    fn test_encode_decode_populated_change_set() {
        let account_key = StoredNibbles::from(vec![1, 2, 3, 4]);
        let storage_key = StorageTrieKey {
            hashed_address: B256::repeat_byte(0x11),
            path: StoredNibbles::from(vec![5, 6, 7, 8]),
        };
        let hashed_storage_key = HashedStorageKey {
            hashed_address: B256::repeat_byte(0x22),
            hashed_storage_key: B256::repeat_byte(0x33),
        };

        let change_set = ChangeSet {
            account_trie_keys: vec![account_key],
            storage_trie_keys: vec![storage_key],
            hashed_account_keys: vec![B256::repeat_byte(0x44)],
            hashed_storage_keys: vec![hashed_storage_key],
        };

        let encoded = change_set.clone().encode();
        let decoded = ChangeSet::decode(&encoded).expect("Failed to decode");
        assert_eq!(change_set, decoded);
    }

    #[test]
    fn test_decode_invalid_data() {
        let invalid_data = vec![0xFF; 32];
        let result = ChangeSet::decode(&invalid_data);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), DatabaseError::Decode));
    }

    #[test]
    fn test_compress_decompress() {
        let change_set = ChangeSet {
            account_trie_keys: vec![StoredNibbles::from(vec![1, 2, 3])],
            storage_trie_keys: vec![StorageTrieKey {
                hashed_address: B256::ZERO,
                path: StoredNibbles::from(vec![4, 5, 6]),
            }],
            hashed_account_keys: vec![B256::ZERO],
            hashed_storage_keys: vec![HashedStorageKey {
                hashed_address: B256::ZERO,
                hashed_storage_key: B256::repeat_byte(0x42),
            }],
        };

        let mut buf = Vec::new();
        change_set.compress_to_buf(&mut buf);

        let decompressed = ChangeSet::decompress(&buf).expect("Failed to decompress");
        assert_eq!(change_set, decompressed);
    }
}
