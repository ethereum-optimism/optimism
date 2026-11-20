//! Tests that the pool a node builds knows the limits its payload builder works with.

use crate::launcher::launch;
use alloy_consensus::{SignableTransaction, TxEip1559, transaction::Recovered};
use alloy_genesis::Genesis;
use alloy_network::eip2718::Encodable2718;
use alloy_primitives::{Address, Signature, TxKind};
use reth_node_builder::NodeConfig;
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::OpNode;
use reth_optimism_primitives::OpTransactionSigned;
use reth_optimism_txpool::OpPooledTransaction;
use reth_transaction_pool::{
    TransactionOrigin, TransactionPool,
    error::{InvalidPoolTransactionError, PoolErrorKind},
};
use std::sync::Arc;
use tempfile::TempDir;

/// The gas limit in the test chain's genesis, unlike reth's 30M default.
const GENESIS_GAS_LIMIT: u64 = 20_000_000;

/// A node's pool is built from the same limits as its payload builder.
#[tokio::test]
async fn pool_knows_the_node_limits() {
    reth_tracing::init_test_tracing();

    let mut genesis: Genesis =
        serde_json::from_str(include_str!("../assets/genesis.json")).unwrap();
    genesis.gas_limit = GENESIS_GAS_LIMIT;
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
    );
    let mut config = NodeConfig::new(chain_spec).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    let op_node = OpNode::default();
    let datadir = TempDir::new().unwrap();
    let node_handle = launch(config, reth_tasks::Runtime::test(), &datadir, op_node.clone())
        .await
        .expect("Failed to launch node");
    let pool = &node_handle.node.pool;

    // `miner_setGasLimit` writes this handle after startup, and the pool must see it.
    op_node.gas_limit_config.set_gas_limit(100_000);
    let tx: OpTransactionSigned = TxEip1559 {
        gas_limit: 200_000,
        to: TxKind::Call(Address::with_last_byte(0x42)),
        ..Default::default()
    }
    .into_signed(Signature::test_signature())
    .into();
    let encoded_len = tx.encode_2718_len();
    let tx = OpPooledTransaction::new(
        Recovered::new_unchecked(tx, Address::with_last_byte(1)),
        encoded_len,
    );
    let err = pool.add_transaction(TransactionOrigin::External, tx).await.unwrap_err();
    assert!(
        matches!(
            err.kind,
            PoolErrorKind::InvalidTransaction(InvalidPoolTransactionError::MaxTxGasLimitExceeded(
                ..
            ))
        ),
        "got {err:?}"
    );

    // Until its first canonical update, reth's validator would otherwise assume 30M.
    assert_eq!(pool.validator().validator().block_gas_limit(), GENESIS_GAS_LIMIT);
}
