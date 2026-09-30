//! RPC integration tests.

use alloy_consensus::{SignableTransaction, TxEip1559};
use alloy_genesis::Genesis;
use alloy_network::{TxSignerSync, eip2718::Encodable2718};
use alloy_primitives::{Address, Bytes, TxKind};
use jsonrpsee::{
    RpcModule,
    core::client::{ClientT, Error as ClientError},
    rpc_params,
    server::ServerBuilder,
    types::error::METHOD_NOT_FOUND_CODE,
};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::wallet::Wallet;
use reth_network::types::NatResolver;
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{
    args::{NetworkArgs, RpcServerArgs, RpcStateCacheArgs},
    node_config::NodeConfig,
};
use reth_optimism_chainspec::{OP_SEPOLIA, OpChainSpecBuilder};
use reth_optimism_node::{OpNode, args::RollupArgs};
use reth_rpc_api::{EthConfigApiClient, servers::AdminApiServer};
use reth_rpc_eth_api::helpers::EthTransactions;
use reth_rpc_server_types::RpcModuleSelection;
use reth_tasks::Runtime;
use reth_transaction_pool::TransactionPool;
use std::sync::Arc;

// <https://github.com/paradigmxyz/reth/issues/19765>
#[tokio::test]
async fn test_admin_external_ip() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let exec = Runtime::test();

    let external_ip = "10.64.128.71".parse().unwrap();
    // Node setup
    let mut network_args = NetworkArgs::default()
        .with_unused_ports()
        .with_nat_resolver(NatResolver::ExternalIp(external_ip));
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    let node_config = NodeConfig::test()
        .map_chain(OP_SEPOLIA.clone())
        .with_network(network_args)
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http());

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(exec).node(OpNode::default()).launch().await?;

    let api = node.add_ons_handle.admin_api();

    let info = api.node_info().await.unwrap();

    assert_eq!(info.ip, external_ip);

    Ok(())
}

#[tokio::test]
async fn test_retain_forwarded_txs() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
    );
    let chain_id = chain_spec.chain_id();
    let signer = Wallet::new(1).with_chain_id(chain_id).wallet_gen().remove(0);

    let mut tx = TxEip1559 {
        chain_id,
        gas_limit: 21_000,
        max_fee_per_gas: 20_000_000_000,
        max_priority_fee_per_gas: 1_000_000_000,
        to: TxKind::Call(Address::random()),
        ..Default::default()
    };
    let signature = signer.sign_transaction_sync(&mut tx)?;
    let signed = tx.into_signed(signature);
    let tx_hash = *signed.hash();
    let raw_tx: Bytes =
        op_alloy_consensus::OpPooledTransaction::Eip1559(signed).encoded_2718().into();

    let server = ServerBuilder::default().build("127.0.0.1:0").await?;
    let sequencer_url = format!("http://{}", server.local_addr()?);
    let mut module = RpcModule::new(());
    module.register_method("eth_sendRawTransaction", move |_params, _ctx, _ext| {
        Ok::<_, jsonrpsee::types::ErrorObjectOwned>(tx_hash)
    })?;
    let _server_handle = server.start(module);

    let exec = Runtime::test();
    for retain_forwarded_txs in [false, true] {
        let mut network_args = NetworkArgs::default().with_unused_ports();
        network_args.discovery.discv5_port = Some(0);
        network_args.discovery.discv5_port_ipv6 = Some(0);
        let node_config = NodeConfig::test()
            .map_chain(chain_spec.clone())
            .with_network(network_args)
            .with_rpc(RpcServerArgs::default().with_unused_ports());
        let op_node = OpNode::new(RollupArgs {
            sequencer: Some(sequencer_url.clone()),
            retain_forwarded_txs,
            ..Default::default()
        });

        let NodeHandle { node, node_exit_future: _ } =
            NodeBuilder::new(node_config).testing_node(exec.clone()).node(op_node).launch().await?;

        let forwarded_hash =
            EthTransactions::send_raw_transaction(node.add_ons_handle.eth_api(), raw_tx.clone())
                .await?;
        assert_eq!(forwarded_hash, tx_hash);
        assert_eq!(node.pool.len(), usize::from(retain_forwarded_txs));
        assert_eq!(node.pool.get(&tx_hash).is_some(), retain_forwarded_txs);
    }

    Ok(())
}

