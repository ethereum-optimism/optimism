use super::*;
use crate::RpcActor;
use jsonrpsee::{
    RpcModule, core::client::ClientT, http_client::HttpClientBuilder, rpc_params, server::Server,
};
use std::time::Duration;
use tokio::time::timeout;
use tokio_util::sync::CancellationToken;

#[tokio::test]
async fn critical_pipeline_reset_failure_stops_service_and_rpc() {
    let error = l2_provider_error().crit();
    let mut engine_client = MockDerivationEngineClient::new();
    engine_client.expect_send_safe_l2_signal().times(0);
    engine_client.expect_send_finalized_l2_block().times(0);
    let mut node = RecordingActor::new(engine_client);
    node.actor.pipeline.attributes.clear();
    node.complete_el_sync_at(DERIVED_HEAD).await;
    node.actor.pipeline.attributes.push_back(derived_attributes());
    node.actor.pipeline.signal_error = Some(error);

    let mut module = RpcModule::new(());
    module.register_method("test_alive", |_, _, _| true).unwrap();
    let server = Server::builder().build("127.0.0.1:0").await.unwrap();
    let address = server.local_addr().unwrap();
    let handle = server.start(module);
    let client = HttpClientBuilder::default()
        .request_timeout(Duration::from_secs(5))
        .build(format!("http://{address}"))
        .unwrap();
    let rpc_actor = RpcActor::new(handle.clone());
    let cancellation = CancellationToken::new();
    let service_cancellation = cancellation.clone();
    let service = tokio::spawn(async move {
        crate::service::spawn_and_wait!(
            service_cancellation,
            actors = [Some(node.actor), Some(rpc_actor)]
        );
        Ok::<(), String>(())
    });
    assert!(client.request::<bool, _>("test_alive", rpc_params![]).await.unwrap());

    node.request_tx
        .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: l2_block(RESUME_HEAD) },
        ))))
        .await
        .unwrap();

    let service_result = timeout(Duration::from_secs(5), service)
        .await
        .expect("failed reset must stop the service")
        .unwrap();
    assert!(service_result.is_err(), "the failed reset must reach the supervisor as an error");
    assert!(cancellation.is_cancelled());
    timeout(Duration::from_secs(5), handle.stopped())
        .await
        .expect("service shutdown must stop the RPC server");
    assert!(client.request::<bool, _>("test_alive", rpc_params![]).await.is_err());
}
