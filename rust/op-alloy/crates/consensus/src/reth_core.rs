//! Implementations of `InMemorySize` for OP Stack consensus types.
//!
//! Ported from reth v1.11.3 (`d6324d63e`):
//! - `crates/primitives-traits/src/size.rs` (behind `cfg(feature = "op")`)
//!
//! Differences from upstream:
//! - `OpTxType` and `TxDeposit` impls are new (upstream only had them for the compound types, but
//!   the `reth-core` crate now requires them standalone).
//! - `OpTxEnvelope::Deposit` explicitly sizes the seal hash + inner tx, whereas upstream delegated
//!   to `Sealed<TxDeposit>::size()` which did the same internally.

use crate::{
    OpDepositReceipt, OpPooledTransaction, OpReceipt, OpTxEnvelope, OpTxType, OpTypedTransaction,
    TxDeposit,
};
use alloy_consensus::InMemorySize;

impl InMemorySize for OpTxType {
    fn size(&self) -> usize {
        core::mem::size_of::<Self>()
    }
}

impl InMemorySize for TxDeposit {
    fn size(&self) -> usize {
        core::mem::size_of::<Self>() + self.input.len()
    }
}

impl InMemorySize for OpDepositReceipt {
    fn size(&self) -> usize {
        self.inner.size() +
            core::mem::size_of_val(&self.deposit_nonce) +
            core::mem::size_of_val(&self.deposit_receipt_version)
    }
}

impl InMemorySize for OpReceipt {
    fn size(&self) -> usize {
        match self {
            Self::Legacy(receipt) |
            Self::Eip2930(receipt) |
            Self::Eip1559(receipt) |
            Self::Eip7702(receipt) |
            Self::PostExec(receipt) => receipt.size(),
            Self::Deposit(receipt) => receipt.size(),
        }
    }
}

impl InMemorySize for OpTypedTransaction {
    fn size(&self) -> usize {
        match self {
            Self::Legacy(tx) => tx.size(),
            Self::Eip2930(tx) => tx.size(),
            Self::Eip1559(tx) => tx.size(),
            Self::Eip7702(tx) => tx.size(),
            Self::Deposit(tx) => tx.size(),
            Self::PostExec(tx) => tx.size(),
        }
    }
}

impl InMemorySize for OpPooledTransaction {
    fn size(&self) -> usize {
        match self {
            Self::Legacy(tx) => tx.size(),
            Self::Eip2930(tx) => tx.size(),
            Self::Eip1559(tx) => tx.size(),
            Self::Eip7702(tx) => tx.size(),
        }
    }
}

impl InMemorySize for OpTxEnvelope {
    fn size(&self) -> usize {
        match self {
            Self::Legacy(tx) => tx.size(),
            Self::Eip2930(tx) => tx.size(),
            Self::Eip1559(tx) => tx.size(),
            Self::Eip7702(tx) => tx.size(),
            Self::Deposit(tx) => core::mem::size_of::<alloy_primitives::B256>() + tx.inner().size(),
            Self::PostExec(tx) => {
                core::mem::size_of::<alloy_primitives::B256>() + tx.inner().size()
            }
        }
    }
}
