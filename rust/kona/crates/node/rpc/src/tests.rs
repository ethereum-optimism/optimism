use super::{
    DevEngineApiServer, DevEngineRpc, HealthzApiServer, HealthzResponse, HealthzRpc, L1State,
    L1WatcherQueries, OutputResponse, RollupNodeApiServer, RollupRpc, WsRPC, WsServer,
};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_provider::{Provider, ProviderBuilder, RootProvider};
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_eth::{Block, EIP1186AccountProofResponse};
use alloy_transport::mock::Asserter;
use jsonrpsee::{
    RpcModule,
    core::server::MethodsError,
    rpc_params,
    server::{Server, middleware::http::ProxyGetRequestLayer},
    types::ErrorObjectOwned,
};
use kona_engine::{EngineState, EngineSyncStateUpdate};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OutputRoot, SyncStatus};
use op_alloy_network::Optimism;
use op_alloy_rpc_types::Transaction;
use serde_json::{Value, json};
use std::{sync::Arc, time::Duration};
use tokio::sync::{Notify, Semaphore, mpsc, watch};

fn block_and_config(isthmus: bool) -> (Block<Transaction>, Arc<RollupConfig>) {
    let mut block = Block::<Transaction>::default();
    block.header.inner.timestamp = 1;
    block.header.inner.state_root = B256::repeat_byte(1);
    if isthmus {
        block.header.inner.withdrawals_root = Some(B256::repeat_byte(2));
    }
    block.header.hash = block.header.inner.hash_slow();
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = block.header.hash;
    config.hardforks.isthmus_time = isthmus.then_some(0);
    (block, Arc::new(config))
}

fn rollup(
    provider: RootProvider<Optimism>,
    config: Arc<RollupConfig>,
    version: &'static str,
) -> (RpcModule<RollupRpc<RootProvider<Optimism>>>, watch::Sender<EngineState>) {
    let (state, receiver) = watch::channel(EngineState::default());
    let (l1_tx, mut l1_rx) = mpsc::channel(4);
    tokio::spawn(async move {
        while let Some(L1WatcherQueries::L1State(reply)) = l1_rx.recv().await {
            let _ = reply.send(L1State {
                current_l1: None,
                current_l1_finalized: None,
                head_l1: None,
                safe_l1: None,
                finalized_l1: None,
            });
        }
    });
    (RollupRpc::new(provider, config, receiver, version, l1_tx).into_rpc(), state)
}

#[tokio::test]
async fn output_proof_uses_the_fetched_hash_when_latest_moves() {
    let (block, config) = block_and_config(false);
    let hash = block.header.hash;
    let root = B256::repeat_byte(3);
    let mut el = RpcModule::new(());
    el.register_method("eth_getBlockByNumber", move |params, (), _| {
        assert_eq!(params.parse::<Value>().unwrap(), json!(["latest", true]));
        Ok::<_, ErrorObjectOwned>(block.clone())
    })
    .unwrap();
    el.register_method("eth_getProof", move |params, (), _| {
        // A number/tag would now resolve to a different block and return a different root.
        let params = params.parse::<Value>().unwrap();
        assert_eq!(
            params,
            json!([
                kona_protocol::Predeploys::L2_TO_L1_MESSAGE_PASSER, [],
                {"blockHash": hash, "requireCanonical": true}
            ])
        );
        Ok::<_, ErrorObjectOwned>(EIP1186AccountProofResponse {
            storage_hash: root,
            ..Default::default()
        })
    })
    .unwrap();
    let server = Server::builder().build("127.0.0.1:0").await.unwrap();
    let provider = ProviderBuilder::new()
        .network::<Optimism>()
        .connect_http(format!("http://{}", server.local_addr().unwrap()).parse().unwrap());
    let handle = server.start(el);
    let (rpc, _state) = rollup(provider.root().clone(), config.clone(), "1.2.3");
    let output: OutputResponse =
        rpc.call("optimism_outputAtBlock", rpc_params!["latest"]).await.unwrap();
    assert_eq!(output.output_root, OutputRoot::from_parts(B256::repeat_byte(1), root, hash).hash());
    assert_eq!(output.block_ref.hash(), hash);
    assert_eq!(output.withdrawal_storage_root, root);
    handle.stop().unwrap();
}

#[tokio::test]
async fn isthmus_output_uses_the_header_without_a_proof() {
    let (block, config) = block_and_config(true);
    let responses = Asserter::new();
    responses.push_success(&block);
    let (rpc, _state) =
        rollup(RootProvider::new(RpcClient::mocked(responses.clone())), config, "1.2.3");
    let output: OutputResponse =
        rpc.call("optimism_outputAtBlock", rpc_params!["latest"]).await.unwrap();
    assert_eq!(output.withdrawal_storage_root, block.header.withdrawals_root.unwrap());
    assert_eq!(
        output.output_root,
        OutputRoot::from_parts(
            block.header.state_root,
            output.withdrawal_storage_root,
            block.header.hash
        )
        .hash()
    );
    assert!(responses.read_q().is_empty());
}

