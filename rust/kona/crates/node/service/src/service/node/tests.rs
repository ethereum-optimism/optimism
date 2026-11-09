use super::rollup::{RPC_HTTP_REQUEST_TIMEOUT, merge_admin_module, rpc_http_middleware};
use jsonrpsee::{RpcModule, core::client::ClientT, http_client::HttpClientBuilder, server::Server};
use kona_engine::{EngineState, test_utils::test_engine_client};
use kona_genesis::RollupConfig;
use kona_rpc::{
    AdminRpc, HealthzApiServer, HealthzResponse, HealthzRpc, RollupNodeApiServer, RollupRpc,
};
use std::{sync::Arc, time::Duration};
use tokio::sync::{mpsc, watch};

#[test]
fn admin_module_registered_only_when_enabled() {
    for enabled in [false, true] {
        let mut modules = RpcModule::new(());
        let admin = AdminRpc::new(None, mpsc::channel(1).0, mpsc::channel(1).0);
        merge_admin_module(&mut modules, enabled, admin).unwrap();
        let methods: Vec<_> = modules.method_names().collect();
        if enabled {
            for method in [
                "admin_postUnsafePayload",
                "admin_startSequencer",
                "admin_stopSequencer",
                "admin_resetDerivationPipeline",
            ] {
                assert!(methods.contains(&method), "missing {method}");
            }
        } else {
            assert!(methods.is_empty(), "admin methods must be disabled by default");
        }
    }
}

#[tokio::test]
async fn registered_endpoints_report_the_binary_version() {
    for version in ["1.2.3", "0.0.0-dev"] {
        let config = Arc::new(RollupConfig::default());
        let (engine, _l1, _l2) = test_engine_client(config.clone());
        let (_state_tx, state_rx) = watch::channel(EngineState::default());
        let (l1_tx, _l1_rx) = mpsc::channel(1);
        let mut rpc = RollupRpc::new(
            version.to_owned(),
            config,
            state_rx,
            engine.query_client(),
            l1_tx,
            None,
        )
        .into_rpc();
        rpc.merge(HealthzRpc::new(version.to_owned()).into_rpc()).unwrap();
        assert_eq!(
            rpc.call::<_, String>("optimism_version", jsonrpsee::rpc_params![]).await.unwrap(),
            version
        );
        assert_eq!(
            rpc.call::<_, HealthzResponse>("healthz", jsonrpsee::rpc_params![])
                .await
                .unwrap()
                .version,
            version
        );
    }
}

/// An admin call waits for the sequencer to finish its current block, which can take a few seconds
/// with a conductor. The server must answer it rather than drop the connection, as a 2s cap did.
#[tokio::test]
async fn rpc_server_answers_a_call_slower_than_two_seconds() {
    let delay = Duration::from_millis(2500);
    assert!(delay < RPC_HTTP_REQUEST_TIMEOUT);
    let mut module = RpcModule::new(());
    module
        .register_async_method("admin_slow", move |_, _, _| async move {
            tokio::time::sleep(delay).await;
            jsonrpsee::core::RpcResult::Ok(7u64)
        })
        .unwrap();
    let server = Server::builder()
        .set_http_middleware(rpc_http_middleware())
        .build("127.0.0.1:0")
        .await
        .unwrap();
    let addr = server.local_addr().unwrap();
    let handle = server.start(module);
    let client = HttpClientBuilder::default()
        .request_timeout(Duration::from_secs(60))
        .build(format!("http://{addr}"))
        .unwrap();
    assert_eq!(client.request::<u64, _>("admin_slow", jsonrpsee::rpc_params![]).await.unwrap(), 7);
    handle.stop().unwrap();
    handle.stopped().await;
}
