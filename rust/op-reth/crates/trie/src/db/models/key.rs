use alloy_primitives::{B256, BlockNumber};
use reth_db::{
    DatabaseError,
    models::sharded_key::ShardedKey,
    table::{Decode, Encode},
};
use reth_trie_common::{Nibbles, StoredNibbles};
use serde::{Deserialize, Serialize};

/// Nibble-subkey layout shared by [`AccountTrieShardedKey`] and [`StorageTrieShardedKey`]: 64
/// nibble bytes right-padded with `0x00`, followed by a 1-byte length suffix. Padding the path
/// to a fixed width and placing nibble bytes ahead of the length byte makes MDBX's byte-wise
/// sort agree with `Nibbles`' lex-by-nibble order.
const NIBBLE_SUBKEY_LEN: usize = 65;
/// Byte length of an encoded [`AccountTrieShardedKey`]: nibble subkey + 8 block number bytes.
const ACCOUNT_TRIE_SHARDED_KEY_LEN: usize = NIBBLE_SUBKEY_LEN + 8;
/// Byte length of an encoded [`StorageTrieShardedKey`]: hashed address + nibble subkey + block.
const STORAGE_TRIE_SHARDED_KEY_LEN: usize = 32 + NIBBLE_SUBKEY_LEN + 8;

/// Encode a nibble path into the fixed-size `[u8; 65]` subkey layout: 64 path bytes right-padded
/// with `0x00`, followed by the actual nibble count in byte 64.
fn encode_nibble_subkey(nibbles: &StoredNibbles) -> [u8; NIBBLE_SUBKEY_LEN] {
    debug_assert!(nibbles.0.len() <= 64, "nibble path exceeds 64");
    let mut buf = [0u8; NIBBLE_SUBKEY_LEN];
    for (i, nibble) in nibbles.0.iter().enumerate() {
        buf[i] = nibble;
    }
    buf[64] = nibbles.0.len() as u8;
    buf
}

/// Inverse of [`encode_nibble_subkey`]: read the length byte at position 64 and reconstruct the
/// [`StoredNibbles`] from the first `len` path bytes.
fn decode_nibble_subkey(buf: &[u8; NIBBLE_SUBKEY_LEN]) -> StoredNibbles {
    let len = buf[64] as usize;
    StoredNibbles::from(Nibbles::from_nibbles_unchecked(&buf[..len]))
}

/// Sharded key for hashed accounts history, keyed by `(hashed_address, block)`.
///
/// Encoded as a fixed-size 40-byte buffer:
///
/// ```text
/// [hashed_address: 32 bytes] ++ [block_number: 8 BE bytes]
/// ```
///
/// MDBX's byte-wise sort groups entries by address (full-width hash, so no padding needed),
/// then orders ascending by block number.
#[derive(Debug, Default, Clone, Eq, PartialEq, Ord, PartialOrd, Serialize, Deserialize, Hash)]
pub struct HashedAccountShardedKey(pub ShardedKey<B256>);

impl HashedAccountShardedKey {
    /// Create a new sharded key for a hashed account.
    pub const fn new(key: B256, highest_block_number: u64) -> Self {
        Self(ShardedKey::new(key, highest_block_number))
    }
}

impl Encode for HashedAccountShardedKey {
    type Encoded = [u8; 40]; // 32 (B256) + 8 (BlockNumber)

    fn encode(self) -> Self::Encoded {
        let mut buf = [0u8; 40];
        buf[..32].copy_from_slice(self.0.key.as_slice());
        buf[32..].copy_from_slice(&self.0.highest_block_number.to_be_bytes());
        buf
    }
}

impl Decode for HashedAccountShardedKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.len() != 40 {
            return Err(DatabaseError::Decode);
        }
        let key = B256::from_slice(&value[..32]);
        let highest_block_number =
            u64::from_be_bytes(value[32..].try_into().map_err(|_| DatabaseError::Decode)?);
        Ok(Self(ShardedKey::new(key, highest_block_number)))
    }
}

/// Sharded key for hashed storage history, keyed by `(hashed_address, storage_key, block)`.
///
/// Encoded as a 72-byte buffer:
///
/// ```text
/// [hashed_address: 32 bytes] ++ [storage_key: 32 bytes] ++ [block_number: 8 BE bytes]
/// ```
///
/// MDBX cursor walks group entries by address, then by storage key, then ascending by block.
#[derive(Debug, Clone, PartialEq, Eq, Ord, PartialOrd, Serialize, Deserialize)]
pub struct HashedStorageShardedKey {
    /// The hashed address of the account owning the storage.
    pub hashed_address: B256,
    /// The sharded key combining the storage key and sharded block number.
    pub sharded_key: ShardedKey<B256>,
}

