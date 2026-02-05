//! Compact codec implementations for OP transaction types.
//!
//! This module provides newtype wrappers around `op-alloy-consensus` transaction types
//! with `Compact` trait implementations for database storage.

use alloc::vec::Vec;
use alloy_consensus::{
    Transaction, constants::EIP7702_TX_TYPE_ID, Signed, TxEip1559, TxEip2930, TxEip7702, TxLegacy,
};
use alloy_primitives::{Address, Bytes, Sealed, Signature, TxKind, B256, U256};
use bytes::{Buf, BufMut};
use core::ops::{Deref, DerefMut};
use op_alloy_consensus::{OpTxEnvelope, OpTypedTransaction};
use reth_codecs::{
    Compact,
    txtype::{
        COMPACT_EXTENDED_IDENTIFIER_FLAG, COMPACT_IDENTIFIER_EIP1559, COMPACT_IDENTIFIER_EIP2930,
        COMPACT_IDENTIFIER_LEGACY,
    },
};

// ===== TxDeposit helper for derive =====

/// Deposit transactions, also known as deposits are initiated on L1, and executed on L2.
///
/// This is a helper type to use derive on it instead of manually managing `bitfield`.
///
/// By deriving `Compact` here, any future changes or enhancements to the `Compact` derive
/// will automatically apply to this type.
///
/// Notice: Make sure this struct is 1:1 with [`op_alloy_consensus::TxDeposit`]
#[derive(Debug, Clone, PartialEq, Eq, Hash, Default, Compact)]
#[cfg_attr(any(test, feature = "arbitrary"), derive(arbitrary::Arbitrary))]
#[reth_codecs(crate = "reth_codecs")]
pub(crate) struct TxDepositCompact {
    source_hash: B256,
    from: Address,
    to: TxKind,
    mint: Option<u128>,
    value: U256,
    gas_limit: u64,
    is_system_transaction: bool,
    input: Bytes,
}

// ===== OpDeposit newtype =====

/// Newtype wrapper around [`op_alloy_consensus::TxDeposit`] for `Compact` implementation.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
#[repr(transparent)]
pub struct OpDeposit(pub op_alloy_consensus::TxDeposit);

impl Deref for OpDeposit {
    type Target = op_alloy_consensus::TxDeposit;
    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for OpDeposit {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl From<op_alloy_consensus::TxDeposit> for OpDeposit {
    fn from(tx: op_alloy_consensus::TxDeposit) -> Self {
        Self(tx)
    }
}

impl From<OpDeposit> for op_alloy_consensus::TxDeposit {
    fn from(tx: OpDeposit) -> Self {
        tx.0
    }
}

impl Compact for OpDeposit {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        let tx = TxDepositCompact {
            source_hash: self.0.source_hash,
            from: self.0.from,
            to: self.0.to,
            mint: match self.0.mint {
                0 => None,
                v => Some(v),
            },
            value: self.0.value,
            gas_limit: self.0.gas_limit,
            is_system_transaction: self.0.is_system_transaction,
            input: self.0.input.clone(),
        };
        tx.to_compact(buf)
    }

    fn from_compact(buf: &[u8], len: usize) -> (Self, &[u8]) {
        let (tx, remaining) = TxDepositCompact::from_compact(buf, len);
        let alloy_tx = op_alloy_consensus::TxDeposit {
            source_hash: tx.source_hash,
            from: tx.from,
            to: tx.to,
            mint: tx.mint.unwrap_or_default(),
            value: tx.value,
            gas_limit: tx.gas_limit,
            is_system_transaction: tx.is_system_transaction,
            input: tx.input,
        };
        (Self(alloy_tx), remaining)
    }
}

// ===== OpTxTy newtype =====

/// Newtype wrapper around [`op_alloy_consensus::OpTxType`] for `Compact` implementation.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[repr(transparent)]
pub struct OpTxTy(pub op_alloy_consensus::OpTxType);

impl Deref for OpTxTy {
    type Target = op_alloy_consensus::OpTxType;
    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl From<op_alloy_consensus::OpTxType> for OpTxTy {
    fn from(ty: op_alloy_consensus::OpTxType) -> Self {
        Self(ty)
    }
}

impl From<OpTxTy> for op_alloy_consensus::OpTxType {
    fn from(ty: OpTxTy) -> Self {
        ty.0
    }
}

impl Compact for OpTxTy {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        use op_alloy_consensus::OpTxType;
        match self.0 {
            OpTxType::Legacy => COMPACT_IDENTIFIER_LEGACY,
            OpTxType::Eip2930 => COMPACT_IDENTIFIER_EIP2930,
            OpTxType::Eip1559 => COMPACT_IDENTIFIER_EIP1559,
            OpTxType::Eip7702 => {
                buf.put_u8(EIP7702_TX_TYPE_ID);
                COMPACT_EXTENDED_IDENTIFIER_FLAG
            }
            OpTxType::Deposit => {
                buf.put_u8(op_alloy_consensus::DEPOSIT_TX_TYPE_ID);
                COMPACT_EXTENDED_IDENTIFIER_FLAG
            }
        }
    }

