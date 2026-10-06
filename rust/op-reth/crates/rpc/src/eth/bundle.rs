//! `eth_sendBundle` support.
//!
//! A bundle wraps a single raw transaction with an execution window (block number or timestamp
//! range) and revert protection, as documented for Unichain at
//! <https://developers.uniswap.org/docs/unichain/technical-information/advanced-txn>:
//!
//! - the transaction is only included while `minBlockNumber <= block <= maxBlockNumber`, or
//!   `minTimestamp <= block timestamp <= maxTimestamp`; the two ranges are mutually exclusive,
//! - without a `maxBlockNumber` the bundle expires [`MAX_BLOCK_RANGE_BLOCKS`] blocks after
//!   submission; a timestamp-ranged bundle expires at `maxTimestamp`, at most
//!   [`MAX_TIMESTAMP_RANGE_SECS`] after the head,
//! - unless its hash is listed in `revertingTxHashes`, the transaction is left out of the block
//!   when it reverts, so the sender is not charged for a failed execution,
//! - once a bundle's window has passed without inclusion, `eth_getTransactionReceipt` for its hash
//!   answers with an invalid-params error rather than `null`.
//!
//! The window is carried by the pool transaction's [`TransactionConditional`], which the payload
//! builder checks before including the transaction and the conditional pool maintenance uses to
//! evict it once passed. The revert protection is carried by
//! [`MaybeRevertProtectedTransaction`], which the payload builder honours.
//!
//! [`MaybeRevertProtectedTransaction`]: reth_optimism_txpool::revert_protection::MaybeRevertProtectedTransaction

use alloy_primitives::{B256, Bytes, TxHash};
use alloy_rpc_types_eth::erc4337::TransactionConditional;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::RpcResult;
use jsonrpsee_types::error::{ErrorObject, INVALID_PARAMS_CODE};
use schnellru::{ByLength, LruMap};
use serde::{Deserialize, Serialize};
use std::{
    fmt,
    sync::{Arc, Mutex},
};

/// Maximum number of blocks a bundle may be scheduled into the future.
///
/// Bounds the lifetime of a block-ranged bundle in the pool: `maxBlockNumber` may not exceed the
/// current block number plus this value, and it defaults to exactly that when omitted.
pub const MAX_BLOCK_RANGE_BLOCKS: u64 = 10;

/// Maximum number of seconds a timestamp-ranged bundle may be scheduled into the future.
///
/// Bounds the lifetime of a timestamp-ranged bundle in the pool: `maxTimestamp` may not exceed
/// the current block's timestamp plus this value, and it defaults to exactly that when omitted.
pub const MAX_TIMESTAMP_RANGE_SECS: u64 = 600;

/// Error message returned by `eth_getTransactionReceipt` for a bundle whose execution window
/// passed without inclusion.
pub const DROPPED_FROM_POOL_MSG: &str = "the transaction was dropped from the pool";

/// Number of submitted bundles whose expiry is remembered for `eth_getTransactionReceipt`.
const PENDING_BUNDLES_CAPACITY: u32 = 4096;

/// `eth_sendBundle` request parameters.
///
/// Unknown fields are rejected rather than ignored, so a constraint this node cannot honour (for
/// example a flashblock range) fails loudly instead of being silently dropped.
#[derive(Serialize, Deserialize, Debug, Clone, Default, PartialEq, Eq)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct Bundle {
    /// Raw (EIP-2718 encoded) signed transactions. Exactly one is supported.
    pub txs: Vec<Bytes>,
    /// Transaction hashes allowed to revert.
    ///
    /// A transaction whose hash is not listed here (including when the field is omitted) is
    /// revert protected: it is only included in a block when its execution succeeds.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub reverting_tx_hashes: Option<Vec<B256>>,
    /// Earliest block number the transaction may be included in (hex quantity).
    #[serde(default, with = "alloy_serde::quantity::opt", skip_serializing_if = "Option::is_none")]
    pub min_block_number: Option<u64>,
    /// Latest block number the transaction may be included in (hex quantity).
    ///
    /// Defaults to the current block number plus [`MAX_BLOCK_RANGE_BLOCKS`].
    #[serde(default, with = "alloy_serde::quantity::opt", skip_serializing_if = "Option::is_none")]
    pub max_block_number: Option<u64>,
    /// Earliest block timestamp (Unix seconds, JSON number) the transaction may be included at.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub min_timestamp: Option<u64>,
    /// Latest block timestamp (Unix seconds, JSON number) the transaction may be included at.
    ///
    /// Defaults to the current block's timestamp plus [`MAX_TIMESTAMP_RANGE_SECS`] when a
    /// `minTimestamp` is given.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub max_timestamp: Option<u64>,
}

