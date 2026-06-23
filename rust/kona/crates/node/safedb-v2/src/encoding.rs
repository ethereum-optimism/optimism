//! Byte-exact key/value encoding for the safe-head database.
//!
//! Keys and values match the `op-node` `safedb` layout exactly so that databases are
//! interchangeable between the Go and Rust implementations.

use crate::{error::SafeDbError, traits::SafeHeadRecord};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;

/// Codec for the "safe head by L1 block number" column.
///
/// Keys are prefixed with a constant byte so that multiple columns can coexist within a single
/// key space.
#[derive(Debug)]
pub(crate) struct SafeByL1BlockNum;

impl SafeByL1BlockNum {
    /// Prefix byte distinguishing this column from any other a future schema may add.
    pub(crate) const PREFIX: u8 = 0;

    /// Length in bytes of an encoded key: one prefix byte plus a big-endian `u64`.
    pub(crate) const KEY_LEN: usize = 9;

    /// Length in bytes of an encoded value: two 32-byte hashes plus a big-endian `u64`.
    pub(crate) const VALUE_LEN: usize = 72;

    /// Encodes the key for the given L1 block number.
    pub(crate) fn key(l1_block_num: u64) -> [u8; Self::KEY_LEN] {
        let mut key = [0u8; Self::KEY_LEN];
        key[0] = Self::PREFIX;
        key[1..].copy_from_slice(&l1_block_num.to_be_bytes());
        key
    }

    /// Returns the maximum possible key, used as the exclusive upper bound for range operations.
    pub(crate) fn max_key() -> [u8; Self::KEY_LEN] {
        Self::key(u64::MAX)
    }

    /// Encodes the value pairing an L1 block with the L2 safe head derived as of that block.
    pub(crate) fn value(l1: BlockNumHash, l2: BlockNumHash) -> [u8; Self::VALUE_LEN] {
        let mut val = [0u8; Self::VALUE_LEN];
        val[0..32].copy_from_slice(l1.hash.as_slice());
        val[32..64].copy_from_slice(l2.hash.as_slice());
        val[64..].copy_from_slice(&l2.number.to_be_bytes());
        val
    }

    /// Decodes a key/value pair back into the L1 block and its recorded L2 safe head.
    ///
    /// The L1 block number lives in the key while both hashes and the L2 number live in the value.
    pub(crate) fn decode(key: &[u8], val: &[u8]) -> Result<SafeHeadRecord, SafeDbError> {
        if key.len() != Self::KEY_LEN || val.len() != Self::VALUE_LEN || key[0] != Self::PREFIX {
            return Err(SafeDbError::InvalidEntry);
        }
        let l1 = BlockNumHash {
            hash: B256::from_slice(&val[0..32]),
            number: u64::from_be_bytes(
                key[1..Self::KEY_LEN].try_into().map_err(|_| SafeDbError::InvalidEntry)?,
            ),
        };
        let safe_head = BlockNumHash {
            hash: B256::from_slice(&val[32..64]),
            number: u64::from_be_bytes(
                val[64..Self::VALUE_LEN].try_into().map_err(|_| SafeDbError::InvalidEntry)?,
            ),
        };
        Ok(SafeHeadRecord { l1, safe_head })
    }
}
