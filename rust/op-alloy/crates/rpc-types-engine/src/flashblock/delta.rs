//! Flashblock delta execution payload types.

use alloc::vec::Vec;
use alloy_eips::eip4895::Withdrawal;
use alloy_primitives::{B256, Bloom, Bytes};

/// Represents the modified portions of an execution payload within a flashblock.
/// This structure contains only the fields that can be updated during block construction,
/// such as state root, receipts, logs, and new transactions. Other immutable block fields
/// like parent hash and block number are excluded since they remain constant throughout
/// the block's construction.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct OpFlashblockPayloadDelta {
    /// The state root of the block.
    pub state_root: B256,
    /// The receipts root of the block.
    pub receipts_root: B256,
    /// The logs bloom of the block.
    pub logs_bloom: Bloom,
    /// The gas used of the block.
    #[cfg_attr(feature = "serde", serde(with = "alloy_serde::quantity"))]
    pub gas_used: u64,
    /// The block hash of the block.
    pub block_hash: B256,
    /// The transactions of the block.
    pub transactions: Vec<Bytes>,
    /// Array of [`Withdrawal`] enabled with V2
    pub withdrawals: Vec<Withdrawal>,
    /// The withdrawals root of the block.
    pub withdrawals_root: B256,
    /// The estimated cumulative blob gas used for the block. Introduced in Jovian.
    /// spec: <https://docs.optimism.io/notices/upgrade-17#block-header-changes>
    /// Defaults to 0 if not present (for pre-Jovian blocks).
    #[cfg_attr(
        feature = "serde",
        serde(
            default,
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )
    )]
    pub blob_gas_used: Option<u64>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;

    #[test]
    #[cfg(feature = "serde")]
    fn test_delta_serde_roundtrip() {
        let delta = OpFlashblockPayloadDelta {
            state_root: B256::random(),
            receipts_root: B256::random(),
            logs_bloom: Bloom::default(),
            gas_used: 21_000,
            block_hash: B256::random(),
            transactions: vec![Bytes::from(vec![1, 2, 3])],
            withdrawals: vec![],
            withdrawals_root: B256::random(),
            blob_gas_used: Some(123456),
        };

        let json = serde_json::to_string(&delta).unwrap();
        let decoded: OpFlashblockPayloadDelta = serde_json::from_str(&json).unwrap();
        assert_eq!(delta, decoded);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_delta_snake_case_serialization() {
        let delta = OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            gas_used: 0,
            block_hash: B256::ZERO,
            transactions: vec![],
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: Some(0),
        };

        let json = serde_json::to_string(&delta).unwrap();
        assert!(json.contains("state_root"));
        assert!(json.contains("receipts_root"));
        assert!(json.contains("logs_bloom"));
        assert!(json.contains("gas_used"));
        assert!(json.contains("block_hash"));
        assert!(json.contains("withdrawals_root"));
        assert!(json.contains("blob_gas_used"));
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_delta_with_withdrawals() {
        let withdrawal = Withdrawal {
            index: 0,
            validator_index: 1,
            address: alloy_primitives::Address::ZERO,
            amount: 1000,
        };

        let delta = OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            gas_used: 0,
            block_hash: B256::ZERO,
            transactions: vec![],
            withdrawals: vec![withdrawal],
            withdrawals_root: B256::ZERO,
            blob_gas_used: Some(0),
        };

        let json = serde_json::to_string(&delta).unwrap();
        let decoded: OpFlashblockPayloadDelta = serde_json::from_str(&json).unwrap();
        assert_eq!(delta.withdrawals.len(), 1);
        assert_eq!(decoded.withdrawals.len(), 1);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_delta_blob_gas_used_none_skipped() {
        // Test that None blob_gas_used is skipped in serialization (pre-Jovian)
        let delta = OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            gas_used: 0,
            block_hash: B256::ZERO,
            transactions: vec![],
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: None,
        };

        let json = serde_json::to_string(&delta).unwrap();
        // Should not contain blob_gas_used when None
        assert!(!json.contains("blob_gas_used"));

        // Deserialization should work and default to None
        let decoded: OpFlashblockPayloadDelta = serde_json::from_str(&json).unwrap();
        assert_eq!(decoded.blob_gas_used, None);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_delta_blob_gas_used_some_included() {
        // Test that Some blob_gas_used is included in serialization (Jovian+)
        let delta = OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            gas_used: 0,
            block_hash: B256::ZERO,
            transactions: vec![],
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: Some(12345),
        };

        let json = serde_json::to_string(&delta).unwrap();
        // Should contain blob_gas_used when Some
        assert!(json.contains("blob_gas_used"));
        assert!(json.contains("0x3039"));
    }

    #[test]
    fn test_delta_default() {
        let delta = OpFlashblockPayloadDelta::default();
        assert_eq!(delta.state_root, B256::ZERO);
        assert_eq!(delta.receipts_root, B256::ZERO);
        assert_eq!(delta.logs_bloom, Bloom::ZERO);
        assert_eq!(delta.gas_used, 0);
        assert_eq!(delta.block_hash, B256::ZERO);
        assert!(delta.transactions.is_empty());
        assert!(delta.withdrawals.is_empty());
        assert_eq!(delta.withdrawals_root, B256::ZERO);
        assert_eq!(delta.blob_gas_used, None);
    }
}
