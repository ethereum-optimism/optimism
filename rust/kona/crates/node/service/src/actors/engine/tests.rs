use super::{EngineActor, QueuedEngineDerivationClient};
use crate::{DerivationActorRequest, NodeActor};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
use kona_engine::{
    Engine, EngineActorRequest, EngineState, EngineSyncStateUpdate, FinalizeBlockId, NoopBlockSink,
    test_utils::{TestChain, expect_l1_block, test_engine_client},
};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_rpc::{AdminApiServer, AdminRpc};
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
async fn admin_reset_recovers_and_completes_original_request(
    #[case] l1_failure: bool,
    #[case] read_delay: Duration,
) {
    // Finalized, safe and unsafe heads 18, 20 and 26 of a real chain, so the reset checks the
    // unsafe head's L1 origin. The sequencing window is long enough that the safe head search
    // stops at the finalized head.
    let chain = TestChain::new();
    let config = Arc::new(RollupConfig { seq_window_size: 1000, ..Default::default() });
    let (client, l1, l2) = test_engine_client(config.clone());
    let client = Arc::new(client);
    if l1_failure {
        chain.expect_heads(&l2, 18, 20, 26);
        l1.expect_error("eth_getBlockByNumber");
    } else {
        l2.expect_error("eth_getBlockByNumber");
    }
    let (derivation_tx, mut derivation_rx) = mpsc::channel(1);
    // Derivation waits for the reset reply and cannot drain its full queue until then.
    derivation_tx
        .send(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::default()))
        .await
        .unwrap();
    let derivation = QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx };
    let (requests, request_rx) = mpsc::channel(8);
    let (state_tx, state_rx) = tokio::sync::watch::channel(EngineState::default());
    let (unsafe_head_tx, unsafe_head_rx) =
        tokio::sync::watch::channel(kona_protocol::L2BlockInfo::default());
    let mut actor = EngineActor::new(
        client,
        config,
        derivation,
        Engine::new(EngineState::default(), state_tx),
        Some(unsafe_head_tx),
        request_rx,
        Arc::new(NoopBlockSink),
    );
    // Validators have no sequencer; admin reset goes directly to the engine.
    let rpc = AdminRpc::new(None, requests, mpsc::channel(1).0);
    let reset = tokio::spawn(async move { rpc.admin_reset_derivation_pipeline().await });
    actor.step().await.unwrap(); // retain the reset
    actor.step().await.unwrap(); // failed attempt, then wait for backoff
    assert!(!reset.is_finished());
    for number in [18, 20, 26] {
        l2.expect_with_delay("eth_getBlockByNumber", chain.rpc(number), read_delay);
    }
    expect_l1_block(&l1, 105, 106);
    for number in (18..26).rev() {
        chain.expect_by_hash(&l2, number);
    }
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
    // Reset acknowledgement must make the recovered heads visible to RPC readers, even
    // when stopped sequencing leaves the engine task queue empty.
    // The unsafe head keeps its canonical L1 origin; the safe heads walk back to finalized.
    let published = state_rx.borrow().sync_state;
    assert_eq!(published.unsafe_head(), chain.info(26));
    for head in [published.local_safe_head(), published.safe_head(), published.finalized_head()] {
        assert_eq!(head, chain.info(18));
    }
    assert_eq!(*unsafe_head_rx.borrow(), chain.info(26));
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
    l1.assert_finished();
    l2.assert_finished();
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
    let (client, _l1, _l2) = test_engine_client(config.clone());
    let client = Arc::new(client);
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
    let mut actor = EngineActor::new(
        client,
        config,
        derivation,
        Engine::new(state, state_tx),
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
