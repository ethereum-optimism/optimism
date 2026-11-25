//! This module contains the top level span batch transaction data type.

use alloc::vec::Vec;
use alloy_consensus::{Transaction, TxEnvelope};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, Signature, U256};
use alloy_rlp::{Bytes, Decodable, Encodable, Header};
use op_alloy_consensus::OpTxType;

use crate::{
    SpanBatchEip1559TransactionData, SpanBatchEip2930TransactionData,
    SpanBatchEip7702TransactionData, SpanBatchError, SpanBatchLegacyTransactionData,
    SpanDecodingError,
};

/// `PostExec` transaction data within a span batch.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SpanBatchPostExecTransactionData {
    /// RLP-encoded `PostExec` payload bytes.
    pub data: Bytes,
}

impl Encodable for SpanBatchPostExecTransactionData {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        out.put_slice(self.data.as_ref());
    }
}

impl Decodable for SpanBatchPostExecTransactionData {
    fn decode(r: &mut &[u8]) -> Result<Self, alloy_rlp::Error> {
        let header = Header::decode(&mut (**r).as_ref())?;
        let len = header.payload_length + header.length();
        if r.len() < len {
            return Err(alloy_rlp::Error::InputTooShort);
        }
        let data = Bytes::from(r[..len].to_vec());
        *r = &r[len..];
        Ok(Self { data })
    }
}

/// The typed transaction data for a transaction within a span batch.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SpanBatchTransactionData {
    /// Legacy transaction data.
    Legacy(SpanBatchLegacyTransactionData),
    /// EIP-2930 transaction data.
    Eip2930(SpanBatchEip2930TransactionData),
    /// EIP-1559 transaction data.
    Eip1559(SpanBatchEip1559TransactionData),
    /// EIP-7702 transaction data.
    Eip7702(SpanBatchEip7702TransactionData),
    /// `PostExec` transaction data.
    PostExec(SpanBatchPostExecTransactionData),
}

impl Encodable for SpanBatchTransactionData {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        match self {
            Self::Legacy(data) => {
                data.encode(out);
            }
            Self::Eip2930(data) => {
                out.put_u8(u8::from(OpTxType::Eip2930));
                data.encode(out);
            }
            Self::Eip1559(data) => {
                out.put_u8(u8::from(OpTxType::Eip1559));
                data.encode(out);
            }
            Self::Eip7702(data) => {
                out.put_u8(u8::from(OpTxType::Eip7702));
                data.encode(out);
            }
            Self::PostExec(data) => {
                out.put_u8(u8::from(OpTxType::PostExec));
                data.encode(out);
            }
        }
    }
}

impl Decodable for SpanBatchTransactionData {
    fn decode(r: &mut &[u8]) -> Result<Self, alloy_rlp::Error> {
        if !r.is_empty() && r[0] > 0x7F {
            // Legacy transaction
            return Ok(Self::Legacy(SpanBatchLegacyTransactionData::decode(r)?));
        }
        // Non-legacy transaction (EIP-2718 envelope encoding)
        Self::decode_typed(r)
    }
}

impl TryFrom<&TxEnvelope> for SpanBatchTransactionData {
    type Error = SpanBatchError;

    fn try_from(tx_envelope: &TxEnvelope) -> Result<Self, Self::Error> {
        match tx_envelope {
            TxEnvelope::Legacy(s) => {
                let s = s.tx();
                Ok(Self::Legacy(SpanBatchLegacyTransactionData {
                    value: s.value,
                    gas_price: U256::from(s.gas_price),
                    data: Bytes::from(s.input().to_vec()),
                }))
            }
            TxEnvelope::Eip2930(s) => {
                let s = s.tx();
                Ok(Self::Eip2930(SpanBatchEip2930TransactionData {
                    value: s.value,
                    gas_price: U256::from(s.gas_price),
                    data: Bytes::from(s.input().to_vec()),
                    access_list: s.access_list.clone(),
                }))
            }
            TxEnvelope::Eip1559(s) => {
                let s = s.tx();
                Ok(Self::Eip1559(SpanBatchEip1559TransactionData {
                    value: s.value,
                    max_fee_per_gas: U256::from(s.max_fee_per_gas),
                    max_priority_fee_per_gas: U256::from(s.max_priority_fee_per_gas),
                    data: Bytes::from(s.input().to_vec()),
                    access_list: s.access_list.clone(),
                }))
            }
            TxEnvelope::Eip7702(s) => {
                let s = s.tx();
                Ok(Self::Eip7702(SpanBatchEip7702TransactionData {
                    value: s.value,
                    max_fee_per_gas: U256::from(s.max_fee_per_gas),
                    max_priority_fee_per_gas: U256::from(s.max_priority_fee_per_gas),
                    data: Bytes::from(s.input().to_vec()),
                    access_list: s.access_list.clone(),
                    authorization_list: s.authorization_list.clone(),
                }))
            }
            _ => Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionType)),
        }
    }
}

