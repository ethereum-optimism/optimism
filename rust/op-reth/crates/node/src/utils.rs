use crate::{OpBuiltPayload, OpNode as OtherOpNode, init_op_engine_defaults};
use alloy_genesis::Genesis;
use alloy_primitives::{Address, B256};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use reth_e2e_test_utils::{
    NodeHelperType, TmpDB, transaction::TransactionTestContext, wallet::Wallet,
};
use reth_node_api::{NodeTypesWithDBAdapter, TreeConfig};
use reth_node_core::node_config::NodeConfig;
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_payload_builder::OpPayloadAttrs;
use reth_provider::providers::BlockchainProvider;
use reth_tracing::tracing::warn;
use std::sync::Arc;
use tokio::sync::Mutex;

/// Optimism Node Helper type
pub(crate) type OpNode =
    NodeHelperType<OtherOpNode, BlockchainProvider<NodeTypesWithDBAdapter<OtherOpNode, TmpDB>>>;

/// Puts the op-reth engine defaults in effect, warning when another `NodeConfig` locked them
/// masked first.
fn init_op_engine_defaults_or_warn() {
    if let Err(err) = init_op_engine_defaults() {
        warn!(target: "reth::cli", %err, "Engine defaults are masked");
    }
}

/// Builds a [`NodeConfig`] for `chain_spec` on unused ports with the op-reth engine defaults in
/// effect, so a node launched from it passes the state masking guard.
pub fn node_config(chain_spec: impl Into<Arc<OpChainSpec>>) -> NodeConfig<OpChainSpec> {
    init_op_engine_defaults_or_warn();
    NodeConfig::new(chain_spec.into()).with_unused_ports()
}

/// The default [`TreeConfig`] with state masking off, which the state masking guard accepts.
///
/// Also puts the op-reth engine defaults in effect, so a `NodeConfig` built after this call
/// runs unmasked.
pub fn unmasked_tree_config() -> TreeConfig {
    init_op_engine_defaults_or_warn();
    TreeConfig::default().with_num_state_masking_blocks(0)
}

/// Creates the initial setup with `num_nodes` of the node config, started and connected.
pub async fn setup(num_nodes: usize) -> eyre::Result<(Vec<OpNode>, Wallet)> {
    let genesis: Genesis =
        serde_json::from_str(include_str!("../tests/assets/genesis.json")).unwrap();
    reth_e2e_test_utils::setup_engine(
        num_nodes,
        Arc::new(
            OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
        ),
        false,
        unmasked_tree_config(),
        optimism_payload_attributes,
    )
    .await
}

/// Advance the chain with sequential payloads returning them in the end.
pub async fn advance_chain(
    length: usize,
    node: &mut OpNode,
    wallet: Arc<Mutex<Wallet>>,
) -> eyre::Result<Vec<OpBuiltPayload>> {
    node.advance(length as u64, |_| {
        let wallet = wallet.clone();
        Box::pin(async move {
            let mut wallet = wallet.lock().await;
            let tx_fut = TransactionTestContext::optimism_l1_block_info_tx(
                wallet.chain_id,
                wallet.inner.clone(),
                wallet.inner_nonce,
            );
            wallet.inner_nonce += 1;
            tx_fut.await
        })
    })
    .await
}

/// Helper function to create a new eth payload attributes
pub const fn optimism_payload_attributes(timestamp: u64) -> OpPayloadAttrs {
    OpPayloadAttrs(OpPayloadAttributes {
        payload_attributes: alloy_rpc_types_engine::PayloadAttributes {
            timestamp,
            prev_randao: B256::ZERO,
            suggested_fee_recipient: Address::ZERO,
            withdrawals: Some(vec![]),
            parent_beacon_block_root: Some(B256::ZERO),
            slot_number: None,
            target_gas_limit: None,
        },
        transactions: None,
        no_tx_pool: None,
        gas_limit: Some(30_000_000),
        eip_1559_params: None,
        min_base_fee: None,
    })
}
