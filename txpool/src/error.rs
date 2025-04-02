use crate::supervisor::{InteropTxValidatorError, InvalidInboxEntry};
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
}

impl PoolTransactionError for InvalidCrossTx {
    fn is_bad_transaction(&self) -> bool {
        match self {
            Self::ValidationError(err) => {
                match err {
                    InteropTxValidatorError::InvalidInboxEntry(err) => match err {
                        // This transaction could have been valid earlier. `SafetyLevel::Invalid`
                        // may be returned if tx has been invalidated as a result of reorg caused
                        // by network congestion (not peer's fault). Any other `SafetyLevel` may
                        // become valid when origin chain progresses, but is not safe enough to
                        // match local minimum safety level (also not peer's fault).
                        InvalidInboxEntry::MinimumSafety { .. } => false,
                        // This tx will not become valid unless supervisor is reconfigured, down
                        // score peer
                        InvalidInboxEntry::UnknownChain(_) => true,
                    },
                    // Rpc error or supervisor haven't responded in time
                    InteropTxValidatorError::RpcClientError(_) |
                    InteropTxValidatorError::ValidationTimeout(_) => false,
                    // Transaction caused unknown (for parsing) error in supervisor
                    InteropTxValidatorError::SupervisorServerError(_) => true,
                }
            }
            Self::CrossChainTxPreInterop => true,
        }
    }

    fn as_any(&self) -> &dyn Any {
        self
    }
}
