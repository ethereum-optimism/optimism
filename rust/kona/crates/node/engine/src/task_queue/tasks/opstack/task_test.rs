//! Tests for the [`OpStackTask`].

use crate::{
    EngineBuildError, EngineState, ImportedBlockSink, InsertTaskError, NoopBlockSink,
    OpStackRequest, OpStackTask, OpenBlockError, SafeLabel, SealTaskError,
    test_utils::{TestEngineStateBuilder, test_engine_client, test_l2_block, test_rpc_block},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceState, ForkchoiceUpdated, PayloadAttributes, PayloadId,
    PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::sync::{Arc, Mutex};
use tokio::sync::{mpsc, watch};

const FINALIZED: u64 = 2;
const SAFE: u64 = 4;
const LOCAL_SAFE: u64 = 6;
const UNSAFE: u64 = 10;

/// The engine's chain, labelled as on a node that follows a cross-chain verifier.
struct Chain {
    blocks: Vec<OpBlock>,
    genesis: kona_genesis::ChainGenesis,
}

impl Chain {
    fn new() -> Self {
        let mut blocks: Vec<OpBlock> = Vec::new();
        for number in 0..=UNSAFE {
            let parent = blocks.last().map_or(B256::ZERO, |block| block.header.hash_slow());
            blocks.push(test_l2_block(number, parent, 0));
        }
        let genesis = kona_genesis::ChainGenesis {
            l2: BlockNumHash { number: 0, hash: blocks[0].header.hash_slow() },
            ..Default::default()
        };
        Self { blocks, genesis }
    }

    fn info(&self, number: u64) -> L2BlockInfo {
        L2BlockInfo::from_block_and_genesis(&self.blocks[number as usize], &self.genesis).unwrap()
    }

    fn engine_state(&self) -> EngineState {
        TestEngineStateBuilder::new()
            .with_safe_label(SafeLabel::Verified { activation_timestamp: 0 })
            .with_unsafe_head(self.info(UNSAFE))
            .with_local_safe_head(self.info(LOCAL_SAFE))
            .with_safe_head(self.info(SAFE))
            .with_finalized_head(self.info(FINALIZED))
            .build()
    }

    fn cfg(&self) -> Arc<RollupConfig> {
        Arc::new(RollupConfig { block_time: 2, genesis: self.genesis, ..Default::default() })
    }

    /// The forkchoice of the engine state with `head` as the head.
    fn forkchoice(&self, head: B256) -> ForkchoiceState {
        ForkchoiceState {
            head_block_hash: head,
            safe_block_hash: self.info(SAFE).block_info.hash,
            finalized_block_hash: self.info(FINALIZED).block_info.hash,
        }
    }

    /// A block at the unsafe head's height that conflicts with it, as a payload and as the block
    /// the engine decodes from it.
    fn sibling_of_unsafe_head(&self) -> (OpExecutionPayloadEnvelope, L2BlockInfo) {
        let mut block = test_l2_block(UNSAFE, self.info(UNSAFE - 1).block_info.hash, 1);
        // V1 round trips restore Some(0); keep the advertised and decoded hashes identical.
        block.header.base_fee_per_gas = Some(0);
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block));
        let decoded: OpBlock = payload.clone().try_into_block().unwrap();
        (payload, L2BlockInfo::from_block_and_genesis(&decoded, &self.genesis).unwrap())
    }
}

/// Records the blocks the engine hands over after an import.
#[derive(Debug, Default)]
struct RecordingSink(Mutex<Vec<L2BlockInfo>>);

impl ImportedBlockSink for RecordingSink {
    fn block_imported(&self, _: OpBlock, info: L2BlockInfo) {
        self.0.lock().unwrap().push(info);
    }
}

fn attributes(timestamp: u64) -> OpPayloadAttributes {
    OpPayloadAttributes {
        payload_attributes: PayloadAttributes { timestamp, ..Default::default() },
        no_tx_pool: Some(true),
        ..Default::default()
    }
}

