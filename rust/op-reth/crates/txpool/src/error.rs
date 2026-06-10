use crate::interop_filter::InteropTxValidatorError;
use reth_transaction_pool::error::PoolTransactionError;
use std::any::Any;

/// Wrapper for [`InteropTxValidatorError`] to implement [`PoolTransactionError`] for it.
#[derive(thiserror::Error, Debug)]
pub enum InvalidCrossTx {
    /// Errors produced by interop filter validation.
    #[error(transparent)]
    ValidationError(#[from] InteropTxValidatorError),
    /// Error cause by cross chain tx during not active interop hardfork
    #[error("cross chain tx is invalid before interop")]
    CrossChainTxPreInterop,
    /// Rejected because failsafe mode is active — all interop txs are blocked.
    #[error("interop failsafe is active")]
    FailsafeEnabled,
}

impl InvalidCrossTx {
    /// Whether an already-pooled interop tx should be evicted when revalidation against the filter
    /// returns this error. Only a permanently-invalid filter verdict (or a cross-chain tx seen
    /// before interop activation) evicts; transient or out-of-sync verdicts keep the tx, and
    /// failsafe is handled separately by the global failsafe eviction path.
    pub const fn should_evict_on_revalidation(&self) -> bool {
        match self {
            Self::CrossChainTxPreInterop => true,
            Self::ValidationError(e) => e.is_message_permanently_invalid(),
            Self::FailsafeEnabled => false,
        }
    }
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
