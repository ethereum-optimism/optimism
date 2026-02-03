//! Optimism specific types related to transactions.

use alloy_consensus::{Transaction as TransactionTrait, Typed2718, transaction::Recovered};
use alloy_eips::{Encodable2718, eip2930::AccessList, eip7702::SignedAuthorization};
use alloy_primitives::{Address, B256, BlockHash, Bytes, ChainId, TxKind, U256};
use alloy_serde::OtherFields;
use op_alloy_consensus::{OpTransaction, OpTxEnvelope, transaction::OpTransactionInfo};
use serde::{Deserialize, Serialize};

mod request;
pub use request::OpTransactionRequest;

/// OP Transaction type
#[derive(
    Clone, Debug, PartialEq, Eq, Serialize, Deserialize, derive_more::Deref, derive_more::DerefMut,
)]
#[cfg_attr(all(any(test, feature = "arbitrary"), feature = "k256"), derive(arbitrary::Arbitrary))]
#[serde(
    try_from = "tx_serde::TransactionSerdeHelper<T>",
    into = "tx_serde::TransactionSerdeHelper<T>",
    bound = "T: TransactionTrait + OpTransaction + Clone + serde::Serialize + serde::de::DeserializeOwned"
)]
pub struct Transaction<T = OpTxEnvelope> {
    /// Ethereum Transaction Types
    #[deref]
    #[deref_mut]
    pub inner: alloy_rpc_types_eth::Transaction<T>,

    /// Nonce for deposit transactions. Only present in RPC responses.
    pub deposit_nonce: Option<u64>,

    /// Deposit receipt version for deposit transactions post-canyon
    pub deposit_receipt_version: Option<u64>,
}

impl<T: OpTransaction + TransactionTrait> Transaction<T> {
    /// Converts a consensus `tx` with an additional context `tx_info` into an RPC [`Transaction`].
    pub fn from_transaction(tx: Recovered<T>, tx_info: OpTransactionInfo) -> Self {
        let base_fee = tx_info.inner.base_fee;
        let effective_gas_price = if tx.is_deposit() {
            // For deposits, we must always set the `gasPrice` field to 0 in rpc
            // deposit tx don't have a gas price field, but serde of `Transaction` will take care of
            // it
            0
        } else {
            base_fee
                .map(|base_fee| {
                    tx.effective_tip_per_gas(base_fee).unwrap_or_default() + base_fee as u128
                })
                .unwrap_or_else(|| tx.max_fee_per_gas())
        };

        Self {
            inner: alloy_rpc_types_eth::Transaction {
                inner: tx,
                block_hash: tx_info.inner.block_hash,
                block_number: tx_info.inner.block_number,
                transaction_index: tx_info.inner.index,
                effective_gas_price: Some(effective_gas_price),
            },
            deposit_nonce: tx_info.deposit_meta.deposit_nonce,
            deposit_receipt_version: tx_info.deposit_meta.deposit_receipt_version,
        }
    }
}

impl<T: Typed2718> Typed2718 for Transaction<T> {
    fn ty(&self) -> u8 {
        self.inner.ty()
    }
}

impl<T: TransactionTrait> TransactionTrait for Transaction<T> {
    fn chain_id(&self) -> Option<ChainId> {
        self.inner.chain_id()
    }

    fn nonce(&self) -> u64 {
        self.inner.nonce()
    }

    fn gas_limit(&self) -> u64 {
        self.inner.gas_limit()
    }

    fn gas_price(&self) -> Option<u128> {
        self.inner.gas_price()
    }

    fn max_fee_per_gas(&self) -> u128 {
        self.inner.max_fee_per_gas()
    }

    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        self.inner.max_priority_fee_per_gas()
    }

    fn max_fee_per_blob_gas(&self) -> Option<u128> {
        self.inner.max_fee_per_blob_gas()
    }

    fn priority_fee_or_price(&self) -> u128 {
        self.inner.priority_fee_or_price()
    }

    fn effective_gas_price(&self, base_fee: Option<u64>) -> u128 {
        self.inner.effective_gas_price(base_fee)
    }

    fn is_dynamic_fee(&self) -> bool {
        self.inner.is_dynamic_fee()
    }

    fn kind(&self) -> TxKind {
        self.inner.kind()
    }

    fn is_create(&self) -> bool {
        self.inner.is_create()
    }

    fn to(&self) -> Option<Address> {
        self.inner.to()
    }

    fn value(&self) -> U256 {
        self.inner.value()
    }

    fn input(&self) -> &Bytes {
        self.inner.input()
    }

    fn access_list(&self) -> Option<&AccessList> {
        self.inner.access_list()
    }

    fn blob_versioned_hashes(&self) -> Option<&[B256]> {
        self.inner.blob_versioned_hashes()
    }

    fn authorization_list(&self) -> Option<&[SignedAuthorization]> {
        self.inner.authorization_list()
    }
}

