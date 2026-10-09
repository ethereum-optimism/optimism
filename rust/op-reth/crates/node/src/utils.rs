use crate::{OpBuiltPayload, OpNode as OtherOpNode};
use alloy_consensus::BlockHeader;
use alloy_genesis::Genesis;
use alloy_primitives::{Address, B256};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use reth_e2e_test_utils::{
    NodeHelperType, TmpDB, transaction::TransactionTestContext, wallet::Wallet,
};
use reth_node_api::NodeTypesWithDBAdapter;
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_payload_builder::OpPayloadAttrs;
use reth_provider::providers::BlockchainProvider;
use std::sync::Arc;
use tokio::sync::Mutex;

/// Optimism Node Helper type
pub(crate) type OpNode =
    NodeHelperType<OtherOpNode, BlockchainProvider<NodeTypesWithDBAdapter<OtherOpNode, TmpDB>>>;

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
        Default::default(),
        optimism_payload_attributes,
    )
    .await
}

/// Advance the chain with sequential payloads returning them in the end.
///
/// Unlike `NodeTestContext::advance`, which expects the injected transaction to open the block,
/// this expects it right after the L1 attributes deposit.
pub async fn advance_chain(
    length: usize,
    node: &mut OpNode,
    wallet: Arc<Mutex<Wallet>>,
) -> eyre::Result<Vec<OpBuiltPayload>> {
    let mut chain = Vec::with_capacity(length);
    for _ in 0..length {
        let raw_tx = {
            let mut wallet = wallet.lock().await;
            let tx_fut = TransactionTestContext::optimism_l1_block_info_tx(
                wallet.chain_id,
                wallet.inner.clone(),
                wallet.inner_nonce,
            );
            wallet.inner_nonce += 1;
            tx_fut.await
        };
        let tx_hash = node.rpc.inject_tx(raw_tx).await?;
        let payload = node.advance_block().await?;
        let block = payload.block();
        assert_eq!(block.body().transactions.get(1).map(|tx| tx.tx_hash()), Some(tx_hash));
        node.wait_block(block.number(), block.hash(), false).await?;
        chain.push(payload);
    }
    Ok(chain)
}

/// Helper function to create a new eth payload attributes. Like a rollup node, it opens the block
/// with an L1 attributes deposit, which block validation requires.
pub fn optimism_payload_attributes(timestamp: u64) -> OpPayloadAttrs {
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
        transactions: Some(vec![crate::node::TX_SET_L1_BLOCK.into()]),
        no_tx_pool: None,
        gas_limit: Some(30_000_000),
        eip_1559_params: None,
        min_base_fee: None,
    })
}
