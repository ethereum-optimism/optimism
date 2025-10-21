use alloy_primitives::{B256, U256};
use derive_more::{Constructor, From, Into};
use reth_db::{
    table::{Compress, Decode, Decompress, Encode},
    DatabaseError,
};
use reth_trie::StoredNibbles;
use serde::{Deserialize, Serialize};

/// Composite key: `(hashed-address, path)` for storage trie branches
///
/// Used to efficiently index storage branches by both account address and trie path.
/// The encoding ensures lexicographic ordering: first by address, then by path.
#[derive(Default, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
pub struct StorageTrieKey {
    /// Hashed account address
    pub hashed_address: B256,
    /// Trie path as nibbles
    pub path: StoredNibbles,
}

impl StorageTrieKey {
    /// Create a new storage branch key
    pub const fn new(hashed_address: B256, path: StoredNibbles) -> Self {
        Self { hashed_address, path }
    }
}

impl Encode for StorageTrieKey {
    type Encoded = Vec<u8>;

    fn encode(self) -> Self::Encoded {
        let mut buf = Vec::with_capacity(32 + self.path.0.len());
        // First encode the address (32 bytes)
        buf.extend_from_slice(self.hashed_address.as_slice());
        // Then encode the path
        buf.extend_from_slice(&self.path.encode());
        buf
    }
}

impl Decode for StorageTrieKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.len() < 32 {
            return Err(DatabaseError::Decode);
        }

        // First 32 bytes are the address
        let hashed_address = B256::from_slice(&value[..32]);

        // Remaining bytes are the path
        let path = StoredNibbles::decode(&value[32..])?;

        Ok(Self { hashed_address, path })
    }
}

/// Composite key: (`hashed_address`, `hashed_storage_key`) for hashed storage values
///
/// Used to efficiently index storage values by both account address and storage key.
/// The encoding ensures lexicographic ordering: first by address, then by storage key.
#[derive(Debug, Default, Clone, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
pub struct HashedStorageKey {
    /// Hashed account address
    pub hashed_address: B256,
    /// Hashed storage key
    pub hashed_storage_key: B256,
}

impl HashedStorageKey {
    /// Create a new hashed storage key
    pub const fn new(hashed_address: B256, hashed_storage_key: B256) -> Self {
        Self { hashed_address, hashed_storage_key }
    }
}

impl Encode for HashedStorageKey {
    type Encoded = [u8; 64];

    fn encode(self) -> Self::Encoded {
        let mut buf = [0u8; 64];
        // First 32 bytes: address
        buf[..32].copy_from_slice(self.hashed_address.as_slice());
        // Next 32 bytes: storage key
        buf[32..].copy_from_slice(self.hashed_storage_key.as_slice());
        buf
    }
}

impl Decode for HashedStorageKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.len() != 64 {
            return Err(DatabaseError::Decode);
        }

        let hashed_address = B256::from_slice(&value[..32]);
        let hashed_storage_key = B256::from_slice(&value[32..64]);

        Ok(Self { hashed_address, hashed_storage_key })
    }
}

/// Storage value wrapper for U256 values
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, From, Into, Constructor)]
pub struct StorageValue(pub U256);

impl Compress for StorageValue {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: bytes::BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        let be: [u8; 32] = self.0.to_be_bytes::<32>();
        buf.put_slice(&be);
    }
}

impl Decompress for StorageValue {
    fn decompress(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.len() != 32 {
            return Err(DatabaseError::Decode);
        }
        let bytes: [u8; 32] = value.try_into().map_err(|_| DatabaseError::Decode)?;
        Ok(Self(U256::from_be_bytes(bytes)))
    }
}

/// Proof Window key for tracking active proof window bounds
///
/// Used to store earliest and latest block numbers in the external storage.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
#[repr(u8)]
pub enum ProofWindowKey {
    /// Earliest block number stored in external storage
    EarliestBlock = 0,
    /// Latest block number stored in external storage
    LatestBlock = 1,
}

impl Encode for ProofWindowKey {
    type Encoded = [u8; 1];

