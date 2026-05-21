use alloy_primitives::B256;
use bytes::BufMut;
use reth_codecs::{Compact, DecompressError};
use reth_db::{
    DatabaseError,
    table::{Compress, Decompress},
};
use reth_primitives_traits::{Account, ValueWithSubKey};
use reth_trie_common::{BranchNodeCompact, StoredNibblesSubKey};
use serde::{Deserialize, Serialize};

/// Account state before a block, keyed by hashed address.
///
/// This is the hashed-address equivalent of reth's
/// `AccountBeforeTx`, designed for our v2 `AccountChangeSets`
/// table where keys are `keccak256(address)`.
///
/// Layout: `[hashed_address: 32 bytes][account: Compact-encoded or empty]`
///
/// - The 32-byte hashed address acts as the [`DupSort::SubKey`].
/// - An empty remainder means the account did not exist before this block (creation).
/// - A non-empty remainder is the [`Account`] state before the block was applied.
///
/// [`DupSort::SubKey`]: reth_db::table::DupSort::SubKey
#[derive(Debug, Default, Clone, Eq, PartialEq, Serialize, Deserialize)]
pub struct HashedAccountBeforeTx {
    /// Hashed address (`keccak256(address)`). Acts as `DupSort::SubKey`.
    pub hashed_address: B256,
    /// Account state before the block. `None` means the account didn't exist.
    pub info: Option<Account>,
}

impl HashedAccountBeforeTx {
    /// Creates a new instance.
    pub const fn new(hashed_address: B256, info: Option<Account>) -> Self {
        Self { hashed_address, info }
    }
}

impl ValueWithSubKey for HashedAccountBeforeTx {
    type SubKey = B256;

    fn get_subkey(&self) -> Self::SubKey {
        self.hashed_address
    }
}

impl Compress for HashedAccountBeforeTx {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        // SubKey: raw 32 bytes (uncompressed so MDBX can seek by it)
        buf.put_slice(self.hashed_address.as_slice());
        // Value: compress the account if present, otherwise write nothing
        if let Some(account) = &self.info {
            account.compress_to_buf(buf);
        }
    }
}

impl Decompress for HashedAccountBeforeTx {
    fn decompress(value: &[u8]) -> Result<Self, DecompressError> {
        if value.len() < 32 {
            return Err(DecompressError::new(DatabaseError::Decode));
        }

        let hashed_address = B256::from_slice(&value[..32]);
        let info = if value.len() > 32 { Some(Account::decompress(&value[32..])?) } else { None };

        Ok(Self { hashed_address, info })
    }
}

/// Trie changeset entry representing the state of a trie node before a block.
///
/// `nibbles` is the subkey when used as a value in the changeset tables.
/// This is a local definition since the upstream `reth-trie-common` crate does
/// not provide this type.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct TrieChangeSetsEntry {
    /// The nibbles of the intermediate node
    pub nibbles: StoredNibblesSubKey,
    /// Node value prior to the block being processed, None indicating it didn't exist.
    pub node: Option<BranchNodeCompact>,
}

impl ValueWithSubKey for TrieChangeSetsEntry {
    type SubKey = StoredNibblesSubKey;

    fn get_subkey(&self) -> Self::SubKey {
        self.nibbles.clone()
    }
}

impl Compress for TrieChangeSetsEntry {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        let _ = self.nibbles.to_compact(buf);
        if let Some(ref node) = self.node {
            let _ = node.to_compact(buf);
        }
    }
}

impl Decompress for TrieChangeSetsEntry {
    fn decompress(value: &[u8]) -> Result<Self, DecompressError> {
        if value.is_empty() {
            return Ok(Self {
                nibbles: StoredNibblesSubKey::from(reth_trie_common::Nibbles::default()),
                node: None,
            });
        }

        let (nibbles, rest) = StoredNibblesSubKey::from_compact(value, 65);
        let node = if rest.is_empty() {
            None
        } else {
            Some(BranchNodeCompact::from_compact(rest, rest.len()).0)
        };
        Ok(Self { nibbles, node })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_db::table::{Compress, Decompress};

    #[test]
    fn test_hashed_account_before_tx_roundtrip_some() {
        let original = HashedAccountBeforeTx {
            hashed_address: B256::repeat_byte(0xaa),
            info: Some(Account {
                nonce: 42,
                balance: alloy_primitives::U256::from(1000u64),
                bytecode_hash: None,
            }),
        };

        let compressed = original.clone().compress();
        assert!(compressed.len() > 32, "Should contain address + account data");

        let decompressed = HashedAccountBeforeTx::decompress(&compressed).unwrap();
        assert_eq!(original, decompressed);
    }

    #[test]
    fn test_hashed_account_before_tx_roundtrip_none() {
        let original =
            HashedAccountBeforeTx { hashed_address: B256::repeat_byte(0xbb), info: None };

        let compressed = original.clone().compress();
        assert_eq!(compressed.len(), 32, "None account should be just the 32-byte address");

        let decompressed = HashedAccountBeforeTx::decompress(&compressed).unwrap();
        assert_eq!(original, decompressed);
    }

    #[test]
    fn test_hashed_account_before_tx_subkey() {
        let addr = B256::repeat_byte(0xcc);
        let entry = HashedAccountBeforeTx::new(addr, None);
        assert_eq!(entry.get_subkey(), addr);
    }

    #[test]
    fn test_trie_changesets_entry_roundtrip_with_node() {
        let nibbles =
            StoredNibblesSubKey(reth_trie_common::Nibbles::from_nibbles_unchecked([0x0A, 0x0B]));
        let node = BranchNodeCompact::new(0b11, 0, 0, vec![], Some(B256::repeat_byte(0xDD)));
        let original = TrieChangeSetsEntry { nibbles, node: Some(node) };

        let compressed = original.clone().compress();
        let decompressed = TrieChangeSetsEntry::decompress(&compressed).unwrap();
        assert_eq!(original, decompressed);
    }

    #[test]
    fn test_trie_changesets_entry_roundtrip_none_node() {
        let nibbles = StoredNibblesSubKey(reth_trie_common::Nibbles::from_nibbles_unchecked([
            0x01, 0x02, 0x03,
        ]));
        let original = TrieChangeSetsEntry { nibbles, node: None };

        let compressed = original.clone().compress();
        let decompressed = TrieChangeSetsEntry::decompress(&compressed).unwrap();
        assert_eq!(original, decompressed);
    }

    #[test]
    fn test_trie_changesets_entry_roundtrip_empty() {
        let original = TrieChangeSetsEntry {
            nibbles: StoredNibblesSubKey(reth_trie_common::Nibbles::default()),
            node: None,
        };

        let compressed = original.clone().compress();
        let decompressed = TrieChangeSetsEntry::decompress(&compressed).unwrap();
        assert_eq!(original, decompressed);
    }

    #[test]
    fn test_trie_changesets_entry_subkey() {
        let nibbles =
            StoredNibblesSubKey(reth_trie_common::Nibbles::from_nibbles_unchecked([0x05, 0x06]));
        let entry = TrieChangeSetsEntry { nibbles: nibbles.clone(), node: None };
        assert_eq!(entry.get_subkey(), nibbles);
    }
}
