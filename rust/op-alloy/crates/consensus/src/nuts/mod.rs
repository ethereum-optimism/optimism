//! Network Upgrade Transaction (NUT) bundle types and conversion.
//!
//! NUT bundles define the L2 deposit transactions that activate a hardfork. Each bundle contains
//! ordered transactions (contract deployments, proxy upgrades, etc.) that the rollup node executes
//! at the fork activation block.
//!
//! Source hashes are derived from a qualified intent string that combines the fork name,
//! transaction index, and per-transaction intent field:
//! `"{fork_name} {index}: {intent}"`.

use alloc::{string::String, vec::Vec};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, Bytes, TxKind, U256};

use crate::{TxDeposit, UpgradeDepositSource};

/// A single network upgrade transaction from a NUT bundle.
#[derive(Debug, Clone)]
pub struct NetworkUpgradeTransaction {
    /// Human-readable intent string, used for source hash derivation.
    pub intent: String,
    /// Sender address.
    pub from: Address,
    /// Recipient address. `None` for contract creation.
    pub to: Option<Address>,
    /// Calldata.
    pub data: Bytes,
    /// Gas limit for this transaction.
    pub gas_limit: u64,
}

/// A parsed NUT (Network Upgrade Transaction) bundle.
///
/// Constructed either from JSON at build time (via `build.rs`) or directly in code.
/// Use [`NutBundle::to_deposit_transactions`] to convert to deposit transactions with
/// correctly derived source hashes.
#[derive(Debug, Clone)]
pub struct NutBundle {
    /// The fork name, used to namespace intents for source hash derivation.
    pub fork_name: String,
    /// The ordered list of upgrade transactions.
    pub transactions: Vec<NetworkUpgradeTransaction>,
}

impl NutBundle {
    /// Returns the total gas required by all transactions in the bundle.
    pub fn total_gas(&self) -> u64 {
        self.transactions.iter().map(|tx| tx.gas_limit).sum()
    }

    /// Converts the bundle into a list of [`TxDeposit`]s.
    ///
    /// Source hashes are derived from a qualified intent string:
    /// `"{fork_name} {index}: {intent}"`.
    ///
    /// Returns an error if any transaction is missing an intent.
    pub fn to_deposit_transactions(&self) -> Result<Vec<TxDeposit>, NutBundleError> {
        self.transactions
            .iter()
            .enumerate()
            .map(|(i, tx)| {
                if tx.intent.is_empty() {
                    return Err(NutBundleError::MissingIntent(i));
                }

                let qualified_intent = alloc::format!("{} {}: {}", self.fork_name, i, tx.intent);
                let source = UpgradeDepositSource { intent: qualified_intent };

                Ok(TxDeposit {
                    source_hash: source.source_hash(),
                    from: tx.from,
                    to: tx.to.map_or(TxKind::Create, TxKind::Call),
                    mint: 0,
                    value: U256::ZERO,
                    gas_limit: tx.gas_limit,
                    is_system_transaction: false,
                    input: tx.data.clone(),
                })
            })
            .collect()
    }

    /// Converts the bundle into EIP-2718 encoded deposit transaction bytes.
    ///
    /// This is the format expected by [`crate::Hardfork::txs`][hardfork-txs].
    ///
    /// [hardfork-txs]: https://docs.rs/kona-hardforks/latest/kona_hardforks/trait.Hardfork.html
    pub fn to_encoded_transactions(&self) -> Result<Vec<Bytes>, NutBundleError> {
        Ok(self
            .to_deposit_transactions()?
            .into_iter()
            .map(|tx| {
                let mut encoded = Vec::new();
                tx.encode_2718(&mut encoded);
                Bytes::from(encoded)
            })
            .collect())
    }
}

/// Errors that can occur when processing a NUT bundle.
#[derive(Debug, thiserror::Error)]
pub enum NutBundleError {
    /// A transaction is missing an intent string.
    #[error("tx {0}: missing intent")]
    MissingIntent(usize),
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::string::ToString;
    use alloy_primitives::{address, b256, hex};

    fn test_bundle() -> NutBundle {
        NutBundle {
            fork_name: "Test".to_string(),
            transactions: alloc::vec![
                NetworkUpgradeTransaction {
                    intent: "First Transaction".to_string(),
                    from: Address::ZERO,
                    to: Some(address!("f39Fd6e51aad88F6F4ce6aB8827279cffFb92266")),
                    data: Bytes::from_static(&hex!("abcdef")),
                    gas_limit: 1_000_000,
                },
                NetworkUpgradeTransaction {
                    intent: "Second Transaction".to_string(),
                    from: address!("000000000000000000000000000000000000abba"),
                    to: Some(address!("4200000000000000000000000000000000000015")),
                    data: Bytes::from_static(&hex!("feedface")),
                    gas_limit: 5_000_000,
                },
            ],
        }
    }

