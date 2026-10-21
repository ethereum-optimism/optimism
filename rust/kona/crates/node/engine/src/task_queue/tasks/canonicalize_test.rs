use super::*;
use crate::{InsertTaskError, NoopBlockSink, SynchronizeTaskError, test_utils::test_engine_client};
use alloy_consensus::{Block, SignableTransaction, TxEip1559};
use alloy_primitives::{Sealed, Signature};
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use op_alloy_consensus::{OpTxEnvelope, TxDeposit};

#[derive(Debug, Default)]
pub(crate) struct RecordingSink(pub(crate) std::sync::Mutex<Vec<L2BlockInfo>>);
impl ImportedBlockSink for RecordingSink {
    fn block_imported(&self, _: op_alloy_consensus::OpBlock, info: L2BlockInfo) {
        self.0.lock().unwrap().push(info);
    }
}

pub(crate) struct PayloadFixture {
    pub(crate) payload: OpExecutionPayloadEnvelope,
    pub(crate) cfg: Arc<RollupConfig>,
}

pub(crate) fn payload_fixture() -> PayloadFixture {
    let mut block = Block::<OpTxEnvelope>::default();
    // V1 round trips restore Some(0); keep the advertised and decoded hashes identical.
    block.header.base_fee_per_gas = Some(0);
    let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block));
    let block: op_alloy_consensus::OpBlock = payload.clone().try_into_block().unwrap();
    let cfg = Arc::new(RollupConfig {
        genesis: kona_genesis::ChainGenesis {
            l2: alloy_eips::BlockNumHash { hash: block.header.hash_slow(), number: 0 },
            ..Default::default()
        },
        ..Default::default()
    });
    PayloadFixture { payload, cfg }
}

