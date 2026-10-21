use super::*;
use crate::QueuedEngineDerivationClient;
use alloy_primitives::Sealed;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use kona_engine::{
    NoopBlockSink,
    test_utils::{TestEngineStateBuilder, test_block_info, test_engine_client},
};
use kona_protocol::L1BlockInfoBedrock;
use op_alloy_consensus::{OpBlock, OpTxEnvelope, TxDeposit};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

/// Exercise the real actor request/drain loop with forty consecutive unsafe blocks, without
/// depending on the removed dev RPC subscriptions. Observe EVERY enqueue, including a burst
/// already waiting in the inbound channel: healthy processing must keep the task queue <= 1.
#[tokio::test(start_paused = true)]
async fn healthy_unsafe_imports_keep_engine_queue_bounded() {
    let config = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(config.clone());
    let initial_head = test_block_info(0);
    let state = TestEngineStateBuilder::new().with_unsafe_head(initial_head).build();
    let (state_tx, state_rx) = watch::channel(state);
    let (requests, request_rx) = mpsc::channel(40);
    let (derivation_tx, _derivation_rx) = mpsc::channel(8);
    let mut actor = EngineActor::new(
        Arc::new(client),
        config,
        QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_tx },
        Engine::new(state, state_tx),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    let mut expected_heads = vec![initial_head];
    for number in 1..=40 {
        let mut block = OpBlock::default();
        block.header.number = number;
        block.header.timestamp = number * 2;
        block.header.parent_hash = expected_heads.last().unwrap().hash();
        block.body.transactions.push(OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
            input: L1BlockInfoBedrock::default().encode_calldata(),
            ..Default::default()
        })));
        let payload = ExecutionPayloadV1::from_block_slow(&block);
        let imported: OpBlock =
            OpExecutionPayloadEnvelope::V1(payload.clone()).try_into_block().unwrap();
        expected_heads.push(
            L2BlockInfo::from_block_and_genesis(&imported, &RollupConfig::default().genesis)
                .unwrap(),
        );
        l2.expect_params(
            "engine_newPayloadV1",
            serde_json::json!([payload]),
            PayloadStatus::from_status(PayloadStatusEnum::Valid),
        );
        l2.expect_params(
            "engine_forkchoiceUpdatedV3",
            serde_json::json!([
                alloy_rpc_types_engine::ForkchoiceState {
                    head_block_hash: expected_heads.last().unwrap().hash(),
                    safe_block_hash: initial_head.hash(),
                    finalized_block_hash: initial_head.hash(),
                },
                null
            ]),
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        requests
            .send(EngineActorRequest::ProcessUnsafeL2Block(Box::new(
                OpExecutionPayloadEnvelope::V1(payload),
            )))
            .await
            .unwrap();
    }
    for expected in &expected_heads[..40] {
        time::timeout(Duration::from_secs(1), actor.step()).await.unwrap().unwrap();
        assert_eq!(actor.engine.len(), 1, "no backlog may accumulate while the EL is healthy");
        assert_eq!(state_rx.borrow().sync_state.unsafe_head(), *expected);
    }
    // The final step imports block forty before discovering that its input has closed.
    drop(requests);
    assert!(matches!(actor.step().await, Err(EngineError::ChannelClosed)));
    assert!(actor.engine.is_empty());
    assert_eq!(state_rx.borrow().sync_state.unsafe_head(), expected_heads[40]);
    l1.assert_finished();
    l2.assert_finished();
}
