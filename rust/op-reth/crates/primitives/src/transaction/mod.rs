//! Optimism transaction types

mod tx_type;

/// Kept for consistency tests
#[cfg(test)]
mod signed;

pub use op_alloy_consensus::{
    OpTransaction, OpTxEnvelope, OpTxType, OpTypedTransaction, POST_EXEC_TX_TYPE_ID,
    PostExecPayload, SDMGasEntry, TxPostExec, build_post_exec_tx,
};

/// Signed transaction.
pub type OpTransactionSigned = OpTxEnvelope;