impl Bundle {
    /// Returns the bundle's single transaction, or an error if the bundle does not hold exactly
    /// one.
    pub fn single_tx(&self) -> Result<&Bytes, BundleError> {
        match self.txs.as_slice() {
            [tx] => Ok(tx),
            _ => Err(BundleError::NotExactlyOneTx { count: self.txs.len() }),
        }
    }

    /// Returns whether the transaction with the given hash must be left out of a block when it
    /// reverts, i.e. it is not listed in `revertingTxHashes`.
    pub fn is_revert_protected(&self, hash: &TxHash) -> bool {
        !self.reverting_tx_hashes.as_ref().is_some_and(|hashes| hashes.contains(hash))
    }

    /// Validates the bundle's execution window against the latest block and converts it into the
    /// [`TransactionConditional`] attached to the pool transaction.
    ///
    /// `latest_block_number` and `latest_timestamp` describe the current chain head; the bundle
    /// can only ever be included in a later block, so a maximum at or before the head is rejected.
    /// The returned conditional always carries an upper bound (block number or timestamp), which
    /// is what expires the bundle.
    pub fn conditional(
        &self,
        latest_block_number: u64,
        latest_timestamp: u64,
    ) -> Result<TransactionConditional, BundleError> {
        let has_block_range = self.min_block_number.is_some() || self.max_block_number.is_some();
        let has_timestamp_range = self.min_timestamp.is_some() || self.max_timestamp.is_some();
        if has_block_range && has_timestamp_range {
            return Err(BundleError::RangesMutuallyExclusive);
        }

        let mut conditional = TransactionConditional {
            known_accounts: Default::default(),
            block_number_min: self.min_block_number,
            block_number_max: None,
            timestamp_min: self.min_timestamp,
            timestamp_max: None,
        };

        if has_timestamp_range {
            let max_allowed = latest_timestamp + MAX_TIMESTAMP_RANGE_SECS;
            let max = self.max_timestamp.unwrap_or(max_allowed);
            if let Some(min) = self.min_timestamp &&
                min > max
            {
                return Err(BundleError::MinTimestampGreaterThanMax { min, max });
            }
            if max <= latest_timestamp {
                return Err(BundleError::MaxTimestampInPast { max, current: latest_timestamp });
            }
            if max > max_allowed {
                return Err(BundleError::MaxTimestampTooHigh {
                    max,
                    current: latest_timestamp,
                    max_allowed,
                });
            }
            conditional.timestamp_max = Some(max);
        } else {
            let max_allowed = latest_block_number + MAX_BLOCK_RANGE_BLOCKS;
            let max = self.max_block_number.unwrap_or(max_allowed);
            if let Some(min) = self.min_block_number &&
                min > max
            {
                return Err(BundleError::MinBlockGreaterThanMax { min, max });
            }
            if max <= latest_block_number {
                return Err(BundleError::MaxBlockInPast { max, current: latest_block_number });
            }
            if max > max_allowed {
                return Err(BundleError::MaxBlockTooHigh {
                    max,
                    current: latest_block_number,
                    max_allowed,
                });
            }
            conditional.block_number_max = Some(max);
        }

        Ok(conditional)
    }
}

/// `eth_sendBundle` response.
#[derive(Serialize, Deserialize, Debug, Clone, Copy, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct BundleResult {
    /// Hash of the bundle's transaction, usable with `eth_getTransactionReceipt`.
    pub bundle_hash: B256,
}

