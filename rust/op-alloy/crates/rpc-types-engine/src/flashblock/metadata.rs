//! Flashblock metadata types.

use alloc::collections::BTreeMap;
use alloy_primitives::{Address, B256, U256};
use op_alloy_consensus::OpReceipt;

/// Provides metadata about the block that may be useful for indexing or analysis.
// Note: this uses mixed camel, snake case: <https://github.com/flashbots/rollup-boost/blob/dd12e8e8366004b4758bfa0cfa98efa6929b7e9f/crates/flashblocks-rpc/src/cache.rs#L31>
#[derive(Clone, Debug, Default, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct OpFlashblockPayloadMetadata {
    /// The number of the block in the L2 chain.
    pub block_number: u64,
    /// A map of addresses to their updated balances after the block execution.
    /// This represents balance changes due to transactions, rewards, or system transfers.
    pub new_account_balances: BTreeMap<Address, U256>,
    /// Execution receipts for all transactions in the block.
    /// Contains logs, gas usage, and other EVM-level metadata.
    #[cfg_attr(feature = "serde", serde(deserialize_with = "deserialize_flashblock_receipts"))]
    pub receipts: BTreeMap<B256, OpReceipt>,
}

#[cfg(feature = "serde")]
/// Supports deserializing flashblocks with externally tag receipts for backwards compatibility.
fn deserialize_flashblock_receipts<'de, D>(
    deserializer: D,
) -> Result<BTreeMap<B256, OpReceipt>, D::Error>
where
    D: serde::Deserializer<'de>,
{
    use alloy_consensus::Receipt;
    use op_alloy_consensus::OpDepositReceipt;
    use serde::Deserialize;

    #[derive(Deserialize)]
    enum ExternallyTagged {
        Legacy(Receipt),
        Eip2930(Receipt),
        Eip1559(Receipt),
        Eip7702(Receipt),
        Deposit(OpDepositReceipt),
    }

    #[derive(Deserialize)]
    #[serde(untagged)]
    enum MaybeExternallyTagged {
        ExternallyTagged(ExternallyTagged),
        InternallyTagged(OpReceipt),
    }

    impl From<MaybeExternallyTagged> for OpReceipt {
        fn from(value: MaybeExternallyTagged) -> Self {
            match value {
                MaybeExternallyTagged::ExternallyTagged(receipt) => match receipt {
                    ExternallyTagged::Legacy(receipt) => Self::Legacy(receipt),
                    ExternallyTagged::Eip2930(receipt) => Self::Eip2930(receipt),
                    ExternallyTagged::Eip1559(receipt) => Self::Eip1559(receipt),
                    ExternallyTagged::Eip7702(receipt) => Self::Eip7702(receipt),
                    ExternallyTagged::Deposit(receipt) => Self::Deposit(receipt),
                },
                MaybeExternallyTagged::InternallyTagged(receipt) => receipt,
            }
        }
    }

    Ok(BTreeMap::<B256, MaybeExternallyTagged>::deserialize(deserializer)?
        .into_iter()
        .map(|(hash, receipt)| (hash, receipt.into()))
        .collect())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec::Vec;
    use alloy_consensus::{Eip658Value, Receipt};
    use alloy_primitives::{Log, address};

    fn sample_metadata() -> OpFlashblockPayloadMetadata {
        let mut balances = BTreeMap::new();
        balances.insert(address!("0000000000000000000000000000000000000001"), U256::from(1000));

        let mut receipts = BTreeMap::new();
        let receipt = OpReceipt::Legacy(Receipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 21000,
            logs: Vec::new(),
        });
        receipts.insert(B256::ZERO, receipt);

        OpFlashblockPayloadMetadata { block_number: 100, new_account_balances: balances, receipts }
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_metadata_serde_roundtrip() {
        let metadata = sample_metadata();

        let json = serde_json::to_string(&metadata).unwrap();
        let decoded: OpFlashblockPayloadMetadata = serde_json::from_str(&json).unwrap();
        assert_eq!(metadata, decoded);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_metadata_snake_case_serialization() {
        let metadata = sample_metadata();

        let json = serde_json::to_string(&metadata).unwrap();
        assert!(json.contains("block_number"));
        assert!(json.contains("new_account_balances"));
        assert!(json.contains("receipts"));
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_address_balance_map_serialization() {
        let mut balances = BTreeMap::new();
        balances.insert(address!("0000000000000000000000000000000000000001"), U256::from(1000));
        balances.insert(address!("0000000000000000000000000000000000000002"), U256::from(2000));

        let metadata = OpFlashblockPayloadMetadata {
            block_number: 1,
            new_account_balances: balances,
            receipts: BTreeMap::new(),
        };

        let json = serde_json::to_value(&metadata).unwrap();
        let balances_obj = json.get("new_account_balances").unwrap();

        // Should be serialized as an object with hex string keys
        assert!(balances_obj.is_object());
        assert!(balances_obj.get("0x0000000000000000000000000000000000000001").is_some());
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_receipt_map_serialization() {
        let mut receipts = BTreeMap::new();
        let receipt1 = OpReceipt::Legacy(Receipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 21000,
            logs: Vec::<Log>::new(),
        });
        receipts.insert(B256::ZERO, receipt1);

        let metadata = OpFlashblockPayloadMetadata {
            block_number: 1,
            new_account_balances: BTreeMap::new(),
            receipts,
        };

        let json = serde_json::to_value(&metadata).unwrap();
        let receipts_obj = json.get("receipts").unwrap();

        // Should be serialized as an object with hex string keys
        assert!(receipts_obj.is_object());
        assert!(
            receipts_obj
                .get("0x0000000000000000000000000000000000000000000000000000000000000000")
                .is_some()
        );
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_receipt_json_format() {
        let mut receipts = BTreeMap::new();
        let receipt = OpReceipt::Legacy(Receipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 21000,
            logs: Vec::<Log>::new(),
        });
        receipts.insert(B256::ZERO, receipt);

        let metadata = OpFlashblockPayloadMetadata {
            block_number: 1,
            new_account_balances: BTreeMap::new(),
            receipts,
        };

        let json = serde_json::to_value(&metadata).unwrap();
        let receipts_obj = json.get("receipts").unwrap();
        let receipt_entry = receipts_obj
            .get("0x0000000000000000000000000000000000000000000000000000000000000000")
            .unwrap();

        assert_eq!(receipt_entry.get("type").unwrap().as_str().unwrap(), "0x0");
    }

    #[test]
    fn test_metadata_default() {
        let metadata = OpFlashblockPayloadMetadata::default();
        assert_eq!(metadata.block_number, 0);
        assert!(metadata.new_account_balances.is_empty());
        assert!(metadata.receipts.is_empty());
    }
}
