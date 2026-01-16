//! This module contains the eip2930 transaction data type for a span batch.

use crate::{SpanBatchError, SpanDecodingError};
use alloy_consensus::{SignableTransaction, Signed, TxEip2930};
use alloy_eips::eip2930::AccessList;
use alloy_primitives::{Address, Signature, TxKind, U256};
use alloy_rlp::{Bytes, RlpDecodable, RlpEncodable};

/// The transaction data for an EIP-2930 transaction within a span batch.
#[derive(Debug, Clone, PartialEq, Eq, RlpEncodable, RlpDecodable)]
pub struct SpanBatchEip2930TransactionData {
    /// The ETH value of the transaction.
    pub value: U256,
    /// The gas price of the transaction.
    pub gas_price: U256,
    /// Transaction calldata.
    pub data: Bytes,
    /// Access list, used to pre-warm storage slots through static declaration.
    pub access_list: AccessList,
}

impl SpanBatchEip2930TransactionData {
    /// Converts [`SpanBatchEip2930TransactionData`] into a signed [`TxEip2930`].
    pub fn to_signed_tx(
        &self,
        nonce: u64,
        gas: u64,
        to: Option<Address>,
        chain_id: u64,
        signature: Signature,
    ) -> Result<Signed<TxEip2930>, SpanBatchError> {
        let access_list_tx = TxEip2930 {
            chain_id,
            nonce,
            gas_price: u128::from_be_bytes(
                self.gas_price.to_be_bytes::<32>()[16..].try_into().map_err(|_| {
                    SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData)
                })?,
            ),
            gas_limit: gas,
            to: to.map_or(TxKind::Create, TxKind::Call),
            value: self.value,
            input: self.data.clone().into(),
            access_list: self.access_list.clone(),
        };
        let signature_hash = access_list_tx.signature_hash();
        Ok(Signed::new_unchecked(access_list_tx, signature, signature_hash))
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::SpanBatchTransactionData;
    use alloc::vec::Vec;
    use alloy_rlp::{Decodable, Encodable};

    #[test]
    fn encode_eip2930_tx_data_roundtrip() {
        let access_list_tx = SpanBatchEip2930TransactionData {
            value: U256::from(0xFF),
            gas_price: U256::from(0xEE),
            data: Bytes::from(alloc::vec![0x01, 0x02, 0x03]),
            access_list: AccessList::default(),
        };
        let mut encoded_buf = Vec::new();
        SpanBatchTransactionData::Eip2930(access_list_tx.clone()).encode(&mut encoded_buf);

        let decoded = SpanBatchTransactionData::decode(&mut encoded_buf.as_slice()).unwrap();
        let SpanBatchTransactionData::Eip2930(access_list_decoded) = decoded else {
            panic!("Expected SpanBatchEip2930TransactionData, got {decoded:?}");
        };

        assert_eq!(access_list_tx, access_list_decoded);
    }
}