/// Reasons a bundle is rejected before it reaches the pool. All map to an invalid-params RPC error.
#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum BundleError {
    /// The bundle does not hold exactly one transaction.
    #[error("bundle must contain exactly one transaction (got {count})")]
    NotExactlyOneTx {
        /// Number of transactions in the bundle.
        count: usize,
    },
    /// Both a block number range and a timestamp range were given.
    #[error("block number range and timestamp range are mutually exclusive")]
    RangesMutuallyExclusive,
    /// `minBlockNumber` exceeds `maxBlockNumber` (or its default).
    #[error("min_block_number ({min}) is greater than max_block_number ({max})")]
    MinBlockGreaterThanMax {
        /// The requested minimum.
        min: u64,
        /// The requested (or default) maximum.
        max: u64,
    },
    /// `maxBlockNumber` is not after the current block.
    #[error("max_block_number ({max}) is a past block (current: {current})")]
    MaxBlockInPast {
        /// The requested maximum.
        max: u64,
        /// The current block number.
        current: u64,
    },
    /// `maxBlockNumber` is more than [`MAX_BLOCK_RANGE_BLOCKS`] blocks ahead.
    #[error(
        "max_block_number ({max}) is too high (current: {current}, max allowed: {max_allowed})"
    )]
    MaxBlockTooHigh {
        /// The requested maximum.
        max: u64,
        /// The current block number.
        current: u64,
        /// The highest accepted maximum.
        max_allowed: u64,
    },
    /// `minTimestamp` exceeds `maxTimestamp` (or its default).
    #[error("min_timestamp ({min}) is greater than max_timestamp ({max})")]
    MinTimestampGreaterThanMax {
        /// The requested minimum.
        min: u64,
        /// The requested (or default) maximum.
        max: u64,
    },
    /// `maxTimestamp` is not after the current block's timestamp.
    #[error("max_timestamp ({max}) is a past timestamp (current: {current})")]
    MaxTimestampInPast {
        /// The requested maximum.
        max: u64,
        /// The current block timestamp.
        current: u64,
    },
    /// `maxTimestamp` is more than [`MAX_TIMESTAMP_RANGE_SECS`] ahead of the current block.
    #[error("max_timestamp ({max}) is too high (current: {current}, max allowed: {max_allowed})")]
    MaxTimestampTooHigh {
        /// The requested maximum.
        max: u64,
        /// The current block timestamp.
        current: u64,
        /// The highest accepted maximum.
        max_allowed: u64,
    },
}

impl From<BundleError> for ErrorObject<'static> {
    fn from(err: BundleError) -> Self {
        ErrorObject::owned(INVALID_PARAMS_CODE, err.to_string(), None::<String>)
    }
}

/// The end of a bundle's execution window, after which it can no longer be included.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct BundleExpiry {
    /// Last block number the bundle may be included in, if block-ranged.
    pub block_number_max: Option<u64>,
    /// Last block timestamp the bundle may be included at, if timestamp-ranged.
    pub timestamp_max: Option<u64>,
}

impl BundleExpiry {
    /// Returns whether a chain head at `block_number` / `timestamp` is past this window: no later
    /// block can include the bundle any more.
    pub fn is_expired_at(&self, block_number: u64, timestamp: u64) -> bool {
        self.block_number_max.is_some_and(|max| block_number >= max) ||
            self.timestamp_max.is_some_and(|max| timestamp >= max)
    }
}

impl From<&TransactionConditional> for BundleExpiry {
    fn from(conditional: &TransactionConditional) -> Self {
        Self {
            block_number_max: conditional.block_number_max,
            timestamp_max: conditional.timestamp_max,
        }
    }
}

/// Expiries of the bundles submitted through this node, keyed by transaction hash.
///
/// Shared between the `eth_sendBundle` handler, which records every accepted bundle (also when
/// it is forwarded to a sequencer), and the `eth` API, whose `eth_getTransactionReceipt` answers
/// with [`DROPPED_FROM_POOL_MSG`] for a bundle whose window has passed without a receipt. Bounded;
/// the oldest entries are forgotten first. Clones share the same map.
#[derive(Clone)]
pub struct PendingBundles(Arc<Mutex<LruMap<TxHash, BundleExpiry, ByLength>>>);

impl Default for PendingBundles {
    fn default() -> Self {
        Self(Arc::new(Mutex::new(LruMap::new(ByLength::new(PENDING_BUNDLES_CAPACITY)))))
    }
}

impl fmt::Debug for PendingBundles {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("PendingBundles").field("len", &self.lock().len()).finish()
    }
}

