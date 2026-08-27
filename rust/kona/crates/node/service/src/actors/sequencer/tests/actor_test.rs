#![cfg(test)]

use crate::{
    NodeActor, SequencerActorError,
    actors::sequencer::tests::test_util::{
        AttributesRequest, BLOCK_TIME, MAX_MULTI_BLOCKS, now_unix, sequencer_fixture,
        sequencer_fixture_with_config, single_block_config,
    },
};
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadId;
use alloy_transport::RpcError;
use kona_derive::{BuilderError, PipelineErrorKind};
use kona_engine::SealTaskError;
use kona_rpc::SequencerAdminAPIError;
use rstest::rstest;
use std::{collections::VecDeque, time::Duration};
use tokio::{sync::oneshot, time::Instant};

/// A head timestamp far enough ahead of the wall clock that the slot deadline leaves room for a
/// whole block group.
fn head_timestamp_ahead_of_clock() -> u64 {
    now_unix() + 30
}

#[rstest]
#[case::temp(PipelineErrorKind::Temporary(BuilderError::Custom(String::new()).into()), false)]
#[case::reset(PipelineErrorKind::Reset(BuilderError::Custom(String::new()).into()), false)]
#[case::critical(PipelineErrorKind::Critical(BuilderError::Custom(String::new()).into()), true)]
#[tokio::test]
async fn test_build_unsealed_payload_prepare_payload_attributes_error(
    #[case] forced_error: PipelineErrorKind,
    #[case] expect_err: bool,
) {
    let is_reset = matches!(forced_error, PipelineErrorKind::Reset(_));
    let mut fixture = sequencer_fixture(head_timestamp_ahead_of_clock(), 0);
    fixture.actor.attributes_builder.outcomes = VecDeque::from([Some(forced_error)]);

    let parent = fixture.engine.lock().unwrap().head;
    let result = fixture
        .actor
        .build_unsealed_payload(parent, fixture.l1_origin, parent.block_info.timestamp + BLOCK_TIME)
        .await;

    if expect_err {
        assert!(matches!(
            result.unwrap_err(),
            SequencerActorError::AttributesBuilder(PipelineErrorKind::Critical(_))
        ));
    } else {
        assert!(result.expect("non-critical errors are retried, not propagated").is_none());
    }
    assert_eq!(fixture.engine.lock().unwrap().resets, usize::from(is_reset));
}

#[tokio::test(start_paused = true)]
async fn test_ready_payloads_fill_a_group_then_the_slot_advances_the_timestamp() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    let mut fixture = sequencer_fixture(head_timestamp, 2);

    fixture.actor.run_slot().await.expect("the first slot builds a full group");
    let group_timestamp = head_timestamp + BLOCK_TIME;
    assert_eq!(
        fixture.sealed_timestamps(),
        vec![group_timestamp; MAX_MULTI_BLOCKS as usize],
        "a sequencer whose payloads are ready at once fills the group"
    );

    fixture.actor.run_slot().await.expect("the second slot starts a new group");
    assert_eq!(
        fixture.sealed_timestamps()[MAX_MULTI_BLOCKS as usize],
        group_timestamp + BLOCK_TIME,
        "the next slot moves on to the next timestamp"
    );
}

#[tokio::test(start_paused = true)]
async fn test_siblings_share_the_parents_epoch_and_extend_the_group() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    // The origin selector is free to answer once per sibling, and answers with a newer L1 block
    // every time: a slot that reselected mid-group would put a later epoch on its siblings.
    let mut fixture = sequencer_fixture(head_timestamp, 1..=MAX_MULTI_BLOCKS as usize);

    fixture.actor.run_slot().await.expect("the slot builds a full group");

    let epoch = fixture.l1_origin.id();
    let requests = fixture.attributes.lock().unwrap().clone();
    assert_eq!(requests.len(), MAX_MULTI_BLOCKS as usize);
    assert!(
        requests.iter().all(|r: &AttributesRequest| r.epoch == epoch),
        "every sibling stays in the epoch the slot selected, {requests:?}"
    );
    assert!(requests.iter().all(|r| r.timestamp == head_timestamp + BLOCK_TIME));

    // Each sibling builds on the one before it.
    let sealed_hashes = fixture.engine.lock().unwrap().sealed_hashes.clone();
    for (request, parent) in requests.iter().skip(1).zip(sealed_hashes.iter()) {
        assert_eq!(request.parent, *parent, "each sibling builds on the one before it");
    }
}