#[tokio::test]
async fn stale_committed_payload_does_not_import_or_change_forkchoice() {
    let PayloadFixture { payload, cfg } = payload_fixture();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let client = Arc::new(client);
    let (tx, mut rx) = mpsc::channel(1);
    let mut state = EngineState::default();
    let mut parent = L2BlockInfo::default();
    parent.block_info.number = 10;
    let task =
        CanonicalizeTask::new(client, cfg, payload, parent, tx, None, Arc::new(NoopBlockSink));
    let original = state;
    task.execute(&mut state).await.unwrap();
    assert!(matches!(
        rx.recv().await.unwrap(),
        Err(CanonicalizeTaskError::UnsafeHeadChangedSinceBuild)
    ));
    assert_eq!(state, original);
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn failed_forkchoice_update_is_relayed_and_retry_imports_the_same_payload() {
    let PayloadFixture { payload, cfg } = payload_fixture();
    let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect("engine_newPayloadV1", valid());
    l2.expect_error("engine_forkchoiceUpdatedV3");
    let client = Arc::new(client);
    let sink = Arc::new(RecordingSink::default());
    let (tx, mut rx) = mpsc::channel(1);
    let mut state = EngineState::default();
    let original = state;
    let task = CanonicalizeTask::new(
        client.clone(),
        cfg,
        payload.clone(),
        L2BlockInfo::default(),
        tx,
        None,
        sink.clone(),
    );
    task.execute(&mut state).await.unwrap();
    assert!(
        matches!(rx.recv().await.unwrap(), Err(CanonicalizeTaskError::PayloadInsertionFailed(err))
        if matches!(*err, InsertTaskError::ForkchoiceUpdateFailed(SynchronizeTaskError::ForkchoiceUpdateFailed(_))))
    );
    assert_eq!(state, original);
    assert!(sink.0.lock().unwrap().is_empty());

    l2.expect("engine_newPayloadV1", valid());
    l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
    task.execute(&mut state).await.unwrap();
    assert_eq!(rx.recv().await.unwrap().unwrap(), payload);
    assert_eq!(state.sync_state.unsafe_head().block_info.hash, payload.block_hash());
    assert_eq!(sink.0.lock().unwrap().len(), 1);
    l1.assert_finished();
    l2.assert_finished();
}

#[rstest::rstest]
#[case::deposits_only(true)]
#[case::with_user_transaction(false)]
#[tokio::test]
async fn invalid_committed_payload_does_not_build_an_uncommitted_replacement(
    #[case] deposits_only: bool,
) {
    let PayloadFixture { cfg, .. } = payload_fixture();
    let mut block = Block::<OpTxEnvelope>::default();
    block.body.transactions.push(OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        input: kona_protocol::L1BlockInfoBedrock::default().encode_calldata(),
        ..Default::default()
    })));
    if !deposits_only {
        block.body.transactions.push(OpTxEnvelope::Eip1559(
            TxEip1559::default().into_signed(Signature::test_signature()),
        ));
    }
    assert_eq!(block.body.transactions.iter().all(OpTxEnvelope::is_deposit), deposits_only);
    let payload_v1 = ExecutionPayloadV1::from_block_slow(&block);
    let payload = OpExecutionPayloadEnvelope::V1(payload_v1.clone());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_params(
        "engine_newPayloadV1",
        serde_json::json!([payload_v1]),
        PayloadStatus::from_status(PayloadStatusEnum::Invalid {
            validation_error: "invalid transaction".into(),
        }),
    );
    let client = Arc::new(client);
    let (tx, mut rx) = mpsc::channel(1);
    let mut state = EngineState::default();
    let original = state;
    let sink = Arc::new(RecordingSink::default());
    let (head_tx, mut head_rx) = watch::channel(L2BlockInfo::default());
    let task = CanonicalizeTask::new(
        client,
        cfg,
        payload,
        L2BlockInfo::default(),
        tx,
        Some(head_tx),
        sink.clone(),
    );
    // Like op-node's processNewPayload/handleInvalid, even deposits-only INVALID is relayed
    // to the sequencer for drop-and-rebuild, rather than terminating the engine task queue.
    task.execute(&mut state).await.unwrap();
    assert!(
        matches!(rx.recv().await.unwrap(), Err(CanonicalizeTaskError::PayloadInsertionFailed(err))
        if matches!(*err, InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. })))
    );
    assert_eq!(state, original);
    assert!(sink.0.lock().unwrap().is_empty());
    assert!(!head_rx.has_changed().unwrap());
    assert_eq!(*head_rx.borrow_and_update(), L2BlockInfo::default());
    // Strict RPC expectations also reject any replacement build or forkchoice update.
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn unsafe_head_is_published_before_the_canonicalization_response() {
    let PayloadFixture { payload, cfg } = payload_fixture();
    let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect("engine_newPayloadV1", valid());
    l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
    let client = Arc::new(client);
    let (tx, mut rx) = mpsc::channel(1);
    // Hold the response send pending. Head publication must happen even before this is drained.
    tx.send(Err(CanonicalizeTaskError::UnsafeHeadChangedSinceBuild)).await.unwrap();
    let (head_tx, mut head_rx) = watch::channel(L2BlockInfo::default());
    let task = CanonicalizeTask::new(
        client,
        cfg,
        payload.clone(),
        L2BlockInfo::default(),
        tx,
        Some(head_tx),
        Arc::new(NoopBlockSink),
    );
    let execution = tokio::spawn(async move { task.execute(&mut EngineState::default()).await });
    tokio::time::timeout(std::time::Duration::from_secs(1), head_rx.changed())
        .await
        .expect("head update must precede the blocked response")
        .unwrap();
    assert_eq!(head_rx.borrow().block_info.hash, payload.block_hash());
    assert!(!execution.is_finished(), "response must still be waiting for channel capacity");
    rx.recv().await.unwrap().unwrap_err();
    assert_eq!(rx.recv().await.unwrap().unwrap(), payload);
    execution.await.unwrap().unwrap();
    l1.assert_finished();
    l2.assert_finished();
}

#[tokio::test]
async fn invalid_forkchoice_state_is_relayed_as_a_reset_error() {
    let PayloadFixture { payload, cfg } = payload_fixture();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect("engine_newPayloadV1", PayloadStatus::from_status(PayloadStatusEnum::Valid));
    l2.expect_error_code("engine_forkchoiceUpdatedV3", -38002);
    let (tx, mut rx) = mpsc::channel(1);
    let mut state = EngineState::default();
    let task = CanonicalizeTask::new(
        Arc::new(client),
        cfg,
        payload,
        L2BlockInfo::default(),
        tx,
        None,
        Arc::new(NoopBlockSink),
    );
    // The engine queue sees success; only the sequencer can act on this reset.
    task.execute(&mut state).await.unwrap();
    let err = rx.recv().await.unwrap().unwrap_err();
    assert!(matches!(
        &err,
        CanonicalizeTaskError::PayloadInsertionFailed(inner)
            if matches!(**inner, InsertTaskError::ForkchoiceUpdateFailed(
                SynchronizeTaskError::InvalidForkchoiceState
            ))
    ));
    assert_eq!(err.severity(), crate::EngineTaskErrorSeverity::Reset);
    assert_eq!(state, EngineState::default());
    l1.assert_finished();
    l2.assert_finished();
}