impl PendingBundles {
    fn lock(&self) -> std::sync::MutexGuard<'_, LruMap<TxHash, BundleExpiry, ByLength>> {
        self.0.lock().unwrap_or_else(|poisoned| poisoned.into_inner())
    }

    /// Records (or replaces) the expiry of the bundle with this transaction hash.
    pub fn insert(&self, hash: TxHash, expiry: BundleExpiry) {
        self.lock().insert(hash, expiry);
    }

    /// Returns the recorded expiry of the bundle with this transaction hash, if any.
    pub fn get(&self, hash: &TxHash) -> Option<BundleExpiry> {
        self.lock().peek(hash).copied()
    }

    /// Returns whether a bundle with this hash was submitted here and its window has passed at a
    /// chain head with the given number and timestamp.
    pub fn is_expired(&self, hash: &TxHash, block_number: u64, timestamp: u64) -> bool {
        self.get(hash).is_some_and(|expiry| expiry.is_expired_at(block_number, timestamp))
    }
}

/// `eth` namespace extension for bundles.
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "eth"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "eth"))]
pub trait OpEthBundleApi {
    /// Submits a single-transaction bundle with an execution window and revert protection.
    ///
    /// Returns the transaction hash as `bundleHash`.
    #[method(name = "sendBundle")]
    async fn send_bundle(&self, bundle: Bundle) -> RpcResult<BundleResult>;
}

#[cfg(test)]
mod tests {
    use super::*;

    const LATEST_BLOCK: u64 = 1000;
    const LATEST_TIMESTAMP: u64 = 1_700_000_000;

    fn conditional(bundle: Bundle) -> Result<TransactionConditional, BundleError> {
        bundle.conditional(LATEST_BLOCK, LATEST_TIMESTAMP)
    }

    #[test]
    fn deserializes_documented_params() {
        let bundle: Bundle = serde_json::from_str(
            r#"{
                "txs": ["0x02f8"],
                "minBlockNumber": "0x3e9",
                "maxBlockNumber": "0x3ed",
                "revertingTxHashes": ["0x0000000000000000000000000000000000000000000000000000000000000001"]
            }"#,
        )
        .unwrap();
        assert_eq!(bundle.txs, vec![Bytes::from_static(&[0x02, 0xf8])]);
        assert_eq!(bundle.min_block_number, Some(1001));
        assert_eq!(bundle.max_block_number, Some(1005));
        assert_eq!(bundle.reverting_tx_hashes, Some(vec![B256::with_last_byte(1)]));
        assert_eq!(bundle.min_timestamp, None);

