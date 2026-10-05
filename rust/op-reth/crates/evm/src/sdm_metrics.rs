//! Reporting for blocks whose `0x7D` post-exec transaction passes or fails structural
//! validation. The engine surfaces a rejected block only as a generic invalid payload, so the
//! failed rule is logged and counted here, where it is still known; passes are counted so a
//! dashboard can show SDM validation activity rather than inferring it from generic block
//! validation.
//!
//! Recorded from the execution-context builders and the Engine API payload preflight. Beyond
//! live import, several paths re-execute stored canonical blocks that already passed this check:
//! RPC replays, the proofs ExEx's periodic re-execution, and pipeline or backfill sync.
//! Re-execution cannot inflate the failure counters, but every pass recounts `ok` — on a
//! verifying proofs node `ok` runs at roughly twice the block rate and it spikes while a node
//! syncs, so the `ok` series is an activity signal, not a canonical block count. Failures the
//! executor raises while verifying the refunds themselves are a separate class and are not
//! counted here.
//!
//! `new_with_labels` resolves the recorder on every call instead of caching it, so a report before
//! the recorder is installed drops that one sample rather than silencing the counter. The same
//! applies to the zero-registration: it only reaches the exporter when a configuration is
//! constructed after the recorder is installed, which the op-reth binary guarantees by installing
//! the recorder before building any component.

use op_alloy_consensus::PostExecPayloadValidationError;
#[cfg(feature = "std")]
use reth_metrics::{
    Metrics,
    metrics::{self, Counter},
};

#[cfg(feature = "std")]
const RESULT_OK: &str = "ok";
#[cfg(feature = "std")]
const RESULT_FAIL: &str = "fail";
#[cfg(feature = "std")]
const ALL_RESULTS: [&str; 2] = [RESULT_OK, RESULT_FAIL];

#[cfg(feature = "std")]
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmPostExecValidationMetrics {
    /// Post-exec structural validation failures, by the first rule the parser rejected on.
    post_exec_validation_failure_total: Counter,
}

#[cfg(feature = "std")]
#[derive(Metrics)]
#[metrics(scope = "op_sdm")]
struct SdmPostExecResultMetrics {
    /// Post-exec validations by outcome: `ok` when a block's post-exec transaction passes
    /// structural validation, `fail` on any rejection. Blocks carrying no post-exec transaction
    /// count under neither outcome, so the series is an SDM activity signal, not a block counter.
    post_exec_validation_total: Counter,
}

#[cfg(feature = "std")]
fn failure_counter(reason: &'static str) -> Counter {
    SdmPostExecValidationMetrics::new_with_labels(&[("reason", reason)])
        .post_exec_validation_failure_total
}

#[cfg(feature = "std")]
fn result_counter(result: &'static str) -> Counter {
    SdmPostExecResultMetrics::new_with_labels(&[("result", result)]).post_exec_validation_total
}

/// Registers every series this module can emit at zero.
///
/// The counters are otherwise label-lazy: a series would first exist at the event that creates
/// it, and `increase()` never counts a series birth, so the first failure a process ever sees
/// could fire no alert. With the zeros registered — and scraped at least once — that failure is
/// a 0 -> 1 step on an existing series. A call before the recorder is installed is dropped
/// silently, and registration is idempotent, so callers run it on every configuration
/// construction.
pub(crate) fn register_sdm_metrics_at_zero() {
    #[cfg(feature = "std")]
    {
        for reason in PostExecPayloadValidationError::ALL_REASONS {
            failure_counter(reason).increment(0);
        }
        for result in ALL_RESULTS {
            result_counter(result).increment(0);
        }
    }
}

/// Counts a block whose post-exec transaction passed structural validation.
pub(crate) fn report_post_exec_validation_ok() {
    #[cfg(feature = "std")]
    result_counter(RESULT_OK).increment(1);
}

/// Logs a block rejected for its post-exec transaction and, with `std`, counts it under the
/// failed rule and as a failed validation.
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
    {
        failure_counter(reason).increment(1);
        result_counter(RESULT_FAIL).increment(1);
    }
}

