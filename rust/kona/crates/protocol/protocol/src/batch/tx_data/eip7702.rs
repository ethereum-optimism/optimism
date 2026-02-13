//! This module contains the eip7702 transaction data type for a span batch.

use crate::SpanBatchError;
use alloc::vec::Vec;
use alloy_consensus::{SignableTransaction, Signed, TxEip7702};
use alloy_eips::{eip2930::AccessList, eip7702::SignedAuthorization};
use alloy_primitives::{Address, Signature, U256};
use alloy_rlp::{Bytes, RlpDecodable, RlpEncodable};

/// The transaction data for an EIP-7702 transaction within a span batch.
#[derive(Debug, Clone, PartialEq, Eq, RlpEncodable, RlpDecodable)]
pub struct SpanBatchEip7702TransactionData {
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
    /// Authorization list, used to allow a signer to delegate code to a contract
    pub authorization_list: Vec<SignedAuthorization>,
}

impl SpanBatchEip7702TransactionData {
    /// Converts [`SpanBatchEip7702TransactionData`] into a signed [`TxEip7702`].
    pub fn to_signed_tx(
        &self,
        nonce: u64,
        gas: u64,
        to: Address,
        chain_id: u64,
        signature: Signature,
    ) -> Result<Signed<TxEip7702>, SpanBatchError> {
        // SAFETY: A U256 as be bytes is always 32 bytes long.
        let mut max_fee_per_gas = [0u8; 16];
        max_fee_per_gas.copy_from_slice(&self.max_fee_per_gas.to_be_bytes::<32>()[16..]);
        let max_fee_per_gas = u128::from_be_bytes(max_fee_per_gas);

        // SAFETY: A U256 as be bytes is always 32 bytes long.
        let mut max_priority_fee_per_gas = [0u8; 16];
        max_priority_fee_per_gas
            .copy_from_slice(&self.max_priority_fee_per_gas.to_be_bytes::<32>()[16..]);
        let max_priority_fee_per_gas = u128::from_be_bytes(max_priority_fee_per_gas);

        let eip7702_tx = TxEip7702 {
            chain_id,
            nonce,
            max_fee_per_gas,
            max_priority_fee_per_gas,
            gas_limit: gas,
            to,
            value: self.value,
            input: self.data.clone().into(),
            access_list: self.access_list.clone(),
            authorization_list: self.authorization_list.clone(),
        };
        let signature_hash = eip7702_tx.signature_hash();
        Ok(Signed::new_unchecked(eip7702_tx, signature, signature_hash))
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::SpanBatchTransactionData;
    use alloc::{vec, vec::Vec};
    use alloy_rlp::{Decodable, Encodable};
    use alloy_rpc_types_eth::Authorization;

    #[test]
    fn test_eip7702_to_signed_tx() {
        let authorization = Authorization {
            chain_id: U256::from(0x01),
            address: Address::left_padding_from(&[0x01, 0x02, 0x03]),
            nonce: 2,
        };
        let signature = Signature::test_signature();
        let arb_authorization: SignedAuthorization = authorization.into_signed(signature);

        let variable_fee_tx = SpanBatchEip7702TransactionData {
            value: U256::from(0xFF),
            max_fee_per_gas: U256::from(0xEE),
            max_priority_fee_per_gas: U256::from(0xDD),
            data: Bytes::from(alloc::vec![0x01, 0x02, 0x03]),
            access_list: AccessList::default(),
            authorization_list: vec![arb_authorization],
        };

        let signed_tx = variable_fee_tx
            .to_signed_tx(0, 0, Address::ZERO, 0, Signature::test_signature())
            .unwrap();

        assert_eq!(*signed_tx.signature(), Signature::test_signature());
    }

    #[test]
    fn encode_eip7702_tx_data_roundtrip() {
        let authorization = Authorization {
            chain_id: U256::from(0x01),
            address: Address::left_padding_from(&[0x01, 0x02, 0x03]),
            nonce: 2,
        };
        let signature = Signature::test_signature();
        let arb_authorization: SignedAuthorization = authorization.into_signed(signature);

        let variable_fee_tx = SpanBatchEip7702TransactionData {
            value: U256::from(0xFF),
            max_fee_per_gas: U256::from(0xEE),
            max_priority_fee_per_gas: U256::from(0xDD),
            data: Bytes::from(alloc::vec![0x01, 0x02, 0x03]),
            access_list: AccessList::default(),
            authorization_list: vec![arb_authorization],
        };

        let mut encoded_buf = Vec::new();
        SpanBatchTransactionData::Eip7702(variable_fee_tx.clone()).encode(&mut encoded_buf);

        let decoded = SpanBatchTransactionData::decode(&mut encoded_buf.as_slice()).unwrap();
        let SpanBatchTransactionData::Eip7702(variable_fee_decoded) = decoded else {
            panic!("Expected SpanBatchEip7702TransactionData, got {decoded:?}");
        };

        assert_eq!(variable_fee_tx, variable_fee_decoded);
    }
}
