use crate::{OpBuiltPayload, OpNode as OtherOpNode};
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
///
/// # Errors
///
/// Returns any [`eyre::Report`] produced while building the chain spec or spawning the test
/// nodes via [`reth_e2e_test_utils::setup_engine`].
///
/// # Panics
///
/// Panics if the bundled genesis JSON cannot be parsed (test-utils bug).
pub async fn setup(num_nodes: usize) -> eyre::Result<(Vec<OpNode>, Wallet)> {
    let genesis: Genesis =
        serde_json::from_str(include_str!("../tests/assets/genesis.json")).unwrap();
    #[allow(
        clippy::default_trait_access,
        reason = "avoid adding reth_engine_primitives dep solely for TreeConfig default"
    )]
    reth_e2e_test_utils::setup_engine(
        num_nodes,
        Arc::new(OpChainSpecBuilder::base_mainnet().genesis(genesis).ecotone_activated().build()),
        false,
        Default::default(),
        optimism_payload_attributes,
    )
    .await
}

/// Advance the chain with sequential payloads returning them in the end.
///
/// # Errors
///
/// Forwards any [`eyre::Report`] produced while advancing the test node `length` blocks.
#[allow(
    clippy::future_not_send,
    reason = "OpNode helper is `!Send` by design in reth test-utils"
)]
#[allow(
    clippy::significant_drop_tightening,
    reason = "wallet mutex must span the whole tx-construction future"
)]
pub async fn advance_chain(
    length: usize,
    node: &mut OpNode,
    wallet: Arc<Mutex<Wallet>>,
) -> eyre::Result<Vec<OpBuiltPayload>> {
    node.advance(length.try_into().unwrap_or(u64::MAX), |_| {
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
#[must_use]
pub const fn optimism_payload_attributes(timestamp: u64) -> OpPayloadAttrs {
    OpPayloadAttrs(OpPayloadAttributes {
        payload_attributes: alloy_rpc_types_engine::PayloadAttributes {
            timestamp,
            prev_randao: B256::ZERO,
            suggested_fee_recipient: Address::ZERO,
            withdrawals: Some(vec![]),
            parent_beacon_block_root: Some(B256::ZERO),
            slot_number: None,
        },
        transactions: None,
        no_tx_pool: None,
        gas_limit: Some(30_000_000),
        eip_1559_params: None,
        min_base_fee: None,
    })
}
