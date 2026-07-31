use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_eth::erc4337::TransactionConditional;
use reth_rpc_eth_types::EthApiError;
use serde::{Deserialize, Serialize};

/// Maximum number of blocks allowed in the block range for bundle execution.
///
/// This constant limits how far into the future a bundle can be scheduled to
/// prevent excessive resource usage and ensure timely execution. When no
/// maximum block number is specified, this value is added to the current block
/// number to set the default upper bound.
pub const MAX_BLOCK_RANGE_BLOCKS: u64 = 10;

/// A bundle represents a collection of transactions that should be executed
/// together with specific conditional constraints.
///
/// Bundles allow for sophisticated transaction ordering and conditional
/// execution based on block numbers, flashblock numbers, and timestamps. They
/// are a key primitive in MEV (Maximal Extractable Value) strategies and block
/// building.
///
/// # Validation
///
/// The following validations are performed before adding the transaction to the
/// mempool:
/// - Block number ranges are valid (min ≤ max)
/// - Maximum block numbers are not in the past
/// - Block ranges don't exceed `MAX_BLOCK_RANGE_BLOCKS` (currently 10)
/// - There's only one transaction in the bundle
/// - Flashblock number ranges are valid (min ≤ max)
#[derive(Serialize, Deserialize, Debug, Clone, Default)]
pub struct Bundle {
    /// List of raw transaction data to be included in the bundle.
    ///
    /// Each transaction is represented as raw bytes that will be decoded and
    /// executed in the specified order when the bundle conditions are met.
    #[serde(rename = "txs")]
    pub transactions: Vec<Bytes>,

    /// Optional list of transaction hashes that are allowed to revert.
    ///
    /// By default, if any transaction in a bundle reverts, the entire bundle is
    /// considered invalid. This field allows specific transactions to revert
    /// without invalidating the bundle, enabling more sophisticated MEV
    /// strategies.
    #[serde(rename = "revertingTxHashes")]
    pub reverting_hashes: Option<Vec<B256>>,

    /// Minimum block number at which this bundle can be included.
    ///
    /// If specified, the bundle will only be considered for inclusion in blocks
    /// at or after this block number. This allows for scheduling bundles for
    /// future execution.
    #[serde(
        default,
        rename = "minBlockNumber",
        with = "alloy_serde::quantity::opt",
        skip_serializing_if = "Option::is_none"
    )]
    pub block_number_min: Option<u64>,

    /// Maximum block number at which this bundle can be included.
    ///
    /// If specified, the bundle will be considered invalid for inclusion in
    /// blocks after this block number. If not specified, defaults to the
    /// current block number plus `MAX_BLOCK_RANGE_BLOCKS`.
    #[serde(
        default,
        rename = "maxBlockNumber",
        with = "alloy_serde::quantity::opt",
        skip_serializing_if = "Option::is_none"
    )]
    pub block_number_max: Option<u64>,

    /// Minimum flashblock number at which this bundle can be included.
    ///
    /// Flashblocks are preconfirmations that are built incrementally. This
    /// field along with `maxFlashblockNumber` allows bundles to be scheduled
    /// for more precise execution.
    #[serde(
        default,
        rename = "minFlashblockNumber",
        with = "alloy_serde::quantity::opt",
        skip_serializing_if = "Option::is_none"
    )]
    pub flashblock_number_min: Option<u64>,

    /// Maximum flashblock number at which this bundle can be included.
    ///
    /// Similar to `minFlashblockNumber`, this sets an upper bound on which
    /// flashblocks can include this bundle.
    #[serde(
        default,
        rename = "maxFlashblockNumber",
        with = "alloy_serde::quantity::opt",
        skip_serializing_if = "Option::is_none"
    )]
    pub flashblock_number_max: Option<u64>,

    /// Minimum timestamp (Unix epoch seconds) for bundle inclusion.
    ///
    /// **Warning**: Not recommended for production use as it depends on the
    /// builder node's clock, which may not be perfectly synchronized with
    /// network time. Block number constraints are preferred for deterministic
    /// behavior.
    #[serde(
        default,
        rename = "minTimestamp",
        skip_serializing_if = "Option::is_none"
    )]
    pub min_timestamp: Option<u64>,

    /// Maximum timestamp (Unix epoch seconds) for bundle inclusion.
    ///
    /// **Warning**: Not recommended for production use as it depends on the
    /// builder node's clock, which may not be perfectly synchronized with
    /// network time. Block number constraints are preferred for deterministic
    /// behavior.
    #[serde(
        default,
        rename = "maxTimestamp",
        skip_serializing_if = "Option::is_none"
    )]
    pub max_timestamp: Option<u64>,
}

