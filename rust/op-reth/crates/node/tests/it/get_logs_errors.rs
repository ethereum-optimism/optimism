//! `eth_getLogs` must answer an error, not a silently empty or partial array, for a range the
//! node cannot serve in full: a `fromBlock` past the head and a range whose receipts the node
//! pruned. An indexer that polls `lastSeen + 1`, or reads a range whose receipts are gone, sees
//! the error instead of mistaking the missing logs for an empty range.

use crate::engine_driver::{EngineDriver, wait_until};
use alloy_consensus::{SignableTransaction, TxEip1559};
use alloy_genesis::Genesis;
use alloy_network::{TxSignerSync, eip2718::Encodable2718};
use alloy_primitives::{Address, Bytes, TxKind};
use alloy_rpc_types_eth::{Filter, Log};
use alloy_signer_local::PrivateKeySigner;
use jsonrpsee::{
    core::client::{ClientT, Error as ClientError},
    http_client::HttpClient,
    rpc_params,
    types::error::INVALID_PARAMS_CODE,
    ws_client::WsClient,
};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::wallet::Wallet;
use reth_node_builder::{NodeBuilder, NodeConfig, NodeHandle, rpc::RethRpcServerHandles};
use reth_node_core::args::{PruningArgs, RpcServerArgs};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::OpNode;
use reth_provider::{ChainSpecProvider, ReceiptProvider};
use reth_tasks::Runtime;
use rstest::rstest;
use std::sync::Arc;

/// The receipts of every block below this one are pruned; the blocks from it on keep theirs.
const RECEIPTS_PRUNED_BEFORE: u64 = 4;
/// The height the chain is built to. The head carries the one log a served range answers with.
const CHAIN_HEIGHT: u64 = 6;
/// Init code that emits an empty `LOG0` and deploys no runtime code: `PUSH1 0 PUSH1 0 LOG0 STOP`.
const LOG_EMITTING_INIT_CODE: [u8; 6] = [0x60, 0x00, 0x60, 0x00, 0xa0, 0x00];
/// EIP-4444's code for history the node no longer holds.
const PRUNED_HISTORY_UNAVAILABLE_CODE: i32 = 4444;

#[derive(Clone, Copy, Debug)]
enum Transport {
    Http,
    Ws,
}

/// A client of the node's user RPC over one transport.
enum RpcClient {
    Http(HttpClient),
    Ws(WsClient),
}

impl RpcClient {
    async fn connect(handles: &RethRpcServerHandles, transport: Transport) -> Self {
        match transport {
            Transport::Http => Self::Http(handles.rpc.http_client().expect("http is served")),
            Transport::Ws => Self::Ws(handles.rpc.ws_client().await.expect("ws is served")),
        }
    }

    async fn get_logs(&self, filter: Filter) -> Result<Vec<Log>, ClientError> {
        match self {
            Self::Http(client) => client.request("eth_getLogs", rpc_params![filter]).await,
            Self::Ws(client) => client.request("eth_getLogs", rpc_params![filter]).await,
        }
    }
}

/// A node whose chain reaches `CHAIN_HEIGHT` with the receipts below `RECEIPTS_PRUNED_BEFORE`
/// pruned and one log, from a contract created at the head, in the receipts it kept.
struct PrunedReceiptsNode<N> {
    handles: RethRpcServerHandles,
    log_address: Address,
    /// Keeps the node behind the RPC handles running.
    _node: N,
}

impl<N> PrunedReceiptsNode<N> {
    async fn client(&self, transport: Transport) -> RpcClient {
        RpcClient::connect(&self.handles, transport).await
    }
}

