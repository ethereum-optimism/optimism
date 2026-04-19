//! Compact codec implementations for OP Stack consensus types.
//!
//! Ported from reth v1.11.3 (`d6324d63e`), where they lived behind the `op` feature:
//! - Transaction codecs: `crates/storage/codecs/src/alloy/transaction/optimism.rs`
//! - Receipt codecs: `crates/storage/codecs/src/alloy/optimism.rs`
//!
//! Differences from upstream:
//! - `CompactOpReceipt` uses `Vec<Log>` instead of `Cow<'a, Vec<Log>>` because the crates.io
//!   `reth-codecs-derive` macro doesn't support lifetime parameters. The wire format is identical;
//!   only serialization performance differs (clone vs borrow).
//! - `Compress`/`Decompress` impls for `OpTxEnvelope` and `OpReceipt` are added here since they
//!   were previously provided by reth's in-tree codecs crate.

use crate::{
    OpReceipt, OpTxEnvelope, OpTxType, OpTypedTransaction, POST_EXEC_TX_TYPE_ID, TxDeposit,
    TxPostExec,
};
use alloc::vec::Vec;
use alloy_consensus::{Receipt, Signed, Transaction};
use alloy_primitives::{Address, B256, Bytes, Log, Signature, TxKind, U256};
use alloy_rlp::Decodable;
use reth_codecs::{
    Compact,
    alloy::transaction::{CompactEnvelope, Envelope, FromTxCompact, ToTxCompact},
    txtype::*,
};

// --- OpTxType ---

impl Compact for OpTxType {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        match self {
            Self::Legacy => COMPACT_IDENTIFIER_LEGACY,
            Self::Eip2930 => COMPACT_IDENTIFIER_EIP2930,
            Self::Eip1559 => COMPACT_IDENTIFIER_EIP1559,
            Self::Eip7702 => {
                buf.put_u8(alloy_consensus::constants::EIP7702_TX_TYPE_ID);
                COMPACT_EXTENDED_IDENTIFIER_FLAG
            }
            Self::Deposit => {
                buf.put_u8(crate::DEPOSIT_TX_TYPE_ID);
                COMPACT_EXTENDED_IDENTIFIER_FLAG
            }
            Self::PostExec => {
                buf.put_u8(POST_EXEC_TX_TYPE_ID);
                COMPACT_EXTENDED_IDENTIFIER_FLAG
            }
        }
    }

    fn from_compact(mut buf: &[u8], identifier: usize) -> (Self, &[u8]) {
        use bytes::Buf;
        match identifier {
            COMPACT_IDENTIFIER_LEGACY => (Self::Legacy, buf),
            COMPACT_IDENTIFIER_EIP2930 => (Self::Eip2930, buf),
            COMPACT_IDENTIFIER_EIP1559 => (Self::Eip1559, buf),
            COMPACT_EXTENDED_IDENTIFIER_FLAG => {
                let extended_identifier = buf.get_u8();
                let ty = match extended_identifier {
                    alloy_consensus::constants::EIP7702_TX_TYPE_ID => Self::Eip7702,
                    crate::DEPOSIT_TX_TYPE_ID => Self::Deposit,
                    POST_EXEC_TX_TYPE_ID => Self::PostExec,
                    _ => panic!("Unsupported OpTxType identifier: {extended_identifier}"),
                };
                (ty, buf)
            }
            _ => panic!("Unknown identifier for OpTxType: {identifier}"),
        }
    }
}

// --- TxDeposit ---

/// Mirror struct for compact encoding of [`TxDeposit`].
#[derive(reth_codecs_derive::Compact)]
#[reth_codecs(crate = "reth_codecs")]
struct CompactTxDeposit {
    source_hash: B256,
    from: Address,
    to: TxKind,
    mint: Option<u128>,
    value: U256,
    gas_limit: u64,
    is_system_transaction: bool,
    input: Bytes,
}

impl From<&TxDeposit> for CompactTxDeposit {
    fn from(tx: &TxDeposit) -> Self {
        Self {
            source_hash: tx.source_hash,
            from: tx.from,
            to: tx.to,
            mint: match tx.mint {
                0 => None,
                v => Some(v),
            },
            value: tx.value,
            gas_limit: tx.gas_limit,
            is_system_transaction: tx.is_system_transaction,
            input: tx.input.clone(),
        }
    }
}

impl From<CompactTxDeposit> for TxDeposit {
    fn from(tx: CompactTxDeposit) -> Self {
        Self {
            source_hash: tx.source_hash,
            from: tx.from,
            to: tx.to,
            mint: tx.mint.unwrap_or_default(),
            value: tx.value,
            gas_limit: tx.gas_limit,
            is_system_transaction: tx.is_system_transaction,
            input: tx.input,
        }
    }
}