    #[test]
    fn test_total_gas() {
        let bundle = test_bundle();
        assert_eq!(bundle.total_gas(), 6_000_000);
    }

    #[test]
    fn test_to_deposit_transactions() {
        let bundle = test_bundle();
        let deposits = bundle.to_deposit_transactions().unwrap();

        assert_eq!(deposits.len(), 2);

        // Verify first tx: qualified intent is "Test 0: First Transaction"
        let expected_source_0 =
            UpgradeDepositSource { intent: "Test 0: First Transaction".to_string() };
        assert_eq!(deposits[0].source_hash, expected_source_0.source_hash());
        assert_eq!(
            deposits[0].source_hash,
            b256!("c3ecdc70c81521aae81240518d3547f601bb33ec07b909e83544b2fe093c78bd")
        );
        assert_eq!(deposits[0].from, Address::ZERO);
        assert_eq!(
            deposits[0].to,
            TxKind::Call(address!("f39Fd6e51aad88F6F4ce6aB8827279cffFb92266"))
        );
        assert_eq!(deposits[0].gas_limit, 1_000_000);
        assert_eq!(deposits[0].value, U256::ZERO);
        assert_eq!(deposits[0].mint, 0);
        assert!(!deposits[0].is_system_transaction);
        assert_eq!(deposits[0].input.as_ref(), hex!("abcdef").as_slice());

        // Verify second tx: qualified intent is "Test 1: Second Transaction"
        let expected_source_1 =
            UpgradeDepositSource { intent: "Test 1: Second Transaction".to_string() };
        assert_eq!(deposits[1].source_hash, expected_source_1.source_hash());
        assert_eq!(
            deposits[1].source_hash,
            b256!("ac61fbb5e3d61e626f66c413c564d376e3b363b94a37aab323b860a2293b7c6c")
        );
        assert_eq!(deposits[1].from, address!("000000000000000000000000000000000000abba"));
        assert_eq!(
            deposits[1].to,
            TxKind::Call(address!("4200000000000000000000000000000000000015"))
        );
        assert_eq!(deposits[1].input.as_ref(), hex!("feedface").as_slice());
        assert_eq!(deposits[1].gas_limit, 5_000_000);

        // Source hashes must be unique.
        assert_ne!(deposits[0].source_hash, deposits[1].source_hash);
    }

    #[test]
    fn test_to_encoded_transactions() {
        let bundle = test_bundle();
        let encoded = bundle.to_encoded_transactions().unwrap();

        assert_eq!(encoded.len(), 2);
        assert_eq!(
            encoded[0].as_ref(),
            hex!("7ef856a0c3ecdc70c81521aae81240518d3547f601bb33ec07b909e83544b2fe093c78bd94000000000000000000000000000000000000000094f39fd6e51aad88f6f4ce6ab8827279cfffb922668080830f42408083abcdef").as_slice()
        );
        assert_eq!(
            encoded[1].as_ref(),
            hex!("7ef857a0ac61fbb5e3d61e626f66c413c564d376e3b363b94a37aab323b860a2293b7c6c94000000000000000000000000000000000000abba9442000000000000000000000000000000000000158080834c4b408084feedface").as_slice()
        );
    }

    #[test]
    fn test_null_to_produces_create() {
        let bundle = NutBundle {
            fork_name: "Test".to_string(),
            transactions: alloc::vec![NetworkUpgradeTransaction {
                intent: "Deploy Contract".to_string(),
                from: address!("4210000000000000000000000000000000000006"),
                to: None,
                data: Bytes::from_static(&hex!("deadbeef")),
                gas_limit: 500_000,
            }],
        };

        let deposits = bundle.to_deposit_transactions().unwrap();
        assert_eq!(deposits[0].to, TxKind::Create);
    }

    #[test]
    fn test_missing_intent_error() {
        let bundle = NutBundle {
            fork_name: "Test".to_string(),
            transactions: alloc::vec![NetworkUpgradeTransaction {
                intent: String::new(),
                from: Address::ZERO,
                to: Some(Address::ZERO),
                data: Bytes::new(),
                gas_limit: 100_000,
            }],
        };

        let err = bundle.to_deposit_transactions().unwrap_err();
        assert!(matches!(err, NutBundleError::MissingIntent(0)));
        assert_eq!(err.to_string(), "tx 0: missing intent");
    }

    #[test]
    fn test_empty_bundle() {
        let bundle = NutBundle { fork_name: "Test".to_string(), transactions: alloc::vec![] };

        assert_eq!(bundle.total_gas(), 0);
        let deposits = bundle.to_deposit_transactions().unwrap();
        assert!(deposits.is_empty());
    }

    #[test]
    fn test_gas_matches_deposit_gas() {
        let bundle = test_bundle();
        let deposits = bundle.to_deposit_transactions().unwrap();

        let sum_gas: u64 = deposits.iter().map(|d| d.gas_limit).sum();
        assert_eq!(bundle.total_gas(), sum_gas);
    }
}
