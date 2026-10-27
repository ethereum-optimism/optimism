use super::{EngineActor, QueuedEngineDerivationClient};
use crate::{DerivationActorRequest, NodeActor};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
use kona_engine::{
    Engine, EngineActorRequest, EngineState, EngineSyncStateUpdate, FinalizeBlockId, NoopBlockSink,
    test_utils::test_engine_client,
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
    let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
        Default::default();
    let mut config = RollupConfig::default();
    config.genesis.l2.hash = block.header.inner.hash_slow();
    // Keep an unsafe block above genesis so both the L1 origin check and the safe walk run.
    let mut unsafe_block = block.clone();
    unsafe_block.header.inner.number = 1;
    unsafe_block.header.inner.parent_hash = config.genesis.l2.hash;
    unsafe_block.header.hash = unsafe_block.header.inner.hash_slow();
    type RpcTransaction =
        <op_alloy_network::Optimism as alloy_provider::Network>::TransactionResponse;
    let deposit = op_alloy_consensus::OpTxEnvelope::Deposit(alloy_primitives::Sealed::new(
        op_alloy_consensus::TxDeposit {
            input: kona_protocol::L1BlockInfoBedrock::new(
                0,
                0,
                0,
                B256::ZERO,
                1,
                Default::default(),
                Default::default(),
                Default::default(),
            )
            .encode_calldata(),
            ..Default::default()
        },
    ));
    unsafe_block.transactions =
        alloy_rpc_types_eth::BlockTransactions::Full(vec![RpcTransaction::from_transaction(
            alloy_consensus::transaction::Recovered::new_unchecked(deposit, Default::default()),
            Default::default(),
        )]);

    let config = Arc::new(config);
    let (client, l1, l2) = test_engine_client(config.clone());
    let client = Arc::new(client);
    if l1_failure {
        for tag in ["finalized", "safe", "latest"] {
            l2.expect_params(
                "eth_getBlockByNumber",
                serde_json::json!([tag, true]),
                if tag == "latest" { &unsafe_block } else { &block },
            );
        }
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
    let (state_tx, _state_rx) = tokio::sync::watch::channel(EngineState::default());
    let mut actor = EngineActor::new(
        client,
        config.clone(),
        derivation,
        Engine::new(EngineState::default(), state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    // Validators have no sequencer; admin reset goes directly to the engine.
    let rpc = AdminRpc::new(None, requests, mpsc::channel(1).0);
    let reset = tokio::spawn(async move { rpc.admin_reset_derivation_pipeline().await });
    actor.step().await.unwrap(); // retain the reset
    actor.step().await.unwrap(); // failed attempt, then wait for backoff
    assert!(!reset.is_finished());
    for response in [&block, &block, &unsafe_block] {
        l2.expect_with_delay("eth_getBlockByNumber", response, read_delay);
    }
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([config.genesis.l2.hash, true]),
        &block,
    );
    l1.expect(
        "eth_getBlockByNumber",
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
