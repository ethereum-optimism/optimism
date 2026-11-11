use super::{ForkchoiceDelivery, synchronize};
use crate::{
    EngineState, EngineSyncState, EngineSyncStateUpdate, EngineTaskError,
    EngineTaskErrorSeverity::{self, Reset, Temporary},
    SafeLabel, SynchronizeTaskError,
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

/// An update that leaves the labels unchanged is skipped only when the caller allows it.
#[rstest]
#[case::changed_labels_are_sent(true, ForkchoiceDelivery::WhenChanged, 1)]
#[case::unchanged_labels_are_skipped(false, ForkchoiceDelivery::WhenChanged, 0)]
#[case::changed_labels_are_sent_always(true, ForkchoiceDelivery::Always, 1)]
#[case::unchanged_labels_are_sent_always(false, ForkchoiceDelivery::Always, 1)]
#[tokio::test]
async fn synchronize_skips_unchanged_labels_only_when_allowed(
    #[case] moves_safe_head: bool,
    #[case] delivery: ForkchoiceDelivery,
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

    synchronize(&client, &mut state, update, delivery).await.unwrap();

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

    let result = synchronize(&client, &mut state, update, ForkchoiceDelivery::WhenChanged).await;

    assert_eq!(result.as_ref().err().map(EngineTaskError::severity), error_severity, "{result:?}");
    assert_eq!(state.sync_state, expected_sync_state);
    assert_eq!(state.el_sync_finished, el_sync_finished);
    l1.assert_finished();
    l2.assert_finished();
}

/// An update that would put a safe label below the finalized head is refused before the
/// execution layer is asked, leaves the labels unchanged, and resets the engine rather than
/// stopping it.
#[rstest]
#[case::safe_head(EngineSyncStateUpdate { safe_head: Some(test_block_info(4)), ..Default::default() }, SafeLabel::Safe)]
#[case::local_safe_head(EngineSyncStateUpdate { local_safe_head: Some(test_block_info(4)), ..Default::default() }, SafeLabel::LocalSafe)]
#[tokio::test]
async fn synchronize_refuses_a_safe_label_below_finality(
    #[case] update: EngineSyncStateUpdate,
    #[case] expected_label: SafeLabel,
) {
    let mut state = engine_state();
    let state_before = state;
    let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));

    let error =
        synchronize(&client, &mut state, update, ForkchoiceDelivery::Always).await.unwrap_err();

    assert!(
        matches!(
            error,
            SynchronizeTaskError::FinalizedAheadOfSafe { finalized_number: 5, label, label_number: 4 }
                if label == expected_label
        ),
        "{error:?}"
    );
    assert_eq!(error.severity(), Reset);
    assert_eq!(state, state_before);
    l1.assert_finished();
    l2.assert_finished();
}