#[tokio::test(start_paused = true)]
async fn test_payload_that_is_never_ready_yields_one_block_at_the_deadline() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    let mut fixture = sequencer_fixture(head_timestamp, 1);
    fixture.engine.lock().unwrap().seals_immediately = false;

    fixture.actor.run_slot().await.expect("the slot seals at its deadline");

    assert_eq!(
        fixture.sealed_timestamps(),
        vec![head_timestamp + BLOCK_TIME],
        "waiting out the deadline leaves no room for a sibling"
    );
    assert!(
        fixture.engine.lock().unwrap().seal_deadlines[0].is_some(),
        "a chain that allows siblings seals against a readiness deadline"
    );
}

#[tokio::test(start_paused = true)]
async fn test_sequencer_behind_the_wall_clock_builds_one_block_per_timestamp() {
    // A head far in the past leaves every slot deadline already expired.
    let head_timestamp = now_unix() - 3_600;
    let mut fixture = sequencer_fixture(head_timestamp, 2);

    fixture.actor.run_slot().await.expect("the first catch-up slot builds one block");
    fixture.actor.run_slot().await.expect("the second catch-up slot builds one block");

    assert_eq!(
        fixture.sealed_timestamps(),
        vec![head_timestamp + BLOCK_TIME, head_timestamp + 2 * BLOCK_TIME]
    );
}

#[tokio::test(start_paused = true)]
async fn test_restarted_sequencer_does_not_extend_a_group_it_did_not_build() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    let mut fixture = sequencer_fixture(head_timestamp, 1);
    // Put the head mid-group, as a sequencer restarted between two siblings would find it.
    {
        let mut engine = fixture.engine.lock().unwrap();
        let sibling = engine.head;
        engine.head.block_info.parent_hash = sibling.block_info.hash;
        engine.head.block_info.number += 1;
        engine.head.block_info.hash = B256::with_last_byte(0x77);
        let head_block = engine.head.block_info;
        engine.blocks.insert(head_block.hash, head_block);
    }

    fixture.actor.run_slot().await.expect("the slot builds on the group without extending it");

    assert_eq!(
        fixture.attributes.lock().unwrap()[0].timestamp,
        head_timestamp + BLOCK_TIME,
        "a restarted sequencer starts a new timestamp rather than joining the group it found"
    );
}

#[tokio::test(start_paused = true)]
async fn test_block_without_the_transaction_pool_ends_the_slot() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    let mut fixture = sequencer_fixture(head_timestamp, 1);
    // Recovery mode is one of the reasons a block is built without the transaction pool.
    fixture.actor.in_recovery_mode = true;

    fixture.actor.run_slot().await.expect("the slot builds a single recovery block");

    assert_eq!(fixture.sealed_timestamps(), vec![head_timestamp + BLOCK_TIME]);
    assert_eq!(
        fixture.engine.lock().unwrap().sealed[0].attributes.no_tx_pool,
        Some(true),
        "the block that ended the slot is the one built without the transaction pool"
    );
}

#[tokio::test(start_paused = true)]
async fn test_engine_reset_mid_group_ends_the_group_and_reselects_the_origin() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    // Two origin selections: one per slot, the second after the reset.
    let mut fixture = sequencer_fixture(head_timestamp, 2);
    fixture.actor.attributes_builder.outcomes = VecDeque::from([
        None,
        Some(PipelineErrorKind::Reset(BuilderError::Custom(String::new()).into())),
    ]);

    fixture.actor.run_slot().await.expect("the reset ends the group without failing the slot");
    assert_eq!(fixture.sealed_timestamps(), vec![head_timestamp + BLOCK_TIME]);
    assert_eq!(fixture.engine.lock().unwrap().resets, 1);

    assert_eq!(fixture.attributes.lock().unwrap().len(), 2, "the group ended at the reset");

    // The origin selector allows exactly one call per slot, so a second slot that succeeds proves
    // the group ended and that this slot selected its origin afresh.
    fixture.actor.run_slot().await.expect("the next slot selects a fresh origin");
    assert_eq!(
        fixture.attributes.lock().unwrap()[2].timestamp,
        head_timestamp + 2 * BLOCK_TIME,
        "the next slot moves on to the next timestamp"
    );
}

