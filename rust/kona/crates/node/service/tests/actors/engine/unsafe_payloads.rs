use super::harness::{EngineHarness, genesis_head, malformed_payload, payload_info, valid_payload};
use alloy_json_rpc::ErrorPayload;
use alloy_rpc_types_engine::{PayloadId, PayloadStatus, PayloadStatusEnum};
use kona_engine::test_utils::{MockEngineCall, MockNewPayloadResponse, TestAttributesBuilder};
use kona_genesis::{HardForkConfig, RollupConfig};
use kona_node_service::{EngineActorRequest, EngineError, NodeActor, SealRequest};
use std::{sync::Arc, time::Duration};
use tokio::time::timeout;

async fn send_unsafe(
    harness: &EngineHarness,
    payload: op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope,
) {
    harness
        .requests
        .send(EngineActorRequest::ProcessUnsafeL2Block(Box::new(payload)))
        .await
        .expect("engine actor request channel closed");
}

async fn drain_until_waiting(harness: &mut EngineHarness) {
    for _ in 0..4 {
        if timeout(Duration::from_millis(1), harness.actor.step()).await.is_err() {
            return;
        }
    }
    panic!("engine actor unexpectedly stopped waiting for work");
}

#[tokio::test]
async fn remote_invalid_payload_does_not_block_later_valid_payload() {
    let config = Arc::new(RollupConfig::default());
    let head = genesis_head();
    let invalid = malformed_payload(1);
    let valid = valid_payload(head, 1);
    let valid_info = payload_info(&valid);
    let mut harness = EngineHarness::new(
        config,
        [
            MockNewPayloadResponse::Status(PayloadStatus::from_status(
                PayloadStatusEnum::Invalid { validation_error: "invalid transaction".to_string() },
            )),
            MockNewPayloadResponse::Status(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        ],
        head,
    );

    send_unsafe(&harness, invalid).await;
    harness.actor.step().await.expect("failed to receive invalid payload");
    drain_until_waiting(&mut harness).await;

    send_unsafe(&harness, valid).await;
    harness.actor.step().await.expect("failed to receive valid payload");
    drain_until_waiting(&mut harness).await;

    assert_eq!(*harness.unsafe_head.borrow(), valid_info);
    assert_eq!(*harness.queue_length.borrow(), 0);
    assert_eq!(
        harness
            .client
            .calls()
            .await
            .into_iter()
            .filter(|call| matches!(call, MockEngineCall::NewPayload(_)))
            .count(),
        2
    );
}

#[tokio::test]
async fn malformed_payload_rpc_error_does_not_block_later_valid_payload() {
    let config = Arc::new(RollupConfig::default());
    let head = genesis_head();
    let invalid = malformed_payload(1);
    let valid = valid_payload(head, 1);
    let valid_info = payload_info(&valid);
    let mut harness = EngineHarness::new(
        config,
        [
            MockNewPayloadResponse::RpcError(ErrorPayload::invalid_params()),
            MockNewPayloadResponse::Status(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        ],
        head,
    );

    send_unsafe(&harness, invalid).await;
    harness.actor.step().await.expect("failed to receive malformed payload");
    drain_until_waiting(&mut harness).await;

    send_unsafe(&harness, valid).await;
    harness.actor.step().await.expect("failed to receive valid payload");
    drain_until_waiting(&mut harness).await;

    assert_eq!(*harness.unsafe_head.borrow(), valid_info);
    assert_eq!(*harness.queue_length.borrow(), 0);
}

#[tokio::test]
async fn engine_capability_error_terminates_actor() {
    let config = Arc::new(RollupConfig::default());
    let head = genesis_head();
    let payload = malformed_payload(1);
    let mut harness = EngineHarness::new(
        config,
        [MockNewPayloadResponse::RpcError(ErrorPayload::method_not_found())],
        head,
    );

    send_unsafe(&harness, payload).await;
    harness.actor.step().await.expect("failed to receive payload");
    let error = harness.actor.step().await.expect_err("capability error must terminate actor");

    assert!(matches!(error, EngineError::EngineTask(_)));
    assert_eq!(
        harness
            .client
            .calls()
            .await
            .into_iter()
            .filter(|call| matches!(call, MockEngineCall::NewPayload(_)))
            .count(),
        1
    );
}

#[tokio::test]
async fn remote_payload_version_mismatch_is_dropped_before_engine_call() {
    let config = Arc::new(RollupConfig {
        hardforks: HardForkConfig { canyon_time: Some(0), ..Default::default() },
        ..Default::default()
    });
    let head = genesis_head();
    let mismatch = malformed_payload(1);
    let mut harness = EngineHarness::new(config, [], head);

    send_unsafe(&harness, mismatch).await;
    harness.actor.step().await.expect("failed to receive mismatched payload");
    drain_until_waiting(&mut harness).await;

    assert!(harness.client.calls().await.is_empty());
    assert_eq!(*harness.queue_length.borrow(), 0);
    assert!(harness.derivation.signals().await.is_empty());
}

#[tokio::test]
async fn local_payload_capability_error_is_returned_to_sequencer() {
    let config = Arc::new(RollupConfig::default());
    let head = genesis_head();
    let payload = valid_payload(head, 1);
    let mut harness = EngineHarness::new(
        config,
        [MockNewPayloadResponse::RpcError(ErrorPayload::method_not_found())],
        head,
    );
    harness.set_payload_to_get(payload).await;
    let attributes = TestAttributesBuilder::new().with_parent(head).with_timestamp(2).build();
    let (result_tx, mut result_rx) = tokio::sync::mpsc::channel(1);

    harness
        .requests
        .send(EngineActorRequest::Seal(Box::new(SealRequest {
            payload_id: PayloadId::new([1; 8]),
            attributes,
            result_tx,
        })))
        .await
        .expect("engine actor request channel closed");
    harness.actor.step().await.expect("failed to receive seal request");
    drain_until_waiting(&mut harness).await;

    let error = result_rx
        .recv()
        .await
        .expect("missing seal result")
        .expect_err("local capability error should fail sealing");
    assert!(error.to_string().contains("Method not found"));
    assert_eq!(
        harness
            .client
            .calls()
            .await
            .into_iter()
            .filter(|call| matches!(call, MockEngineCall::NewPayload(_)))
            .count(),
        1
    );
}
