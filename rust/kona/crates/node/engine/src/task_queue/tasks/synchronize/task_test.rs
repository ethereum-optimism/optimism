//! Forkchoice-status tests: what a `SYNCING` answer to a forkchoice update may and may not do
//! to the engine's view of the chain.
//!
//! The dividing line is EL sync. While the execution layer is still performing its initial sync,
//! `SYNCING` is the expected answer and the update is adopted — that is what drives the sync
//! (op-node's EL-sync regime, `op-node/rollup/engine/engine_controller.go:587-594`). Once EL sync
//! has finished, op-node accepts only `VALID`: a `SYNCING` answer means the execution layer
//! cannot canonicalize the head it was handed, and adopting that head anyway detaches the node's
//! forkchoice from the chain the EL actually has (`engine_controller.go:586-595` and `:700-706`).
//! Against a sync-tester EL that detachment wedged derivation permanently: consolidation asked
//! the EL for a block it never had, forever.

use crate::{
    EngineState, EngineSyncStateUpdate, EngineTaskError, EngineTaskErrorSeverity, EngineTaskExt,
    SynchronizeTask, SynchronizeTaskError,
    test_utils::{MockEngineClient, TestEngineStateBuilder, test_engine_client_builder},
};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
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

fn advance_unsafe_to(
    client: Arc<MockEngineClient>,
    head: L2BlockInfo,
) -> SynchronizeTask<MockEngineClient> {
    SynchronizeTask::new(
        client,
        Arc::new(RollupConfig::default()),
        EngineSyncStateUpdate { unsafe_head: Some(head), ..Default::default() },
    )
}

/// After EL sync has finished, a `SYNCING` answer must not move the heads: op-node treats
/// anything but `VALID` as a failure then and re-syncs with the execution layer's actual state
/// instead of adopting a head the EL does not have
/// (`op-node/rollup/engine/engine_controller.go:586-595`, `:700-706`, `:873-879`).
#[tokio::test]
async fn a_syncing_fcu_after_el_sync_is_rejected_and_not_adopted() {
    let mut state = state_at(block(3), true);

    let result = advance_unsafe_to(syncing_client(), block(5)).execute(&mut state).await;

    assert!(
        matches!(result, Err(SynchronizeTaskError::ForkchoiceUpdatedSyncing)),
        "expected the SYNCING answer to be rejected, got {result:?}"
    );
    assert_eq!(
        state.sync_state.unsafe_head(),
        block(3),
        "a head the execution layer answered SYNCING for must not be adopted"
    );
}

/// The rejection asks for a reset, not a retry of the same update: op-node's answer to a
/// post-EL-sync `SYNCING` forkchoice status is to re-discover the EL's actual chain state
/// (`op-node/rollup/engine/engine_controller.go:700-706`).
#[test]
fn the_post_el_sync_syncing_rejection_resets() {
    assert_eq!(
        SynchronizeTaskError::ForkchoiceUpdatedSyncing.severity(),
        EngineTaskErrorSeverity::Reset
    );
}

/// During the initial EL sync the same answer is expected and the update is adopted — this is
/// what drives EL sync toward the gossiped tip (op-node's EL-sync regime,
/// `op-node/rollup/engine/engine_controller.go:587-594`).
#[tokio::test]
async fn a_syncing_fcu_during_el_sync_is_adopted() {
    let mut state = state_at(block(3), false);

    advance_unsafe_to(syncing_client(), block(5))
        .execute(&mut state)
        .await
        .expect("a SYNCING answer during EL sync is accepted");

    assert_eq!(
        state.sync_state.unsafe_head(),
        block(5),
        "during EL sync the optimistic head drives the sync and is adopted"
    );
}