impl Compact for TxDeposit {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        CompactTxDeposit::from(self).to_compact(buf)
    }

    fn from_compact(buf: &[u8], len: usize) -> (Self, &[u8]) {
        let (compact, buf) = CompactTxDeposit::from_compact(buf, len);
        (compact.into(), buf)
    }
}

impl Compact for TxPostExec {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        self.input().to_compact(buf)
    }

    fn from_compact(buf: &[u8], len: usize) -> (Self, &[u8]) {
        let (input, buf) = Bytes::from_compact(buf, len);
        let mut slice = input.as_ref();
        let tx = Self::decode(&mut slice).expect("valid compact post-exec tx");
        (tx, buf)
    }
}

// --- OpTypedTransaction ---

impl Compact for OpTypedTransaction {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        let tx_type = self.tx_type();
        let identifier = tx_type.to_compact(buf);
        match self {
            Self::Legacy(tx) => {
                tx.to_compact(buf);
            }
            Self::Eip2930(tx) => {
                tx.to_compact(buf);
            }
            Self::Eip1559(tx) => {
                tx.to_compact(buf);
            }
            Self::Eip7702(tx) => {
                tx.to_compact(buf);
            }
            Self::Deposit(tx) => {
                tx.to_compact(buf);
            }
            Self::PostExec(tx) => {
                tx.to_compact(buf);
            }
        }
        identifier
    }

    fn from_compact(buf: &[u8], identifier: usize) -> (Self, &[u8]) {
        let (tx_type, buf) = OpTxType::from_compact(buf, identifier);
        match tx_type {
            OpTxType::Legacy => {
                let (tx, buf) = alloy_consensus::TxLegacy::from_compact(buf, buf.len());
                (Self::Legacy(tx), buf)
            }
            OpTxType::Eip2930 => {
                let (tx, buf) = alloy_consensus::TxEip2930::from_compact(buf, buf.len());
                (Self::Eip2930(tx), buf)
            }
            OpTxType::Eip1559 => {
                let (tx, buf) = alloy_consensus::TxEip1559::from_compact(buf, buf.len());
                (Self::Eip1559(tx), buf)
            }
            OpTxType::Eip7702 => {
                let (tx, buf) = alloy_consensus::TxEip7702::from_compact(buf, buf.len());
                (Self::Eip7702(tx), buf)
            }
            OpTxType::Deposit => {
                let (tx, buf) = TxDeposit::from_compact(buf, buf.len());
                (Self::Deposit(tx), buf)
            }
            OpTxType::PostExec => {
                let (tx, buf) = TxPostExec::from_compact(buf, buf.len());
                (Self::PostExec(tx), buf)
            }
        }
    }
}

// --- OpTxEnvelope ---

impl Envelope for OpTxEnvelope {
    fn signature(&self) -> &Signature {
        match self {
            Self::Legacy(tx) => tx.signature(),
            Self::Eip2930(tx) => tx.signature(),
            Self::Eip1559(tx) => tx.signature(),
            Self::Eip7702(tx) => tx.signature(),
            Self::Deposit(_) | Self::PostExec(_) => {
                const DEPOSIT_SIG: Signature = Signature::new(U256::ZERO, U256::ZERO, false);
                &DEPOSIT_SIG
            }
        }
    }

    fn tx_type(&self) -> Self::TxType {
        alloy_consensus::Typed2718::ty(self).try_into().expect("valid op tx type")
    }
}

impl ToTxCompact for OpTxEnvelope {
    fn to_tx_compact(&self, buf: &mut (impl bytes::BufMut + AsMut<[u8]>)) {
        // Only write the tx body without the type prefix. The type is serialized separately
        // by CompactEnvelope.
        match self {
            Self::Legacy(tx) => {
                tx.tx().to_compact(buf);
            }
            Self::Eip2930(tx) => {
                tx.tx().to_compact(buf);
            }
            Self::Eip1559(tx) => {
                tx.tx().to_compact(buf);
            }
            Self::Eip7702(tx) => {
                tx.tx().to_compact(buf);
            }
            Self::Deposit(tx) => {
                tx.inner().to_compact(buf);
            }
            Self::PostExec(tx) => {
                tx.inner().to_compact(buf);
            }
        };
    }
}

impl FromTxCompact for OpTxEnvelope {
    type TxType = OpTxType;