impl From<BundleConditionalError> for EthApiError {
    fn from(err: BundleConditionalError) -> Self {
        EthApiError::InvalidParams(err.to_string())
    }
}

#[derive(Debug, thiserror::Error)]
pub enum BundleConditionalError {
    #[error("block_number_min ({min}) is greater than block_number_max ({max})")]
    MinGreaterThanMax { min: u64, max: u64 },
    #[error("block_number_max ({max}) is a past block (current: {current})")]
    MaxBlockInPast { max: u64, current: u64 },
    /// To prevent resource exhaustion and ensure timely execution, bundles
    /// cannot be scheduled more than `MAX_BLOCK_RANGE_BLOCKS` blocks into the
    /// future.
    #[error(
        "block_number_max ({max}) is too high (current: {current}, max allowed: {max_allowed})"
    )]
    MaxBlockTooHigh {
        max: u64,
        current: u64,
        max_allowed: u64,
    },
    /// When no explicit maximum block number is provided, the system uses
    /// `current_block + MAX_BLOCK_RANGE_BLOCKS` as the default maximum. This
    /// error occurs when the specified minimum exceeds this default maximum.
    #[error(
        "block_number_min ({min}) is too high with default max range (max allowed: {max_allowed})"
    )]
    MinTooHighForDefaultRange { min: u64, max_allowed: u64 },
    #[error("flashblock_number_min ({min}) is greater than flashblock_number_max ({max})")]
    FlashblockMinGreaterThanMax { min: u64, max: u64 },
}

pub struct BundleConditional {
    pub transaction_conditional: TransactionConditional,
    pub flashblock_number_min: Option<u64>,
    pub flashblock_number_max: Option<u64>,
}

impl Bundle {
    pub fn conditional(
        &self,
        last_block_number: u64,
    ) -> Result<BundleConditional, BundleConditionalError> {
        let mut block_number_max = self.block_number_max;
        let block_number_min = self.block_number_min;

        // Validate block number ranges
        if let Some(max) = block_number_max {
            // Check if min > max
            if let Some(min) = block_number_min
                && min > max
            {
                return Err(BundleConditionalError::MinGreaterThanMax { min, max });
            }

            // The max block cannot be a past block
            if max <= last_block_number {
                return Err(BundleConditionalError::MaxBlockInPast {
                    max,
                    current: last_block_number,
                });
            }

            // Validate that it is not greater than the max_block_range
            let max_allowed = last_block_number + MAX_BLOCK_RANGE_BLOCKS;
            if max > max_allowed {
                return Err(BundleConditionalError::MaxBlockTooHigh {
                    max,
                    current: last_block_number,
                    max_allowed,
                });
            }
        } else {
            // If no upper bound is set, use the maximum block range
            let default_max = last_block_number + MAX_BLOCK_RANGE_BLOCKS;
            block_number_max = Some(default_max);

            // Ensure that the new max is not smaller than the min
            if let Some(min) = block_number_min
                && min > default_max
            {
                return Err(BundleConditionalError::MinTooHighForDefaultRange {
                    min,
                    max_allowed: default_max,
                });
            }
        }

        // Validate flashblock number range
        if let Some(min) = self.flashblock_number_min
            && let Some(max) = self.flashblock_number_max
            && min > max
        {
            return Err(BundleConditionalError::FlashblockMinGreaterThanMax { min, max });
        }

        Ok(BundleConditional {
            transaction_conditional: TransactionConditional {
                block_number_min,
                block_number_max,
                known_accounts: Default::default(),
                timestamp_max: self.max_timestamp,
                timestamp_min: self.min_timestamp,
            },
            flashblock_number_min: self.flashblock_number_min,
            flashblock_number_max: self.flashblock_number_max,
        })
    }
}

