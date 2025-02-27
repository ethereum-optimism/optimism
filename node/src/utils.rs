use crate::{OpBuiltPayload, OpNode as OtherOpNode, OpPayloadBuilderAttributes};
use alloy_genesis::Genesis;
use alloy_primitives::{Address, B256};
use alloy_rpc_types_engine::PayloadAttributes;
use reth_e2e_test_utils::{
    transaction::TransactionTestContext, wallet::Wallet, NodeHelperType, TmpDB,
};
use reth_node_api::NodeTypesWithDBAdapter;
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_payload_builder::EthPayloadBuilderAttributes;
use reth_provider::providers::BlockchainProvider;
use reth_tasks::TaskManager;
use std::sync::Arc;
use tokio::sync::Mutex;

/// Optimism Node Helper type
pub(crate) type OpNode =
    NodeHelperType<OtherOpNode, BlockchainProvider<NodeTypesWithDBAdapter<OtherOpNode, TmpDB>>>;

/// Creates the initial setup with `num_nodes` of the node config, started and connected.
pub async fn setup(num_nodes: usize) -> eyre::Result<(Vec<OpNode>, TaskManager, Wallet)> {
    let genesis: Genesis =
        serde_json::from_str(include_str!("../tests/assets/genesis.json")).unwrap();
    reth_e2e_test_utils::setup_engine(
        num_nodes,
        Arc::new(OpChainSpecBuilder::base_mainnet().genesis(genesis).ecotone_activated().build()),
        false,
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
pub fn optimism_payload_attributes<T>(timestamp: u64) -> OpPayloadBuilderAttributes<T> {
    let attributes = PayloadAttributes {
        timestamp,
        prev_randao: B256::ZERO,
        suggested_fee_recipient: Address::ZERO,
        withdrawals: Some(vec![]),
        parent_beacon_block_root: Some(B256::ZERO),
    };

    OpPayloadBuilderAttributes {
        payload_attributes: EthPayloadBuilderAttributes::new(B256::ZERO, attributes),
        transactions: vec![],
        no_tx_pool: false,
        gas_limit: Some(30_000_000),
        eip_1559_params: None,
    }
}