impl Encode for HashedStorageShardedKey {
    type Encoded = Vec<u8>;
    fn encode(self) -> Self::Encoded {
        let mut buf = Vec::with_capacity(32 + 32 + 8);
        buf.extend_from_slice(self.hashed_address.as_slice());
        // ShardedKey<B256>: Key (32 bytes) + BlockNumber (8 bytes BE)
        buf.extend_from_slice(self.sharded_key.key.as_slice());
        buf.extend_from_slice(&self.sharded_key.highest_block_number.to_be_bytes());
        buf
    }
}

impl Decode for HashedStorageShardedKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        // 32 (Addr) + 32 (Key) + 8 (Block) = 72 bytes
        if value.len() < 72 {
            return Err(DatabaseError::Decode);
        }
        let (addr, rest) = value.split_at(32);
        let hashed_address = B256::from_slice(addr);
        let key = B256::from_slice(&rest[..32]);
        let highest_block_number =
            u64::from_be_bytes(rest[32..40].try_into().map_err(|_| DatabaseError::Decode)?);
        Ok(Self { hashed_address, sharded_key: ShardedKey::new(key, highest_block_number) })
    }
}

/// Sharded key for account trie history.
///
/// Encoded as a fixed-size 73-byte buffer. The nibble portion is right-padded with `0x00` and
/// followed by a length byte so MDBX's byte-wise sort agrees with `Nibbles`' lex-by-nibble
/// order:
///
/// ```text
/// [nibbles: 64 bytes, right-padded 0x00] ++ [length: 1 byte] ++ [block_number: 8 BE bytes]
/// ```
///
/// See [`StorageTrieShardedKey`] for the same layout extended with a per-account address.
#[derive(Debug, Default, Clone, Eq, PartialEq, Ord, PartialOrd, Serialize, Deserialize, Hash)]
pub struct AccountTrieShardedKey {
    /// Trie path as nibbles.
    pub key: StoredNibbles,
    /// Highest block number in this shard (or `u64::MAX` for the sentinel).
    pub highest_block_number: u64,
}

impl AccountTrieShardedKey {
    /// Create a new sharded key for an account trie path.
    pub const fn new(key: StoredNibbles, highest_block_number: u64) -> Self {
        Self { key, highest_block_number }
    }
}

impl Encode for AccountTrieShardedKey {
    type Encoded = [u8; ACCOUNT_TRIE_SHARDED_KEY_LEN];

    fn encode(self) -> Self::Encoded {
        let mut buf = [0u8; ACCOUNT_TRIE_SHARDED_KEY_LEN];
        buf[..NIBBLE_SUBKEY_LEN].copy_from_slice(&encode_nibble_subkey(&self.key));
        buf[NIBBLE_SUBKEY_LEN..].copy_from_slice(&self.highest_block_number.to_be_bytes());
        buf
    }
}

impl Decode for AccountTrieShardedKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        let bytes: &[u8; ACCOUNT_TRIE_SHARDED_KEY_LEN] =
            value.try_into().map_err(|_| DatabaseError::Decode)?;
        let nibble_buf: &[u8; NIBBLE_SUBKEY_LEN] =
            bytes[..NIBBLE_SUBKEY_LEN].try_into().map_err(|_| DatabaseError::Decode)?;
        let key = decode_nibble_subkey(nibble_buf);
        let highest_block_number = u64::from_be_bytes(
            bytes[NIBBLE_SUBKEY_LEN..].try_into().map_err(|_| DatabaseError::Decode)?,
        );
        Ok(Self { key, highest_block_number })
    }
}

/// Sharded key for storage trie history, keyed by `(hashed_address, nibble_path, block)`.
///
/// Encoded as a fixed-size 105-byte buffer; the 32-byte hashed address sits at position 0 so
/// MDBX cursor walks naturally group all entries for the same account, then sort by trie path
/// (lex-by-nibble) and block number — see [`AccountTrieShardedKey`] for the nibble-subkey
/// rationale.
///
/// ```text
/// [hashed_address: 32 bytes] ++ [nibbles: 64 bytes, right-padded 0x00]
///                            ++ [length: 1 byte]
///                            ++ [block_number: 8 BE bytes]
/// ```
#[derive(Debug, Clone, PartialEq, Eq, Ord, PartialOrd, Serialize, Deserialize)]
pub struct StorageTrieShardedKey {
    /// The hashed address of the account owning the storage trie.
    pub hashed_address: B256,
    /// The trie path (nibbles).
    pub key: StoredNibbles,
    /// Highest block number in this shard (or `u64::MAX` for the sentinel).
    pub highest_block_number: u64,
}