/// Result returned after successfully submitting a bundle for inclusion.
///
/// This struct contains the unique identifier for the submitted bundle, which
/// can be used to track the bundle's status and inclusion in future blocks.
#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct BundleResult {
    /// Transaction hash of the single transaction in the bundle.
    ///
    /// This hash can be used to:
    /// - Track bundle inclusion in blocks
    /// - Query bundle status
    /// - Reference the bundle in subsequent operations
    #[serde(rename = "bundleHash")]
    pub bundle_hash: B256,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_bundle_conditional_no_bounds() {
        let bundle = Bundle {
            transactions: vec![],
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle
            .conditional(last_block)
            .unwrap()
            .transaction_conditional;

        assert_eq!(result.block_number_min, None);
        assert_eq!(
            result.block_number_max,
            Some(last_block + MAX_BLOCK_RANGE_BLOCKS)
        );
    }

    #[test]
    fn test_bundle_conditional_with_valid_bounds() {
        let bundle = Bundle {
            block_number_max: Some(1005),
            block_number_min: Some(1002),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle
            .conditional(last_block)
            .unwrap()
            .transaction_conditional;

        assert_eq!(result.block_number_min, Some(1002));
        assert_eq!(result.block_number_max, Some(1005));
    }

    #[test]
    fn test_bundle_conditional_min_greater_than_max() {
        let bundle = Bundle {
            block_number_max: Some(1005),
            block_number_min: Some(1010),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block);

        assert!(matches!(
            result,
            Err(BundleConditionalError::MinGreaterThanMax {
                min: 1010,
                max: 1005
            })
        ));
    }

    #[test]
    fn test_bundle_conditional_max_in_past() {
        let bundle = Bundle {
            block_number_max: Some(999),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block);

        assert!(matches!(
            result,
            Err(BundleConditionalError::MaxBlockInPast {
                max: 999,
                current: 1000
            })
        ));
    }

    #[test]
    fn test_bundle_conditional_max_too_high() {
        let bundle = Bundle {
            block_number_max: Some(1020),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block);

        assert!(matches!(
            result,
            Err(BundleConditionalError::MaxBlockTooHigh {
                max: 1020,
                current: 1000,
                max_allowed: 1010
            })
        ));
    }

    #[test]
    fn test_bundle_conditional_min_too_high_for_default_range() {
        let bundle = Bundle {
            block_number_min: Some(1015),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block);

        assert!(matches!(
            result,
            Err(BundleConditionalError::MinTooHighForDefaultRange {
                min: 1015,
                max_allowed: 1010
            })
        ));
    }

    #[test]
    fn test_bundle_conditional_with_only_min() {
        let bundle = Bundle {
            block_number_min: Some(1005),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle
            .conditional(last_block)
            .unwrap()
            .transaction_conditional;

        assert_eq!(result.block_number_min, Some(1005));
        assert_eq!(result.block_number_max, Some(1010)); // last_block + MAX_BLOCK_RANGE_BLOCKS
    }

    #[test]
    fn test_bundle_conditional_with_only_max() {
        let bundle = Bundle {
            block_number_max: Some(1008),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle
            .conditional(last_block)
            .unwrap()
            .transaction_conditional;

        assert_eq!(result.block_number_min, None);
        assert_eq!(result.block_number_max, Some(1008));
    }

    #[test]
    fn test_bundle_conditional_min_lower_than_last_block() {
        let bundle = Bundle {
            block_number_min: Some(999),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle
            .conditional(last_block)
            .unwrap()
            .transaction_conditional;

        assert_eq!(result.block_number_min, Some(999));
        assert_eq!(result.block_number_max, Some(1010));
    }

    #[test]
    fn test_bundle_conditional_flashblock_min_greater_than_max() {
        let bundle = Bundle {
            flashblock_number_min: Some(105),
            flashblock_number_max: Some(100),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block);

        assert!(matches!(
            result,
            Err(BundleConditionalError::FlashblockMinGreaterThanMax { min: 105, max: 100 })
        ));
    }

    #[test]
    fn test_bundle_conditional_with_valid_flashblock_range() {
        let bundle = Bundle {
            flashblock_number_min: Some(100),
            flashblock_number_max: Some(105),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block).unwrap();

        assert_eq!(result.flashblock_number_min, Some(100));
        assert_eq!(result.flashblock_number_max, Some(105));
    }

    #[test]
    fn test_bundle_conditional_with_only_flashblock_min() {
        let bundle = Bundle {
            flashblock_number_min: Some(100),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block).unwrap();

        assert_eq!(result.flashblock_number_min, Some(100));
        assert_eq!(result.flashblock_number_max, None);
    }

    #[test]
    fn test_bundle_conditional_with_only_flashblock_max() {
        let bundle = Bundle {
            flashblock_number_max: Some(105),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block).unwrap();

        assert_eq!(result.flashblock_number_min, None);
        assert_eq!(result.flashblock_number_max, Some(105));
    }

    #[test]
    fn test_bundle_conditional_flashblock_equal_values() {
        let bundle = Bundle {
            flashblock_number_min: Some(100),
            flashblock_number_max: Some(100),
            ..Default::default()
        };

        let last_block = 1000;
        let result = bundle.conditional(last_block).unwrap();

        assert_eq!(result.flashblock_number_min, Some(100));
        assert_eq!(result.flashblock_number_max, Some(100));
    }
}
