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
    /// Whether a revalidation verdict means the tx is now invalid and must be evicted.
    ///
    /// Distinct from [`is_bad_transaction`](Self::is_bad_transaction), which decides peer
    /// penalization. Only a decisive invalid verdict is evicted; transient or non-decisive results
    /// (quorum miss, timeout, data-unavailable, disagreement) are retained so an unreachable
    /// interop filter cannot drain the pool. Failsafe is handled by the whole-pool eviction.
    pub const fn is_now_invalid(&self) -> bool {
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

    /// A definitive invalid verdict (`InvalidEntry`) must be evicted on revalidation.
    #[test]
    fn invalid_entry_is_now_invalid() {
        let err = InvalidCrossTx::ValidationError(InteropTxValidatorError::InvalidEntry(
            SuperchainDAError::ConflictingData,
        ));
        assert!(err.is_now_invalid(), "a definitive InvalidEntry verdict must be evicted");
    }

    /// A definitive `Rejected` verdict (generic filter rejection) must be evicted on revalidation.
    #[test]
    fn rejected_is_now_invalid() {
        let err = InvalidCrossTx::ValidationError(InteropTxValidatorError::Rejected {
            code: -32602,
            message: "failed to parse access entry".to_string(),
        });
        assert!(err.is_now_invalid(), "a definitive Rejected verdict must be evicted");
    }

    /// A pre-interop cross-chain tx must be evicted (preserves prior behavior).
    #[test]
    fn pre_interop_is_now_invalid() {
        assert!(InvalidCrossTx::CrossChainTxPreInterop.is_now_invalid());
    }

    /// A transient quorum miss must NOT evict: a flapping/unreachable interop filter must not drain
    /// the pool.
    #[test]
    fn quorum_not_reached_is_retained() {
        let err = InvalidCrossTx::ValidationError(InteropTxValidatorError::QuorumNotReached {
            received: 1,
            required: 2,
        });
        assert!(!err.is_now_invalid(), "a transient quorum miss must not evict");
    }

    /// Data-not-yet-available (out-of-sync endpoint) is a soft, non-decisive result: retained.
    #[test]
    fn data_unavailable_is_retained() {
        let err = InvalidCrossTx::ValidationError(InteropTxValidatorError::DataUnavailable {
            code: -321401,
        });
        assert!(!err.is_now_invalid(), "a soft out-of-sync result must not evict");
    }

    /// An endpoint timeout is non-decisive: retained.
    #[test]
    fn timeout_is_retained() {
        let err = InvalidCrossTx::ValidationError(InteropTxValidatorError::Timeout(2));
        assert!(!err.is_now_invalid(), "a timeout must not evict");
    }

    /// Endpoints splitting on the verdict is not a clean "found invalid": retained.
    #[test]
    fn disagreement_is_retained() {
        let err = InvalidCrossTx::ValidationError(InteropTxValidatorError::Disagreement {
            valid: 1,
            invalid: 1,
        });
        assert!(!err.is_now_invalid(), "endpoint disagreement must not evict");
    }

    /// Failsafe is handled by the whole-pool eviction at the top of the maintenance loop, so the
    /// per-tx revalidation predicate must not also act on it.
    #[test]
    fn failsafe_is_not_evicted_by_this_predicate() {
        assert!(!InvalidCrossTx::FailsafeEnabled.is_now_invalid());
        let err = InvalidCrossTx::ValidationError(InteropTxValidatorError::FailsafeEnabled);
        assert!(!err.is_now_invalid());
    }

    /// `is_now_invalid` must remain independent of reth's peer-penalization predicate: a definitive
    /// invalid verdict warrants eviction but not peer penalization.
    #[test]
    fn eviction_predicate_is_distinct_from_peer_penalization() {
        let err = InvalidCrossTx::ValidationError(InteropTxValidatorError::InvalidEntry(
            SuperchainDAError::ConflictingData,
        ));
        assert!(err.is_now_invalid(), "definitive invalid verdict is evictable");
        assert!(
            !err.is_bad_transaction(),
            "but it does not warrant peer penalization (reth semantics unchanged)"
        );
    }
}