#[tokio::test]
async fn test_eth_config_endpoint_exists() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let exec = Runtime::test();

    // Node setup
    let network = OP_SEPOLIA.clone();
    let mut network_args = NetworkArgs::default().with_unused_ports();
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    let node_config = NodeConfig::test()
        .map_chain(network.clone())
        .with_network(network_args)
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http());

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(exec).node(OpNode::default()).launch().await?;

    let client = node.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();
    let config = client.config().await?;
    assert_eq!(config.current.chain_id, network.clone().chain_id());

    Ok(())
}

/// Methods that re-execute a block to compute its EIP-7928 block access list.
const BLOCK_ACCESS_LIST_METHODS: [&str; 5] = [
    "eth_getBlockAccessListByBlockHash",
    "eth_getBlockAccessListByBlockNumber",
    "eth_getBlockAccessList",
    "eth_getBlockAccessListRaw",
    "debug_getRawBlockAccessList",
];

async fn assert_block_access_list_methods_unserved(client: &impl ClientT, transport: &str) {
    for method in BLOCK_ACCESS_LIST_METHODS {
        let res = client.request::<serde_json::Value, _>(method, rpc_params![]).await;
        match res {
            Err(ClientError::Call(err)) if err.code() == METHOD_NOT_FOUND_CODE => {}
            other => panic!("{method} is served over {transport}: {other:?}"),
        }
    }
}

#[tokio::test]
async fn test_block_access_list_methods_unserved() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let exec = Runtime::test();
    let mut network_args = NetworkArgs::default().with_unused_ports();
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    let node_config =
        NodeConfig::test().map_chain(OP_SEPOLIA.clone()).with_network(network_args).with_rpc(
            RpcServerArgs::default()
                .with_unused_ports()
                .with_http()
                .with_http_api(RpcModuleSelection::All),
        );

    let NodeHandle { node, node_exit_future: _ } =
        NodeBuilder::new(node_config).testing_node(exec).node(OpNode::default()).launch().await?;
    let handles = node.add_ons_handle.rpc_server_handles();

    assert_block_access_list_methods_unserved(&handles.rpc.http_client().unwrap(), "http").await;
    assert_block_access_list_methods_unserved(&handles.auth.http_client(), "auth").await;

    Ok(())
}

#[tokio::test]
async fn test_computed_bal_cache_options_rejected() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let exec = Runtime::test();
    for (flag, cache) in [
        (
            "--rpc-cache.prewarm-bals",
            RpcStateCacheArgs { prewarm_bals: Some(0), ..Default::default() },
        ),
        (
            "--rpc-cache.cache-computed-bals",
            RpcStateCacheArgs { cache_computed_bals: true, ..Default::default() },
        ),
    ] {
        let mut network_args = NetworkArgs::default().with_unused_ports();
        network_args.discovery.discv5_port = Some(0);
        network_args.discovery.discv5_port_ipv6 = Some(0);
        let mut rpc = RpcServerArgs::default().with_unused_ports().with_http();
        rpc.rpc_state_cache = cache;
        let node_config = NodeConfig::test()
            .map_chain(OP_SEPOLIA.clone())
            .with_network(network_args)
            .with_rpc(rpc);

        let launched = NodeBuilder::new(node_config)
            .testing_node(exec.clone())
            .node(OpNode::default())
            .launch()
            .await;
        let err = launched.err().unwrap_or_else(|| panic!("node launched with {flag}"));
        assert!(err.to_string().contains(flag), "unexpected error for {flag}: {err:#}");
    }

    Ok(())
}
