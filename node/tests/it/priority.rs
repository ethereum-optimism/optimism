//! Node builder test that customizes priority of transactions in the block.

use alloy_consensus::{transaction::Recovered, SignableTransaction, Transaction, TxEip1559};
use alloy_genesis::Genesis;
use alloy_network::TxSignerSync;
use alloy_primitives::{Address, ChainId, TxKind};
use reth_chainspec::EthChainSpec;
use reth_db::test_utils::create_test_rw_db_with_path;
use reth_e2e_test_utils::{
    node::NodeTestContext, transaction::TransactionTestContext, wallet::Wallet,
};
use reth_node_api::FullNodeTypes;
use reth_node_builder::{
    components::{BasicPayloadServiceBuilder, ComponentsBuilder},
    EngineNodeLauncher, NodeBuilder, NodeConfig,
};
use reth_node_core::args::DatadirArgs;
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{
    args::RollupArgs,
    node::{
        OpAddOns, OpConsensusBuilder, OpExecutorBuilder, OpNetworkBuilder, OpNodeTypes,
        OpPayloadBuilder, OpPoolBuilder,
    },
    txpool::OpPooledTransaction,
    utils::optimism_payload_attributes,
    OpNode,
};
use reth_optimism_payload_builder::builder::OpPayloadTransactions;
use reth_payload_util::{
    BestPayloadTransactions, PayloadTransactions, PayloadTransactionsChain,
    PayloadTransactionsFixed,
};
use reth_provider::providers::BlockchainProvider;
use reth_tasks::TaskManager;
use reth_transaction_pool::PoolTransaction;
use std::sync::Arc;
use tokio::sync::Mutex;

#[derive(Clone, Debug)]
struct CustomTxPriority {
    chain_id: ChainId,
}

impl OpPayloadTransactions<OpPooledTransaction> for CustomTxPriority {
    fn best_transactions<Pool>(
        &self,
        pool: Pool,
        attr: reth_transaction_pool::BestTransactionsAttributes,
    ) -> impl PayloadTransactions<Transaction = OpPooledTransaction>
    where
        Pool: reth_transaction_pool::TransactionPool<Transaction = OpPooledTransaction>,
    {
        // Block composition:
        // 1. Best transactions from the pool (up to 250k gas)
        // 2. End-of-block transaction created by the node (up to 100k gas)

        // End of block transaction should send a 0-value transfer to a random address.
        let sender = Wallet::default().inner;
        let mut end_of_block_tx = TxEip1559 {
            chain_id: self.chain_id,
            nonce: 1, // it will be 2nd tx after L1 block info tx that uses the same sender
            gas_limit: 21000,
            max_fee_per_gas: 20e9 as u128,
            to: TxKind::Call(Address::random()),
            value: 0.try_into().unwrap(),
            ..Default::default()
        };
        let signature = sender.sign_transaction_sync(&mut end_of_block_tx).unwrap();
        let end_of_block_tx = OpPooledTransaction::from_pooled(Recovered::new_unchecked(
            op_alloy_consensus::OpPooledTransaction::Eip1559(
                end_of_block_tx.into_signed(signature),
            ),
            sender.address(),
        ));

        PayloadTransactionsChain::new(
            BestPayloadTransactions::new(pool.best_transactions_with_attributes(attr)),
            // Allow 250k gas for the transactions from the pool
            Some(250_000),
            PayloadTransactionsFixed::single(end_of_block_tx),
            // Allow 100k gas for the end-of-block transaction
            Some(100_000),
        )
    }
}

/// Builds the node with custom transaction priority service within default payload builder.
fn build_components<Node>(
    chain_id: ChainId,
) -> ComponentsBuilder<
    Node,
    OpPoolBuilder,
    BasicPayloadServiceBuilder<OpPayloadBuilder<CustomTxPriority>>,
    OpNetworkBuilder,
    OpExecutorBuilder,
    OpConsensusBuilder,
>
where
    Node: FullNodeTypes<Types: OpNodeTypes>,
{
    let RollupArgs { disable_txpool_gossip, compute_pending_block, discovery_v4, .. } =
        RollupArgs::default();
    ComponentsBuilder::default()
        .node_types::<Node>()
        .pool(OpPoolBuilder::default())
        .executor(OpExecutorBuilder::default())
        .payload(BasicPayloadServiceBuilder::new(
            OpPayloadBuilder::new(compute_pending_block)
                .with_transactions(CustomTxPriority { chain_id }),
        ))
        .network(OpNetworkBuilder { disable_txpool_gossip, disable_discovery_v4: !discovery_v4 })
        .consensus(OpConsensusBuilder::default())
}

#[tokio::test]
async fn test_custom_block_priority_config() {
    reth_tracing::init_test_tracing();

    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json")).unwrap();
    let chain_spec =
        Arc::new(OpChainSpecBuilder::base_mainnet().genesis(genesis).ecotone_activated().build());

    // This wallet is going to send:
    // 1. L1 block info tx
    // 2. End-of-block custom tx
    let wallet = Arc::new(Mutex::new(Wallet::default().with_chain_id(chain_spec.chain().into())));

    // Configure and launch the node.
    let config = NodeConfig::new(chain_spec).with_datadir_args(DatadirArgs {
        datadir: reth_db::test_utils::tempdir_path().into(),
        ..Default::default()
    });
    let db = create_test_rw_db_with_path(
        config
            .datadir
            .datadir
            .unwrap_or_chain_default(config.chain.chain(), config.datadir.clone())
            .db(),
    );
    let tasks = TaskManager::current();
    let node_handle = NodeBuilder::new(config.clone())
        .with_database(db)
        .with_types_and_provider::<OpNode, BlockchainProvider<_>>()
        .with_components(build_components(config.chain.chain_id()))
        .with_add_ons(OpAddOns::default())
        .launch_with_fn(|builder| {
            let launcher = EngineNodeLauncher::new(
                tasks.executor(),
                builder.config.datadir(),
                Default::default(),
            );
            builder.launch_with(launcher)
        })
        .await
        .expect("Failed to launch node");

    // Advance the chain with a single block.
    let block_payloads = NodeTestContext::new(node_handle.node, optimism_payload_attributes)
        .await
        .unwrap()
        .advance(1, |_| {
            let wallet = wallet.clone();
            Box::pin(async move {
                let mut wallet = wallet.lock().await;
                let tx_fut = TransactionTestContext::optimism_l1_block_info_tx(
                    wallet.chain_id,
                    wallet.inner.clone(),
                    // This doesn't matter in the current test (because it's only one block),
                    // but make sure you're not reusing the nonce from end-of-block tx
                    // if they have the same signer.
                    wallet.inner_nonce * 2,
                );
                wallet.inner_nonce += 1;
                tx_fut.await
            })
        })
        .await
        .unwrap();
    assert_eq!(block_payloads.len(), 1);
    let block_payload = block_payloads.first().unwrap();
    let block_payload = block_payload.block().clone();
    assert_eq!(block_payload.body().transactions.len(), 2); // L1 block info tx + end-of-block custom tx

    // Check that last transaction in the block looks like a transfer to a random address.
    let end_of_block_tx = block_payload.body().transactions.last().unwrap();
    let Some(tx) = end_of_block_tx.as_eip1559() else {
        panic!("expected EIP-1559 transaction");
    };
    assert_eq!(tx.tx().nonce(), 1);
    assert_eq!(tx.tx().gas_limit(), 21_000);
    assert!(tx.tx().input().is_empty());
}
