//! Transaction types for Optimism.

mod deposit;
pub use deposit::{DepositTransaction, TxDeposit};

mod tx_type;
pub use tx_type::DEPOSIT_TX_TYPE_ID;

mod envelope;
pub use envelope::{OpTransaction, OpTxEnvelope, OpTxType};

#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
pub use envelope::serde_bincode_compat as envelope_serde_bincode_compat;

mod typed;
pub use typed::OpTypedTransaction;

mod pooled;
pub use pooled::OpPooledTransaction;

#[cfg(feature = "serde")]
pub use deposit::serde_deposit_tx_rpc;

mod meta;
pub use meta::{OpDepositInfo, OpTransactionInfo};

/// Bincode-compatible serde implementations for transaction types.
#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
pub mod serde_bincode_compat {
    pub use super::{deposit::serde_bincode_compat::TxDeposit, envelope::serde_bincode_compat::*};
}
