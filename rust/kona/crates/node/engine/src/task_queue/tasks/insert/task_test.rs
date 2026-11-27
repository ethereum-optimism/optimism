//! Tests for the [`InsertTask`].

use crate::{
    DeniedBlocks, EngineTask, ExecutableEngineTask, InsertTask, InsertTaskError, NoopBlockSink,
    test_utils::{TestEngineStateBuilder, test_engine_client, test_l2_block},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayloadV1, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use rstest::rstest;
use std::sync::Arc;

const UNSAFE: u64 = 10;

/// The chain up to [`UNSAFE`], its rollup config, and the payload of the next block.
fn chain() -> (Vec<OpBlock>, Arc<RollupConfig>, OpExecutionPayloadEnvelope) {
    let mut blocks: Vec<OpBlock> = Vec::new();
    for number in 0..=UNSAFE {
        let parent = blocks.last().map_or(B256::ZERO, |block| block.header.hash_slow());
        blocks.push(test_l2_block(number, parent, 0));
    }
    let cfg = Arc::new(RollupConfig {
        genesis: kona_genesis::ChainGenesis {
            l2: BlockNumHash { number: 0, hash: blocks[0].header.hash_slow() },
            ..Default::default()
        },
        ..Default::default()
    });
    let mut next = test_l2_block(UNSAFE + 1, blocks[UNSAFE as usize].header.hash_slow(), 0);
    // V1 round trips restore Some(0); keep the advertised and decoded hashes identical.
    next.header.base_fee_per_gas = Some(0);
    let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&next));
    (blocks, cfg, payload)
}

/// While a cross-chain verifier denied a block above the finalized head, an unsafe payload is
/// dropped before it reaches the execution layer, as op-node gates unsafe payloads until finality
/// passes the denied block. A derived payload, and an unsafe one once finality passed the denied
/// block, are imported.
#[rstest]
#[case::unsafe_while_denial_above_finality(8, 7, false, false)]
#[case::derived_while_denial_above_finality(8, 7, true, true)]
#[case::unsafe_once_finality_passed(8, 8, false, true)]
#[tokio::test]
async fn unsafe_payload_waits_for_finality_to_pass_a_denial(
    #[case] denied: u64,
    #[case] finalized: u64,
    #[case] derived: bool,
    #[case] imported: bool,
) {
    let (blocks, cfg, payload) = chain();
    let info = |number: u64| {
        L2BlockInfo::from_block_and_genesis(&blocks[number as usize], &cfg.genesis).unwrap()
    };
    let (client, l1, l2) = test_engine_client(cfg.clone());
    if imported {
        let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
        l2.expect("engine_newPayloadV2", valid());
        l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
    }
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(info(UNSAFE))
        .with_safe_head(info(finalized))
        .with_finalized_head(info(finalized))
        .build();
    let task =
        InsertTask::new(Arc::new(client), cfg.clone(), payload, derived, Arc::new(NoopBlockSink))
            .with_denied_blocks(DeniedBlocks::applied(vec![BlockNumHash {
                number: denied,
                hash: B256::repeat_byte(0xde),
            }]));

    let result = task.execute(&mut state).await;

    if imported {
        assert_eq!(result.unwrap().block_info.number, UNSAFE + 1);
        assert_eq!(state.sync_state.unsafe_head().block_info.number, UNSAFE + 1);
    } else {
        assert!(matches!(
            result,
            Err(InsertTaskError::DeniedChainRecovery { highest_denied, finalized: at })
                if highest_denied == denied && at == finalized
        ));
        assert_eq!(state.sync_state.unsafe_head(), info(UNSAFE));
        // The queue drops the payload rather than retrying it.
        EngineTask::Insert(Box::new(task)).execute(&mut state).await.unwrap();
    }
    l1.assert_finished();
    l2.assert_finished();
}

/// An unsafe payload at or below the local-safe head is dropped before it reaches the execution
/// layer, as op-node drops one at or below its safe head: importing it could only fail the
/// forkchoice update, or move the unsafe head below local-safe.
#[rstest]
#[case::at_local_safe(UNSAFE + 1)]
#[case::below_local_safe(UNSAFE + 2)]
#[tokio::test]
async fn unsafe_payload_at_or_below_local_safe_is_dropped(#[case] local_safe: u64) {
    let (blocks, cfg, payload) = chain();
    let head = L2BlockInfo::from_block_and_genesis(&blocks[UNSAFE as usize], &cfg.genesis).unwrap();
    let mut safe = head;
    safe.block_info.number = local_safe;
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(safe)
        .with_local_safe_head(safe)
        .with_safe_head(head)
        .with_finalized_head(head)
        .build();
    let before = state;
    let task = InsertTask::new(Arc::new(client), cfg, payload, false, Arc::new(NoopBlockSink));

    assert!(matches!(
        task.execute(&mut state).await,
        Err(InsertTaskError::AtOrBelowLocalSafe { number, local_safe: at })
            if number == UNSAFE + 1 && at == local_safe
    ));
    assert_eq!(state, before);
    l1.assert_finished();
    l2.assert_finished();
}

/// A verifier that has not answered yet holds off no unsafe payload.
#[tokio::test]
async fn unsafe_payload_is_imported_before_the_verifier_answers() {
    let (blocks, cfg, payload) = chain();
    let head = L2BlockInfo::from_block_and_genesis(&blocks[UNSAFE as usize], &cfg.genesis).unwrap();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let valid = || PayloadStatus::from_status(PayloadStatusEnum::Valid);
    l2.expect("engine_newPayloadV2", valid());
    l2.expect("engine_forkchoiceUpdatedV3", ForkchoiceUpdated::new(valid()));
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(head).build();
    let (_sender, unknown) = DeniedBlocks::from_verifier(0);
    let task = InsertTask::new(Arc::new(client), cfg, payload, false, Arc::new(NoopBlockSink))
        .with_denied_blocks(unknown);

    assert_eq!(task.execute(&mut state).await.unwrap().block_info.number, UNSAFE + 1);
    l1.assert_finished();
    l2.assert_finished();
}
