//! `eth_getLogs` must answer an error, not a silently empty or partial array, for a range the
//! node cannot serve in full: a range reaching past the head and a range or block whose receipts
//! the node pruned. An indexer that polls `lastSeen + 1`, or reads a range whose receipts are gone,
//! sees the error instead of mistaking the missing logs for an empty range.

use crate::engine_driver::{EngineDriver, wait_until};
use alloy_consensus::{SignableTransaction, TxEip1559};
use alloy_genesis::Genesis;
use alloy_network::{TxSignerSync, eip2718::Encodable2718};
use alloy_primitives::{Address, B256, Bytes, TxKind};
use alloy_rpc_types_eth::{Filter, Log};
use alloy_signer_local::PrivateKeySigner;
use jsonrpsee::{
    core::client::{ClientT, Error as ClientError},
    rpc_params,
    types::error::INVALID_PARAMS_CODE,
};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::wallet::Wallet;
use reth_node_builder::{NodeBuilder, NodeConfig, NodeHandle, rpc::RethRpcServerHandles};
use reth_node_core::args::{PruningArgs, RpcServerArgs};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::OpNode;
use reth_provider::{BlockHashReader, ChainSpecProvider, ReceiptProvider};
use reth_tasks::Runtime;
use std::sync::Arc;

/// The receipts of every block below this one are pruned; the blocks from it on keep theirs.
const RECEIPTS_PRUNED_BEFORE: u64 = 4;
/// The height the chain is built to. The head carries the one log a served range answers with.
const CHAIN_HEIGHT: u64 = 6;
/// Init code that emits an empty `LOG0` and deploys no runtime code: `PUSH1 0 PUSH1 0 LOG0 STOP`.
const LOG_EMITTING_INIT_CODE: [u8; 6] = [0x60, 0x00, 0x60, 0x00, 0xa0, 0x00];
/// EIP-4444's code for history the node no longer holds.
const PRUNED_HISTORY_UNAVAILABLE_CODE: i32 = 4444;

/// A node whose chain reaches `CHAIN_HEIGHT` with the receipts below `RECEIPTS_PRUNED_BEFORE`
/// pruned and one log, from a contract created at the head, in the receipts it kept.
struct PrunedReceiptsNode<N> {
    handles: RethRpcServerHandles,
    last_pruned_block_hash: B256,
    log_address: Address,
    /// Keeps the node behind the RPC handles running.
    _node: N,
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
        // The pruner runs after every persisted block, and a minimum distance of one lets it
        // reach a receipts prune target this close to the head of a short chain.
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
    // Under the v2 layout, op-reth's default, the receipts of a short chain share one static file,
    // which the pruner only drops whole, so the v2 pruned-receipts path is not covered here; the
    // v1 layout holds them in the database, pruned block by block.
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

    let last_pruned_block = RECEIPTS_PRUNED_BEFORE - 1;
    wait_until("the receipts below the prune target were never pruned", || {
        matches!(node.provider.receipts_by_block(last_pruned_block.into()), Ok(None))
    })
    .await;
    assert!(
        node.provider
            .receipts_by_block(RECEIPTS_PRUNED_BEFORE.into())?
            .is_some_and(|receipts| !receipts.is_empty()),
        "the first block past the prune target must keep its receipts"
    );
    let last_pruned_block_hash =
        node.provider.block_hash(last_pruned_block)?.expect("a pruned block keeps its header");

    Ok(PrunedReceiptsNode { handles, last_pruned_block_hash, log_address, _node: node })
}

/// A contract creation whose init code emits one log and deploys nothing.
fn log_emitting_creation_tx(chain_id: u64, creator: &PrivateKeySigner) -> Bytes {
    let mut creation_tx = TxEip1559 {
        chain_id,
        gas_limit: 100_000,
        input: Bytes::from_static(&LOG_EMITTING_INIT_CODE),
        max_fee_per_gas: 20_000_000_000,
        max_priority_fee_per_gas: 1_000_000_000,
        to: TxKind::Create,
        ..Default::default()
    };
    let signature = creator.sign_transaction_sync(&mut creation_tx).expect("the creation signs");
    op_alloy_consensus::OpPooledTransaction::Eip1559(creation_tx.into_signed(signature))
        .encoded_2718()
        .into()
}

