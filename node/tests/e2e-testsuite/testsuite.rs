use alloy_primitives::{Address, B256};
use eyre::Result;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use reth_e2e_test_utils::testsuite::{
    actions::AssertMineBlock,
    setup::{NetworkSetup, Setup},
    TestBuilder,
};
use reth_optimism_chainspec::{OpChainSpecBuilder, OP_MAINNET};
use reth_optimism_node::{OpEngineTypes, OpNode};
use std::sync::Arc;

#[tokio::test]
async fn test_testsuite_op_assert_mine_block() -> Result<()> {
    reth_tracing::init_test_tracing();

    let setup = Setup::default()
        .with_chain_spec(Arc::new(
            OpChainSpecBuilder::default()
                .chain(OP_MAINNET.chain)
                .genesis(serde_json::from_str(include_str!("../assets/genesis.json")).unwrap())
                .build()
                .into(),
        ))
        .with_network(NetworkSetup::single_node());

    let test =
        TestBuilder::new().with_setup(setup).with_action(AssertMineBlock::<OpEngineTypes>::new(
            0,
            vec![],
            Some(B256::ZERO),
            // TODO: refactor once we have actions to generate payload attributes.
            OpPayloadAttributes {
                payload_attributes: alloy_rpc_types_engine::PayloadAttributes {
                    timestamp: std::time::SystemTime::now()
                        .duration_since(std::time::UNIX_EPOCH)
                        .unwrap()
                        .as_secs(),
                    prev_randao: B256::random(),
                    suggested_fee_recipient: Address::random(),
                    withdrawals: None,
                    parent_beacon_block_root: None,
                },
                transactions: None,
                no_tx_pool: None,
                eip_1559_params: None,
                gas_limit: Some(30_000_000),
            },
        ));

    test.run::<OpNode>().await?;

    Ok(())
}