/// An external sequencer builds on any block the execution layer has. Here it builds a sibling of
/// the unsafe head, which the node's own build task refuses as stale. Opening moves no label.
#[tokio::test]
async fn open_builds_on_a_parent_below_the_unsafe_head() {
    let chain = Chain::new();
    let parent = chain.info(UNSAFE - 1);
    let attributes = attributes(parent.block_info.timestamp + 2);
    let payload_id = PayloadId::new([7; 8]);
    let (client, l1, l2) = test_engine_client(chain.cfg());
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([parent.block_info.hash, true]),
        test_rpc_block(&chain.blocks[(UNSAFE - 1) as usize]),
    );
    l2.expect_params(
        "engine_forkchoiceUpdatedV2",
        serde_json::json!([chain.forkchoice(parent.block_info.hash), attributes]),
        ForkchoiceUpdated {
            payload_status: PayloadStatus::from_status(PayloadStatusEnum::Valid),
            payload_id: Some(payload_id),
        },
    );
    let (result_tx, mut result_rx) = mpsc::channel(1);
    let request = OpStackRequest::Open {
        parent: parent.block_info.id(),
        attributes: Box::new(attributes),
        result_tx,
    };
    let mut state = chain.engine_state();
    let before = state;

    OpStackTask::new(Arc::new(client), chain.cfg(), request, None, Arc::new(NoopBlockSink))
        .execute(&mut state)
        .await;

    assert_eq!(result_rx.recv().await.unwrap().unwrap(), payload_id);
    assert_eq!(state, before);
    l1.assert_finished();
    l2.assert_finished();
}

/// A parent the execution layer does not have is reported without a forkchoice update.
#[tokio::test]
async fn open_on_an_unknown_parent_is_reported() {
    let chain = Chain::new();
    let unknown = BlockNumHash { number: 3, hash: B256::repeat_byte(0xab) };
    let (client, l1, l2) = test_engine_client(chain.cfg());
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([unknown.hash, true]),
        None::<crate::test_utils::TestRpcBlock>,
    );
    let (result_tx, mut result_rx) = mpsc::channel(1);
    let request =
        OpStackRequest::Open { parent: unknown, attributes: Box::new(attributes(8)), result_tx };
    let mut state = chain.engine_state();

    OpStackTask::new(Arc::new(client), chain.cfg(), request, None, Arc::new(NoopBlockSink))
        .execute(&mut state)
        .await;

    assert!(matches!(
        result_rx.recv().await.unwrap(),
        Err(OpenBlockError::ParentNotFound(hash)) if hash == unknown.hash
    ));
    l1.assert_finished();
    l2.assert_finished();
}

/// A build the execution layer rejects is the caller's to handle. Unlike a rejected build of the
/// node's own sequencer, it does not reset the engine: the task cannot fail the queue.
#[tokio::test]
async fn open_rejected_by_the_execution_layer_is_reported() {
    let chain = Chain::new();
    let parent = chain.info(UNSAFE);
    let (client, l1, l2) = test_engine_client(chain.cfg());
    l2.expect_params(
        "eth_getBlockByHash",
        serde_json::json!([parent.block_info.hash, true]),
        test_rpc_block(&chain.blocks[UNSAFE as usize]),
    );
    l2.expect(
        "engine_forkchoiceUpdatedV2",
        ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Invalid {
            validation_error: "bad attributes".into(),
        })),
    );
    let (result_tx, mut result_rx) = mpsc::channel(1);
    let request = OpStackRequest::Open {
        parent: parent.block_info.id(),
        attributes: Box::new(attributes(parent.block_info.timestamp + 2)),
        result_tx,
    };
    let mut state = chain.engine_state();
    let before = state;

    OpStackTask::new(Arc::new(client), chain.cfg(), request, None, Arc::new(NoopBlockSink))
        .execute(&mut state)
        .await;

    assert!(matches!(
        result_rx.recv().await.unwrap(),
        Err(OpenBlockError::Build(EngineBuildError::InvalidPayload(_)))
    ));
    assert_eq!(state, before);
    l1.assert_finished();
    l2.assert_finished();
}

