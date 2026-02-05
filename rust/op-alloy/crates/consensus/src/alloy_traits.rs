//! Implementations of alloy-consensus traits for OP consensus types.
//!
//! This module provides implementations of [`InMemorySize`], [`SignedTransaction`],
//! and [`SerdeBincodeCompat`] for Optimism consensus types.

use alloy_consensus::size::InMemorySize;

// ===== InMemorySize implementations =====

impl InMemorySize for crate::OpTxType {
    fn size(&self) -> usize {
        core::mem::size_of::<Self>()
    }
}

impl InMemorySize for crate::TxDeposit {
    fn size(&self) -> usize {
        self.size()
    }
}

impl InMemorySize for crate::OpDepositReceipt {
    fn size(&self) -> usize {
        let Self { inner, deposit_nonce, deposit_receipt_version } = self;
        inner.size() +
            core::mem::size_of_val(deposit_nonce) +
            core::mem::size_of_val(deposit_receipt_version)
    }
}

impl InMemorySize for crate::OpReceipt {
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

impl InMemorySize for crate::OpTypedTransaction {
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

impl InMemorySize for crate::OpPooledTransaction {
    fn size(&self) -> usize {
        match self {
            Self::Legacy(tx) => tx.size(),
            Self::Eip2930(tx) => tx.size(),
            Self::Eip1559(tx) => tx.size(),
            Self::Eip7702(tx) => tx.size(),
        }
    }
}

impl InMemorySize for crate::OpTxEnvelope {
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

// ===== SignedTransaction implementations =====

#[cfg(feature = "k256")]
impl alloy_consensus::SignedTransaction for crate::OpTxEnvelope {}

#[cfg(feature = "k256")]
impl alloy_consensus::SignedTransaction for crate::OpPooledTransaction {}

// ===== SerdeBincodeCompat implementations =====

#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
impl alloy_consensus::SerdeBincodeCompat for crate::OpTxEnvelope {
    type BincodeRepr<'a> =
        crate::serde_bincode_compat::transaction::OpTxEnvelope<'a>;

    fn as_repr(&self) -> Self::BincodeRepr<'_> {
        self.into()
    }

    fn from_repr(repr: Self::BincodeRepr<'_>) -> Self {
        repr.into()
    }
}

#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
impl alloy_consensus::SerdeBincodeCompat for crate::OpReceipt {
    type BincodeRepr<'a> = crate::serde_bincode_compat::OpReceipt<'a>;

    fn as_repr(&self) -> Self::BincodeRepr<'_> {
        self.into()
    }

    fn from_repr(repr: Self::BincodeRepr<'_>) -> Self {
        repr.into()
    }
}
