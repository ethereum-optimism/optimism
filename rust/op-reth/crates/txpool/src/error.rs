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
    /// Whether a validation verdict means the tx is definitively invalid and must be evicted.
    ///
    /// Distinct from [`is_bad_transaction`](Self::is_bad_transaction), which decides peer
    /// penalization. Only a decisive invalid verdict is evicted; transient or non-decisive results
    /// (quorum miss, timeout, data-unavailable, disagreement) are retained so an unreachable
    /// interop filter cannot drain the pool. Failsafe is handled by the whole-pool eviction.
    pub const fn is_definitive_invalid(&self) -> bool {
        match self {
            Self::CrossChainTxPreInterop => true,
            Self::ValidationError(err) => err.is_definitive_invalid(),
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::interop_filter::InteropTxValidatorError;
    use op_alloy_rpc_types::SuperchainDAError;
    use rstest::rstest;

    #[rstest]
    #[case::invalid_entry(
        InvalidCrossTx::ValidationError(InteropTxValidatorError::InvalidEntry(
            SuperchainDAError::ConflictingData,
        )),
        true,
        false
    )]
    #[case::rejected(
        InvalidCrossTx::ValidationError(InteropTxValidatorError::Rejected {
            code: -32602,
            message: "failed to parse access entry".to_string(),
        }),
        true,
        false
    )]
    #[case::pre_interop(InvalidCrossTx::CrossChainTxPreInterop, true, true)]
    #[case::outer_failsafe(InvalidCrossTx::FailsafeEnabled, false, false)]
    #[case::quorum_not_reached(
        InvalidCrossTx::ValidationError(InteropTxValidatorError::QuorumNotReached {
            received: 1,
            required: 2,
        }),
        false,
        false
    )]
    #[case::data_unavailable(
        InvalidCrossTx::ValidationError(InteropTxValidatorError::DataUnavailable { code: -321401 }),
        false,
        false
    )]
    #[case::timeout(
        InvalidCrossTx::ValidationError(InteropTxValidatorError::Timeout(2)),
        false,
        false
    )]
    #[case::disagreement(
        InvalidCrossTx::ValidationError(InteropTxValidatorError::Disagreement {
            valid: 1,
            invalid: 1,
        }),
        false,
        false
    )]
    #[case::inner_failsafe(
        InvalidCrossTx::ValidationError(InteropTxValidatorError::FailsafeEnabled),
        false,
        false
    )]
    fn invalid_cross_tx_predicates(
        #[case] err: InvalidCrossTx,
        #[case] expected_definitive_invalid: bool,
        #[case] expected_bad_transaction: bool,
    ) {
        assert_eq!(err.is_definitive_invalid(), expected_definitive_invalid);
        assert_eq!(err.is_bad_transaction(), expected_bad_transaction);
    }
}
