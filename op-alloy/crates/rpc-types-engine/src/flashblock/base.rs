//! Flashblock base execution payload types.

use alloy_primitives::{Address, B256, Bytes, U256};

/// Immutable block properties shared across all flashblocks in a sequence.
///
/// These properties remain constant throughout the block construction process
/// and are set at the beginning of the flashblock sequence.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct OpFlashblockPayloadBase {
    /// Parent beacon block root.
    pub parent_beacon_block_root: B256,
    /// Hash of the parent block.
    pub parent_hash: B256,
    /// Address that receives fees for this block.
    pub fee_recipient: Address,
    /// The previous randao value.
    pub prev_randao: B256,
    /// Block number.
    #[cfg_attr(feature = "serde", serde(with = "alloy_serde::quantity"))]
    pub block_number: u64,
    /// Gas limit for this block.
    #[cfg_attr(feature = "serde", serde(with = "alloy_serde::quantity"))]
    pub gas_limit: u64,
    /// Block timestamp.
    #[cfg_attr(feature = "serde", serde(with = "alloy_serde::quantity"))]
    pub timestamp: u64,
    /// Extra data for the block.
    pub extra_data: Bytes,
    /// Base fee per gas for this block.
    pub base_fee_per_gas: U256,
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;

    #[test]
    #[cfg(feature = "serde")]
    fn test_base_serde_roundtrip() {
        let base = OpFlashblockPayloadBase {
            parent_beacon_block_root: B256::random(),
            parent_hash: B256::random(),
            fee_recipient: Address::random(),
            prev_randao: B256::random(),
            block_number: 100,
            gas_limit: 30_000_000,
            timestamp: 1234567890,
            extra_data: Bytes::from(vec![1, 2, 3]),
            base_fee_per_gas: U256::from(1000000000u64),
        };

        let json = serde_json::to_string(&base).unwrap();
        let decoded: OpFlashblockPayloadBase = serde_json::from_str(&json).unwrap();
        assert_eq!(base, decoded);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_base_snake_case_serialization() {
        let base = OpFlashblockPayloadBase {
            parent_beacon_block_root: B256::ZERO,
            parent_hash: B256::ZERO,
            fee_recipient: Address::ZERO,
            prev_randao: B256::ZERO,
            block_number: 1,
            gas_limit: 30_000_000,
            timestamp: 1234567890,
            extra_data: Bytes::default(),
            base_fee_per_gas: U256::from(1000000000u64),
        };

        let json = serde_json::to_string(&base).unwrap();
        assert!(json.contains("parent_beacon_block_root"));
        assert!(json.contains("parent_hash"));
        assert!(json.contains("fee_recipient"));
        assert!(json.contains("prev_randao"));
        assert!(json.contains("block_number"));
        assert!(json.contains("gas_limit"));
        assert!(json.contains("base_fee_per_gas"));
    }

    #[test]
    fn test_base_default() {
        let base = OpFlashblockPayloadBase::default();
        assert_eq!(base.parent_beacon_block_root, B256::ZERO);
        assert_eq!(base.parent_hash, B256::ZERO);
        assert_eq!(base.fee_recipient, Address::ZERO);
        assert_eq!(base.block_number, 0);
        assert_eq!(base.gas_limit, 0);
        assert_eq!(base.timestamp, 0);
        assert_eq!(base.extra_data, Bytes::default());
        assert_eq!(base.base_fee_per_gas, U256::ZERO);
    }
}
