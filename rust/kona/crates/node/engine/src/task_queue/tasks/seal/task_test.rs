use crate::{
    BuildSealCoupling::{self, Atomic, Detached},
    EngineTaskExt, SealTask, SealTaskError,
    test_utils::{
        TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client_builder,
    },
};
use alloy_rpc_types_engine::PayloadId;
use kona_genesis::RollupConfig;
use rstest::rstest;
use std::sync::Arc;
use tokio::sync::mpsc;

/// The two paths the unsafe-head check can steer `execute` into: aborting the seal as stale, or
/// proceeding to the payload fetch — which the unconfigured mock client fails, surfacing as
/// [`SealTaskError::GetPayloadFailed`].
#[derive(Debug, PartialEq, Eq)]
enum SealOutcome {
    AbortedAsStale,
    ProceededToSeal,
}

fn classify(err: &SealTaskError) -> SealOutcome {
    match err {
        SealTaskError::UnsafeHeadChangedSinceBuild => SealOutcome::AbortedAsStale,
        SealTaskError::GetPayloadFailed(_) => SealOutcome::ProceededToSeal,
        other => panic!("unexpected seal error: {other:?}"),
    }
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
        PayloadId::new([1u8; 8]),
        attributes,
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
