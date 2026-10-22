use super::{EngineActor, EngineActorRequest, QueuedEngineDerivationClient, ResetRequest};
use crate::{DerivationActorRequest, NodeActor};
use alloy_eips::BlockId;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
use kona_engine::{
    Engine, EngineState, EngineSyncStateUpdate, FinalizeBlockId, NoopBlockSink,
    test_utils::MockEngineClient,
};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
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
    let client = Arc::new(
        MockEngineClient::builder()
            .with_config(config.clone())
            .with_l2_block(BlockId::finalized(), block.clone())
            .with_l2_block(BlockId::safe(), block.clone())
            .with_l2_block(BlockId::latest(), block)
            .with_l1_block(config.genesis.l1.hash.into(), Default::default())
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(
                PayloadStatus::from_status(PayloadStatusEnum::Valid),
            ))
            .build(),
    );
    let storage = client.storage();
    {
        let mut data = storage.write().await;
        if l1_failure {
            data.l1_read_error = Some("L1 timeout".into());
        } else {
            data.l2_read_error = Some("L2 timeout".into());
        }
    }
    let (derivation_tx, mut derivation_rx) = mpsc::channel(1);
    // Derivation waits for the reset reply and cannot drain its full queue until then.
    derivation_tx
        .send(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::default()))
        .await
        .unwrap();
    let derivation = QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx };
    let (requests, request_rx) = mpsc::channel(8);
    let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
    let (queue_tx, _queue_rx) = tokio::sync::watch::channel(0);
    let mut actor = EngineActor::new(
        client,
        config,
        derivation,
        Engine::new(EngineState::default(), state_tx, queue_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    let (reply, mut reply_rx) = mpsc::channel(1);
    requests
        .send(EngineActorRequest::Reset(Box::new(ResetRequest { result_tx: reply })))
        .await
        .unwrap();
    actor.step().await.unwrap(); // retain the reset
    actor.step().await.unwrap(); // failed attempt, then wait for backoff
    assert!(reply_rx.try_recv().is_err());
    {
        let mut data = storage.write().await;
        data.l1_read_error = None;
        data.l2_read_error = None;
        data.l2_read_delay = read_delay;
    }
    // A step performs recovery and then waits for another message; the response arrives first.
    let mut tasks = tokio::task::JoinSet::new();
    tasks.spawn(async move {
        loop {
            actor.step().await?;
        }
        #[allow(unreachable_code)]
        Ok::<(), crate::EngineError>(())
    });
    time::timeout(Duration::from_secs(20), reply_rx.recv()).await.unwrap().unwrap().unwrap();
    assert!(matches!(
        derivation_rx.recv().await.unwrap(),
        DerivationActorRequest::ProcessL1HeadUpdateRequest(_)
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
}

fn l2_block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            hash: B256::with_last_byte(number as u8),
            number,
            ..Default::default()
        },
        ..Default::default()
    }
}

/// Derivation must be handed the local-safe head, not the safe head, on both the safe head
/// update and the sync completion. The two differ here so a label mix-up cannot pass.
#[tokio::test(start_paused = true)]
async fn derivation_is_handed_the_local_safe_head() {
    let local_safe_head = l2_block(30);
    let safe_head = l2_block(20);
    let config = Arc::new(RollupConfig::default());
    let client = Arc::new(MockEngineClient::builder().with_config(config.clone()).build());
    let state = EngineState {
        sync_state: EngineState::default().sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(l2_block(40)),
            local_safe_head: Some(local_safe_head),
            safe_head: Some(safe_head),
            finalized_head: Some(l2_block(10)),
        }),
        el_sync_finished: true,
        ..Default::default()
    };
    let (derivation_tx, mut derivation_rx) = mpsc::channel(8);
    let derivation = QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx };
    let (requests, request_rx) = mpsc::channel(8);
    let (state_tx, _state_rx) = tokio::sync::watch::channel(state);
    let (queue_tx, _queue_rx) = tokio::sync::watch::channel(0);
    let mut actor = EngineActor::new(
        client,
        config,
        derivation,
        Engine::new(state, state_tx, queue_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    // Any request lets the step return once the drain has notified derivation.
    requests
        .send(EngineActorRequest::ProcessFinalizedL2Block(Box::new(FinalizeBlockId::ByNumber(10))))
        .await
        .unwrap();

    actor.step().await.unwrap();

    let Some(DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(sent)) =
        derivation_rx.recv().await
    else {
        panic!("expected a safe head update first");
    };
    assert_eq!(*sent, local_safe_head);
    let Some(DerivationActorRequest::ProcessEngineSyncCompletionRequest(sent)) =
        derivation_rx.recv().await
    else {
        panic!("expected the sync completion second");
    };
    assert_eq!(*sent, local_safe_head);
}
