use bytes::{Buf, BufMut};
use reth_db::{
    table::{Compress, Decompress},
    DatabaseError,
};
use serde::{Deserialize, Serialize};

/// Wrapper type for `Option<T>` that implements `Compress` and `Decompress`
///
/// Encoding:
/// - `None` => empty byte array (length 0)
/// - `Some(value)` => compressed bytes of value (length > 0)
///
/// This assumes the inner type `T` always compresses to non-empty bytes when it exists.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct MaybeDeleted<T>(pub Option<T>);

impl<T> From<Option<T>> for MaybeDeleted<T> {
    fn from(opt: Option<T>) -> Self {
        Self(opt)
    }
}

impl<T> From<MaybeDeleted<T>> for Option<T> {
    fn from(maybe: MaybeDeleted<T>) -> Self {
        maybe.0
    }
}

impl<T: Compress> Compress for MaybeDeleted<T> {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        match &self.0 {
            None => {
                // Empty = deleted, write nothing
            }
            Some(value) => {
                // Compress the inner value to the buffer
                value.compress_to_buf(buf);
            }
        }
    }
}

impl<T: Decompress> Decompress for MaybeDeleted<T> {
    fn decompress(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.is_empty() {
            // Empty = deleted
            Ok(Self(None))
        } else {
            // Non-empty = present
            let inner = T::decompress(value)?;
            Ok(Self(Some(inner)))
        }
    }
}

/// Versioned value wrapper for `DupSort` tables
///
/// For `DupSort` tables in MDBX, the Value type must contain the `SubKey` as a field.
/// This wrapper combines a `block_number` (the `SubKey`) with the actual value.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct VersionedValue<T> {
    /// Block number (`SubKey` for `DupSort`)
    pub block_number: u64,
    /// The actual value (may be deleted)
    pub value: MaybeDeleted<T>,
}

impl<T> VersionedValue<T> {
    /// Create a new versioned value
    pub const fn new(block_number: u64, value: MaybeDeleted<T>) -> Self {
        Self { block_number, value }
    }
}

impl<T: Compress> Compress for VersionedValue<T> {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        // Encode block number first (8 bytes, big-endian)
        buf.put_u64(self.block_number);
        // Then encode the value
        self.value.compress_to_buf(buf);
    }
}

impl<T: Decompress> Decompress for VersionedValue<T> {
    fn decompress(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.len() < 8 {
            return Err(DatabaseError::Decode);
        }

        let mut buf: &[u8] = value;
        let block_number = buf.get_u64();
        let value = MaybeDeleted::<T>::decompress(&value[8..])?;

        Ok(Self { block_number, value })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_primitives_traits::Account;
    use reth_trie::BranchNodeCompact;

    #[test]
    fn test_maybe_deleted_none() {
        let none: MaybeDeleted<Account> = MaybeDeleted(None);
        let compressed = none.compress();
        assert!(compressed.is_empty(), "None should compress to empty bytes");

        let decompressed = MaybeDeleted::<Account>::decompress(&compressed).unwrap();
        assert_eq!(decompressed.0, None);
    }

    #[test]
    fn test_maybe_deleted_some_account() {
        let account = Account {
            nonce: 42,
            balance: alloy_primitives::U256::from(1000u64),
            bytecode_hash: None,
        };
        let some = MaybeDeleted(Some(account));
        let compressed = some.compress();
        assert!(!compressed.is_empty(), "Some should compress to non-empty bytes");

        let decompressed = MaybeDeleted::<Account>::decompress(&compressed).unwrap();
        assert_eq!(decompressed.0, Some(account));
    }

    #[test]
    fn test_maybe_deleted_some_branch() {
        // Create a simple valid BranchNodeCompact (empty is valid)
        let branch = BranchNodeCompact::new(
            0,      // state_mask
            0,      // tree_mask
            0,      // hash_mask
            vec![], // hashes
            None,   // root_hash
        );
        let some = MaybeDeleted(Some(branch.clone()));
        let compressed = some.compress();
        assert!(!compressed.is_empty(), "Some should compress to non-empty bytes");

        let decompressed = MaybeDeleted::<BranchNodeCompact>::decompress(&compressed).unwrap();
        assert_eq!(decompressed.0, Some(branch));
    }

    #[test]
    fn test_maybe_deleted_roundtrip() {
        let test_cases = vec![
            MaybeDeleted(None),
            MaybeDeleted(Some(Account {
                nonce: 0,
                balance: alloy_primitives::U256::ZERO,
                bytecode_hash: None,
            })),
            MaybeDeleted(Some(Account {
                nonce: 999,
                balance: alloy_primitives::U256::MAX,
                bytecode_hash: Some([0xff; 32].into()),
            })),
        ];

        for original in test_cases {
            let compressed = original.clone().compress();
            let decompressed = MaybeDeleted::<Account>::decompress(&compressed).unwrap();
            assert_eq!(original, decompressed);
        }
    }
}