async fn launch_pruned_receipts_node() -> eyre::Result<PrunedReceiptsNode<impl Sized>> {
    super::crash_backtrace::install();
    reth_tracing::init_test_tracing();

    let genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).ecotone_activated().build(),
    );
    let chain_id = chain_spec.chain_id();
    let mut config = NodeConfig::test()
        .map_chain(chain_spec)
        .with_unused_ports()
        .with_rpc(RpcServerArgs::default().with_unused_ports().with_http().with_ws())
        // The pruner runs after every persisted block, and the minimum distance lifts the
        // 64-block margin reth otherwise keeps ahead of a receipts prune target.
        .with_pruning(PruningArgs {
            block_interval: Some(1),
            minimum_distance: Some(1),
            receipts_before: Some(RECEIPTS_PRUNED_BEFORE),
            ..Default::default()
        });
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);
    // Every block is persisted as soon as it is canonical, so the pruner sees the chain's height.
    config.engine.persistence_threshold = 0;
    config.engine.memory_block_buffer_target = Some(0);
    config.engine.num_state_masking_blocks = 0;
    // Under the v2 layout the receipts of a short chain share one static file, which the pruner
    // only drops whole; the v1 layout holds them in the database, pruned block by block.
    config.storage.v2 = false;
    // The RPC cache would keep serving receipts the pruner removed; on a real chain the pruned
    // range lies far outside its window.
    config.rpc.rpc_state_cache.set_zero_lengths();

    let NodeHandle { node, node_exit_future: _ } = NodeBuilder::new(config)
        .testing_node(Runtime::test())
        .node(OpNode::default())
        .launch()
        .await?;
    let handles = node.add_ons_handle.rpc_server_handles().clone();
    let mut driver = EngineDriver {
        client: node.auth_server_handle().http_client(),
        engine: node.add_ons_handle.beacon_engine_handle.clone(),
        head: node.provider.chain_spec().genesis_hash(),
        wallet: Wallet::default().with_chain_id(chain_id),
    };

    for number in 1..CHAIN_HEIGHT {
        driver.build_block(number).await?;
    }
    let creator = driver.spare_signer();
    let log_address = creator.address().create(0);
    driver
        .build_block_with(CHAIN_HEIGHT, vec![log_emitting_creation_tx(chain_id, &creator)])
        .await?;

    let receipts_of = |number: u64| node.provider.receipts_by_block(number.into()).unwrap();
    wait_until("the receipts below the prune target were never pruned", || {
        receipts_of(RECEIPTS_PRUNED_BEFORE - 1).is_none()
    })
    .await;
    assert!(
        receipts_of(RECEIPTS_PRUNED_BEFORE).is_some_and(|receipts| !receipts.is_empty()),
        "the first block past the prune target must keep its receipts"
    );

    Ok(PrunedReceiptsNode { handles, log_address, _node: node })
}

/// A contract creation whose init code emits one log and deploys nothing.
fn log_emitting_creation_tx(chain_id: u64, creator: &PrivateKeySigner) -> Bytes {
    let mut tx = TxEip1559 {
        chain_id,
        gas_limit: 100_000,
        input: Bytes::from_static(&LOG_EMITTING_INIT_CODE),
        max_fee_per_gas: 20_000_000_000,
        max_priority_fee_per_gas: 1_000_000_000,
        to: TxKind::Create,
        ..Default::default()
    };
    let signature = creator.sign_transaction_sync(&mut tx).expect("the creation signs");
    op_alloy_consensus::OpPooledTransaction::Eip1559(tx.into_signed(signature))
        .encoded_2718()
        .into()
}

#[track_caller]
fn assert_rpc_error(result: Result<Vec<Log>, ClientError>, code: i32, message: &str) {
    match result {
        Err(ClientError::Call(err)) => {
            assert_eq!(err.code(), code, "unexpected error code: {err:?}");
            assert_eq!(err.message(), message, "unexpected error message: {err:?}");
        }
        other => panic!("expected error {code} `{message}`, got {other:?}"),
    }
}

#[rstest]
#[case::http(Transport::Http)]
#[case::ws(Transport::Ws)]
#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
#[ignore = "reth 2.5.2 answers `[]` for a fromBlock past the head; reth 2.6 answers `-32602 invalid block \
            range params` (reth#26891). Un-ignore with the reth 2.6 bump (#23401)"]
async fn from_block_past_the_head_is_rejected(#[case] transport: Transport) -> eyre::Result<()> {
    let node = launch_pruned_receipts_node().await?;
    let client = node.client(transport).await;

    let result = client.get_logs(Filter::new().from_block(CHAIN_HEIGHT + 1)).await;

    assert_rpc_error(result, INVALID_PARAMS_CODE, "invalid block range params");
    Ok(())
}

#[rstest]
#[case::http(Transport::Http)]
#[case::ws(Transport::Ws)]
#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
#[ignore = "reth 2.5.2 skips the blocks whose receipts are pruned and answers the logs of the rest; reth 2.6 \
            answers `4444 pruned history unavailable` (reth#27002). Un-ignore with the reth 2.6 bump (#23401)"]
async fn range_over_pruned_receipts_is_rejected(#[case] transport: Transport) -> eyre::Result<()> {
    let node = launch_pruned_receipts_node().await?;
    let client = node.client(transport).await;

    let result = client.get_logs(Filter::new().from_block(1u64).to_block(CHAIN_HEIGHT)).await;

    assert_rpc_error(result, PRUNED_HISTORY_UNAVAILABLE_CODE, "pruned history unavailable");
    Ok(())
}

#[rstest]
#[case::http(Transport::Http)]
#[case::ws(Transport::Ws)]
#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn range_within_kept_receipts_is_served(#[case] transport: Transport) -> eyre::Result<()> {
    let node = launch_pruned_receipts_node().await?;
    let client = node.client(transport).await;

    let logs = client
        .get_logs(Filter::new().from_block(RECEIPTS_PRUNED_BEFORE).to_block(CHAIN_HEIGHT))
        .await?;

    assert_eq!(logs.len(), 1, "the kept range holds exactly the created contract's log: {logs:?}");
    assert_eq!(logs[0].address(), node.log_address);
    assert_eq!(logs[0].block_number, Some(CHAIN_HEIGHT));
    Ok(())
}
