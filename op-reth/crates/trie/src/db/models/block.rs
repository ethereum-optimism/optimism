use alloy_primitives::B256;
use bytes::BufMut;
use derive_more::{Constructor, From, Into};
use reth_db::{
    table::{Compress, Decompress},
    DatabaseError,
};
use serde::{Deserialize, Serialize};

/// Wrapper for block number and block hash tuple to implement [`Compress`]/[`Decompress`].
///
/// Used for storing block metadata (number + hash).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, From, Into, Constructor)]
pub struct BlockNumberHash(pub u64, pub B256);

impl Compress for BlockNumberHash {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        // Encode block number (8 bytes, big-endian) + hash (32 bytes) = 40 bytes total
        buf.put_u64(self.0);
        buf.put_slice(self.1.as_slice());
    }
}

impl Decompress for BlockNumberHash {
    fn decompress(value: &[u8]) -> Result<Self, DatabaseError> {
        if value.len() != 40 {
            return Err(DatabaseError::Decode);
        }

        let block_number =
            u64::from_be_bytes(value[..8].try_into().map_err(|_| DatabaseError::Decode)?);
        let hash = B256::from_slice(&value[8..40]);

        Ok(Self(block_number, hash))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;

    #[test]
    fn test_block_number_hash_roundtrip() {
        let test_cases = vec![
            BlockNumberHash(0, B256::ZERO),
            BlockNumberHash(42, B256::repeat_byte(0xaa)),
            BlockNumberHash(u64::MAX, B256::repeat_byte(0xff)),
        ];

        for original in test_cases {
            let compressed = original.compress();
            let decompressed = BlockNumberHash::decompress(&compressed).unwrap();
            assert_eq!(original, decompressed);
        }
    }
}