impl<T: TransactionTrait + Encodable2718> alloy_network_primitives::TransactionResponse
    for Transaction<T>
{
    fn tx_hash(&self) -> alloy_primitives::TxHash {
        self.inner.tx_hash()
    }

    fn block_hash(&self) -> Option<BlockHash> {
        self.inner.block_hash()
    }

    fn block_number(&self) -> Option<u64> {
        self.inner.block_number()
    }

    fn transaction_index(&self) -> Option<u64> {
        self.inner.transaction_index()
    }

    fn from(&self) -> Address {
        self.inner.from()
    }
}

/// Optimism specific transaction fields
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[doc(alias = "OptimismTxFields")]
#[serde(rename_all = "camelCase")]
pub struct OpTransactionFields {
    /// The ETH value to mint on L2
    #[serde(default, skip_serializing_if = "Option::is_none", with = "alloy_serde::quantity::opt")]
    pub mint: Option<u128>,
    /// Hash that uniquely identifies the source of the deposit.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source_hash: Option<B256>,
    /// Field indicating whether the transaction is a system transaction, and therefore
    /// exempt from the L2 gas limit.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub is_system_tx: Option<bool>,
    /// Deposit receipt version for deposit transactions post-canyon
    #[serde(default, skip_serializing_if = "Option::is_none", with = "alloy_serde::quantity::opt")]
    pub deposit_receipt_version: Option<u64>,
}

impl From<OpTransactionFields> for OtherFields {
    fn from(value: OpTransactionFields) -> Self {
        serde_json::to_value(value).unwrap().try_into().unwrap()
    }
}

impl<T> AsRef<T> for Transaction<T> {
    fn as_ref(&self) -> &T {
        self.inner.as_ref()
    }
}

mod tx_serde {
    //! Helper module for serializing and deserializing OP [`Transaction`].
    //!
    //! This is needed because we might need to deserialize the `from` field into both
    //! [`alloy_consensus::transaction::Recovered::signer`] which resides in
    //! [`alloy_rpc_types_eth::Transaction::inner`] and [`op_alloy_consensus::TxDeposit::from`].
    //!
    //! Additionally, we need similar logic for the `gasPrice` field
    use super::*;
    use alloy_consensus::transaction::Recovered;
    use op_alloy_consensus::OpTransaction;
    use serde::de::Error;