impl StorageTrieShardedKey {
    /// Create a new storage trie sharded key.
    pub const fn new(hashed_address: B256, key: StoredNibbles, highest_block_number: u64) -> Self {
        Self { hashed_address, key, highest_block_number }
    }
}

impl Encode for StorageTrieShardedKey {
    type Encoded = [u8; STORAGE_TRIE_SHARDED_KEY_LEN];

    fn encode(self) -> Self::Encoded {
        let mut buf = [0u8; STORAGE_TRIE_SHARDED_KEY_LEN];
        buf[..32].copy_from_slice(self.hashed_address.as_slice());
        buf[32..32 + NIBBLE_SUBKEY_LEN].copy_from_slice(&encode_nibble_subkey(&self.key));
        buf[32 + NIBBLE_SUBKEY_LEN..].copy_from_slice(&self.highest_block_number.to_be_bytes());
        buf
    }
}

impl Decode for StorageTrieShardedKey {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        let bytes: &[u8; STORAGE_TRIE_SHARDED_KEY_LEN] =
            value.try_into().map_err(|_| DatabaseError::Decode)?;
        let hashed_address = B256::from_slice(&bytes[..32]);
        let nibble_buf: &[u8; NIBBLE_SUBKEY_LEN] =
            bytes[32..32 + NIBBLE_SUBKEY_LEN].try_into().map_err(|_| DatabaseError::Decode)?;
        let key = decode_nibble_subkey(nibble_buf);
        let highest_block_number = u64::from_be_bytes(
            bytes[32 + NIBBLE_SUBKEY_LEN..].try_into().map_err(|_| DatabaseError::Decode)?,
        );
        Ok(Self { hashed_address, key, highest_block_number })
    }
}

/// Key for the storage `ChangeSets` table, keyed by `(block, hashed_address)`.
///
/// Encoded as a fixed-size 40-byte buffer:
///
/// ```text
/// [block_number: 8 BE bytes] ++ [hashed_address: 32 bytes]
/// ```
///
/// Block goes first so MDBX cursor walks iterate change sets in block order, with
/// address-grouping within each block. Replaces upstream `BlockNumberAddress`, which keyed by
/// the unhashed account address.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Ord, PartialOrd, Serialize, Deserialize)]
pub struct BlockNumberHashedAddress(pub (BlockNumber, B256));

impl Encode for BlockNumberHashedAddress {
    type Encoded = [u8; 40]; // 8 + 32
    fn encode(self) -> Self::Encoded {
        let mut buf = [0u8; 40];
        buf[..8].copy_from_slice(&self.0.0.to_be_bytes());
        buf[8..].copy_from_slice(self.0.1.as_slice());
        buf
    }
}