#[tokio::test(start_paused = true)]
async fn test_admin_stop_between_siblings_hands_off_the_last_sealed_sibling() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    let mut fixture = sequencer_fixture(head_timestamp, 1);

    let (tx, rx) = oneshot::channel();
    fixture
        .admin_tx
        .send(crate::SequencerAdminQuery::StopSequencer(tx))
        .await
        .expect("the admin channel is open");

    fixture.actor.run_slot().await.expect("the stop ends the group without failing the slot");

    let stopped_at: Result<_, SequencerAdminAPIError> =
        rx.await.expect("the stop query is answered");
    assert_eq!(
        stopped_at.expect("stopping succeeds"),
        fixture.engine.lock().unwrap().head.hash(),
        "the hand-off hash is the sibling the sequencer had just sealed"
    );
    assert_eq!(
        fixture.sealed_timestamps(),
        vec![head_timestamp + BLOCK_TIME],
        "the group ends at the first sibling instead of filling up"
    );
    assert!(!fixture.actor.is_active);
}

#[tokio::test(start_paused = true)]
async fn test_repeated_seal_failures_back_off_to_the_slot_deadline() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    let mut fixture = sequencer_fixture(head_timestamp, 3);
    fixture.engine.lock().unwrap().seal_outcomes = VecDeque::from([
        Some(SealTaskError::PayloadJobUnknown(PayloadId::new([1u8; 8]))),
        Some(SealTaskError::GetPayloadFailed(RpcError::NullResp)),
    ]);

    fixture.actor.step().await.expect("a non-fatal seal failure ends the group, not the actor");
    let after_first_slot = Instant::now();
    fixture.actor.step().await.expect("the actor retries the timestamp at once");
    let retried_after = Instant::now();
    fixture.actor.step().await.expect("the actor keeps sequencing");
    let sealed_after = Instant::now();

    assert!(
        retried_after.duration_since(after_first_slot) < Duration::from_secs(1),
        "a slot that sealed nothing retries without waiting, {:?}",
        retried_after.duration_since(after_first_slot)
    );
    assert!(
        sealed_after.duration_since(retried_after) >= Duration::from_secs(BLOCK_TIME),
        "a second failure at the same timestamp waits out the slot, {:?}",
        sealed_after.duration_since(retried_after)
    );
    let sealed = fixture.sealed_timestamps();
    assert!(
        !sealed.is_empty() && sealed.iter().all(|ts| *ts == head_timestamp + BLOCK_TIME),
        "the failed seals left the chain where it was, so the retries kept the timestamp, {sealed:?}"
    );
}

#[tokio::test(start_paused = true)]
async fn test_chain_without_siblings_builds_one_block_per_block_time_at_the_deadline() {
    let head_timestamp = head_timestamp_ahead_of_clock();
    let mut fixture = sequencer_fixture_with_config(head_timestamp, 2, single_block_config());

    let start = Instant::now();
    fixture.actor.run_slot().await.expect("the first slot builds one block");
    let after_first_slot = Instant::now();
    fixture.actor.run_slot().await.expect("the second slot builds one block");

    assert_eq!(
        fixture.sealed_timestamps(),
        vec![head_timestamp + BLOCK_TIME, head_timestamp + 2 * BLOCK_TIME],
        "a chain that does not allow siblings advances one timestamp per block"
    );
    assert!(
        fixture.engine.lock().unwrap().seal_deadlines.iter().all(Option::is_none),
        "the execution layer is never asked when the payload is worth sealing"
    );
    assert!(
        after_first_slot.duration_since(start) >= Duration::from_secs(30),
        "the block is sealed at its slot deadline, not when it was built, {:?}",
        after_first_slot.duration_since(start)
    );
    assert!(
        Instant::now().duration_since(after_first_slot) >= Duration::from_secs(BLOCK_TIME),
        "the slots that follow wait for their own deadline too"
    );
}
