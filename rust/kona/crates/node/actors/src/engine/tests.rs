use super::{EngineActor, QueuedEngineDerivationClient};
use crate::{DerivationActorRequest, EngineActorRequest, NodeActor, ResetRequest};
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
use kona_engine::{Engine, EngineState, NoopBlockSink, test_utils::test_engine_client};
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::{
    sync::mpsc,
    time::{self, Duration},
};

/// Both L1 and L2 outages during a reset must retain the original request and finish it
/// after recovery, without a new reset request or L1 head to wake the actor.
#[rstest::rstest]
#[case(true, Duration::ZERO)]
#[case(false, Duration::ZERO)]
#[case(false, Duration::from_secs(4))]
#[tokio::test(start_paused = true)]
async fn reset_recovers_and_completes_original_request(
    #[case] l1_failure: bool,
    #[case] read_delay: Duration,
) {
    let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
        Default::default();
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = block.header.inner.hash_slow();
    let config = Arc::new(config);
    let (client, l1, l2) = test_engine_client(config.clone());
    let client = Arc::new(client);
    if l1_failure {
        for tag in ["finalized", "safe", "latest"] {
            l2.expect_params("eth_getBlockByNumber", serde_json::json!([tag, true]), &block);
        }
        l1.expect_error("eth_getBlockByHash");
    } else {
        l2.expect_error("eth_getBlockByNumber");
    }
    let (derivation_tx, mut derivation_rx) = mpsc::channel(1);
    // Derivation waits for the reset reply and cannot drain its full queue until then.
    derivation_tx
        .send(DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(Box::default()))
        .await
        .unwrap();
    let derivation = QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx };
    let (requests, request_rx) = mpsc::channel(8);
    let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
    let mut actor = EngineActor::new(
        client,
        config,
        derivation,
        Engine::new(EngineState::default(), state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    let reset = tokio::spawn(async move {
        let (result_tx, mut result_rx) = mpsc::channel(1);
        requests
            .send(EngineActorRequest::Reset(Box::new(ResetRequest { result_tx })))
            .await
            .unwrap();
        result_rx.recv().await.unwrap()
    });
    actor.step().await.unwrap(); // retain the reset
    actor.step().await.unwrap(); // failed attempt, then wait for backoff
    assert!(!reset.is_finished());
    for _ in 0..3 {
        l2.expect_with_delay("eth_getBlockByNumber", &block, read_delay);
    }
    l1.expect(
        "eth_getBlockByHash",
        alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default(),
    );
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
    );
    // A step performs recovery and then waits for another message; the response arrives first.
    let mut tasks = tokio::task::JoinSet::new();
    tasks.spawn(async move {
        loop {
            actor.step().await?;
        }
        #[allow(unreachable_code)]
        Ok::<(), crate::EngineError>(())
    });
    time::timeout(Duration::from_secs(20), reset).await.unwrap().unwrap().unwrap();
    assert!(matches!(
        derivation_rx.recv().await.unwrap(),
        DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(_)
    ));
    assert!(matches!(
        derivation_rx.recv().await.unwrap(),
        DerivationActorRequest::ProcessEngineSignalRequest(_)
    ));
    assert!(matches!(
        derivation_rx.recv().await.unwrap(),
        DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(_)
    ));
    assert!(matches!(
        derivation_rx.recv().await.unwrap(),
        DerivationActorRequest::ProcessEngineSyncCompletionRequest(_)
    ));
    tasks.abort_all();
    l1.assert_finished();
    l2.assert_finished();
}