#[cfg(all(test, feature = "std"))]
mod tests {
    use super::{RESULT_FAIL, RESULT_OK};
    use crate::{OpEvmConfig, PostExecMode};
    use alloy_consensus::{Block, BlockBody, Header, Sealable};
    use alloy_genesis::Genesis;
    use metrics_exporter_prometheus::{PrometheusBuilder, PrometheusHandle, PrometheusRecorder};
    use metrics_util::layers::{Layer, Prefix, PrefixLayer};
    use op_alloy_consensus::{
        PostExecPayloadValidationError, SDMGasEntry, TxDeposit, build_post_exec_tx,
    };
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
    const RESULTS: &str = "reth_op_sdm_post_exec_validation_total";
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
    /// returns whether the executor will reject it alongside the rendered exposition.
    fn import(block: SealedBlock<OpBlock>) -> (bool, String) {
        let (recorder, handle) = recorder();
        let context = with_local_recorder(&recorder, || {
            evm_config().context_for_block(&block).expect("context construction is infallible")
        });
        (matches!(context.post_exec_mode, PostExecMode::Invalid(_)), handle.render())
    }

    /// Reads one labeled series out of the exposition. `None` distinguishes a series that was
    /// never registered from one reporting 0.
    fn series(exposition: &str, metric: &str, label: &str, value: &str) -> Option<f64> {
        let series = format!("{metric}{{{label}=\"{value}\"}} ");
        exposition.lines().find_map(|line| line.strip_prefix(series.as_str())?.trim().parse().ok())
    }

    fn failures(exposition: &str, reason: &str) -> Option<f64> {
        series(exposition, FAILURES, "reason", reason)
    }

    fn results(exposition: &str, result: &str) -> Option<f64> {
        series(exposition, RESULTS, "result", result)
    }

    /// Asserts every failure series is present, at 1 for `failed` and 0 everywhere else.
    /// Presence at zero is the property under test: an absent series is a birth `increase()`
    /// cannot count.
    fn assert_failures(exposition: &str, failed: Option<&str>) {
        for reason in PostExecPayloadValidationError::ALL_REASONS {
            let expected = if Some(reason) == failed { 1.0 } else { 0.0 };
            assert_eq!(failures(exposition, reason), Some(expected), "{reason}: {exposition}");
        }
    }

    fn assert_results(exposition: &str, ok: f64, fail: f64) {
        assert_eq!(results(exposition, RESULT_OK), Some(ok), "{exposition}");
        assert_eq!(results(exposition, RESULT_FAIL), Some(fail), "{exposition}");
    }

    #[test]
    fn fresh_config_registers_every_series_at_zero() {
        let (recorder, handle) = recorder();
        with_local_recorder(&recorder, || {
            let _ = evm_config();
        });
        let exposition = handle.render();

        assert_failures(&exposition, None);
        assert_results(&exposition, 0.0, 0.0);
    }

    #[rstest]
    #[case::before_activation(INACTIVE, vec![post_exec(BLOCK)], "unexpected_post_exec_tx")]
    #[case::two_post_exec_txs(ACTIVE, vec![post_exec(BLOCK), post_exec(BLOCK)], "multiple_post_exec_txs")]
    #[case::not_last(ACTIVE, vec![post_exec(BLOCK), deposit()], "post_exec_tx_not_last")]
    #[case::wrong_anchor(ACTIVE, vec![post_exec(BLOCK + 1)], "block_number_mismatch")]
    #[case::too_many_entries(ACTIVE, vec![post_exec(BLOCK)], "too_many_gas_refund_entries")]
    fn rejected_block_counts_under_the_failed_rule_and_as_fail(
        #[case] timestamp: u64,
        #[case] transactions: Vec<OpTransactionSigned>,
        #[case] reason: &str,
    ) {
        let (rejected, exposition) = import(block(timestamp, transactions));

        assert!(rejected, "block is rejected");
        assert_failures(&exposition, Some(reason));
        assert_results(&exposition, /* ok */ 0.0, /* fail */ 1.0);
    }

