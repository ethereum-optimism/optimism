//! Lagoon-at-genesis fixtures and SDM node helpers shared by the integration tests.

use alloy_genesis::Genesis;
use alloy_primitives::B64;
use alloy_rpc_types_trace::{
    common::TraceResult,
    geth::{
        GethDebugBuiltInTracerType, GethDebugTracerConfig, GethDebugTracerType,
        GethDebugTracingOptions, GethTrace, PreStateConfig,
    },
};
use op_alloy_consensus::encode_jovian_extra_data;
use reth_chainspec::BaseFeeParams;
use reth_e2e_test_utils::{NodeHelperType, node::NodeTestContext};
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{
    args::{NetworkArgs, RpcServerArgs},
    node_config::NodeConfig,
};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_node::{OpNode, args::RollupArgs, utils::optimism_payload_attributes};
use reth_optimism_payload_builder::OpPayloadAttrs;
use reth_rpc_server_types::RpcModuleSelection;
use reth_tasks::Runtime;
use std::sync::Arc;

/// Gas used by a plain transfer before the SDM refund.
pub(crate) const TRANSFER_GAS: u64 = 21_000;

/// Gas used before the SDM refund by a deployment of the one-byte init code `STOP`: intrinsic
/// 21000, CREATE 32000, one zero calldata byte 4, one init code word 2, and no code deposit.
pub(crate) const DEPLOY_GAS: u64 = 53_006;

/// Builds a Lagoon-at-genesis spec that funds the e2e test wallet.
pub(crate) fn chain_spec() -> eyre::Result<Arc<OpChainSpec>> {
    chain_spec_with(|_| {})
}

/// Builds a Lagoon-at-genesis spec that funds the e2e test wallet, with `customize` applied to the
/// genesis before the spec is built.
pub(crate) fn chain_spec_with(
    customize: impl FnOnce(&mut Genesis),
) -> eyre::Result<Arc<OpChainSpec>> {
    let mut genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    // Lagoon is active at genesis, so the first child's base fee uses Jovian extra data.
    genesis.extra_data = encode_jovian_extra_data(B64::ZERO, BaseFeeParams::optimism(), 0)?;
    genesis.base_fee_per_gas = Some(1_000_000_000);
    customize(&mut genesis);
    Ok(Arc::new(OpChainSpecBuilder::optimism_sepolia().genesis(genesis).lagoon_activated().build()))
}

/// Payload attributes carrying the Jovian fields a Lagoon block requires.
pub(crate) const fn payload_attributes(timestamp: u64) -> OpPayloadAttrs {
    let mut attributes = optimism_payload_attributes(timestamp);
    attributes.0.eip_1559_params = Some(B64::ZERO);
    attributes.0.min_base_fee = Some(0);
    attributes
}

/// Returns a test node config serving every RPC module over HTTP and `WebSocket` on unused ports.
pub(crate) fn rpc_node_config(chain_spec: &Arc<OpChainSpec>) -> NodeConfig<OpChainSpec> {
    let network_args = NetworkArgs::default().with_unused_ports();
    NodeConfig::test().map_chain(chain_spec.clone()).with_network(network_args).with_rpc(
        RpcServerArgs::default()
            .with_unused_ports()
            .with_http()
            // `debug_*` is not part of the default HTTP module selection.
            .with_http_api(RpcModuleSelection::All)
            // debug_subscribe is WebSocket-only.
            .with_ws()
            .with_ws_api(RpcModuleSelection::All),
    )
}

/// Launches a Lagoon node that opts into SDM and sequences one refund per regular transaction.
pub(crate) async fn launch_sdm_node(
    chain_spec: &Arc<OpChainSpec>,
) -> eyre::Result<NodeHelperType<OpNode>> {
    launch_sdm_node_with_attributes(chain_spec, payload_attributes).await
}

/// Like [`launch_sdm_node`], with the payload attributes supplied per block timestamp.
pub(crate) async fn launch_sdm_node_with_attributes(
    chain_spec: &Arc<OpChainSpec>,
    attributes: impl Fn(u64) -> OpPayloadAttrs + Send + Sync + 'static,
) -> eyre::Result<NodeHelperType<OpNode>> {
    let op_node = OpNode::new(RollupArgs {
        operator_sdm_opt_in: true,
        testing_sdm_fixed_policy: Some(None),
        ..Default::default()
    });
    let NodeHandle { node, node_exit_future: _ } = NodeBuilder::new(rpc_node_config(chain_spec))
        .testing_node(Runtime::test())
        .node(op_node)
        .launch()
        .await?;
    NodeTestContext::new(node, attributes).await
}

/// Returns tracing options selecting the prestate tracer with the given config.
pub(crate) fn prestate_tracer(config: PreStateConfig) -> GethDebugTracingOptions {
    GethDebugTracingOptions::default()
        .with_tracer(GethDebugTracerType::BuiltInTracer(GethDebugBuiltInTracerType::PreStateTracer))
        .with_config(GethDebugTracerConfig::from(config))
}

/// Returns the gas reported by a successful default (struct logger) trace.
pub(crate) fn default_trace_gas(trace: &TraceResult<GethTrace, String>) -> u64 {
    match trace {
        TraceResult::Success { result: GethTrace::Default(frame), .. } => frame.gas,
        other => panic!("unexpected trace shape: {other:?}"),
    }
}
