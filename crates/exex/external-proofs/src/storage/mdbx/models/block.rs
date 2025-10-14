use bytes::BufMut;
use reth_db_api::table::{Compress, Decompress};
use serde::{Deserialize, Serialize};

/// Newtype wrapper for (u64, B256) to implement Compress/Decompress
///
/// Used for storing block metadata (number + hash).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct BlockNumberHash(pub u64, pub alloy_primitives::B256);

impl From<(u64, alloy_primitives::B256)> for BlockNumberHash {
    fn from((number, hash): (u64, alloy_primitives::B256)) -> Self {
        Self(number, hash)
    }
}

impl From<BlockNumberHash> for (u64, alloy_primitives::B256) {
    fn from(bnh: BlockNumberHash) -> Self {
        (bnh.0, bnh.1)
    }
}

impl BlockNumberHash {
    /// Create a new block number and hash pair
    pub const fn new(block_number: u64, hash: alloy_primitives::B256) -> Self {
        Self(block_number, hash)
    }

    /// Destructure into components
    pub const fn into_components(self) -> (u64, alloy_primitives::B256) {
        (self.0, self.1)
    }
}

impl Compress for BlockNumberHash {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        // Encode block number (8 bytes, big-endian) + hash (32 bytes) = 40 bytes total
        buf.put_u64(self.0);
        buf.put_slice(self.1.as_slice());
    }
}

impl Decompress for BlockNumberHash {
    fn decompress(value: &[u8]) -> Result<Self, reth_db_api::DatabaseError> {
        if value.len() != 40 {
            return Err(reth_db_api::DatabaseError::Decode);
        }

        let block_number = u64::from_be_bytes(
            value[..8].try_into().map_err(|_| reth_db_api::DatabaseError::Decode)?,
        );
        let hash = alloy_primitives::B256::from_slice(&value[8..40]);

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
