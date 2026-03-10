//! L2 block and reference types.

use alloy_consensus::Header;
use alloy_primitives::{B256, Sealed};
use op_alloy_consensus::{OpReceiptEnvelope, OpTxEnvelope};

/// A fully constructed L2 block with header, transactions, and receipts.
#[derive(Debug, Clone)]
pub struct L2Block {
    /// Sealed header with computed block hash.
    pub header: Sealed<Header>,
    /// All transactions including deposits.
    pub transactions: Vec<OpTxEnvelope>,
    /// Execution receipts.
    pub receipts: Vec<OpReceiptEnvelope>,
    /// Withdrawals root (present for Isthmus+).
    pub withdrawals_root: Option<B256>,
}

/// Lightweight reference to an L2 block by index.
#[derive(Debug, Clone, Copy)]
pub struct L2BlockRef {
    /// Index into the chain builder's block list.
    pub index: usize,
}
