//! Implementations of `InMemorySize` for OP Stack consensus types.

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
            Self::Eip7702(receipt) => receipt.size(),
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
        }
    }
}
