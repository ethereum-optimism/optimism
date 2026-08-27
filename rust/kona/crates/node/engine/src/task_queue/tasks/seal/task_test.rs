use crate::{
    BuildSealCoupling::{self, Atomic, Detached},
    EngineTaskExt, PayloadReadiness, SealTask, SealTaskError,
    test_utils::{
        MockEngineClient, TestAttributesBuilder, TestEngineStateBuilder, test_block_info,
        test_engine_client_builder,
    },
};
use alloy_rpc_types_engine::PayloadId;
use kona_genesis::RollupConfig;
use rstest::rstest;
use std::{sync::Arc, time::Duration};
use tokio::{sync::mpsc, time::Instant};

/// The payload id every task in this module seals.
fn payload_id() -> PayloadId {
    PayloadId::new([1u8; 8])
}

/// The paths `execute` can be steered into: aborting the seal as stale, abandoning a build job the
/// execution layer has forgotten, or proceeding to the payload fetch — which the unconfigured mock
/// client fails, surfacing as [`SealTaskError::GetPayloadFailed`].
#[derive(Debug, PartialEq, Eq)]
enum SealOutcome {
    AbortedAsStale,
    PayloadJobGone,
    ProceededToSeal,
}

fn classify(err: &SealTaskError) -> SealOutcome {
    match err {
        SealTaskError::UnsafeHeadChangedSinceBuild => SealOutcome::AbortedAsStale,
        SealTaskError::PayloadJobUnknown(_) => SealOutcome::PayloadJobGone,
        SealTaskError::GetPayloadFailed(_) => SealOutcome::ProceededToSeal,
        other => panic!("unexpected seal error: {other:?}"),
    }
}

/// A seal task for a payload built on `parent_block`, sealed through `client`.
fn seal_task(
    client: Arc<MockEngineClient>,
    ready_deadline: Option<Instant>,
    coupling: BuildSealCoupling,
) -> SealTask<MockEngineClient> {
    SealTask::new(
        client,
        Arc::new(RollupConfig::default()),
        payload_id(),
        TestAttributesBuilder::new().with_parent(test_block_info(10)).build(),
        ready_deadline,
        false,
        coupling,
        None,
        Arc::new(crate::NoopBlockSink),
    )
}

#[rstest]
#[case::detached_with_moved_unsafe_head(Detached, false, SealOutcome::AbortedAsStale)]
#[case::detached_with_current_unsafe_head(Detached, true, SealOutcome::ProceededToSeal)]
#[case::atomic_with_reorged_unsafe_head(Atomic, false, SealOutcome::ProceededToSeal)]
#[case::atomic_with_current_unsafe_head(Atomic, true, SealOutcome::ProceededToSeal)]
#[tokio::test]
async fn unsafe_head_check_variants(
    #[case] coupling: BuildSealCoupling,
    #[case] unsafe_head_at_parent: bool,
    #[case] expected: SealOutcome,
    #[values(true, false)] with_channel: bool,
) {
    let parent_block = test_block_info(10);
    let unsafe_head = if unsafe_head_at_parent { parent_block } else { test_block_info(15) };

    let attributes = TestAttributesBuilder::new().with_parent(parent_block).build();
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();

    let (tx, mut rx) = mpsc::channel(1);
    let task = SealTask::new(
        Arc::new(test_engine_client_builder().build()),
        Arc::new(RollupConfig::default()),
        payload_id(),
        attributes,
        None,
        false,
        coupling,
        with_channel.then_some(tx),
        Arc::new(crate::NoopBlockSink),
    );

    let result = task.execute(&mut state).await;

    if with_channel {
        // With a result channel, the task itself succeeds and the error is relayed to the caller.
        result.expect("task with a result channel should succeed");
        let relayed = rx.recv().await.expect("channel should receive the seal result");
        assert_eq!(classify(&relayed.expect_err("seal should fail against the mock")), expected);
    } else {
        assert_eq!(classify(&result.expect_err("seal should fail against the mock")), expected);
    }
}

#[rstest]
#[case::ready(PayloadReadiness::Ready, SealOutcome::ProceededToSeal)]
#[case::pending(PayloadReadiness::Pending, SealOutcome::ProceededToSeal)]
#[case::unknown(PayloadReadiness::Unknown, SealOutcome::PayloadJobGone)]
#[tokio::test]
async fn readiness_decides_whether_the_payload_is_fetched(
    #[case] readiness: PayloadReadiness,
    #[case] expected: SealOutcome,
) {
    let parent_block = test_block_info(10);
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent_block).build();
    let client = Arc::new(test_engine_client_builder().with_payload_readiness(readiness).build());

    let deadline = Instant::now() + Duration::from_millis(50);
    let result = seal_task(Arc::clone(&client), Some(deadline), Atomic).execute(&mut state).await;

    assert_eq!(classify(&result.expect_err("seal should fail against the mock")), expected);
}

#[tokio::test]
async fn readiness_is_awaited_only_against_a_deadline() {
    let parent_block = test_block_info(10);
    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent_block).build();
    let client = Arc::new(test_engine_client_builder().build());

    let _ = seal_task(Arc::clone(&client), None, Atomic).execute(&mut state).await;
    assert!(
        client.await_payload_ready_calls().await.is_empty(),
        "a seal the caller already decided is due does not wait for readiness"
    );

    let max_wait = Duration::from_millis(50);
    let _ = seal_task(Arc::clone(&client), Some(Instant::now() + max_wait), Atomic)
        .execute(&mut state)
        .await;

    let calls = client.await_payload_ready_calls().await;
    assert_eq!(calls.len(), 1);
    assert_eq!(calls[0].0, payload_id());
    assert!(calls[0].1 <= max_wait, "the wait is bounded by the deadline, {:?}", calls[0].1);

    let _ = seal_task(Arc::clone(&client), Some(Instant::now()), Atomic).execute(&mut state).await;
    assert_eq!(
        client.await_payload_ready_calls().await.len(),
        1,
        "a deadline that has already passed leaves nothing to wait for"
    );
}