#[tokio::test]
async fn output_read_failures_return_internal_errors() {
    for case in
        ["missing block", "block RPC", "proof RPC", "missing withdrawals root", "invalid block"]
    {
        let (mut block, mut config) = block_and_config(case == "missing withdrawals root");
        if case == "missing withdrawals root" {
            block.header.inner.withdrawals_root = None;
            block.header.hash = block.header.inner.hash_slow();
            Arc::make_mut(&mut config).genesis.l2.hash = block.header.hash;
        }
        if case == "invalid block" {
            block.header.inner.number = 1;
        }
        let responses = Asserter::new();
        match case {
            "missing block" => responses.push_success(&Value::Null),
            "block RPC" => responses.push_failure_msg("unavailable"),
            _ => responses.push_success(&block),
        }
        if case == "proof RPC" {
            responses.push_failure_msg("requested hash is no longer canonical");
        }
        let (rpc, _state) =
            rollup(RootProvider::new(RpcClient::mocked(responses)), config, "1.2.3");
        let error = rpc
            .call::<_, OutputResponse>("optimism_outputAtBlock", rpc_params!["latest"])
            .await
            .unwrap_err();
        assert!(matches!(error, MethodsError::JsonRpc(error) if error.code() == -32603), "{case}");
    }
}

#[tokio::test]
async fn stalled_proof_does_not_block_other_reads() {
    let (block, config) = block_and_config(false);
    let started = Arc::new(Notify::new());
    let release = Arc::new(Semaphore::new(0));
    let mut el = RpcModule::new((started.clone(), release.clone()));
    el.register_method("eth_getBlockByNumber", move |_, _, _| {
        Ok::<_, ErrorObjectOwned>(block.clone())
    })
    .unwrap();
    el.register_async_method("eth_getProof", |_, context, _| async move {
        context.0.notify_one();
        context.1.acquire().await.unwrap().forget();
        Ok::<_, ErrorObjectOwned>(EIP1186AccountProofResponse::default())
    })
    .unwrap();
    let server = Server::builder().build("127.0.0.1:0").await.unwrap();
    let provider = ProviderBuilder::new()
        .network::<Optimism>()
        .connect_http(format!("http://{}", server.local_addr().unwrap()).parse().unwrap());
    let handle = server.start(el);
    let (rpc, state) = rollup(provider.root().clone(), config.clone(), "1.2.3");
    let output_rpc = rpc.clone();
    let output = tokio::spawn(async move {
        output_rpc.call::<_, OutputResponse>("optimism_outputAtBlock", rpc_params!["latest"]).await
    });
    tokio::time::timeout(Duration::from_secs(2), started.notified()).await.unwrap();
    let head = L2BlockInfo {
        block_info: kona_protocol::BlockInfo { number: 7, ..Default::default() },
        ..Default::default()
    };
    state.send_modify(|state| {
        state.sync_state = state
            .sync_state
            .apply_update(EngineSyncStateUpdate { unsafe_head: Some(head), ..Default::default() })
    });
    tokio::time::timeout(Duration::from_secs(2), async {
        assert_eq!(
            rpc.call::<_, String>("optimism_version", rpc_params![]).await.unwrap(),
            "1.2.3"
        );
        assert_eq!(
            rpc.call::<_, RollupConfig>("optimism_rollupConfig", rpc_params![]).await.unwrap(),
            *config
        );
        assert_eq!(
            rpc.call::<_, SyncStatus>("optimism_syncStatus", rpc_params![])
                .await
                .unwrap()
                .unsafe_l2,
            head
        );
    })
    .await
    .unwrap();
    assert!(!output.is_finished());
    release.add_permits(1);
    output.await.unwrap().unwrap();
    handle.stop().unwrap();
}

#[tokio::test]
async fn registered_version_and_health_use_the_binary_release() {
    for version in ["1.2.3", "0.0.0-dev"] {
        let (_, config) = block_and_config(false);
        let (mut rpc, _state) =
            rollup(RootProvider::new(RpcClient::mocked(Asserter::new())), config, version);
        rpc.merge(HealthzRpc::new(version).into_rpc()).unwrap();
        assert_eq!(
            rpc.call::<_, String>("optimism_version", rpc_params![]).await.unwrap(),
            version
        );
        assert_eq!(
            rpc.call::<_, HealthzResponse>("healthz", rpc_params![]).await.unwrap().version,
            version
        );
        let middleware = tower::ServiceBuilder::new()
            .layer(ProxyGetRequestLayer::new([("/healthz", "healthz")]).unwrap());
        let server =
            Server::builder().set_http_middleware(middleware).build("127.0.0.1:0").await.unwrap();
        let address = server.local_addr().unwrap();
        let handle = server.start(rpc);
        let health: HealthzResponse = reqwest::get(format!("http://{address}/healthz"))
            .await
            .unwrap()
            .error_for_status()
            .unwrap()
            .json()
            .await
            .unwrap();
        assert_eq!(health.version, version);
        handle.stop().unwrap();
    }
}

