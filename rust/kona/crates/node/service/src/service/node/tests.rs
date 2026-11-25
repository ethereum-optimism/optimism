use kona_engine::{EngineState, test_utils::test_engine_client};
use kona_genesis::RollupConfig;
use kona_rpc::{HealthzApiServer, HealthzResponse, HealthzRpc, RollupNodeApiServer, RollupRpc};
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

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
            None,
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