    /// Helper struct which will be flattened into the transaction and will only contain `from`
    /// field if inner [`OpTxEnvelope`] did not consume it.
    #[derive(Serialize, Deserialize)]
    struct OptionalFields {
        #[serde(default, skip_serializing_if = "Option::is_none")]
        from: Option<Address>,
        #[serde(
            default,
            rename = "gasPrice",
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )]
        effective_gas_price: Option<u128>,
        #[serde(
            default,
            rename = "nonce",
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )]
        deposit_nonce: Option<u64>,
    }

    #[derive(Serialize, Deserialize)]
    #[serde(rename_all = "camelCase")]
    pub(crate) struct TransactionSerdeHelper<T> {
        #[serde(flatten)]
        inner: T,
        #[serde(default)]
        block_hash: Option<BlockHash>,
        #[serde(default, with = "alloy_serde::quantity::opt")]
        block_number: Option<u64>,
        #[serde(default, with = "alloy_serde::quantity::opt")]
        transaction_index: Option<u64>,
        #[serde(
            default,
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )]
        deposit_receipt_version: Option<u64>,

        #[serde(flatten)]
        other: OptionalFields,
    }

    impl<T: TransactionTrait + OpTransaction> From<Transaction<T>> for TransactionSerdeHelper<T> {
        fn from(value: Transaction<T>) -> Self {
            let Transaction {
                inner:
                    alloy_rpc_types_eth::Transaction {
                        inner,
                        block_hash,
                        block_number,
                        transaction_index,
                        effective_gas_price,
                    },
                deposit_receipt_version,
                deposit_nonce,
            } = value;

            // if inner transaction is a deposit, then don't serialize `from` directly
            let from = if inner.as_deposit().is_some() { None } else { Some(inner.signer()) };

            // if inner transaction has its own `gasPrice` don't serialize it in this struct.
            let effective_gas_price = effective_gas_price.filter(|_| inner.gas_price().is_none());

            Self {
                inner: inner.into_inner(),
                block_hash,
                block_number,
                transaction_index,
                deposit_receipt_version,
                other: OptionalFields { from, effective_gas_price, deposit_nonce },
            }
        }
    }

    impl<T: TransactionTrait + OpTransaction> TryFrom<TransactionSerdeHelper<T>> for Transaction<T> {
        type Error = serde_json::Error;

        fn try_from(value: TransactionSerdeHelper<T>) -> Result<Self, Self::Error> {
            let TransactionSerdeHelper {
                inner,
                block_hash,
                block_number,
                transaction_index,
                deposit_receipt_version,
                other,
            } = value;

            // Try to get `from` field from inner envelope or from `MaybeFrom`, otherwise return
            // error
            let from = if let Some(from) = other.from {
                from
            } else {
                inner
                    .as_deposit()
                    .map(|v| v.from)
                    .ok_or_else(|| serde_json::Error::custom("missing `from` field"))?
            };

            // Only serialize deposit_nonce if inner transaction is deposit to avoid duplicated keys
            let deposit_nonce = other.deposit_nonce.filter(|_| inner.is_deposit());

            let effective_gas_price = other.effective_gas_price.or(inner.gas_price());

            Ok(Self {
                inner: alloy_rpc_types_eth::Transaction {
                    inner: Recovered::new_unchecked(inner, from),
                    block_hash,
                    block_number,
                    transaction_index,
                    effective_gas_price,
                },
                deposit_receipt_version,
                deposit_nonce,
            })
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn can_deserialize_deposit() {
        // cast rpc eth_getTransactionByHash
        // 0xbc9329afac05556497441e2b3ee4c5d4da7ca0b2a4c212c212d0739e94a24df9 --rpc-url optimism
        let rpc_tx = r#"{"blockHash":"0x9d86bb313ebeedf4f9f82bf8a19b426be656a365648a7c089b618771311db9f9","blockNumber":"0x798ad0b","hash":"0xbc9329afac05556497441e2b3ee4c5d4da7ca0b2a4c212c212d0739e94a24df9","transactionIndex":"0x0","type":"0x7e","nonce":"0x152ea95","input":"0x440a5e200000146b000f79c50000000000000003000000006725333f000000000141e287000000000000000000000000000000000000000000000000000000012439ee7e0000000000000000000000000000000000000000000000000000000063f363e973e96e7145ff001c81b9562cba7b6104eeb12a2bc4ab9f07c27d45cd81a986620000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f32985","mint":"0x0","sourceHash":"0x04e9a69416471ead93b02f0c279ab11ca0b635db5c1726a56faf22623bafde52","r":"0x0","s":"0x0","v":"0x0","yParity":"0x0","gas":"0xf4240","from":"0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001","to":"0x4200000000000000000000000000000000000015","depositReceiptVersion":"0x1","value":"0x0","gasPrice":"0x0"}"#;

        let tx = serde_json::from_str::<Transaction>(rpc_tx).unwrap();

        let OpTxEnvelope::Deposit(inner) = tx.as_ref() else {
            panic!("Expected deposit transaction");
        };
        assert_eq!(tx.inner.inner.signer(), inner.from);
        assert_eq!(tx.deposit_nonce, Some(22211221));
        assert_eq!(tx.inner.effective_gas_price, Some(0));

        let deserialized = serde_json::to_value(&tx).unwrap();
        let expected = serde_json::from_str::<serde_json::Value>(rpc_tx).unwrap();
        similar_asserts::assert_eq!(deserialized, expected);
    }
}
