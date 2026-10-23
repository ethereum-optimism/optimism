use super::rollup::merge_admin_module;
use crate::QueuedEngineRpcClient;
use jsonrpsee::RpcModule;
use kona_rpc::{HealthzApiServer, HealthzResponse, HealthzRpc, RollupNodeApiServer, RollupRpc};
use tokio::sync::mpsc;

fn admin_method_names(enable_admin: bool) -> Vec<String> {
    let mut modules = RpcModule::new(());
    let (network_admin_tx, _rx) = mpsc::channel(1);
    merge_admin_module(&mut modules, enable_admin, None, network_admin_tx)
        .expect("admin module registration");
    modules.method_names().map(ToString::to_string).collect()
}

#[test]
fn admin_module_registered_only_when_enabled() {
    // Without `--rpc.enable-admin`, no admin methods are exposed.
    assert!(
        admin_method_names(false).is_empty(),
        "admin namespace must not be registered when disabled"
    );

    // With it enabled, the sequencer-control and payload-injection methods the acceptance
    // suite relies on are present.
    let enabled = admin_method_names(true);
    for method in ["admin_postUnsafePayload", "admin_startSequencer", "admin_stopSequencer"] {
        assert!(enabled.iter().any(|m| m == method), "missing {method}");
    }
}

#[tokio::test]
async fn registered_endpoints_report_the_binary_version() {
    for version in ["1.2.3", "0.0.0-dev"] {
        let (engine_tx, _engine_rx) = mpsc::channel(1);
        let (l1_tx, _l1_rx) = mpsc::channel(1);
        let engine_client = QueuedEngineRpcClient::new(engine_tx);
        let mut rpc = RollupRpc::new(version, engine_client, l1_tx, None).into_rpc();
        rpc.merge(HealthzRpc::new(version).into_rpc()).unwrap();
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
