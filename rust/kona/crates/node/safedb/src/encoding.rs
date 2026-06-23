//! Byte-exact key/value encoding for the safe-head database.
//!
//! Keys and values match the `op-node` `safedb` layout exactly so that databases are
//! interchangeable between the Go and Rust implementations.

use crate::error::SafeDbError;
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;

/// Prefix byte distinguishing the "safe head by L1 block number" column.
///
/// Keys are prefixed with a constant byte to allow differentiating columns within the data.
pub(crate) const KEY_PREFIX_SAFE_BY_L1_BLOCK_NUM: u8 = 0;

/// Length in bytes of an encoded key: one prefix byte plus a big-endian `u64`.
pub(crate) const KEY_LEN: usize = 9;

/// Length in bytes of an encoded value: two 32-byte hashes plus a big-endian `u64`.
pub(crate) const VALUE_LEN: usize = 72;

/// Encodes the key for the given L1 block number.
pub(crate) fn safe_by_l1_block_num_key(l1_block_num: u64) -> [u8; KEY_LEN] {
    let mut key = [0u8; KEY_LEN];
    key[0] = KEY_PREFIX_SAFE_BY_L1_BLOCK_NUM;
    key[1..].copy_from_slice(&l1_block_num.to_be_bytes());
    key
}

/// Returns the maximum possible key, used as the exclusive upper bound for range operations.
pub(crate) fn max_key() -> [u8; KEY_LEN] {
    safe_by_l1_block_num_key(u64::MAX)
}

/// Encodes the value pairing an L1 block with the L2 safe head derived as of that block.
pub(crate) fn safe_by_l1_block_num_value(l1: BlockNumHash, l2: BlockNumHash) -> [u8; VALUE_LEN] {
    let mut val = [0u8; VALUE_LEN];
    val[0..32].copy_from_slice(l1.hash.as_slice());
    val[32..64].copy_from_slice(l2.hash.as_slice());
    val[64..].copy_from_slice(&l2.number.to_be_bytes());
    val
}

/// Decodes a key/value pair back into the L1 block and its recorded L2 safe head.
///
/// The L1 block number lives in the key while both hashes and the L2 number live in the value.
pub(crate) fn decode_safe_by_l1_block_num(
    key: &[u8],
    val: &[u8],
) -> Result<(BlockNumHash, BlockNumHash), SafeDbError> {
    if key.len() != KEY_LEN || val.len() != VALUE_LEN || key[0] != KEY_PREFIX_SAFE_BY_L1_BLOCK_NUM {
        return Err(SafeDbError::InvalidEntry);
    }
    let l1 = BlockNumHash {
        hash: B256::from_slice(&val[0..32]),
        number: u64::from_be_bytes(key[1..KEY_LEN].try_into().expect("checked length")),
    };
    let l2 = BlockNumHash {
        hash: B256::from_slice(&val[32..64]),
        number: u64::from_be_bytes(val[64..VALUE_LEN].try_into().expect("checked length")),
    };
    Ok((l1, l2))
}
