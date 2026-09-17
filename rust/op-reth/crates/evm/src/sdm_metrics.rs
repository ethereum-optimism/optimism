//! Reporting for blocks whose `0x7D` post-exec transaction fails structural validation. The
//! engine surfaces such a block only as a generic invalid payload, so the failed rule is logged
//! and counted here, where it is still known.

use op_alloy_consensus::PostExecPayloadValidationError;
#[cfg(feature = "std")]
use reth_metrics::{
    Metrics,
    metrics::{self, Counter},
};

#[cfg(feature = "std")]
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmPostExecValidationMetrics {
    /// Blocks whose post-exec transaction failed structural validation, by failed rule.
    post_exec_validation_failure_total: Counter,
}

/// Logs a block rejected for its post-exec transaction and, with `std`, counts it under the
/// failed rule.
pub(crate) fn report_post_exec_validation_failure(
    block_number: u64,
    error: &PostExecPayloadValidationError,
) {
    let reason = failure_reason(error);
    tracing::warn!(
        target: "op_evm",
        block_number,
        reason,
        %error,
        "block rejected: post-exec transaction failed structural validation"
    );
    #[cfg(feature = "std")]
    SdmPostExecValidationMetrics::new_with_labels(&[("reason", reason)])
        .post_exec_validation_failure_total
        .increment(1);
}

/// The failed rule as a stable label value, independent of the error's display text.
pub(crate) const fn failure_reason(error: &PostExecPayloadValidationError) -> &'static str {
    match error {
        PostExecPayloadValidationError::UnexpectedPostExecTx { .. } => "unexpected_post_exec_tx",
        PostExecPayloadValidationError::MultiplePostExecTxs { .. } => "multiple_post_exec_txs",
        PostExecPayloadValidationError::PostExecTxNotLast { .. } => "post_exec_tx_not_last",
        PostExecPayloadValidationError::BlockNumberMismatch { .. } => "block_number_mismatch",
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use PostExecPayloadValidationError as Error;
    use rstest::rstest;

    #[rstest]
    #[case::unexpected(
        /* error */ Error::UnexpectedPostExecTx { tx_index: 0 },
        /* expected */ "unexpected_post_exec_tx"
    )]
    #[case::multiple(
        /* error */ Error::MultiplePostExecTxs { first_index: 0, duplicate_index: 1 },
        /* expected */ "multiple_post_exec_txs"
    )]
    #[case::not_last(
        /* error */ Error::PostExecTxNotLast { tx_index: 0, last_index: 1 },
        /* expected */ "post_exec_tx_not_last"
    )]
    #[case::block_number(
        /* error */ Error::BlockNumberMismatch { payload_block_number: 1, block_number: 2 },
        /* expected */ "block_number_mismatch"
    )]
    fn failure_reason_names_the_failed_rule(#[case] error: Error, #[case] expected: &str) {
        assert_eq!(failure_reason(&error), expected);
    }
}