    fn from_tx_compact(buf: &[u8], tx_type: Self::TxType, signature: Signature) -> (Self, &[u8])
    where
        Self: Sized,
    {
        // Deserialize the tx body directly based on tx_type. The type prefix was already
        // consumed by CompactEnvelope.
        match tx_type {
            OpTxType::Legacy => {
                let (tx, buf) = alloy_consensus::TxLegacy::from_compact(buf, buf.len());
                (Self::Legacy(Signed::new_unhashed(tx, signature)), buf)
            }
            OpTxType::Eip2930 => {
                let (tx, buf) = alloy_consensus::TxEip2930::from_compact(buf, buf.len());
                (Self::Eip2930(Signed::new_unhashed(tx, signature)), buf)
            }
            OpTxType::Eip1559 => {
                let (tx, buf) = alloy_consensus::TxEip1559::from_compact(buf, buf.len());
                (Self::Eip1559(Signed::new_unhashed(tx, signature)), buf)
            }
            OpTxType::Eip7702 => {
                let (tx, buf) = alloy_consensus::TxEip7702::from_compact(buf, buf.len());
                (Self::Eip7702(Signed::new_unhashed(tx, signature)), buf)
            }
            OpTxType::Deposit => {
                let (tx, buf) = TxDeposit::from_compact(buf, buf.len());
                (Self::Deposit(alloy_consensus::Sealed::new(tx)), buf)
            }
            OpTxType::PostExec => {
                let (tx, buf) = TxPostExec::from_compact(buf, buf.len());
                (Self::PostExec(alloy_consensus::Sealed::new(tx)), buf)
            }
        }
    }
}

impl Compact for OpTxEnvelope {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        <Self as CompactEnvelope>::to_compact(self, buf)
    }

    fn from_compact(buf: &[u8], len: usize) -> (Self, &[u8]) {
        <Self as CompactEnvelope>::from_compact(buf, len)
    }
}

impl reth_codecs::Compress for OpTxEnvelope {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: bytes::BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        let _ = Compact::to_compact(self, buf);
    }
}

impl reth_codecs::Decompress for OpTxEnvelope {
    fn decompress(value: &[u8]) -> Result<Self, reth_codecs::DecompressError> {
        let (obj, _) = Compact::from_compact(value, value.len());
        Ok(obj)
    }
}

// --- OpReceipt ---

/// Mirror struct for compact encoding of [`crate::OpDepositReceipt`].
#[derive(reth_codecs_derive::CompactZstd)]
#[reth_codecs(crate = "reth_codecs")]
#[reth_zstd(
    compressor = reth_zstd_compressors::with_receipt_compressor,
    decompressor = reth_zstd_compressors::with_receipt_decompressor
)]
struct CompactOpReceipt {
    tx_type: OpTxType,
    success: bool,
    cumulative_gas_used: u64,
    logs: Vec<Log>,
    deposit_nonce: Option<u64>,
    deposit_receipt_version: Option<u64>,
}

impl From<&OpReceipt> for CompactOpReceipt {
    fn from(receipt: &OpReceipt) -> Self {
        use alloy_consensus::TxReceipt;
        let (deposit_nonce, deposit_receipt_version) = match receipt {
            OpReceipt::Deposit(deposit) => (deposit.deposit_nonce, deposit.deposit_receipt_version),
            _ => (None, None),
        };
        Self {
            tx_type: receipt.tx_type(),
            success: receipt.status(),
            cumulative_gas_used: receipt.cumulative_gas_used(),
            logs: receipt.as_receipt().logs.clone(),
            deposit_nonce,
            deposit_receipt_version,
        }
    }
}

impl From<CompactOpReceipt> for OpReceipt {
    fn from(compact: CompactOpReceipt) -> Self {
        let receipt = Receipt {
            status: compact.success.into(),
            cumulative_gas_used: compact.cumulative_gas_used,
            logs: compact.logs,
        };
        match compact.tx_type {
            OpTxType::Legacy => Self::Legacy(receipt),
            OpTxType::Eip2930 => Self::Eip2930(receipt),
            OpTxType::Eip1559 => Self::Eip1559(receipt),
            OpTxType::Eip7702 => Self::Eip7702(receipt),
            OpTxType::PostExec => Self::PostExec(receipt),
            OpTxType::Deposit => Self::Deposit(crate::OpDepositReceipt {
                inner: receipt,
                deposit_nonce: compact.deposit_nonce,
                deposit_receipt_version: compact.deposit_receipt_version,
            }),
        }
    }
}

impl Compact for OpReceipt {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        CompactOpReceipt::from(self).to_compact(buf)
    }

    fn from_compact(buf: &[u8], len: usize) -> (Self, &[u8]) {
        let (compact, buf) = CompactOpReceipt::from_compact(buf, len);
        (compact.into(), buf)
    }
}

impl reth_codecs::Compress for OpReceipt {
    type Compressed = Vec<u8>;

    fn compress_to_buf<B: bytes::BufMut + AsMut<[u8]>>(&self, buf: &mut B) {
        let _ = Compact::to_compact(self, buf);
    }
}

impl reth_codecs::Decompress for OpReceipt {
    fn decompress(value: &[u8]) -> Result<Self, reth_codecs::DecompressError> {
        let (obj, _) = Compact::from_compact(value, value.len());
        Ok(obj)
    }
}