    fn encode(self) -> Self::Encoded {
        [self as u8]
    }
}

impl Decode for ProofWindowKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        match value.first() {
            Some(&0) => Ok(Self::EarliestBlock),
            Some(&1) => Ok(Self::LatestBlock),
            _ => Err(DatabaseError::Decode),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_trie::Nibbles;

    #[test]
    fn test_storage_branch_subkey_encode_decode() {
        let addr = B256::from([1u8; 32]);
        let path = StoredNibbles(Nibbles::from_nibbles_unchecked([1, 2, 3, 4]));
        let key = StorageTrieKey::new(addr, path.clone());

        let encoded = key.clone().encode();
        let decoded = StorageTrieKey::decode(&encoded).unwrap();

        assert_eq!(key, decoded);
        assert_eq!(decoded.hashed_address, addr);
        assert_eq!(decoded.path, path);
    }

    #[test]
    fn test_storage_branch_subkey_ordering() {
        let addr1 = B256::from([1u8; 32]);
        let addr2 = B256::from([2u8; 32]);
        let path1 = StoredNibbles(Nibbles::from_nibbles_unchecked([1, 2]));
        let path2 = StoredNibbles(Nibbles::from_nibbles_unchecked([1, 3]));

        let key1 = StorageTrieKey::new(addr1, path1.clone());
        let key2 = StorageTrieKey::new(addr1, path2);
        let key3 = StorageTrieKey::new(addr2, path1);

        // Encoded bytes should be sortable: first by address, then by path
        let enc1 = key1.encode();
        let enc2 = key2.encode();
        let enc3 = key3.encode();

        assert!(enc1 < enc2, "Same address, path1 < path2");
        assert!(enc1 < enc3, "addr1 < addr2");
        assert!(enc2 < enc3, "addr1 < addr2 (even with larger path)");
    }

    #[test]
    fn test_hashed_storage_subkey_encode_decode() {
        let addr = B256::from([1u8; 32]);
        let storage_key = B256::from([2u8; 32]);
        let key = HashedStorageKey::new(addr, storage_key);

        let encoded = key.clone().encode();
        let decoded = HashedStorageKey::decode(&encoded).unwrap();

        assert_eq!(key, decoded);
        assert_eq!(decoded.hashed_address, addr);
        assert_eq!(decoded.hashed_storage_key, storage_key);
    }

    #[test]
    fn test_hashed_storage_subkey_ordering() {
        let addr1 = B256::from([1u8; 32]);
        let addr2 = B256::from([2u8; 32]);
        let storage1 = B256::from([10u8; 32]);
        let storage2 = B256::from([20u8; 32]);

        let key1 = HashedStorageKey::new(addr1, storage1);
        let key2 = HashedStorageKey::new(addr1, storage2);
        let key3 = HashedStorageKey::new(addr2, storage1);

        // Encoded bytes should be sortable: first by address, then by storage key
        let enc1 = key1.encode();
        let enc2 = key2.encode();
        let enc3 = key3.encode();

        assert!(enc1 < enc2, "Same address, storage1 < storage2");
        assert!(enc1 < enc3, "addr1 < addr2");
        assert!(enc2 < enc3, "addr1 < addr2 (even with larger storage key)");
    }

    #[test]
    fn test_hashed_storage_subkey_size() {
        let addr = B256::from([1u8; 32]);
        let storage_key = B256::from([2u8; 32]);
        let key = HashedStorageKey::new(addr, storage_key);

        let encoded = key.encode();
        assert_eq!(encoded.len(), 64, "Encoded size should be exactly 64 bytes");
    }

    #[test]
    fn test_metadata_key_encode_decode() {
        let key = ProofWindowKey::EarliestBlock;
        let encoded = key.encode();
        let decoded = ProofWindowKey::decode(&encoded).unwrap();
        assert_eq!(key, decoded);

        let key = ProofWindowKey::LatestBlock;
        let encoded = key.encode();
        let decoded = ProofWindowKey::decode(&encoded).unwrap();
        assert_eq!(key, decoded);
    }
}