        let bundle: Bundle = serde_json::from_str(
            r#"{"txs": ["0x02f8"], "minTimestamp": 1700000060, "maxTimestamp": 1700000300}"#,
        )
        .unwrap();
        assert_eq!(bundle.min_timestamp, Some(1_700_000_060));
        assert_eq!(bundle.max_timestamp, Some(1_700_000_300));
        assert_eq!(bundle.max_block_number, None);
    }

    #[test]
    fn rejects_unknown_fields() {
        assert!(
            serde_json::from_str::<Bundle>(r#"{"txs": ["0x02f8"], "minFlashblockNumber": "0x1"}"#)
                .is_err()
        );
    }

    #[test]
    fn serializes_result_as_bundle_hash() {
        let json =
            serde_json::to_value(BundleResult { bundle_hash: B256::with_last_byte(1) }).unwrap();
        assert_eq!(json["bundleHash"], B256::with_last_byte(1).to_string());
    }

    #[test]
    fn single_tx() {
        assert_eq!(Bundle::default().single_tx(), Err(BundleError::NotExactlyOneTx { count: 0 }));
        let two = Bundle { txs: vec![Bytes::new(), Bytes::new()], ..Default::default() };
        assert_eq!(two.single_tx(), Err(BundleError::NotExactlyOneTx { count: 2 }));
        let one = Bundle { txs: vec![Bytes::from_static(&[1])], ..Default::default() };
        assert_eq!(one.single_tx(), Ok(&Bytes::from_static(&[1])));
    }

    #[test]
    fn revert_protection_follows_reverting_tx_hashes() {
        let hash = B256::with_last_byte(1);
        assert!(Bundle::default().is_revert_protected(&hash));
        let empty = Bundle { reverting_tx_hashes: Some(vec![]), ..Default::default() };
        assert!(empty.is_revert_protected(&hash));
        let listed = Bundle { reverting_tx_hashes: Some(vec![hash]), ..Default::default() };
        assert!(!listed.is_revert_protected(&hash));
        assert!(listed.is_revert_protected(&B256::with_last_byte(2)));
    }

    #[test]
    fn no_bounds_defaults_max_block() {
        let cond = conditional(Bundle::default()).unwrap();
        assert_eq!(cond.block_number_min, None);
        assert_eq!(cond.block_number_max, Some(LATEST_BLOCK + MAX_BLOCK_RANGE_BLOCKS));
        assert_eq!(cond.timestamp_min, None);
        assert_eq!(cond.timestamp_max, None);
        assert!(cond.known_accounts.is_empty());
    }

    #[test]
    fn valid_block_range() {
        let cond = conditional(Bundle {
            min_block_number: Some(1002),
            max_block_number: Some(1005),
            ..Default::default()
        })
        .unwrap();
        assert_eq!(cond.block_number_min, Some(1002));
        assert_eq!(cond.block_number_max, Some(1005));

        // Only a minimum: the default maximum still applies.
        let cond =
            conditional(Bundle { min_block_number: Some(1005), ..Default::default() }).unwrap();
        assert_eq!(cond.block_number_min, Some(1005));
        assert_eq!(cond.block_number_max, Some(1010));

        // A minimum at or below the head is accepted; the pool just treats it as already met.
        let cond =
            conditional(Bundle { min_block_number: Some(999), ..Default::default() }).unwrap();
        assert_eq!(cond.block_number_min, Some(999));

        // A single-block window is accepted.
        let cond = conditional(Bundle {
            min_block_number: Some(1001),
            max_block_number: Some(1001),
            ..Default::default()
        })
        .unwrap();
        assert_eq!(cond.block_number_max, Some(1001));
    }

    #[test]
    fn invalid_block_ranges() {
        assert_eq!(
            conditional(Bundle {
                min_block_number: Some(1010),
                max_block_number: Some(1005),
                ..Default::default()
            }),
            Err(BundleError::MinBlockGreaterThanMax { min: 1010, max: 1005 })
        );
        assert_eq!(
            conditional(Bundle { max_block_number: Some(1000), ..Default::default() }),
            Err(BundleError::MaxBlockInPast { max: 1000, current: 1000 })
        );
        assert_eq!(
            conditional(Bundle { max_block_number: Some(1011), ..Default::default() }),
            Err(BundleError::MaxBlockTooHigh { max: 1011, current: 1000, max_allowed: 1010 })
        );
        // A minimum beyond the default maximum can never be met.
        assert_eq!(
            conditional(Bundle { min_block_number: Some(1011), ..Default::default() }),
            Err(BundleError::MinBlockGreaterThanMax { min: 1011, max: 1010 })
        );
    }

    #[test]
    fn valid_timestamp_range_expires_by_timestamp() {
        // The documented example: a window that opens a minute from now and closes after five.
        let cond = conditional(Bundle {
            min_timestamp: Some(LATEST_TIMESTAMP + 60),
            max_timestamp: Some(LATEST_TIMESTAMP + 300),
            ..Default::default()
        })
        .unwrap();
        assert_eq!(cond.timestamp_min, Some(LATEST_TIMESTAMP + 60));
        assert_eq!(cond.timestamp_max, Some(LATEST_TIMESTAMP + 300));
        assert_eq!(cond.block_number_min, None);
        // No block-number expiry: it would close the window before `minTimestamp` is reached.
        assert_eq!(cond.block_number_max, None);

        // Only a minimum: the default maximum applies.
        let cond = conditional(Bundle {
            min_timestamp: Some(LATEST_TIMESTAMP + 60),
            ..Default::default()
        })
        .unwrap();
        assert_eq!(cond.timestamp_max, Some(LATEST_TIMESTAMP + MAX_TIMESTAMP_RANGE_SECS));
        assert_eq!(cond.block_number_max, None);
    }

    #[test]
    fn invalid_timestamp_ranges() {
        assert_eq!(
            conditional(Bundle {
                min_timestamp: Some(LATEST_TIMESTAMP + 300),
                max_timestamp: Some(LATEST_TIMESTAMP + 60),
                ..Default::default()
            }),
            Err(BundleError::MinTimestampGreaterThanMax {
                min: LATEST_TIMESTAMP + 300,
                max: LATEST_TIMESTAMP + 60
            })
        );
        assert_eq!(
            conditional(Bundle { max_timestamp: Some(LATEST_TIMESTAMP), ..Default::default() }),
            Err(BundleError::MaxTimestampInPast {
                max: LATEST_TIMESTAMP,
                current: LATEST_TIMESTAMP
            })
        );
        let too_high = LATEST_TIMESTAMP + MAX_TIMESTAMP_RANGE_SECS + 1;
        assert_eq!(
            conditional(Bundle { max_timestamp: Some(too_high), ..Default::default() }),
            Err(BundleError::MaxTimestampTooHigh {
                max: too_high,
                current: LATEST_TIMESTAMP,
                max_allowed: LATEST_TIMESTAMP + MAX_TIMESTAMP_RANGE_SECS
            })
        );
        assert_eq!(
            conditional(Bundle { min_timestamp: Some(too_high), ..Default::default() }),
            Err(BundleError::MinTimestampGreaterThanMax {
                min: too_high,
                max: LATEST_TIMESTAMP + MAX_TIMESTAMP_RANGE_SECS
            })
        );
    }

    #[test]
    fn block_and_timestamp_ranges_are_mutually_exclusive() {
        assert_eq!(
            conditional(Bundle {
                max_block_number: Some(1005),
                min_timestamp: Some(LATEST_TIMESTAMP + 1),
                ..Default::default()
            }),
            Err(BundleError::RangesMutuallyExclusive)
        );
        assert_eq!(
            conditional(Bundle {
                min_block_number: Some(1001),
                max_timestamp: Some(LATEST_TIMESTAMP + 1),
                ..Default::default()
            }),
            Err(BundleError::RangesMutuallyExclusive)
        );
    }

    #[test]
    fn bundle_error_is_invalid_params() {
        let err: ErrorObject<'static> = BundleError::RangesMutuallyExclusive.into();
        assert_eq!(err.code(), INVALID_PARAMS_CODE);
    }

    #[test]
    fn expiry_follows_the_window_upper_bound() {
        let by_block = BundleExpiry::from(&conditional(Bundle::default()).unwrap());
        assert_eq!(by_block, BundleExpiry { block_number_max: Some(1010), timestamp_max: None });
        // A head at the maximum means the next block is already past the window.
        assert!(!by_block.is_expired_at(1009, u64::MAX));
        assert!(by_block.is_expired_at(1010, 0));

        let by_timestamp = BundleExpiry::from(
            &conditional(Bundle {
                max_timestamp: Some(LATEST_TIMESTAMP + 30),
                ..Default::default()
            })
            .unwrap(),
        );
        assert_eq!(
            by_timestamp,
            BundleExpiry { block_number_max: None, timestamp_max: Some(LATEST_TIMESTAMP + 30) }
        );
        assert!(!by_timestamp.is_expired_at(u64::MAX, LATEST_TIMESTAMP + 29));
        assert!(by_timestamp.is_expired_at(0, LATEST_TIMESTAMP + 30));
    }

    #[test]
    fn pending_bundles_report_expiry_only_for_known_hashes() {
        let pending = PendingBundles::default();
        let hash = B256::with_last_byte(1);
        assert!(!pending.is_expired(&hash, u64::MAX, u64::MAX));

        pending.insert(hash, BundleExpiry { block_number_max: Some(1010), timestamp_max: None });
        assert_eq!(
            pending.get(&hash),
            Some(BundleExpiry { block_number_max: Some(1010), timestamp_max: None })
        );
        assert!(!pending.is_expired(&hash, 1009, 0));
        assert!(pending.is_expired(&hash, 1010, 0));

        // Resubmission replaces the window.
        pending.insert(hash, BundleExpiry { block_number_max: Some(1020), timestamp_max: None });
        assert!(!pending.is_expired(&hash, 1010, 0));
    }
}
