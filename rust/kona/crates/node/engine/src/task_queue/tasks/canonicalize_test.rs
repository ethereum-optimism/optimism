use super::*;
use crate::{InsertTaskError, NoopBlockSink, SynchronizeTaskError, test_utils::test_engine_client};
use alloy_consensus::Block;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use op_alloy_consensus::OpTxEnvelope;

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
    assert!(matches!(rx.recv().await.unwrap(), Err(SealTaskError::UnsafeHeadChangedSinceBuild)));
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
    assert!(matches!(rx.recv().await.unwrap(), Err(SealTaskError::PayloadInsertionFailed(err))
        if matches!(*err, InsertTaskError::ForkchoiceUpdateFailed(SynchronizeTaskError::ForkchoiceUpdateFailed(_)))));
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

#[tokio::test]
async fn invalid_committed_payload_does_not_build_an_uncommitted_replacement() {
    let PayloadFixture { payload, cfg } = payload_fixture();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect(
        "engine_newPayloadV1",
        PayloadStatus::from_status(PayloadStatusEnum::Invalid {
            validation_error: "invalid transaction".into(),
        }),
    );
    let client = Arc::new(client);
    let (tx, mut rx) = mpsc::channel(1);
    let mut state = EngineState::default();
    let original = state;
    let task = CanonicalizeTask::new(
        client,
        cfg,
        payload,
        L2BlockInfo::default(),
        tx,
        None,
        Arc::new(NoopBlockSink),
    );
    task.execute(&mut state).await.unwrap();
    assert!(matches!(rx.recv().await.unwrap(), Err(SealTaskError::PayloadInsertionFailed(err))
        if matches!(*err, InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. }))));
    assert_eq!(state, original);
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
    tx.send(Err(SealTaskError::UnsafeHeadChangedSinceBuild)).await.unwrap();
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
