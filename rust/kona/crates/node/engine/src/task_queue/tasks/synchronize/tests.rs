use super::{synchronize, synchronize_unconditionally};
use crate::{
    EngineState, EngineSyncState, EngineSyncStateUpdate, EngineTaskError,
    EngineTaskErrorSeverity::{self, Reset, Temporary},
    test_utils::{RpcMock, TestEngineStateBuilder, test_block_info, test_engine_client},
};
use alloy_rpc_types_engine::{ForkchoiceState, INVALID_FORK_CHOICE_STATE_ERROR, PayloadStatusEnum};
use kona_genesis::RollupConfig;
use rstest::rstest;
use std::sync::Arc;

const FORKCHOICE_UPDATED: &str = "engine_forkchoiceUpdatedV3";

/// How the execution layer answers the forkchoice update.
#[derive(Clone, Copy, Debug)]
enum Reply {
    Invalid,
    InvalidForkchoiceState,
    Syncing,
    TransportError,
    Valid,
}

/// Which labels the engine holds after the forkchoice update.
#[derive(Clone, Copy, Debug)]
enum Labels {
    Kept,
    Moved,
}

/// An engine state that has not seen a `VALID` answer yet.
fn engine_state() -> EngineState {
    TestEngineStateBuilder::new()
        .with_unsafe_head(test_block_info(20))
        .with_safe_head(test_block_info(10))
        .with_finalized_head(test_block_info(5))
        .with_el_sync_finished(false)
        .build()
}

/// An update that moves the safe head, or one that leaves every label unchanged.
fn state_update(moves_safe_head: bool) -> EngineSyncStateUpdate {
    EngineSyncStateUpdate {
        safe_head: moves_safe_head.then(|| test_block_info(15)),
        ..Default::default()
    }
}

const fn labels_after(
    labels: Labels,
    before: EngineSyncState,
    moved: EngineSyncState,
) -> EngineSyncState {
    match labels {
        Labels::Kept => before,
        Labels::Moved => moved,
    }
}

fn expect_reply(l2: &RpcMock, forkchoice: ForkchoiceState, reply: Reply) {
    match reply {
        Reply::Invalid => l2.expect_forkchoice_updated(
            forkchoice,
            PayloadStatusEnum::Invalid { validation_error: "invalid".into() },
        ),
        Reply::InvalidForkchoiceState => {
            l2.expect_error_code(FORKCHOICE_UPDATED, INVALID_FORK_CHOICE_STATE_ERROR as i64)
        }
        Reply::Syncing => l2.expect_forkchoice_updated(forkchoice, PayloadStatusEnum::Syncing),
        Reply::TransportError => l2.expect_transport_error(FORKCHOICE_UPDATED),
        Reply::Valid => l2.expect_forkchoice_updated(forkchoice, PayloadStatusEnum::Valid),
    }
}

/// `synchronize` skips an update that leaves the labels unchanged.
#[rstest]
#[case::changed_labels_are_sent(true, 1)]
#[case::unchanged_labels_are_skipped(false, 0)]
#[tokio::test]
async fn synchronize_skips_unchanged_labels(
    #[case] moves_safe_head: bool,
    #[case] forkchoice_updates: usize,
) {
    let mut state = engine_state();
    let update = state_update(moves_safe_head);
    let expected_sync_state = state.sync_state.apply_update(update);
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    for _ in 0..forkchoice_updates {
        l2.expect_forkchoice_updated(
            expected_sync_state.create_forkchoice_state(),
            PayloadStatusEnum::Valid,
        );
    }

    synchronize(&client, &mut state, update).await.unwrap();

    assert_eq!(state.sync_state, expected_sync_state);
    l1.assert_finished();
    l2.assert_finished();
}

/// `synchronize_unconditionally` sends the update even when it leaves the labels unchanged.
#[rstest]
#[case::changed_labels_are_sent(true)]
#[case::unchanged_labels_are_sent(false)]
#[tokio::test]
async fn synchronize_unconditionally_sends_unchanged_labels(#[case] moves_safe_head: bool) {
    let mut state = engine_state();
    let update = state_update(moves_safe_head);
    let expected_sync_state = state.sync_state.apply_update(update);
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    l2.expect_forkchoice_updated(
        expected_sync_state.create_forkchoice_state(),
        PayloadStatusEnum::Valid,
    );

    synchronize_unconditionally(&client, &mut state, update).await.unwrap();

    assert_eq!(state.sync_state, expected_sync_state);
    l1.assert_finished();
    l2.assert_finished();
}

/// Only a `VALID` or `SYNCING` answer moves the labels, and only `VALID` marks the execution layer
/// as synced. Failures leave the engine state untouched.
#[rstest]
#[case::valid(Reply::Valid, Labels::Moved, true, None)]
#[case::syncing(Reply::Syncing, Labels::Moved, false, None)]
#[case::invalid_payload(Reply::Invalid, Labels::Kept, false, Some(Reset))]
#[case::invalid_forkchoice_state(Reply::InvalidForkchoiceState, Labels::Kept, false, Some(Reset))]
#[case::transport_error(Reply::TransportError, Labels::Kept, false, Some(Temporary))]
#[tokio::test]
async fn synchronize_applies_labels_only_when_accepted(
    #[case] reply: Reply,
    #[case] labels: Labels,
    #[case] el_sync_finished: bool,
    #[case] error_severity: Option<EngineTaskErrorSeverity>,
) {
    let mut state = engine_state();
    let update = state_update(true);
    let moved_sync_state = state.sync_state.apply_update(update);
    let expected_sync_state = labels_after(labels, state.sync_state, moved_sync_state);
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
    expect_reply(&l2, moved_sync_state.create_forkchoice_state(), reply);

    let result = synchronize(&client, &mut state, update).await;

    assert_eq!(result.as_ref().err().map(EngineTaskError::severity), error_severity, "{result:?}");
    assert_eq!(state.sync_state, expected_sync_state);
    assert_eq!(state.el_sync_finished, el_sync_finished);
    l1.assert_finished();
    l2.assert_finished();
}
