//! Deterministic recovery through the real channel reader, batch queue, and actor handshake.

use super::*;
use crate::{
    DerivationActor, DerivationActorRequest, QueuedDerivationEngineClient,
    QueuedEngineDerivationClient,
};
use alloy_consensus::{SignableTransaction, TxEip1559};
use alloy_eips::{
    BlockNumHash,
    eip2718::{Decodable2718, Encodable2718},
};
use alloy_primitives::{Address, B256, Bytes, Sealed, Signature, U256};
use alloy_rlp::Encodable;
use alloy_rpc_types_engine::{
    ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2, ExecutionPayloadV1, ForkchoiceState,
    ForkchoiceUpdated, PayloadAttributes, PayloadId, PayloadStatus, PayloadStatusEnum,
};
use kona_derive::{
    AttributesQueue, BatchQueue, BatchStream, ChannelReader, ChannelReaderProvider,
    DerivationPipeline, OriginAdvancer, OriginProvider, PipelineError, PipelineResult, Stage,
    test_utils::{TestAttributesBuilder, TestL2ChainProvider},
};
use kona_engine::{
    EngineState, EngineSyncStateUpdate, NoopBlockSink,
    test_utils::{RpcMock, test_engine_client},
};
use kona_genesis::SystemConfig;
use kona_protocol::{
    Batch, BatchReader, BlockInfo, L1BlockInfoBedrock, L2BlockInfo, SingleBatch, SpanBatch,
};
use kona_safedb::{SafeDatabase, SafeDb, SafeDbError};
use op_alloy_consensus::{OpBlock, OpTxEnvelope, TxDeposit};
use op_alloy_rpc_types_engine::OpPayloadAttributes;

/// No more L1 data after the single channel supplied directly to `ChannelReader`. Any reset or
/// flush would lose the remaining valid batch and is a test failure.
#[derive(Debug)]
struct ChannelSource(BlockInfo);
impl OriginProvider for ChannelSource {
    fn origin(&self) -> Option<BlockInfo> {
        Some(self.0)
    }
}
#[async_trait]
impl OriginAdvancer for ChannelSource {
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        Err(PipelineError::Eof.temp())
    }
}
#[async_trait]
impl ChannelReaderProvider for ChannelSource {
    async fn next_data(&mut self) -> PipelineResult<Option<Bytes>> {
        Err(PipelineError::Eof.temp())
    }
}
#[async_trait]
impl Stage for ChannelSource {
    async fn reset(&mut self, _: BlockNumHash, _: SystemConfig) -> PipelineResult<()> {
        panic!("must not replay the channel")
    }
    async fn activate(&mut self) -> PipelineResult<()> {
        panic!("no hardfork transition")
    }
    async fn flush_channel(&mut self) -> PipelineResult<()> {
        panic!("must preserve later batches in the channel")
    }
}

fn valid() -> PayloadStatus {
    PayloadStatus::from_status(PayloadStatusEnum::Valid)
}
fn fcu() -> ForkchoiceUpdated {
    ForkchoiceUpdated::new(valid())
}

fn user_transaction(chain_id: u64, nonce: u64) -> Bytes {
    TxEip1559 { chain_id, nonce, gas_limit: 21_000, ..Default::default() }
        .into_signed(Signature::test_signature())
        .encoded_2718()
        .into()
}

fn attributes(parent: L2BlockInfo, origin: BlockInfo, timestamp: u64) -> OpPayloadAttributes {
    let deposit = OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        input: L1BlockInfoBedrock::new(
            origin.number,
            origin.timestamp,
            0,
            origin.hash,
            parent.seq_num + 1,
            Address::ZERO,
            U256::ZERO,
            U256::ZERO,
        )
        .encode_calldata(),
        ..Default::default()
    }));
    OpPayloadAttributes {
        payload_attributes: PayloadAttributes { timestamp, ..Default::default() },
        transactions: Some(vec![deposit.encoded_2718().into()]),
        no_tx_pool: Some(true),
        gas_limit: Some(30_000_000),
        ..Default::default()
    }
}

fn payload(parent: L2BlockInfo, attributes: &OpPayloadAttributes) -> ExecutionPayloadV1 {
    let mut block = OpBlock::default();
    block.header.number = parent.block_info.number + 1;
    block.header.parent_hash = parent.hash();
    block.header.timestamp = attributes.payload_attributes.timestamp;
    block.header.gas_limit = attributes.gas_limit.unwrap();
    block.header.base_fee_per_gas = Some(0);
    block.body.transactions = attributes
        .transactions
        .as_ref()
        .unwrap()
        .iter()
        .map(|tx| OpTxEnvelope::decode_2718(&mut tx.as_ref()).unwrap())
        .collect();
    ExecutionPayloadV1::from_block_slow(&block)
}

