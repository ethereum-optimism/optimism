//! Forkchoice-status tests: what a `SYNCING` answer to a forkchoice update does to the engine's
//! view of the chain.
//!
//! A `SYNCING` answer is accepted and the heads it carries are adopted — during the initial EL
//! sync *and* afterwards. This engine has no reqresp sync client and no unsafe payload buffer,
//! so pointing the execution layer at the gossiped tip is its only unsafe-chain catch-up channel:
//! op-node's EL-sync regime (`op-node/rollup/engine/engine_controller.go:587-594`), kept past the
//! initial sync because a verifier that falls behind — a restart, a dropped peer — re-enters it.
//! op-node's CL-sync regime refuses these heads, but it has the reqresp/payload-queue machinery
//! to catch up without them; dropping them here instead stranded verifiers behind a live
//! sequencer with no recovery channel (`TestUnsafeChainNotStalling_*`, `TestSequencerRestart`).
//! The head this adopts may name blocks the EL cannot serve yet; consolidation answers that
//! fetch miss with op-node's stall/reset split (see the consolidate task) rather than retrying
//! it forever.

use crate::{
    EngineState, EngineSyncStateUpdate, EngineTaskExt, SynchronizeTask,
    test_utils::{MockEngineClient, TestEngineStateBuilder, test_engine_client_builder},
};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use rstest::rstest;
use std::sync::Arc;

fn block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            number,
            hash: B256::repeat_byte(number as u8),
            parent_hash: B256::repeat_byte(number.saturating_sub(1) as u8),
            timestamp: number * 2,
        },
        ..Default::default()
    }
}

/// A client whose every forkchoice update answers `SYNCING`.
fn syncing_client() -> Arc<MockEngineClient> {
    Arc::new(
        test_engine_client_builder()
            .with_config(Arc::new(RollupConfig::default()))
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(PayloadStatus::new(
                PayloadStatusEnum::Syncing,
                None,
            )))
            .build(),
    )
}

/// A state whose heads sit at `head`, with EL sync in the given phase.
fn state_at(head: L2BlockInfo, el_sync_finished: bool) -> EngineState {
    TestEngineStateBuilder::new()
        .with_unsafe_head(head)
        .with_local_safe_head(head)
        .with_finalized_head(head)
        .with_el_sync_finished(el_sync_finished)
        .build()
}

/// A `SYNCING` forkchoice answer adopts the heads in both EL-sync phases: driving the execution
/// layer toward the optimistic head is this engine's only unsafe-chain catch-up channel, initial
/// sync or not.
#[rstest]
#[case::during_initial_el_sync(false)]
#[case::after_el_sync(true)]
#[tokio::test]
async fn a_syncing_fcu_adopts_the_heads(#[case] el_sync_finished: bool) {
    let mut state = state_at(block(3), el_sync_finished);

    SynchronizeTask::new(
        syncing_client(),
        Arc::new(RollupConfig::default()),
        EngineSyncStateUpdate { unsafe_head: Some(block(5)), ..Default::default() },
    )
    .execute(&mut state)
    .await
    .expect("a SYNCING answer is accepted");

    assert_eq!(
        state.sync_state.unsafe_head(),
        block(5),
        "the optimistic head drives EL sync and is adopted (el_sync_finished={el_sync_finished})"
    );
}
