//! Optimism transaction types

mod tx_type;

/// Kept for consistency tests
#[cfg(test)]
mod signed;

pub use op_alloy_consensus::{OpTransaction, OpTxType, OpTypedTransaction};

/// Signed transaction.
pub type OpTransactionSigned = op_alloy_consensus::OpTxEnvelope;
