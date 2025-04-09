//! Optimism transaction types

pub mod signed;
pub mod tx_type;

/// A trait that represents an optimism transaction, mainly used to indicate whether or not the
/// transaction is a deposit transaction.
pub trait OpTransaction {
    /// Whether or not the transaction is a dpeosit transaction.
    fn is_deposit(&self) -> bool;
}

impl OpTransaction for op_alloy_consensus::OpTxEnvelope {
    fn is_deposit(&self) -> bool {
        Self::is_deposit(self)
    }
}