    fn from_compact(mut buf: &[u8], identifier: usize) -> (Self, &[u8]) {
        use op_alloy_consensus::OpTxType;
        (
            Self(match identifier {
                COMPACT_IDENTIFIER_LEGACY => OpTxType::Legacy,
                COMPACT_IDENTIFIER_EIP2930 => OpTxType::Eip2930,
                COMPACT_IDENTIFIER_EIP1559 => OpTxType::Eip1559,
                COMPACT_EXTENDED_IDENTIFIER_FLAG => {
                    let extended_identifier = buf.get_u8();
                    match extended_identifier {
                        EIP7702_TX_TYPE_ID => OpTxType::Eip7702,
                        op_alloy_consensus::DEPOSIT_TX_TYPE_ID => OpTxType::Deposit,
                        _ => panic!("Unsupported OpTxType identifier: {extended_identifier}"),
                    }
                }
                _ => panic!("Unknown identifier for TxType: {identifier}"),
            }),
            buf,
        )
    }
}

// ===== OpTypedTx newtype =====

/// Newtype wrapper around [`op_alloy_consensus::OpTypedTransaction`] for `Compact`
/// implementation.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
#[repr(transparent)]
pub struct OpTypedTx(pub OpTypedTransaction);

impl Deref for OpTypedTx {
    type Target = OpTypedTransaction;
    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for OpTypedTx {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl From<OpTypedTransaction> for OpTypedTx {
    fn from(tx: OpTypedTransaction) -> Self {
        Self(tx)
    }
}

impl From<OpTypedTx> for OpTypedTransaction {
    fn from(tx: OpTypedTx) -> Self {
        tx.0
    }
}

#[cfg(any(test, feature = "arbitrary"))]
impl<'a> arbitrary::Arbitrary<'a> for OpTypedTx {
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        OpTypedTransaction::arbitrary(u).map(Self)
    }
}

impl Compact for OpTypedTx {
    fn to_compact<B>(&self, out: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        let identifier = OpTxTy(self.0.tx_type()).to_compact(out);
        match &self.0 {
            OpTypedTransaction::Legacy(tx) => tx.to_compact(out),
            OpTypedTransaction::Eip2930(tx) => tx.to_compact(out),
            OpTypedTransaction::Eip1559(tx) => tx.to_compact(out),
            OpTypedTransaction::Eip7702(tx) => tx.to_compact(out),
            OpTypedTransaction::Deposit(tx) => OpDeposit(tx.clone()).to_compact(out),
        };
        identifier
    }