impl Decode for BlockNumberHashedAddress {
    fn decode(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.len() < 40 {
            return Err(DatabaseError::Decode);
        }
        let block_num = u64::from_be_bytes(value[..8].try_into().unwrap());
        let hash = B256::from_slice(&value[8..40]);
        Ok(Self((block_num, hash)))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_db::table::{Decode, Encode};
    use reth_trie_common::Nibbles;

    #[test]
    fn hashed_account_sharded_key_roundtrip() {
        let original = HashedAccountShardedKey::new(B256::repeat_byte(0xaa), 42);
        let decoded = HashedAccountShardedKey::decode(&original.clone().encode()).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn hashed_storage_sharded_key_roundtrip() {
        let original = HashedStorageShardedKey {
            hashed_address: B256::repeat_byte(0xaa),
            sharded_key: ShardedKey::new(B256::repeat_byte(0xbb), 100),
        };
        let decoded = HashedStorageShardedKey::decode(&original.clone().encode()).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn account_trie_sharded_key_roundtrip() {
        let nibbles = StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x0a, 0x0b, 0x0c]));
        let original = AccountTrieShardedKey::new(nibbles, 500);
        let decoded = AccountTrieShardedKey::decode(&original.clone().encode()).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn account_trie_sharded_key_roundtrip_empty_nibbles() {
        let original =
            AccountTrieShardedKey::new(StoredNibbles::from(Nibbles::default()), u64::MAX);
        let decoded = AccountTrieShardedKey::decode(&original.clone().encode()).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn storage_trie_sharded_key_roundtrip() {
        let nibbles = StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x01, 0x02]));
        let original = StorageTrieShardedKey::new(B256::repeat_byte(0xcc), nibbles, 999);
        let decoded = StorageTrieShardedKey::decode(&original.clone().encode()).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn storage_trie_sharded_key_roundtrip_empty_nibbles() {
        let original = StorageTrieShardedKey::new(
            B256::repeat_byte(0xdd),
            StoredNibbles::from(Nibbles::default()),
            0,
        );
        let decoded = StorageTrieShardedKey::decode(&original.clone().encode()).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn block_number_hashed_address_roundtrip() {
        let original = BlockNumberHashedAddress((42, B256::repeat_byte(0xdd)));
        let decoded = BlockNumberHashedAddress::decode(&original.encode()).unwrap();
        assert_eq!(original, decoded);
    }

    #[test]
    fn account_trie_shorter_nibbles_sort_before_longer() {
        let key_a = AccountTrieShardedKey::new(
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x01])),
            256,
        );
        let key_b = AccountTrieShardedKey::new(
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x01, 0x00])),
            1,
        );

        assert!(
            key_a.encode() < key_b.encode(),
            "shorter nibble path must sort before longer in encoded form"
        );
    }

    #[test]
    fn account_trie_same_nibbles_ordered_by_block() {
        let nibbles = StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x0a, 0x0b]));

        let lo = AccountTrieShardedKey::new(nibbles.clone(), 10);
        let hi = AccountTrieShardedKey::new(nibbles, 20);

        assert!(
            lo.encode() < hi.encode(),
            "same nibbles: lower block must sort before higher block"
        );
    }

    #[test]
    fn account_trie_nibbles_resembling_block_bytes_are_unambiguous() {
        let key_a = AccountTrieShardedKey::new(
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05,
            ])),
            1,
        );
        let key_b = AccountTrieShardedKey::new(StoredNibbles::from(Nibbles::default()), 5);

        let enc_a = key_a.encode();
        let enc_b = key_b.encode();
        assert_ne!(enc_a, enc_b, "different logical keys must never produce identical encodings");
        // Empty nibbles (all-zero padded path) must sort before non-empty paths whose first
        // non-zero nibble is greater than zero.
        assert!(enc_b < enc_a, "empty nibbles must sort before non-empty nibbles");
    }

    /// Regression: under the old length-prefixed encoding, the length byte at position 0
    /// dominated MDBX's byte-wise sort, so `[0x05]` (len=1) would have sorted before
    /// `[0x01, 0x05]` (len=2) — opposite of the logical nibble lex order. The fixed-size
    /// layout places nibble bytes first, so the actual path content drives ordering.
    #[test]
    fn account_trie_sort_follows_nibble_lex_order_not_length() {
        let key_short = AccountTrieShardedKey::new(
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x05])),
            0,
        );
        let key_long = AccountTrieShardedKey::new(
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x01, 0x05])),
            0,
        );

        assert!(
            key_long.encode() < key_short.encode(),
            "[0x01, 0x05] must sort before [0x05] because nibble 0x01 < 0x05",
        );
    }

    /// Storage-trie variant of the same regression.
    #[test]
    fn storage_trie_sort_follows_nibble_lex_order_not_length() {
        let addr = B256::repeat_byte(0x44);
        let key_short = StorageTrieShardedKey::new(
            addr,
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x05])),
            0,
        );
        let key_long = StorageTrieShardedKey::new(
            addr,
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x01, 0x05])),
            0,
        );

        assert!(
            key_long.encode() < key_short.encode(),
            "[0x01, 0x05] must sort before [0x05] within the same address",
        );
    }

    #[test]
    fn storage_trie_shorter_nibbles_sort_before_longer() {
        let addr = B256::repeat_byte(0x11);

        let key_a = StorageTrieShardedKey::new(
            addr,
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x0f])),
            256,
        );
        let key_b = StorageTrieShardedKey::new(
            addr,
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x0f, 0x00])),
            1,
        );

        assert!(
            key_a.encode() < key_b.encode(),
            "shorter nibble path must sort before longer in encoded form"
        );
    }

    #[test]
    fn storage_trie_same_nibbles_ordered_by_block() {
        let addr = B256::repeat_byte(0x22);
        let nibbles = StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x0a]));

        let lo = StorageTrieShardedKey::new(addr, nibbles.clone(), 10);
        let hi = StorageTrieShardedKey::new(addr, nibbles, 20);

        assert!(
            lo.encode() < hi.encode(),
            "same nibbles: lower block must sort before higher block"
        );
    }

    #[test]
    fn storage_trie_nibbles_resembling_block_bytes_are_unambiguous() {
        let addr = B256::repeat_byte(0x33);

        let key_a = StorageTrieShardedKey::new(
            addr,
            StoredNibbles::from(Nibbles::from_nibbles_unchecked([
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05,
            ])),
            1,
        );
        let key_b = StorageTrieShardedKey::new(addr, StoredNibbles::from(Nibbles::default()), 5);

        let enc_a = key_a.encode();
        let enc_b = key_b.encode();
        assert_ne!(enc_a, enc_b, "different logical keys must never produce identical encodings");
        assert!(enc_b < enc_a, "empty nibbles must sort before non-empty nibbles");
    }
}
