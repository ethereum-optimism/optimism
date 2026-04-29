//! Optimism transaction types

use alloc::vec::Vec;
use alloy_consensus::{Sealable, transaction::Recovered};
use alloy_primitives::Address;
use reth_primitives_traits::SignedTransaction;

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

/// Capability trait for signed transaction types that can synthesize the OP post-exec tx.
pub trait BuildPostExecTransaction: SignedTransaction + OpTransaction + Sized {
    /// Builds the post-exec tx for the given block and refund entries.
    fn build_post_exec(block_number: u64, gas_refund_entries: Vec<SDMGasEntry>) -> Self;

    /// Builds a recovered post-exec tx with the canonical zero signer.
    #[must_use]
    fn build_recovered_post_exec(
        block_number: u64,
        gas_refund_entries: Vec<SDMGasEntry>,
    ) -> Recovered<Self> {
        Recovered::new_unchecked(
            Self::build_post_exec(block_number, gas_refund_entries),
            Address::ZERO,
        )
    }
}

impl BuildPostExecTransaction for OpTransactionSigned {
    fn build_post_exec(block_number: u64, gas_refund_entries: Vec<SDMGasEntry>) -> Self {
        Self::PostExec(build_post_exec_tx(block_number, gas_refund_entries).seal_slow())
    }
}