    fn from_compact(buf: &[u8], identifier: usize) -> (Self, &[u8]) {
        use op_alloy_consensus::OpTxType;
        let (tx_type, buf) = OpTxTy::from_compact(buf, identifier);
        match tx_type.0 {
            OpTxType::Legacy => {
                let (tx, buf) = Compact::from_compact(buf, buf.len());
                (Self(OpTypedTransaction::Legacy(tx)), buf)
            }
            OpTxType::Eip2930 => {
                let (tx, buf) = Compact::from_compact(buf, buf.len());
                (Self(OpTypedTransaction::Eip2930(tx)), buf)
            }
            OpTxType::Eip1559 => {
                let (tx, buf) = Compact::from_compact(buf, buf.len());
                (Self(OpTypedTransaction::Eip1559(tx)), buf)
            }
            OpTxType::Eip7702 => {
                let (tx, buf) = Compact::from_compact(buf, buf.len());
                (Self(OpTypedTransaction::Eip7702(tx)), buf)
            }
            OpTxType::Deposit => {
                let (tx, buf) = OpDeposit::from_compact(buf, buf.len());
                (Self(OpTypedTransaction::Deposit(tx.0)), buf)
            }
        }
    }
}

// ===== OpTx newtype =====

/// Newtype wrapper around [`OpTxEnvelope`] for `Compact` implementation.
///
/// Uses the same compact encoding format as the bare `OpTxEnvelope` implementation in
/// `reth-codecs`: a leading byte of bitflags (IsCompressed, TxType, Signature), followed
/// by the signature and the (optionally zstd-compressed) transaction body.
#[derive(Debug, Clone, PartialEq, Eq)]
#[repr(transparent)]
pub struct OpTx(pub OpTxEnvelope);

impl Deref for OpTx {
    type Target = OpTxEnvelope;
    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for OpTx {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl From<OpTxEnvelope> for OpTx {
    fn from(tx: OpTxEnvelope) -> Self {
        Self(tx)
    }
}

impl From<OpTx> for OpTxEnvelope {
    fn from(tx: OpTx) -> Self {
        tx.0
    }
}

const DEPOSIT_SIGNATURE: Signature = Signature::new(U256::ZERO, U256::ZERO, false);

/// Returns the signature for this envelope variant.
fn envelope_signature(tx: &OpTxEnvelope) -> &Signature {
    match tx {
        OpTxEnvelope::Legacy(tx) => tx.signature(),
        OpTxEnvelope::Eip2930(tx) => tx.signature(),
        OpTxEnvelope::Eip1559(tx) => tx.signature(),
        OpTxEnvelope::Eip7702(tx) => tx.signature(),
        OpTxEnvelope::Deposit(_) => &DEPOSIT_SIGNATURE,
    }
}

/// Serializes the inner transaction body (without signature or type) using Compact encoding.
fn tx_to_compact(tx: &OpTxEnvelope, buf: &mut (impl BufMut + AsMut<[u8]>)) {
    match tx {
        OpTxEnvelope::Legacy(tx) => {
            tx.tx().to_compact(buf);
        }
        OpTxEnvelope::Eip2930(tx) => {
            tx.tx().to_compact(buf);
        }
        OpTxEnvelope::Eip1559(tx) => {
            tx.tx().to_compact(buf);
        }
        OpTxEnvelope::Eip7702(tx) => {
            tx.tx().to_compact(buf);
        }
        OpTxEnvelope::Deposit(tx) => {
            OpDeposit((**tx).clone()).to_compact(buf);
        }
    };
}

/// Deserializes a transaction from the given tx type, buffer and signature.
fn tx_from_compact(
    buf: &[u8],
    tx_type: OpTxTy,
    signature: Signature,
) -> (OpTxEnvelope, &[u8]) {
    use op_alloy_consensus::OpTxType;
    match tx_type.0 {
        OpTxType::Legacy => {
            let (tx, buf) = TxLegacy::from_compact(buf, buf.len());
            let tx = Signed::new_unhashed(tx, signature);
            (OpTxEnvelope::Legacy(tx), buf)
        }
        OpTxType::Eip2930 => {
            let (tx, buf) = TxEip2930::from_compact(buf, buf.len());
            let tx = Signed::new_unhashed(tx, signature);
            (OpTxEnvelope::Eip2930(tx), buf)
        }
        OpTxType::Eip1559 => {
            let (tx, buf) = TxEip1559::from_compact(buf, buf.len());
            let tx = Signed::new_unhashed(tx, signature);
            (OpTxEnvelope::Eip1559(tx), buf)
        }
        OpTxType::Eip7702 => {
            let (tx, buf) = TxEip7702::from_compact(buf, buf.len());
            let tx = Signed::new_unhashed(tx, signature);
            (OpTxEnvelope::Eip7702(tx), buf)
        }
        OpTxType::Deposit => {
            let (tx, buf) = OpDeposit::from_compact(buf, buf.len());
            let tx = Sealed::new(tx.0);
            (OpTxEnvelope::Deposit(tx), buf)
        }
    }
}

impl Compact for OpTx {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: BufMut + AsMut<[u8]>,
    {
        let start = buf.as_mut().len();

        // Placeholder for bitflags.
        // The first byte uses 4 bits as flags: IsCompressed[1bit], TxType[2bits], Signature[1bit]
        buf.put_u8(0);

        let sig_bit = envelope_signature(&self.0).to_compact(buf) as u8;
        let zstd_bit = self.0.input().len() >= 32;

        let tx_bits = if zstd_bit {
            // compress the tx prefixed with txtype
            let mut tx_buf = Vec::with_capacity(256);
            let tx_bits = OpTxTy(OpTxEnvelope::tx_type(&self.0)).to_compact(&mut tx_buf) as u8;
            tx_to_compact(&self.0, &mut tx_buf);

            buf.put_slice(
                &reth_zstd_compressors::TRANSACTION_COMPRESSOR.with(|compressor| {
                    compressor.borrow_mut().compress(&tx_buf)
                })
                .expect("Failed to compress"),
            );
            tx_bits
        } else {
            let tx_bits =
                OpTxTy(OpTxEnvelope::tx_type(&self.0)).to_compact(buf) as u8;
            tx_to_compact(&self.0, buf);
            tx_bits
        };

        let flags = sig_bit | (tx_bits << 1) | ((zstd_bit as u8) << 3);
        buf.as_mut()[start] = flags;

        buf.as_mut().len() - start
    }

    fn from_compact(mut buf: &[u8], _len: usize) -> (Self, &[u8]) {
        let flags = buf.get_u8() as usize;

        let sig_bit = flags & 1;
        let tx_bits = (flags & 0b110) >> 1;
        let zstd_bit = flags >> 3;

        let (signature, buf) = Signature::from_compact(buf, sig_bit);

        let (transaction, buf) = if zstd_bit != 0 {
            reth_zstd_compressors::TRANSACTION_DECOMPRESSOR.with(|decompressor| {
                let mut decompressor = decompressor.borrow_mut();
                let decompressed = decompressor.decompress(buf);
                let (tx_type, tx_buf) = OpTxTy::from_compact(decompressed, tx_bits);
                let (tx, _) = tx_from_compact(tx_buf, tx_type, signature);
                (tx, buf)
            })
        } else {
            let (tx_type, buf) = OpTxTy::from_compact(buf, tx_bits);
            tx_from_compact(buf, tx_type, signature)
        };

        (Self(transaction), buf)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_codecs::generate_tests;

    generate_tests!(#[reth_codecs, compact] OpTypedTx, OpTypedTxTests);
}