/// The execution layer's answer to `engine_getPayload` reaches the caller, including an unknown
/// payload ID, which the RPC reports as op-node does.
#[tokio::test]
async fn get_payload_reports_an_unknown_payload() {
    let chain = Chain::new();
    let (client, l1, l2) = test_engine_client(chain.cfg());
    l2.expect_error_code("engine_getPayloadV2", -38001);
    let (result_tx, mut result_rx) = mpsc::channel(1);
    let request =
        OpStackRequest::GetPayload { payload_id: PayloadId::new([7; 8]), timestamp: 22, result_tx };
    let mut state = chain.engine_state();

    OpStackTask::new(Arc::new(client), chain.cfg(), request, None, Arc::new(NoopBlockSink))
        .execute(&mut state)
        .await;

    let Err(SealTaskError::GetPayloadFailed(err)) = result_rx.recv().await.unwrap() else {
        panic!("expected the getPayload failure");
    };
    assert_eq!(err.as_error_resp().unwrap().code, -38001);
    l1.assert_finished();
    l2.assert_finished();
}

/// A committed block becomes the unsafe head, also when it replaces the unsafe head. It is an
/// unsafe block: neither the local-safe head nor the verifier's safe label moves. The node's
/// sequencer sees the new head before the caller hears of the commit.
#[tokio::test]
async fn commit_moves_only_the_unsafe_head() {
    let chain = Chain::new();
    let (payload, committed) = chain.sibling_of_unsafe_head();
    let (client, l1, l2) = test_engine_client(chain.cfg());
    l2.expect("engine_newPayloadV2", PayloadStatus::from_status(PayloadStatusEnum::Valid));
    l2.expect_forkchoice_updated(
        chain.forkchoice(committed.block_info.hash),
        PayloadStatusEnum::Valid,
    );
    let (unsafe_head_tx, unsafe_head_rx) = watch::channel(chain.info(UNSAFE));
    let sink = Arc::new(RecordingSink::default());
    let (result_tx, mut result_rx) = mpsc::channel(1);
    let request = OpStackRequest::Commit { payload: Box::new(payload), result_tx };
    let mut state = chain.engine_state();

    OpStackTask::new(Arc::new(client), chain.cfg(), request, Some(unsafe_head_tx), sink.clone())
        .execute(&mut state)
        .await;

    result_rx.recv().await.unwrap().unwrap();
    assert_eq!(state.sync_state.unsafe_head(), committed);
    assert_eq!(state.sync_state.local_safe_head(), chain.info(LOCAL_SAFE));
    assert_eq!(state.sync_state.safe_head(), chain.info(SAFE));
    assert_eq!(state.sync_state.finalized_head(), chain.info(FINALIZED));
    assert_eq!(*unsafe_head_rx.borrow(), committed);
    assert_eq!(sink.0.lock().unwrap().as_slice(), &[committed]);
    l1.assert_finished();
    l2.assert_finished();
}

/// A payload the execution layer finds invalid is reported, and leaves the labels as they were.
#[tokio::test]
async fn commit_of_an_invalid_payload_is_reported() {
    let chain = Chain::new();
    let (payload, _) = chain.sibling_of_unsafe_head();
    let (client, l1, l2) = test_engine_client(chain.cfg());
    l2.expect(
        "engine_newPayloadV2",
        PayloadStatus::from_status(PayloadStatusEnum::Invalid {
            validation_error: "bad block".into(),
        }),
    );
    let (unsafe_head_tx, unsafe_head_rx) = watch::channel(chain.info(UNSAFE));
    let (result_tx, mut result_rx) = mpsc::channel(1);
    let request = OpStackRequest::Commit { payload: Box::new(payload), result_tx };
    let mut state = chain.engine_state();
    let before = state;

    OpStackTask::new(
        Arc::new(client),
        chain.cfg(),
        request,
        Some(unsafe_head_tx),
        Arc::new(NoopBlockSink),
    )
    .execute(&mut state)
    .await;

    assert!(matches!(
        result_rx.recv().await.unwrap(),
        Err(InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Invalid { .. }))
    ));
    assert_eq!(state, before);
    assert_eq!(*unsafe_head_rx.borrow(), chain.info(UNSAFE));
    l1.assert_finished();
    l2.assert_finished();
}
