//! Tests for `SealTask::execute`.

use crate::{
    EngineTaskExt, SealTask, SealTaskError,
    test_utils::{
        TestAttributesBuilder, TestEngineStateBuilder, test_block_info, test_engine_client_builder,
    },
};
use alloy_rpc_types_engine::PayloadId;
use kona_genesis::RollupConfig;
use rstest::rstest;
use std::sync::Arc;

#[rstest]
#[case::sequencer(false, true)]
#[case::derived(true, false)]
#[tokio::test]
async fn rejects_stale_sequencer_build_but_allows_derived_reorg(
    #[case] is_attributes_derived: bool,
    #[case] expect_stale: bool,
) {
    let parent = test_block_info(1);
    let unsafe_head = test_block_info(2);
    let attributes = TestAttributesBuilder::new()
        .with_parent(parent)
        .with_timestamp(parent.block_info.timestamp + 2)
        .build();
    let client = Arc::new(test_engine_client_builder().build());
    let task = SealTask::new(
        client,
        Arc::new(RollupConfig::default()),
        PayloadId::new([1; 8]),
        attributes,
        is_attributes_derived,
        None,
    );
    let mut state = TestEngineStateBuilder::new()
        .with_unsafe_head(unsafe_head)
        .with_safe_head(parent)
        .with_finalized_head(parent)
        .build();

    let err = task.execute(&mut state).await.unwrap_err();
    assert_eq!(matches!(err, SealTaskError::UnsafeHeadChangedSinceBuild), expect_stale);
    if is_attributes_derived {
        // The mock has no payload configured, so reaching the get-payload call proves the derived
        // replacement was allowed past the stale-sequencer-build guard.
        assert!(matches!(err, SealTaskError::GetPayloadFailed(_)));
    }
}
