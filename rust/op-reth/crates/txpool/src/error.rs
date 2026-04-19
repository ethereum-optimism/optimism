use crate::supervisor::InteropTxValidatorError;
use reth_transaction_pool::error::PoolTransactionError;
use std::any::Any;

/// Wrapper for [`InteropTxValidatorError`] to implement [`PoolTransactionError`] for it.
#[derive(thiserror::Error, Debug)]
pub enum InvalidCrossTx {
    /// Errors produced by supervisor validation
    #[error(transparent)]
    ValidationError(#[from] InteropTxValidatorError),
    /// Error cause by cross chain tx during not active interop hardfork
    #[error("cross chain tx is invalid before interop")]
    CrossChainTxPreInterop,
    /// Rejected because failsafe mode is active — all interop txs are blocked.
    #[error("interop failsafe is active")]
    FailsafeEnabled,
}

impl PoolTransactionError for InvalidCrossTx {
    fn is_bad_transaction(&self) -> bool {
        match self {
            Self::CrossChainTxPreInterop => true,
            Self::ValidationError(_) | Self::FailsafeEnabled => false,
        }
    }

    fn as_any(&self) -> &dyn Any {
        self
    }
}