async fn get_logs(client: &impl ClientT, filter: Filter) -> Result<Vec<Log>, ClientError> {
    client.request("eth_getLogs", rpc_params![filter]).await
}

/// Ranges the node cannot serve in full, each with the error code and exact message it answers.
fn unservable_ranges(last_pruned_block_hash: B256) -> [(&'static str, Filter, i32, String); 4] {
    [
        (
            "fromBlock past the head",
            Filter::new().from_block(CHAIN_HEIGHT + 1),
            INVALID_PARAMS_CODE,
            "invalid block range params".to_owned(),
        ),
        (
            "toBlock past the head",
            Filter::new().from_block(1u64).to_block(CHAIN_HEIGHT + 1),
            INVALID_PARAMS_CODE,
            format!(
                "block range extends beyond current head block: requested {}, head {CHAIN_HEIGHT}",
                CHAIN_HEIGHT + 1
            ),
        ),
        (
            "range over pruned receipts",
            Filter::new().from_block(1u64).to_block(CHAIN_HEIGHT),
            PRUNED_HISTORY_UNAVAILABLE_CODE,
            "pruned history unavailable".to_owned(),
        ),
        (
            "block hash with pruned receipts",
            Filter::new().at_block_hash(last_pruned_block_hash),
            PRUNED_HISTORY_UNAVAILABLE_CODE,
            "pruned history unavailable".to_owned(),
        ),
    ]
}

#[track_caller]
fn assert_rpc_error(result: Result<Vec<Log>, ClientError>, case: &str, code: i32, message: &str) {
    match result {
        Err(ClientError::Call(err)) => {
            assert_eq!(err.code(), code, "unexpected error code for {case}: {err:?}");
            assert_eq!(err.message(), message, "unexpected error message for {case}: {err:?}");
        }
        other => panic!("expected error {code} `{message}` for {case}, got {other:?}"),
    }
}

async fn assert_unservable_ranges_rejected(
    client: &impl ClientT,
    transport: &str,
    last_pruned_block_hash: B256,
) {
    for (range, filter, code, message) in unservable_ranges(last_pruned_block_hash) {
        let result = get_logs(client, filter).await;

        assert_rpc_error(result, &format!("{range} over {transport}"), code, &message);
    }
}

async fn assert_kept_range_served(
    client: &impl ClientT,
    transport: &str,
    log_address: Address,
) -> eyre::Result<()> {
    let logs =
        get_logs(client, Filter::new().from_block(RECEIPTS_PRUNED_BEFORE).to_block(CHAIN_HEIGHT))
            .await?;

    assert_eq!(
        logs.len(),
        1,
        "the kept range over {transport} holds exactly the created contract's log: {logs:?}"
    );
    assert_eq!(logs[0].address(), log_address, "log address over {transport}");
    assert_eq!(logs[0].block_number, Some(CHAIN_HEIGHT), "log block over {transport}");
    Ok(())
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn get_logs_rejects_only_ranges_it_cannot_serve_in_full() -> eyre::Result<()> {
    let node = launch_pruned_receipts_node().await?;
    let http_client = node.handles.rpc.http_client().expect("http is served");
    let ws_client = node.handles.rpc.ws_client().await.expect("ws is served");

    assert_unservable_ranges_rejected(&http_client, "http", node.last_pruned_block_hash).await;
    assert_unservable_ranges_rejected(&ws_client, "ws", node.last_pruned_block_hash).await;
    assert_kept_range_served(&http_client, "http", node.log_address).await?;
    assert_kept_range_served(&ws_client, "ws", node.log_address).await?;
    Ok(())
}
