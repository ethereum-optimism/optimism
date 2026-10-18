//! The conductor uses the configured transport deadline for both publication gates.

use alloy_rpc_types_engine::ExecutionPayloadV1;
use jsonrpsee::{RpcModule, server::ServerBuilder, types::ErrorObjectOwned};
use kona_node_service::{Conductor, ConductorClient};
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{sync::Arc, time::Duration};
use tokio::{sync::Notify, time};

#[rstest::rstest]
#[case::leader("conductor_leader")]
#[case::commit("conductor_commitUnsafePayload")]
#[tokio::test]
async fn configured_deadline_bounds_conductor_calls(#[case] method: &'static str) {
    let received = Arc::new(Notify::new());
    let mut module = RpcModule::new(received.clone());
    module
        .register_async_method(method, |_, received, _| async move {
            received.notify_one();
            std::future::pending::<Result<(), ErrorObjectOwned>>().await
        })
        .unwrap();
    let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
    let url = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
    let server = server.start(module);
    // Different from the one-second default: a dropped timeout override must fail this test.
    let timeout = Duration::from_secs(3);
    let client = ConductorClient::new(kona_engine::rpc_client(url, timeout));
    let request = tokio::spawn(async move {
        let started = time::Instant::now();
        let result = match method {
            "conductor_leader" => client.leader().await.map(|_| ()),
            "conductor_commitUnsafePayload" => {
                let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
                    &OpBlock::default(),
                ));
                client.commit_unsafe_payload(&payload).await
            }
            _ => unreachable!(),
        };
        let elapsed = started.elapsed();
        assert!(
            elapsed > timeout - Duration::from_secs(1),
            "deadline fired too early: {elapsed:?}"
        );
        assert!(
            elapsed <= timeout + Duration::from_secs(1),
            "deadline fired too late: {elapsed:?}"
        );
        result
    });
    // Finish real socket I/O before using virtual time for the stalled response.
    time::timeout(Duration::from_secs(2), received.notified()).await.unwrap();
    time::pause();
    let error = time::timeout(timeout * 2, request)
        .await
        .expect("conductor request has no deadline")
        .unwrap()
        .unwrap_err();
    time::resume();
    server.stop().unwrap();
    assert!(error.to_string().contains("RPC request timed out"), "{error}");
}