    /// Drives the Engine API preflight helper with raw encoded transactions against a recorder
    /// private to the test, returning the result alongside the rendered exposition.
    fn preflight(transactions: Vec<alloy_primitives::Bytes>) -> (Result<(), ()>, String) {
        let (recorder, handle) = recorder();
        let result = with_local_recorder(&recorder, || {
            super::register_sdm_metrics_at_zero();
            crate::preflight_post_exec_payload(&transactions, BLOCK)
        });
        (result.map_err(drop), handle.render())
    }

    fn encoded_post_exec(entries: usize) -> alloy_primitives::Bytes {
        use alloy_eips::eip2718::Encodable2718;
        build_post_exec_tx(
            BLOCK,
            (0..entries).map(|i| SDMGasEntry { index: i as u64, gas_refund: 1 }).collect(),
        )
        .encoded_2718()
        .into()
    }

    const fn filler() -> alloy_primitives::Bytes {
        alloy_primitives::Bytes::from_static(&[0x01])
    }

    const fn bare_post_exec() -> alloy_primitives::Bytes {
        alloy_primitives::Bytes::from_static(&[op_alloy_consensus::POST_EXEC_TX_TYPE_ID])
    }

    fn versioned_post_exec(version: u8) -> alloy_primitives::Bytes {
        use op_alloy_consensus::{POST_EXEC_TX_TYPE_ID, PostExecPayload};
        let payload = PostExecPayload { version, block_number: BLOCK, gas_refund_entries: vec![] };
        let mut tx = vec![POST_EXEC_TX_TYPE_ID];
        tx.extend_from_slice(payload.to_rlp_bytes().as_ref());
        tx.into()
    }

    #[rstest]
    #[case::multiple(vec![bare_post_exec(), bare_post_exec()], "multiple_post_exec_txs")]
    #[case::not_last(vec![bare_post_exec(), filler()], "post_exec_tx_not_last")]
    #[case::too_many_entries(vec![filler(), encoded_post_exec(2)], "too_many_gas_refund_entries")]
    #[case::malformed(vec![bare_post_exec()], "malformed_post_exec_payload")]
    #[case::unsupported_version(vec![filler(), versioned_post_exec(2)], "unsupported_post_exec_payload_version")]
    fn rejected_payload_counts_on_the_engine_api_path(
        #[case] transactions: Vec<alloy_primitives::Bytes>,
        #[case] reason: &str,
    ) {
        let (result, exposition) = preflight(transactions);

        assert!(result.is_err(), "preflight rejects");
        assert_failures(&exposition, Some(reason));
        assert_results(&exposition, /* ok */ 0.0, /* fail */ 1.0);
    }

    /// The preflight does not count `ok`: a pass is not yet a validated post-exec transaction —
    /// the parse on the decoded transactions counts that.
    #[rstest]
    #[case::with_post_exec(vec![filler(), encoded_post_exec(1)])]
    #[case::without_post_exec(vec![filler()])]
    fn accepted_payload_counts_nothing_at_preflight(
        #[case] transactions: Vec<alloy_primitives::Bytes>,
    ) {
        let (result, exposition) = preflight(transactions);

        assert!(result.is_ok(), "preflight passes");
        assert_failures(&exposition, None);
        assert_results(&exposition, /* ok */ 0.0, /* fail */ 0.0);
    }

    #[rstest]
    #[case::well_formed(ACTIVE, vec![deposit(), post_exec(BLOCK)], /* ok */ 1.0)]
    #[case::active_without_post_exec_tx(ACTIVE, vec![deposit()], /* ok */ 0.0)]
    #[case::inactive_without_post_exec_tx(INACTIVE, vec![deposit()], /* ok */ 0.0)]
    fn accepted_block_counts_ok_only_when_post_exec_present(
        #[case] timestamp: u64,
        #[case] transactions: Vec<OpTransactionSigned>,
        #[case] expected_ok: f64,
    ) {
        let (rejected, exposition) = import(block(timestamp, transactions));

        assert!(!rejected, "block parses");
        assert_failures(&exposition, None);
        assert_results(&exposition, expected_ok, /* fail */ 0.0);
    }
}
