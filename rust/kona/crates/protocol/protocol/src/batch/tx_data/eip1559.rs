//! This module contains the eip1559 transaction data type for a span batch.

use crate::{SpanBatchError, SpanDecodingError};
use alloy_consensus::{SignableTransaction, Signed, TxEip1559};
use alloy_eips::eip2930::AccessList;
use alloy_primitives::{Address, Signature, TxKind, U256};
use alloy_rlp::{Bytes, RlpDecodable, RlpEncodable};

/// The transaction data for an EIP-1559 transaction within a span batch.
#[derive(Debug, Clone, PartialEq, Eq, RlpEncodable, RlpDecodable)]
pub struct SpanBatchEip1559TransactionData {
    /// The ETH value of the transaction.
    pub value: U256,
    /// Maximum priority fee per gas.
    pub max_priority_fee_per_gas: U256,
    /// Maximum fee per gas.
    pub max_fee_per_gas: U256,
    /// Transaction calldata.
    pub data: Bytes,
    /// Access list, used to pre-warm storage slots through static declaration.
    pub access_list: AccessList,
}

impl SpanBatchEip1559TransactionData {
    /// Converts [`SpanBatchEip1559TransactionData`] into a signed [`TxEip1559`].
    pub fn to_signed_tx(
        &self,
        nonce: u64,
        gas: u64,
        to: Option<Address>,
        chain_id: u64,
        signature: Signature,
    ) -> Result<Signed<TxEip1559>, SpanBatchError> {
        let eip1559_tx = TxEip1559 {
            chain_id,
            nonce,
            max_fee_per_gas: u128::from_be_bytes(
                self.max_fee_per_gas.to_be_bytes::<32>()[16..].try_into().map_err(|_| {
                    SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData)
                })?,
            ),
            max_priority_fee_per_gas: u128::from_be_bytes(
                self.max_priority_fee_per_gas.to_be_bytes::<32>()[16..].try_into().map_err(
                    |_| SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData),
                )?,
            ),
            gas_limit: gas,
            to: to.map_or(TxKind::Create, TxKind::Call),
            value: self.value,
            input: self.data.clone().into(),
            access_list: self.access_list.clone(),
        };
        let signature_hash = eip1559_tx.signature_hash();
        Ok(Signed::new_unchecked(eip1559_tx, signature, signature_hash))
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::SpanBatchTransactionData;
    use alloc::vec::Vec;
    use alloy_rlp::{Decodable, Encodable};

    #[test]
    fn encode_eip1559_tx_data_roundtrip() {
        let variable_fee_tx = SpanBatchEip1559TransactionData {
            value: U256::from(0xFF),
            max_fee_per_gas: U256::from(0xEE),
            max_priority_fee_per_gas: U256::from(0xDD),
            data: Bytes::from(alloc::vec![0x01, 0x02, 0x03]),
            access_list: AccessList::default(),
        };
        let mut encoded_buf = Vec::new();
        SpanBatchTransactionData::Eip1559(variable_fee_tx.clone()).encode(&mut encoded_buf);

        let decoded = SpanBatchTransactionData::decode(&mut encoded_buf.as_slice()).unwrap();
        let SpanBatchTransactionData::Eip1559(variable_fee_decoded) = decoded else {
            panic!("Expected SpanBatchEip1559TransactionData, got {decoded:?}");
        };

        assert_eq!(variable_fee_tx, variable_fee_decoded);
    }
}