fn expect_build(
    rpc: &RpcMock,
    forkchoice: ForkchoiceState,
    attributes: &OpPayloadAttributes,
    payload: &ExecutionPayloadV1,
) {
    rpc.expect_params(
        "engine_forkchoiceUpdatedV2",
        serde_json::json!([forkchoice, attributes]),
        ForkchoiceUpdated { payload_status: valid(), payload_id: Some(PayloadId::new([1; 8])) },
    );
    rpc.expect(
        "engine_getPayloadV2",
        ExecutionPayloadEnvelopeV2 {
            execution_payload: ExecutionPayloadFieldV2::V1(payload.clone()),
            block_value: Default::default(),
        },
    );
}

/// A valid prefix must stay pending even if L1 finalizes before the next block is rejected.
/// The bad input is never made valid: replaying it, flushing, or resetting fails the strict RPC
/// script or `ChannelSource`. The later valid batch is encoded after the span in the SAME channel.
#[rstest::rstest]
#[case::invalid_attributes(true)]
#[case::invalid_payload(false)]
#[tokio::test(start_paused = true)]
async fn rejected_span_rolls_back_without_replaying_and_keeps_later_channel_batch(
    #[case] rejected_at_build: bool,
) {
    let origin = BlockInfo { number: 1, hash: B256::repeat_byte(1), ..Default::default() };
    let inclusion =
        BlockInfo { number: 2, hash: B256::repeat_byte(2), timestamp: 12, ..Default::default() };
    let boundary = L2BlockInfo {
        block_info: BlockInfo {
            number: 10,
            hash: B256::repeat_byte(10),
            timestamp: 20,
            ..Default::default()
        },
        l1_origin: origin.id(),
        seq_num: 10,
    };
    let mut cfg = RollupConfig {
        block_time: 2,
        seq_window_size: 100,
        max_sequencer_drift: 600,
        ..Default::default()
    };
    cfg.hardforks.delta_time = Some(0);
    let cfg = Arc::new(cfg);
    let bad_transaction = user_transaction(cfg.l2_chain_id.id(), u64::MAX);
    let good_transaction = user_transaction(cfg.l2_chain_id.id(), 0);
    let batch = |timestamp, transactions| SingleBatch {
        parent_hash: boundary.hash(),
        epoch_num: origin.number,
        epoch_hash: origin.hash,
        timestamp,
        transactions,
    };
    let mut span = SpanBatch { chain_id: cfg.l2_chain_id.id(), ..Default::default() };
    for single in [batch(22, vec![]), batch(24, vec![bad_transaction.clone()]), batch(26, vec![])] {
        span.append_singular_batch(single, 1).unwrap();
    }
    let mut channel = Vec::new();
    for batch in [Batch::Span(span), Batch::Single(batch(22, vec![good_transaction.clone()]))] {
        let mut encoded = Vec::new();
        batch.encode(&mut encoded).unwrap();
        Bytes::from(encoded).encode(&mut channel);
    }
    // Start at the decompressed channel boundary; both batches share one reader and cursor.
    let mut reader = ChannelReader::new(ChannelSource(inclusion), cfg.clone());
    let mut batch_reader = BatchReader::new(Vec::new(), channel.len(), inclusion.timestamp);
    batch_reader.decompressed = channel;
    reader.next_batch = Some(batch_reader);
    let stream = BatchStream::new(reader, cfg.clone(), TestL2ChainProvider::default());
    let mut batches = BatchQueue::new(cfg.clone(), stream, TestL2ChainProvider::default());
    batches.origin = Some(inclusion);
    batches.l1_blocks = vec![origin, inclusion];
    let prefix_attributes = attributes(boundary, origin, 22);
    let prefix_payload = payload(boundary, &prefix_attributes);
    let prefix_block: OpBlock =
        op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope::V1(prefix_payload.clone())
            .try_into_block()
            .unwrap();
    let prefix = L2BlockInfo::from_block_and_genesis(&prefix_block, &cfg.genesis).unwrap();
    let invalid_base = attributes(prefix, origin, 24);
    let mut invalid_attributes = invalid_base.clone();
    invalid_attributes.transactions.as_mut().unwrap().push(bad_transaction);
    let mut good_attributes = attributes(boundary, origin, 22);
    let good_base = good_attributes.clone();
    good_attributes.transactions.as_mut().unwrap().push(good_transaction);
    let good_payload = payload(boundary, &good_attributes);
    let good_block: OpBlock =
        op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope::V1(good_payload.clone())
            .try_into_block()
            .unwrap();
    let good = L2BlockInfo::from_block_and_genesis(&good_block, &cfg.genesis).unwrap();
    let builder = TestAttributesBuilder {
        attributes: vec![Ok(good_base), Ok(invalid_base), Ok(prefix_attributes.clone())],
    };
    let pipeline = DerivationPipeline::new(
        AttributesQueue::new(cfg.clone(), batches, builder),
        cfg.clone(),
        TestL2ChainProvider::default(),
    );
    let state = EngineState {
        sync_state: EngineState::default().sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(boundary),
            local_safe_head: Some(boundary),
            safe_head: Some(boundary),
            finalized_head: Some(boundary),
        }),
        el_sync_finished: true,
        ..Default::default()
    };
    let (state_tx, state_rx) = watch::channel(state);
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let (engine_tx, engine_rx) = mpsc::channel(8);
    let (derivation_tx, derivation_rx) = mpsc::channel(8);
    let dir = tempfile::TempDir::new().unwrap();
    let db = Arc::new(SafeDatabase::new(dir.path()).unwrap());
    let earlier = L2BlockInfo {
        block_info: BlockInfo { number: 9, hash: B256::repeat_byte(9), ..Default::default() },
        ..Default::default()
    };
    db.safe_head_updated(earlier, BlockNumHash::default()).unwrap();
    db.safe_head_updated(boundary, origin.id()).unwrap();
    let mut derivation = DerivationActor::new(
        QueuedDerivationEngineClient::new(engine_tx.clone()),
        derivation_rx,
        pipeline,
        Some(db.clone()),
        BlockNumHash::default(),
        watch::channel(None).0,
    );
    let mut engine = EngineActor::new(
        Arc::new(client),
        cfg,
        QueuedEngineDerivationClient::new(derivation_tx.clone()),
        Engine::new(state, state_tx),
        None,
        engine_rx,
        Arc::new(NoopBlockSink),
    );
    engine.el_sync_complete = true;
    engine.last_derivation_parent_sent = boundary;
    derivation_tx
        .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(boundary)))
        .await
        .unwrap();
    derivation.step().await.unwrap();
    expect_build(
        &l2,
        state.sync_state.create_forkchoice_state(),
        &prefix_attributes,
        &prefix_payload,
    );
    l2.expect("engine_newPayloadV1", valid());
    l2.expect_params(
        "engine_forkchoiceUpdatedV3",
        serde_json::json!([
            ForkchoiceState {
                head_block_hash: prefix.hash(),
                ..state.sync_state.create_forkchoice_state()
            },
            null
        ]),
        fcu(),
    );
    engine.step().await.unwrap();
    engine.drain().await.unwrap();
    assert_eq!(state_rx.borrow().pending_safe_head, Some(prefix));
    assert_eq!(state_rx.borrow().sync_state.local_safe_head(), boundary);
    assert_eq!(state_rx.borrow().sync_state.safe_head(), boundary);
    derivation.step().await.unwrap(); // confirm prefix, derive the invalid second block
    derivation_tx
        .send(DerivationActorRequest::ProcessFinalizedL1Block(Box::new(inclusion)))
        .await
        .unwrap();
    derivation.step().await.unwrap();
    assert_eq!(db.last_entry().unwrap().safe_head, boundary.block_info.id());
    assert!(matches!(
        db.l1_at_safe_head(prefix.block_info.number),
        Err(SafeDbError::L1AtSafeHeadNotFound)
    ));
    // Only the invalid attributes may be queued; premature finalization adds another request.
    assert_eq!(engine.inbound_request_rx.len(), 1);
    if rejected_at_build {
        l2.expect_error_code("engine_forkchoiceUpdatedV2", -38003);
    } else {
        expect_build(
            &l2,
            ForkchoiceState {
                head_block_hash: prefix.hash(),
                ..state.sync_state.create_forkchoice_state()
            },
            &invalid_attributes,
            &payload(prefix, &invalid_attributes),
        );
        l2.expect(
            "engine_newPayloadV1",
            PayloadStatus::from_status(PayloadStatusEnum::Invalid {
                validation_error: "deterministic invalid nonce".into(),
            }),
        );
    }
    l2.expect_error("engine_forkchoiceUpdatedV3");
    engine.step().await.unwrap();
    engine.drain().await.unwrap(); // retain rollback after a transport failure
    assert!(engine.engine.state().need_fcu_call_backup_unsafe_reorg);
    assert_eq!(engine.engine.len(), 1);
    assert_eq!(derivation_tx.capacity(), 8);
    l2.expect(
        "engine_forkchoiceUpdatedV3",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Syncing)),
    );
    engine.drain().await.unwrap(); // SYNCING cannot acknowledge the rollback either
    assert!(engine.engine.state().need_fcu_call_backup_unsafe_reorg);
    l2.expect_params(
        "engine_forkchoiceUpdatedV3",
        serde_json::json!([state.sync_state.create_forkchoice_state(), null]),
        fcu(),
    );
    engine.drain().await.unwrap();
    assert!(engine.engine.is_empty());
    assert_eq!(state_rx.borrow().sync_state, state.sync_state);
    assert_eq!(state_rx.borrow().pending_safe_head, None);
    derivation.step().await.unwrap(); // old parent discards cached span suffix, reads later batch
    expect_build(&l2, state.sync_state.create_forkchoice_state(), &good_attributes, &good_payload);
    l2.expect("engine_newPayloadV1", valid());
    l2.expect_params(
        "engine_forkchoiceUpdatedV3",
        serde_json::json!([
            ForkchoiceState {
                head_block_hash: good.hash(),
                safe_block_hash: good.hash(),
                finalized_block_hash: boundary.hash()
            },
            null
        ]),
        fcu(),
    );
    engine.step().await.unwrap();
    engine.drain().await.unwrap();
    derivation.step().await.unwrap();
    assert_eq!(state_rx.borrow().sync_state.local_safe_head(), good);
    assert_eq!(state_rx.borrow().sync_state.finalized_head(), boundary);
    assert_eq!(db.first_entry().unwrap().safe_head, earlier.block_info.id());
    assert_eq!(db.l1_at_safe_head(10).unwrap().l1, origin.id());
    assert_eq!(db.last_entry().unwrap().l1, inclusion.id());
    assert_eq!(db.last_entry().unwrap().safe_head, good.block_info.id());
    assert_eq!(db.l1_at_safe_head(good.block_info.number).unwrap().l1, inclusion.id());
    derivation_tx
        .send(DerivationActorRequest::ProcessFinalizedL1Block(Box::new(inclusion)))
        .await
        .unwrap();
    derivation.step().await.unwrap();
    assert!(
        matches!(engine.inbound_request_rx.try_recv().unwrap(), EngineActorRequest::ProcessFinalizedL2Block(id) if id.number() == good.block_info.number)
    );
    l1.assert_finished();
    l2.assert_finished();
}

