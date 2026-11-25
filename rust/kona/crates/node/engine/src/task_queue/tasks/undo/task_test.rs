//! Tests for the [`UndoTask`].

use crate::{
    EngineState, EngineSyncStateUpdate, EngineTaskError, EngineTaskErrorSeverity,
    ExecutableEngineTask, SafeLabel, UndoTask, UndoTaskError,
    test_utils::{TestEngineStateBuilder, test_engine_client, test_l2_block},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;
use tokio::sync::mpsc;

const FINALIZED: u64 = 5;
const SAFE: u64 = 8;
const LOCAL_SAFE: u64 = 15;
const UNSAFE: u64 = 20;

fn chain() -> Vec<L2BlockInfo> {
    let mut blocks: Vec<op_alloy_consensus::OpBlock> = Vec::new();
    for number in 0..=UNSAFE {
        let parent = blocks.last().map_or(B256::ZERO, |block| block.header.hash_slow());
        blocks.push(test_l2_block(number, parent, 0));
    }
    let genesis = kona_genesis::ChainGenesis {
        l2: BlockNumHash { number: 0, hash: blocks[0].header.hash_slow() },
        ..Default::default()
    };
    blocks
        .iter()
        .map(|block| L2BlockInfo::from_block_and_genesis(block, &genesis).unwrap())
        .collect()
}

fn engine_state(chain: &[L2BlockInfo], finalized: Option<u64>) -> EngineState {
    let builder = TestEngineStateBuilder::new()
        .with_safe_label(SafeLabel::Verified { activation_timestamp: 0 })
        .with_unsafe_head(chain[UNSAFE as usize])
        .with_local_safe_head(chain[LOCAL_SAFE as usize])
        .with_safe_head(chain[SAFE as usize]);
    // A default finalized head is no finalized head, as before the engine's first reset.
    builder
        .with_finalized_head(
            finalized.map_or_else(L2BlockInfo::default, |number| chain[number as usize]),
        )
        .build()
}

/// The undo moves the local-safe head and the safe label back to the finalized head, keeps the
/// unsafe head, reports that the forkchoice update took effect, and restarts derivation.
#[tokio::test]
async fn undo_moves_local_safe_and_safe_back_to_the_finalized_head() {
    let chain = chain();
    let mut state = engine_state(&chain, Some(FINALIZED));
    let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
        local_safe_head: Some(chain[FINALIZED as usize]),
        safe_head: Some(chain[FINALIZED as usize]),
        ..Default::default()
    });
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    l2.expect_forkchoice_updated(expected.create_forkchoice_state(), PayloadStatusEnum::Valid);
    let (done, mut heard) = mpsc::channel(1);
    let task = UndoTask::new(Arc::new(client), done);

    let err = task.execute(&mut state).await.unwrap_err();

    assert!(
        matches!(err, UndoTaskError::Undone { local_safe } if local_safe == chain[FINALIZED as usize].block_info.id()),
        "{err:?}"
    );
    assert_eq!(err.severity(), EngineTaskErrorSeverity::Rederive);
    assert_eq!(state.sync_state.local_safe_head(), chain[FINALIZED as usize]);
    assert_eq!(state.sync_state.safe_head(), chain[FINALIZED as usize]);
    assert_eq!(state.sync_state.unsafe_head(), chain[UNSAFE as usize]);
    assert_eq!(heard.try_recv(), Ok(()));
    l1.assert_finished();
    l2.assert_finished();
}

/// Before the engine knows a finalized head the undo waits, without reporting it took effect.
#[tokio::test]
async fn undo_waits_for_a_finalized_head() {
    let chain = chain();
    let mut state = engine_state(&chain, None);
    let state_before = state;
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    let (done, mut heard) = mpsc::channel(1);

    let err = UndoTask::new(Arc::new(client), done).execute(&mut state).await.unwrap_err();

    assert!(matches!(err, UndoTaskError::NoFinalizedHead), "{err:?}");
    assert_eq!(err.severity(), EngineTaskErrorSeverity::Temporary);
    assert_eq!(state, state_before);
    assert!(heard.try_recv().is_err());
    l1.assert_finished();
    l2.assert_finished();
}