impl SpanBatchTransactionData {
    /// Returns the transaction type of the [`SpanBatchTransactionData`].
    pub const fn tx_type(&self) -> OpTxType {
        match self {
            Self::Legacy(_) => OpTxType::Legacy,
            Self::Eip2930(_) => OpTxType::Eip2930,
            Self::Eip1559(_) => OpTxType::Eip1559,
            Self::Eip7702(_) => OpTxType::Eip7702,
            Self::PostExec(_) => OpTxType::PostExec,
        }
    }

    /// Decodes a typed transaction into a [`SpanBatchTransactionData`] from a byte slice.
    pub fn decode_typed(b: &[u8]) -> Result<Self, alloy_rlp::Error> {
        if b.len() <= 1 {
            return Err(alloy_rlp::Error::Custom("Invalid transaction data"));
        }

        let tx_type = OpTxType::try_from(b[0])
            .map_err(|_| alloy_rlp::Error::Custom("Invalid transaction type"))?;
        match tx_type {
            OpTxType::Eip2930 => {
                Ok(Self::Eip2930(SpanBatchEip2930TransactionData::decode(&mut &b[1..])?))
            }
            OpTxType::Eip1559 => {
                Ok(Self::Eip1559(SpanBatchEip1559TransactionData::decode(&mut &b[1..])?))
            }
            OpTxType::Eip7702 => {
                Ok(Self::Eip7702(SpanBatchEip7702TransactionData::decode(&mut &b[1..])?))
            }
            OpTxType::PostExec => {
                Ok(Self::PostExec(SpanBatchPostExecTransactionData::decode(&mut &b[1..])?))
            }
            OpTxType::Legacy | OpTxType::Deposit => {
                Err(alloy_rlp::Error::Custom("Invalid transaction type"))
            }
        }
    }

    /// Converts the [`SpanBatchTransactionData`] into a signed transaction as [`TxEnvelope`].
    pub fn to_signed_tx(
        &self,
        nonce: u64,
        gas: u64,
        to: Option<Address>,
        chain_id: u64,
        signature: Signature,
        is_protected: bool,
    ) -> Result<TxEnvelope, SpanBatchError> {
        Ok(match self {
            Self::Legacy(data) => TxEnvelope::Legacy(data.to_signed_tx(
                nonce,
                gas,
                to,
                chain_id,
                signature,
                is_protected,
            )?),
            Self::Eip2930(data) => {
                TxEnvelope::Eip2930(data.to_signed_tx(nonce, gas, to, chain_id, signature)?)
            }
            Self::Eip1559(data) => {
                TxEnvelope::Eip1559(data.to_signed_tx(nonce, gas, to, chain_id, signature)?)
            }
            Self::Eip7702(data) => {
                let Some(addr) = to else {
                    return Err(SpanBatchError::Decoding(
                        SpanDecodingError::InvalidTransactionData,
                    ));
                };
                TxEnvelope::Eip7702(data.to_signed_tx(nonce, gas, addr, chain_id, signature)?)
            }
            Self::PostExec(_) => {
                return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
            }
        })
    }

    /// Encodes the transaction as a full EIP-2718 transaction.
    pub fn to_full_tx_bytes(
        &self,
        nonce: u64,
        gas: u64,
        to: Option<Address>,
        chain_id: u64,
        signature: Signature,
        is_protected: bool,
    ) -> Result<Vec<u8>, SpanBatchError> {
        match self {
            Self::PostExec(data) => {
                let mut out = Vec::with_capacity(1 + data.data.len());
                out.push(u8::from(OpTxType::PostExec));
                out.extend_from_slice(data.data.as_ref());
                Ok(out)
            }
            _ => {
                let tx = self.to_signed_tx(nonce, gas, to, chain_id, signature, is_protected)?;
                let mut out = Vec::new();
                tx.encode_2718(&mut out);
                Ok(out)
            }
        }
    }
}
