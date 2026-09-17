//! Exposition test for `reth_op_sdm_effective`.
//!
//! Its own integration binary, holding exactly one test, because the metrics recorder is global:
//! every locally sequenced build in the process writes this gauge, and the assertions below read
//! back a single shared value. A second test here would race with it.

use alloy_consensus::Header;
use alloy_primitives::B64;
use metrics_exporter_prometheus::PrometheusBuilder;
use reth_basic_payload_builder::PayloadConfig;
use reth_chainspec::{ForkCondition, Hardfork};
use reth_node_metrics::recorder::{
    install_prometheus_recorder, try_install_prometheus_recorder_with_builder,
};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder, OpHardfork};
use reth_optimism_evm::OpEvmConfig;
use reth_optimism_payload_builder::{
    OpPayloadBuilderAttributes, builder::OpPayloadBuilderCtx, config::OpBuilderConfig,
};
use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
use reth_primitives_traits::SealedHeader;
use reth_revm::{database::StateProviderDatabase, db::State, test_utils::StateProviderTest};
use std::sync::Arc;

const EFFECTIVE: &str = "reth_op_sdm_effective";

type TestCtx = OpPayloadBuilderCtx<
    OpEvmConfig<OpChainSpec, OpPrimitives>,
    OpChainSpec,
    OpPayloadBuilderAttributes<OpTransactionSigned>,
>;

fn lagoon_at_genesis() -> Arc<OpChainSpec> {
    Arc::new(OpChainSpecBuilder::optimism_mainnet().lagoon_activated().build())
}

/// Rolls back every post-Regolith fork, Lagoon included, matching `sdm_admin`'s inactive spec.
fn lagoon_unscheduled() -> Arc<OpChainSpec> {
    Arc::new(OpChainSpecBuilder::optimism_mainnet().regolith_activated().build())
}

fn lagoon_at_timestamp(timestamp: u64) -> Arc<OpChainSpec> {
    Arc::new(
        OpChainSpecBuilder::optimism_mainnet()
            .lagoon_activated()
            .with_fork(OpHardfork::Lagoon.boxed(), ForkCondition::Timestamp(timestamp))
            .build(),
    )
}

/// `no_tx_pool` picks rebuilding a derived block over local sequencing; `opt_in` sets the operator
/// production gate; `block_timestamp` is the timestamp of the block being built.
fn ctx_on(
    chain_spec: Arc<OpChainSpec>,
    block_timestamp: u64,
    no_tx_pool: bool,
    opt_in: bool,
) -> TestCtx {
    let gas_limit = 1_000_000;
    let parent = SealedHeader::seal_slow(Header {
        gas_limit,
        number: 0,
        timestamp: 0,
        ..Default::default()
    });
    let attributes = OpPayloadBuilderAttributes {
        timestamp: block_timestamp,
        gas_limit: Some(gas_limit),
        no_tx_pool,
        eip_1559_params: Some(B64::ZERO),
        min_base_fee: Some(0),
        ..Default::default()
    };
    let builder_config = OpBuilderConfig::default();
    builder_config.operator_sdm_opt_in.set(opt_in);

    OpPayloadBuilderCtx {
        evm_config: OpEvmConfig::optimism(chain_spec.clone()),
        builder_config,
        chain_spec,
        config: PayloadConfig {
            parent_header: Arc::new(parent),
            parent_block_info: None,
            payload_id: attributes.id,
            attributes,
        },
        cancel: Default::default(),
        best_payload: None,
    }
}

/// Reads one unlabeled gauge out of the exposition. `None` distinguishes a series that was never
/// registered from one reporting 0, which a substring match cannot.
fn gauge(exposition: &str, name: &str) -> Option<f64> {
    exposition.lines().find_map(|line| line.strip_prefix(name)?.trim().parse().ok())
}

/// Resolving the post-exec mode for a locally sequenced block reports whether SDM production is
/// effective — both gates, so either one being off reports 0 — and the paths that are not this
/// node deciding what to produce leave the gauge alone.
///
/// One sequential test rather than parameterized cases, because they share this process's
/// recorder; as separate `#[test]` functions they would race. Each must-not-report case is
/// preceded by a build that establishes the value a regression would *change*: the derived-rebuild
/// case sets 1 first, because dropping the guard would report a non-`Produce` mode as 0, and the
/// witness case sets 0 first, because resolving through the reporting path would report 1.
#[test]
fn effective_gauge_reports_both_gates_while_sequencing() {
    // Install before the first record: the generated `Default` latches whichever recorder is
    // global on the process's first write, permanently.
    let recorder = try_install_prometheus_recorder_with_builder(PrometheusBuilder::new())
        .unwrap_or_else(|_| install_prometheus_recorder());
    let effective = || gauge(&recorder.handle().render(), EFFECTIVE);

    ctx_on(lagoon_at_genesis(), 1, /* no_tx_pool */ false, /* opt_in */ true)
        .post_exec_mode()
        .expect("mode resolves");
    assert_eq!(effective(), Some(1.0), "both gates on reports 1");

    ctx_on(lagoon_at_genesis(), 1, /* no_tx_pool */ false, /* opt_in */ false)
        .post_exec_mode()
        .expect("mode resolves");
    assert_eq!(effective(), Some(0.0), "operator gate off reports 0");

    ctx_on(lagoon_unscheduled(), 1, /* no_tx_pool */ false, /* opt_in */ true)
        .post_exec_mode()
        .expect("mode resolves");
    assert_eq!(effective(), Some(0.0), "protocol gate off reports 0");

    // `block_builder` serves witness and replay builds. Gauge is 0 going in and both gates are on,
    // so resolving through the reporting path would show up as 1.
    let state_provider = StateProviderTest::default();
    let mut db = State::builder()
        .with_database(StateProviderDatabase::new(&state_provider))
        .with_bundle_update()
        .build();
    ctx_on(lagoon_at_genesis(), 1, /* no_tx_pool */ false, /* opt_in */ true)
        .block_builder(&mut db)
        .expect("block builder can be created");
    assert_eq!(effective(), Some(0.0), "witness build does not report");

    ctx_on(lagoon_at_genesis(), 1, /* no_tx_pool */ false, /* opt_in */ true)
        .post_exec_mode()
        .expect("mode resolves");
    assert_eq!(effective(), Some(1.0), "re-armed for the derived-rebuild case");

    // Rebuilding a derived block reproduces what the chain already committed to, so it says
    // nothing about whether this operator wants to produce. Gauge is 1 going in, and this mode is
    // never `Produce`, so reporting it at all would show up as 0.
    ctx_on(lagoon_at_genesis(), 1, /* no_tx_pool */ true, /* opt_in */ true)
        .post_exec_mode()
        .expect("mode resolves");
    assert_eq!(effective(), Some(1.0), "rebuilding a derived block does not report");

    // The protocol gate turns on at a timestamp, with no event to hook, so resolving per build is
    // what keeps the gauge current without a timer.
    let activation = 1_000;
    let spec = lagoon_at_timestamp(activation);
    ctx_on(spec.clone(), activation - 1, /* no_tx_pool */ false, /* opt_in */ true)
        .post_exec_mode()
        .expect("mode resolves");
    assert_eq!(effective(), Some(0.0), "block before activation reports 0");

    ctx_on(spec, activation, /* no_tx_pool */ false, /* opt_in */ true)
        .post_exec_mode()
        .expect("mode resolves");
    assert_eq!(effective(), Some(1.0), "first block at activation reports 1");
}