/// A rejection at the first block still needs a confirmation: suppressing an unchanged parent
/// leaves derivation waiting forever for an acknowledgement.
#[tokio::test]
async fn first_block_rejection_acknowledges_the_unchanged_parent() {
    let boundary = L2BlockInfo {
        block_info: BlockInfo {
            number: 10,
            hash: B256::repeat_byte(10),
            timestamp: 20,
            ..Default::default()
        },
        ..Default::default()
    };
    let cfg = Arc::new(RollupConfig::default());
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let state = kona_engine::test_utils::TestEngineStateBuilder::new()
        .with_unsafe_head(boundary)
        .with_safe_head(L2BlockInfo {
            block_info: BlockInfo { number: 9, hash: B256::repeat_byte(9), ..Default::default() },
            ..Default::default()
        })
        .with_finalized_head(L2BlockInfo {
            block_info: BlockInfo { number: 8, hash: B256::repeat_byte(8), ..Default::default() },
            ..Default::default()
        })
        .build();
    let attributes = kona_engine::test_utils::TestAttributesBuilder::new()
        .with_parent(boundary)
        .with_timestamp(22)
        .with_transactions(vec![user_transaction(cfg.l2_chain_id.id(), u64::MAX)])
        .build();
    let (derivation_tx, mut derivation_rx) = mpsc::channel(8);
    let (_request_tx, request_rx) = mpsc::channel(8);
    let mut actor = EngineActor::new(
        Arc::new(client),
        cfg,
        QueuedEngineDerivationClient::new(derivation_tx),
        Engine::new(state, watch::channel(state).0),
        None,
        request_rx,
        Arc::new(NoopBlockSink),
    );
    actor.el_sync_complete = true;
    actor.last_derivation_parent_sent = boundary;
    actor.engine.enqueue(EngineTask::Consolidate(Box::new(ConsolidateTask::new(
        actor.client.clone(),
        actor.rollup.clone(),
        attributes.into(),
        Arc::new(NoopBlockSink),
    ))));
    l2.expect_error_code("engine_forkchoiceUpdatedV2", -38003);
    l2.expect_params(
        "engine_forkchoiceUpdatedV3",
        serde_json::json!([state.sync_state.create_forkchoice_state(), null]),
        fcu(),
    );
    actor.drain().await.unwrap();
    assert!(
        matches!(derivation_rx.try_recv().unwrap(), DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(head) if *head == boundary)
    );
    assert!(actor.engine.is_empty());
    assert!(!actor.reset_pending);
    assert!(derivation_rx.try_recv().is_err());
    actor.drain().await.unwrap();
    assert!(derivation_rx.try_recv().is_err(), "a rejection is acknowledged exactly once");
    l1.assert_finished();
    l2.assert_finished();
}
