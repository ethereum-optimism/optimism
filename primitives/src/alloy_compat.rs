//! Common conversions from alloy types.

use crate::OpTransactionSigned;
use alloy_consensus::TxEnvelope;
use alloy_network::{AnyRpcTransaction, AnyTxEnvelope};
use alloy_rpc_types_eth::{ConversionError, Transaction as AlloyRpcTransaction};
use alloy_serde::WithOtherFields;
use op_alloy_consensus::{OpTypedTransaction, TxDeposit};

impl TryFrom<AnyRpcTransaction> for OpTransactionSigned {
    type Error = ConversionError;

    fn try_from(tx: AnyRpcTransaction) -> Result<Self, Self::Error> {
        let WithOtherFields { inner: AlloyRpcTransaction { inner, .. }, other: _ } = tx.0;
        let from = inner.signer();

        let (transaction, signature, hash) = match inner.into_inner() {
            AnyTxEnvelope::Ethereum(TxEnvelope::Legacy(tx)) => {
                let (tx, signature, hash) = tx.into_parts();
                (OpTypedTransaction::Legacy(tx), signature, hash)
            }
            AnyTxEnvelope::Ethereum(TxEnvelope::Eip2930(tx)) => {
                let (tx, signature, hash) = tx.into_parts();
                (OpTypedTransaction::Eip2930(tx), signature, hash)
            }
            AnyTxEnvelope::Ethereum(TxEnvelope::Eip1559(tx)) => {
                let (tx, signature, hash) = tx.into_parts();
                (OpTypedTransaction::Eip1559(tx), signature, hash)
            }
            AnyTxEnvelope::Ethereum(TxEnvelope::Eip7702(tx)) => {
                let (tx, signature, hash) = tx.into_parts();
                (OpTypedTransaction::Eip7702(tx), signature, hash)
            }
            AnyTxEnvelope::Unknown(mut tx) => {
                // Re-insert `from` field which was consumed by outer `Transaction`.
                // Ref hack in op-alloy <https://github.com/alloy-rs/op-alloy/blob/7d50b698631dd73f8d20f9f60ee78cd0597dc278/crates/rpc-types/src/transaction.rs#L236-L237>
                tx.inner
                    .fields
                    .insert_value("from".to_string(), from)
                    .map_err(|err| ConversionError::Custom(err.to_string()))?;
                let hash = tx.hash;
                (OpTypedTransaction::Deposit(tx.try_into()?), TxDeposit::signature(), hash)
            }
            _ => return Err(ConversionError::Custom("unknown transaction type".to_string())),
        };

        Ok(Self::new(transaction, signature, hash))
    }
}

impl<T> From<AlloyRpcTransaction<T>> for OpTransactionSigned
where
    Self: From<T>,
{
    fn from(value: AlloyRpcTransaction<T>) -> Self {
        value.inner.into_inner().into()
    }
}

#[cfg(test)]
mod tests {
    use alloy_network::AnyRpcTransaction;

    use crate::OpTransactionSigned;

    #[test]
    fn test_tx_deposit() {
        let json = r#"{
            "hash": "0x3f44a72b1faf70be7295183f1f30cfb51ede92d7c44441ca80c9437a6a22e5a5",
            "type": "0x7e",
            "depositReceiptVersion": "0x1",
            "gas": "0x77f2e",
            "gasPrice": "0x0",
            "input": "0xd764ad0b000100000000000000000000000000000000000000000000000000000005dacf0000000000000000000000003154cf16ccdb4c6d922629664174b904d80f2c3500000000000000000000000042000000000000000000000000000000000000100000000000000000000000000000000000000000000000000001c6bf526340000000000000000000000000000000000000000000000000000000000000030d4000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000c41635f5fd000000000000000000000000212241ad6a6a0a7553c4290f7bc517c1041df0be000000000000000000000000d153f571d0a5a07133114bfe623d4d71e03ea5d70000000000000000000000000000000000000000000000000001c6bf5263400000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000018307837333735373036353732363237323639363436373635000000000000000000000000000000000000000000000000000000000000000000000000",
            "mint": "0x1c6bf52634000",
            "nonce": "0x5dacf",
            "r": "0x0",
            "s": "0x0",
            "sourceHash": "0x26eb0df3ebdb8eb11892a336b798948ec846c28200862a67aa32cfd57ca7da4f",
            "to": "0x4200000000000000000000000000000000000007",
            "v": "0x0",
            "value": "0x1c6bf52634000",
            "blockHash": "0x0d7f8b9def6f5d3ba2cbeee2e31e730da81e2c474fa8c3c9e8d0e6b96e37d182",
            "blockNumber": "0x1966297",
            "transactionIndex": "0x1",
            "from": "0x977f82a600a1414e583f7f13623f1ac5d58b1c0b"
        }"#;
        let tx: AnyRpcTransaction = serde_json::from_str(json).unwrap();
        OpTransactionSigned::try_from(tx).unwrap();
    }
}
