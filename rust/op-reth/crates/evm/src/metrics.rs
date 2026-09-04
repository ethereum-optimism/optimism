//! Metrics for Optimism-specific EVM validation.

#[cfg(feature = "std")]
use metrics::{counter, describe_counter};
use op_alloy_consensus::PostExecPayloadValidationError;

/// Counter incremented when an execution-client path rejects `PostExec` block structure.
pub const POST_EXEC_VALIDATION_FAILURES: &str = "optimism_post_exec.validation_failures";

/// Stable, bounded reason labels for `PostExec` structural validation failures.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PostExecValidationFailureReason {
    /// The `0x7D` transaction or its schema could not be decoded.
    InvalidEncodingOrSchema,
    /// A block contained `0x7D` before SDM activation.
    SdmInactive,
    /// A block contained more than one `0x7D` transaction.
    MultipleTransactions,
    /// The `0x7D` transaction was not the final transaction.
    NotLast,
    /// The payload block number did not match the containing block.
    BlockNumberMismatch,
}

impl PostExecValidationFailureReason {
    /// Returns the stable metric label for this failure reason.
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::InvalidEncodingOrSchema => "invalid_encoding_or_schema",
            Self::SdmInactive => "sdm_inactive",
            Self::MultipleTransactions => "multiple_transactions",
            Self::NotLast => "not_last",
            Self::BlockNumberMismatch => "block_number_mismatch",
        }
    }
}

impl From<&PostExecPayloadValidationError> for PostExecValidationFailureReason {
    fn from(error: &PostExecPayloadValidationError) -> Self {
        match error {
            PostExecPayloadValidationError::UnexpectedPostExecTx { .. } => Self::SdmInactive,
            PostExecPayloadValidationError::MultiplePostExecTxs { .. } => {
                Self::MultipleTransactions
            }
            PostExecPayloadValidationError::PostExecTxNotLast { .. } => Self::NotLast,
            PostExecPayloadValidationError::BlockNumberMismatch { .. } => Self::BlockNumberMismatch,
        }
    }
}

/// Records a `PostExec` structural validation failure.
///
/// Callers must classify failures with [`PostExecValidationFailureReason`] so the `reason` label
/// remains low-cardinality.
pub fn record_post_exec_validation_failure(reason: PostExecValidationFailureReason) {
    #[cfg(feature = "std")]
    {
        describe_counter!(
            POST_EXEC_VALIDATION_FAILURES,
            "PostExec structural validation failures by reason"
        );
        counter!(POST_EXEC_VALIDATION_FAILURES, "reason" => reason.as_str()).increment(1);
    }
    #[cfg(not(feature = "std"))]
    let _ = reason;
}

#[cfg(all(test, feature = "std"))]
mod tests {
    use super::*;
    use metrics_util::debugging::{DebugValue, DebuggingRecorder, Snapshot};

    fn failure_count(snapshot: Snapshot, reason: PostExecValidationFailureReason) -> u64 {
        snapshot
            .into_vec()
            .into_iter()
            .find_map(|(composite_key, _, _, value)| {
                let key = composite_key.key();
                let matches = key.name() == POST_EXEC_VALIDATION_FAILURES &&
                    key.labels().any(|label| {
                        label.key() == "reason" && label.value() == reason.as_str()
                    });
                matches.then_some(match value {
                    DebugValue::Counter(value) => value,
                    _ => 0,
                })
            })
            .unwrap_or_default()
    }

    #[test]
    fn records_reason_labeled_post_exec_validation_failure() {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            record_post_exec_validation_failure(
                PostExecValidationFailureReason::BlockNumberMismatch,
            );
        });

        assert_eq!(
            failure_count(
                snapshotter.snapshot(),
                PostExecValidationFailureReason::BlockNumberMismatch
            ),
            1
        );
    }

    #[test]
    fn maps_structural_errors_to_stable_reasons() {
        for (error, expected) in [
            (
                PostExecPayloadValidationError::UnexpectedPostExecTx { tx_index: 0 },
                PostExecValidationFailureReason::SdmInactive,
            ),
            (
                PostExecPayloadValidationError::MultiplePostExecTxs {
                    first_index: 0,
                    duplicate_index: 1,
                },
                PostExecValidationFailureReason::MultipleTransactions,
            ),
            (
                PostExecPayloadValidationError::PostExecTxNotLast { tx_index: 0, last_index: 1 },
                PostExecValidationFailureReason::NotLast,
            ),
            (
                PostExecPayloadValidationError::BlockNumberMismatch {
                    payload_block_number: 1,
                    block_number: 2,
                },
                PostExecValidationFailureReason::BlockNumberMismatch,
            ),
        ] {
            assert_eq!(PostExecValidationFailureReason::from(&error), expected);
        }
    }
}
