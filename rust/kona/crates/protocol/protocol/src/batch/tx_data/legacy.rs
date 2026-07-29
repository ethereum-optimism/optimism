//! This module contains the legacy transaction data type for a span batch.

use crate::{SpanBatchError, SpanDecodingError};
use alloy_consensus::{SignableTransaction, Signed, TxLegacy};
use alloy_primitives::{Address, Signature, TxKind, U256};
use alloy_rlp::{Bytes, RlpDecodable, RlpEncodable};

/// The transaction data for a legacy transaction within a span batch.
#[derive(Debug, Clone, PartialEq, Eq, RlpEncodable, RlpDecodable)]
pub struct SpanBatchLegacyTransactionData {
    /// The ETH value of the transaction.
    pub value: U256,
    /// The gas price of the transaction.
    pub gas_price: U256,
    /// Transaction calldata.
    pub data: Bytes,
}

impl SpanBatchLegacyTransactionData {
    /// Converts [`SpanBatchLegacyTransactionData`] into a signed [`TxLegacy`].
    pub fn to_signed_tx(
        &self,
        nonce: u64,
        gas: u64,
        to: Option<Address>,
        chain_id: u64,
        signature: Signature,
        is_protected: bool,
    ) -> Result<Signed<TxLegacy>, SpanBatchError> {
        let legacy_tx = TxLegacy {
            chain_id: is_protected.then_some(chain_id),
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
        };
        let signature_hash = legacy_tx.signature_hash();
        Ok(Signed::new_unchecked(legacy_tx, signature, signature_hash))
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::SpanBatchTransactionData;
    use alloc::vec::Vec;
    use alloy_rlp::{Decodable, Encodable as _};

    #[test]
    fn encode_legacy_tx_data_roundtrip() {
        let legacy_tx = SpanBatchLegacyTransactionData {
            value: U256::from(0xFF),
            gas_price: U256::from(0xEE),
            data: Bytes::from(alloc::vec![0x01, 0x02, 0x03]),
        };

        let mut encoded_buf = Vec::new();
        SpanBatchTransactionData::Legacy(legacy_tx.clone()).encode(&mut encoded_buf);

        let decoded = SpanBatchTransactionData::decode(&mut encoded_buf.as_slice()).unwrap();
        let SpanBatchTransactionData::Legacy(legacy_decoded) = decoded else {
            panic!("Expected SpanBatchLegacyTransactionData, got {decoded:?}");
        };

        assert_eq!(legacy_tx, legacy_decoded);
    }
}
