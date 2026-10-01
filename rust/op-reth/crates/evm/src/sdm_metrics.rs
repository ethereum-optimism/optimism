//! Reporting for blocks whose `0x7D` post-exec transaction fails structural validation. The
//! engine surfaces such a block only as a generic invalid payload, so the failed rule is logged
//! and counted here, where it is still known.
//!
//! Recorded from the execution-context builders. The RPC callers that reach them replay stored
//! canonical blocks, which already passed this check at import, so replays cannot inflate the
//! counter. Failures the executor raises while verifying the refunds themselves are a separate
//! class and are not counted here.
//!
//! `new_with_labels` resolves the recorder on every call instead of caching it, so a report before
//! the recorder is installed drops that one sample rather than silencing the counter.

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
    /// Post-exec structural validation failures, by the first rule the parser rejected on.
    post_exec_validation_failure_total: Counter,
}

/// Logs a block rejected for its post-exec transaction and, with `std`, counts it under the
/// failed rule.
pub(crate) fn report_post_exec_validation_failure(
    block_number: u64,
    error: PostExecPayloadValidationError,
) {
    let reason = error.as_reason();
    tracing::warn!(
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

#[cfg(all(test, feature = "std"))]
mod tests {
    use crate::OpEvmConfig;
    use alloy_consensus::{Block, BlockBody, Header, Sealable};
    use alloy_genesis::Genesis;
    use metrics_exporter_prometheus::{PrometheusBuilder, PrometheusHandle, PrometheusRecorder};
    use metrics_util::layers::{Layer, Prefix, PrefixLayer};
    use op_alloy_consensus::{SDMGasEntry, TxDeposit, build_post_exec_tx};
    use reth_chainspec::ForkCondition;
    use reth_evm::ConfigureEvm;
    use reth_metrics::metrics::with_local_recorder;
    use reth_optimism_chainspec::OpChainSpecBuilder;
    use reth_optimism_forks::OpHardfork;
    use reth_optimism_primitives::{OpBlock, OpTransactionSigned};
    use reth_primitives_traits::SealedBlock;
    use rstest::rstest;
    use std::sync::Arc;

    const FAILURES: &str = "reth_op_sdm_post_exec_validation_failure_total";
    const ACTIVE: u64 = 100;
    const INACTIVE: u64 = ACTIVE - 1;
    const BLOCK: u64 = 7;

    fn evm_config() -> OpEvmConfig {
        OpEvmConfig::optimism(Arc::new(
            OpChainSpecBuilder::default()
                .chain(10.into())
                .genesis(Genesis::default())
                .with_fork(OpHardfork::Lagoon, ForkCondition::Timestamp(ACTIVE))
                .build(),
        ))
    }

    fn post_exec(anchored_block_number: u64) -> OpTransactionSigned {
        OpTransactionSigned::PostExec(
            build_post_exec_tx(
                anchored_block_number,
                vec![SDMGasEntry { index: 0, gas_refund: 1 }],
            )
            .seal_slow(),
        )
    }

    fn deposit() -> OpTransactionSigned {
        OpTransactionSigned::Deposit(TxDeposit::default().seal_slow())
    }

    fn block(timestamp: u64, transactions: Vec<OpTransactionSigned>) -> SealedBlock<OpBlock> {
        SealedBlock::new_unhashed(Block {
            header: Header { number: BLOCK, timestamp, ..Default::default() },
            body: BlockBody { transactions, ..Default::default() },
        })
    }

    /// A recorder private to one test, prefixed like the node's recorder stack so the rendered
    /// series names are the ones operators query.
    fn recorder() -> (Prefix<PrometheusRecorder>, PrometheusHandle) {
        let recorder = PrometheusBuilder::new().build_recorder();
        let handle = recorder.handle();
        (PrefixLayer::new("reth").layer(recorder), handle)
    }

    /// Builds the execution context for `block` against a recorder private to this test and
    /// returns the rendered exposition alongside the result.
    fn import(block: SealedBlock<OpBlock>) -> (Result<(), ()>, String) {
        let (recorder, handle) = recorder();
        let result = with_local_recorder(&recorder, || evm_config().context_for_block(&block));
        (result.map(drop).map_err(drop), handle.render())
    }

    /// Reads the counter for one `reason` out of the exposition. `None` distinguishes a series that
    /// was never registered from one reporting 0.
    fn failures(exposition: &str, reason: &str) -> Option<f64> {
        let series = format!("{FAILURES}{{reason=\"{reason}\"}} ");
        exposition.lines().find_map(|line| line.strip_prefix(series.as_str())?.trim().parse().ok())
    }

    fn failure_series(exposition: &str) -> usize {
        exposition.lines().filter(|line| line.starts_with(FAILURES)).count()
    }

    #[rstest]
    #[case::before_activation(INACTIVE, vec![post_exec(BLOCK)], "unexpected_post_exec_tx")]
    #[case::two_post_exec_txs(ACTIVE, vec![post_exec(BLOCK), post_exec(BLOCK)], "multiple_post_exec_txs")]
    #[case::not_last(ACTIVE, vec![post_exec(BLOCK), deposit()], "post_exec_tx_not_last")]
    #[case::wrong_anchor(ACTIVE, vec![post_exec(BLOCK + 1)], "block_number_mismatch")]
    #[case::too_many_entries(ACTIVE, vec![post_exec(BLOCK)], "too_many_gas_refund_entries")]
    fn rejected_block_counts_under_the_failed_rule(
        #[case] timestamp: u64,
        #[case] transactions: Vec<OpTransactionSigned>,
        #[case] reason: &str,
    ) {
        let (result, exposition) = import(block(timestamp, transactions));

        assert!(result.is_err(), "block is rejected");
        assert_eq!(failures(&exposition, reason), Some(1.0), "{exposition}");
        assert_eq!(failure_series(&exposition), 1, "only the failed rule registers: {exposition}");
    }

    #[rstest]
    #[case::well_formed(ACTIVE, vec![deposit(), post_exec(BLOCK)])]
    #[case::no_post_exec_tx(INACTIVE, vec![deposit()])]
    fn accepted_block_registers_nothing(
        #[case] timestamp: u64,
        #[case] transactions: Vec<OpTransactionSigned>,
    ) {
        let (result, exposition) = import(block(timestamp, transactions));

        assert!(result.is_ok(), "block parses");
        assert_eq!(failure_series(&exposition), 0, "{exposition}");
    }
}
