//! Exposition test for `reth_op_sdm_post_exec_validation_failure_total`.
//!
//! Its own integration binary, holding exactly one test, because the metrics recorder is global:
//! the crate's unit tests also feed rejected blocks through `context_for_block`, and a counter
//! assertion sharing a process with them would race.

use alloy_consensus::{Block, BlockBody, Header, Sealable};
use alloy_genesis::Genesis;
use metrics_exporter_prometheus::PrometheusBuilder;
use op_alloy_consensus::{SDMGasEntry, TxDeposit, build_post_exec_tx};
use reth_chainspec::ForkCondition;
use reth_evm::ConfigureEvm;
use reth_node_metrics::recorder::{
    install_prometheus_recorder, try_install_prometheus_recorder_with_builder,
};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_evm::OpEvmConfig;
use reth_optimism_forks::OpHardfork;
use reth_optimism_primitives::{OpBlock, OpTransactionSigned};
use reth_primitives_traits::SealedBlock;
use std::sync::Arc;

const FAILURES: &str = "reth_op_sdm_post_exec_validation_failure_total";
const LAGOON_ACTIVATION: u64 = 100;
const BLOCK_NUMBER: u64 = 7;

fn evm_config() -> OpEvmConfig {
    OpEvmConfig::optimism(Arc::new(
        OpChainSpecBuilder::default()
            .chain(10.into())
            .genesis(Genesis::default())
            .with_fork(OpHardfork::Lagoon, ForkCondition::Timestamp(LAGOON_ACTIVATION))
            .build(),
    ))
}

fn post_exec_tx(anchored_block_number: u64) -> OpTransactionSigned {
    OpTransactionSigned::PostExec(
        build_post_exec_tx(anchored_block_number, vec![SDMGasEntry { index: 0, gas_refund: 1 }])
            .seal_slow(),
    )
}

fn deposit_tx() -> OpTransactionSigned {
    OpTransactionSigned::Deposit(TxDeposit::default().seal_slow())
}

fn block(timestamp: u64, transactions: Vec<OpTransactionSigned>) -> SealedBlock<OpBlock> {
    SealedBlock::new_unhashed(Block {
        header: Header { number: BLOCK_NUMBER, timestamp, ..Default::default() },
        body: BlockBody { transactions, ..Default::default() },
    })
}

/// Reads the counter for one `reason` out of the exposition. `None` distinguishes a series that
/// was never registered from one reporting 0.
fn failures(exposition: &str, reason: &str) -> Option<f64> {
    let series = format!("{FAILURES}{{reason=\"{reason}\"}} ");
    exposition.lines().find_map(|line| line.strip_prefix(series.as_str())?.trim().parse().ok())
}

/// Each structural rule a block's post-exec transaction can break increments the counter under
/// its own `reason`, and a block that parses cleanly increments nothing.
///
/// One sequential test rather than parameterized cases, because they share this process's
/// recorder; as separate `#[test]` functions they would race.
#[test]
fn failure_counter_names_the_broken_rule() {
    // Install before the first record so every increment below lands on this recorder.
    let recorder = try_install_prometheus_recorder_with_builder(PrometheusBuilder::new())
        .unwrap_or_else(|_| install_prometheus_recorder());
    let read = |reason: &str| failures(&recorder.handle().render(), reason);
    let evm_config = evm_config();
    let active = LAGOON_ACTIVATION;

    evm_config
        .context_for_block(&block(active - 1, vec![post_exec_tx(BLOCK_NUMBER)]))
        .expect_err("post-exec tx before activation is rejected");
    assert_eq!(read("unexpected_post_exec_tx"), Some(1.0));

    evm_config
        .context_for_block(&block(
            active,
            vec![post_exec_tx(BLOCK_NUMBER), post_exec_tx(BLOCK_NUMBER)],
        ))
        .expect_err("two post-exec txs are rejected");
    assert_eq!(read("multiple_post_exec_txs"), Some(1.0));

    evm_config
        .context_for_block(&block(active, vec![post_exec_tx(BLOCK_NUMBER), deposit_tx()]))
        .expect_err("post-exec tx that is not last is rejected");
    assert_eq!(read("post_exec_tx_not_last"), Some(1.0));

    evm_config
        .context_for_block(&block(active, vec![post_exec_tx(BLOCK_NUMBER + 1)]))
        .expect_err("post-exec tx anchored to another block is rejected");
    assert_eq!(read("block_number_mismatch"), Some(1.0));

    evm_config
        .context_for_block(&block(active, vec![deposit_tx(), post_exec_tx(BLOCK_NUMBER)]))
        .expect("well-formed post-exec tx parses");
    evm_config
        .context_for_block(&block(active - 1, vec![deposit_tx()]))
        .expect("block without post-exec tx parses");
    let exposition = recorder.handle().render();
    assert_eq!(
        exposition
            .lines()
            .filter(|line| line.starts_with(FAILURES) && !line.starts_with("# "))
            .count(),
        4,
        "well-formed blocks register no further series: {exposition}"
    );
    for reason in [
        "unexpected_post_exec_tx",
        "multiple_post_exec_txs",
        "post_exec_tx_not_last",
        "block_number_mismatch",
    ] {
        assert_eq!(failures(&exposition, reason), Some(1.0), "{reason} unchanged by valid blocks");
    }
}
