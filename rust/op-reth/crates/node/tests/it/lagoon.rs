//! Lagoon-at-genesis fixtures and SDM node helpers shared by the integration tests.

use alloy_genesis::Genesis;
use alloy_network::eip2718::Encodable2718;
use alloy_primitives::{B64, Bytes, TxKind};
use alloy_rpc_types_eth::{TransactionInput, TransactionRequest};
use alloy_rpc_types_trace::{
    common::TraceResult,
    geth::{
        GethDebugBuiltInTracerType, GethDebugTracerConfig, GethDebugTracerType,
        GethDebugTracingOptions, GethTrace, PreStateConfig,
    },
};
use jsonrpsee::http_client::HttpClient;
use op_alloy_consensus::encode_jovian_extra_data;
use reth_chainspec::BaseFeeParams;
use reth_e2e_test_utils::{
    NodeHelperType, node::NodeTestContext, transaction::TransactionTestContext, wallet::Wallet,
};
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{args::RpcServerArgs, node_config::NodeConfig};
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
    NodeConfig::test().map_chain(chain_spec.clone()).with_rpc(
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

/// Launches a node on `chain_spec` with [`rpc_node_config`].
pub(crate) async fn launch_node(
    chain_spec: &Arc<OpChainSpec>,
    rollup_args: RollupArgs,
) -> eyre::Result<NodeHelperType<OpNode>> {
    launch_node_with_attributes(chain_spec, rollup_args, payload_attributes).await
}

/// Like [`launch_node`], with the payload attributes supplied per block timestamp.
pub(crate) async fn launch_node_with_attributes(
    chain_spec: &Arc<OpChainSpec>,
    rollup_args: RollupArgs,
    attributes: impl Fn(u64) -> OpPayloadAttrs + Send + Sync + 'static,
) -> eyre::Result<NodeHelperType<OpNode>> {
    let NodeHandle { node, node_exit_future: _ } = NodeBuilder::new(rpc_node_config(chain_spec))
        .testing_node(Runtime::test())
        .node(OpNode::new(rollup_args))
        .launch()
        .await?;
    NodeTestContext::new(node, attributes).await
}

/// Rollup arguments that opt into SDM and sequence one refund per regular transaction.
pub(crate) fn sdm_rollup_args() -> RollupArgs {
    RollupArgs {
        operator_sdm_opt_in: true,
        testing_sdm_fixed_policy: Some(None),
        ..Default::default()
    }
}

/// The HTTP client of a node launched with [`rpc_node_config`].
pub(crate) fn http_client(node: &NodeHelperType<OpNode>) -> HttpClient {
    node.inner.add_ons_handle.rpc_server_handles().rpc.http_client().expect("HTTP RPC on")
}

/// An EIP-1559 transaction from `wallet` paying `fee` per gas as both fee cap and tip, signed and
/// encoded for `eth_sendRawTransaction` or forced payload attributes.
pub(crate) async fn signed_transaction(
    wallet: &Wallet,
    nonce: u64,
    gas_limit: u64,
    fee: u128,
    to: TxKind,
    input: Bytes,
) -> Bytes {
    let request = TransactionRequest {
        chain_id: Some(wallet.chain_id),
        gas: Some(gas_limit),
        input: TransactionInput::new(input),
        max_fee_per_gas: Some(fee),
        max_priority_fee_per_gas: Some(fee),
        nonce: Some(nonce),
        to: Some(to),
        ..Default::default()
    };
    TransactionTestContext::sign_tx(wallet.inner.clone(), request).await.encoded_2718().into()
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