#[tokio::test]
async fn closed_engine_publication_does_not_serve_stale_status() {
    let (_, config) = block_and_config(false);
    let (rpc, state) =
        rollup(RootProvider::new(RpcClient::mocked(Asserter::new())), config, "1.2.3");
    drop(state);
    assert!(rpc.call::<_, SyncStatus>("optimism_syncStatus", rpc_params![]).await.is_err());
    assert!(
        rpc.call::<_, OutputResponse>(
            "optimism_outputAtBlock",
            rpc_params![BlockNumberOrTag::Latest]
        )
        .await
        .is_err()
    );
}

#[tokio::test]
async fn dev_and_head_subscriptions_follow_publications_and_close() {
    let (queue, receiver) = watch::channel(2usize);
    let dev = DevEngineRpc::new(receiver).into_rpc();
    assert_eq!(dev.call::<_, usize>("dev_taskQueueLength", rpc_params![]).await.unwrap(), 2);
    let mut subscription =
        dev.subscribe_unbounded("dev_subscribe_engine_queue_size", rpc_params![]).await.unwrap();
    assert_eq!(subscription.next::<usize>().await.unwrap().unwrap().0, 2);
    queue.send_replace(5);
    assert_eq!(subscription.next::<usize>().await.unwrap().unwrap().0, 5);
    drop(queue);
    assert!(
        tokio::time::timeout(Duration::from_secs(2), subscription.next::<usize>())
            .await
            .unwrap()
            .is_none()
    );
    assert!(dev.call::<_, usize>("dev_taskQueueLength", rpc_params![]).await.is_err());

    for (method, number) in [
        ("ws_subscribe_unsafe_head", 9),
        ("ws_subscribe_safe_head", 8),
        ("ws_subscribe_finalized_head", 7),
    ] {
        let (state, receiver) = watch::channel(EngineState::default());
        let rpc = WsRPC::new(receiver).into_rpc();
        let mut subscription = rpc.subscribe_unbounded(method, rpc_params![]).await.unwrap();
        let head = L2BlockInfo {
            block_info: kona_protocol::BlockInfo { number, ..Default::default() },
            ..Default::default()
        };
        state.send_modify(|state| {
            state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
                unsafe_head: Some(L2BlockInfo {
                    block_info: kona_protocol::BlockInfo { number: 9, ..Default::default() },
                    ..Default::default()
                }),
                safe_head: Some(L2BlockInfo {
                    block_info: kona_protocol::BlockInfo { number: 8, ..Default::default() },
                    ..Default::default()
                }),
                finalized_head: Some(L2BlockInfo {
                    block_info: kona_protocol::BlockInfo { number: 7, ..Default::default() },
                    ..Default::default()
                }),
                ..Default::default()
            });
        });
        assert_eq!(
            tokio::time::timeout(Duration::from_secs(2), subscription.next::<L2BlockInfo>())
                .await
                .unwrap()
                .unwrap()
                .unwrap()
                .0,
            head
        );
        drop(state);
        assert!(
            tokio::time::timeout(Duration::from_secs(2), subscription.next::<L2BlockInfo>())
                .await
                .unwrap()
                .is_none()
        );
    }
}

#[tokio::test]
async fn idle_disconnect_releases_subscription_receivers() {
    let (queue, receiver) = watch::channel(0usize);
    let dev = DevEngineRpc::new(receiver).into_rpc();
    let (state, receiver) = watch::channel(EngineState::default());
    let ws = WsRPC::new(receiver).into_rpc();
    let dev_subscription =
        dev.subscribe_unbounded("dev_subscribe_engine_queue_size", rpc_params![]).await.unwrap();
    let ws_subscription =
        ws.subscribe_unbounded("ws_subscribe_safe_head", rpc_params![]).await.unwrap();
    assert_eq!(queue.receiver_count(), 2);
    assert_eq!(state.receiver_count(), 2);
    drop(dev_subscription);
    drop(ws_subscription);
    tokio::time::timeout(Duration::from_secs(2), async {
        while queue.receiver_count() != 1 || state.receiver_count() != 1 {
            tokio::task::yield_now().await;
        }
    })
    .await
    .unwrap();
}
