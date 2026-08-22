//! Tests for the [`SealTask`]: the unsafe-head staleness check per [`BuildSealCoupling`],
//! and the pairing a seal hands to the [`InsertTask`] it runs.
//!
//! [`SealTask`]: super::SealTask
//! [`InsertTask`]: crate::InsertTask
//! [`BuildSealCoupling`]: crate::BuildSealCoupling

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

/// The local-safe pairing a seal hands to the insert it runs.
mod pairing {
    use crate::{
        LocalSafeOrigin, SealTask,
        test_utils::{MockEngineClient, TestAttributesBuilder, test_engine_client_builder},
    };
    use alloy_primitives::Bytes;
    use alloy_rpc_types_engine::PayloadId;
    use kona_genesis::RollupConfig;
    use kona_protocol::{BlockInfo, OpAttributesWithParent};
    use std::sync::Arc;

    fn l1(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    fn seal_task(
        attributes: OpAttributesWithParent,
        is_derived: bool,
    ) -> SealTask<MockEngineClient> {
        let cfg = Arc::new(RollupConfig::default());
        SealTask {
            engine: Arc::new(test_engine_client_builder().with_config(cfg.clone()).build()),
            cfg,
            payload_id: PayloadId::default(),
            attributes,
            is_attributes_derived: is_derived,
            coupling: crate::BuildSealCoupling::Detached,
            result_tx: None,
            block_sink: Arc::new(crate::NoopBlockSink),
        }
    }

    /// A derived block is a local-safe write, and the L1 key it is paired with comes from the
    /// attributes the task already holds — not from a lookup after the fact.
    #[test]
    fn a_derived_block_carries_the_attributes_origin() {
        let attributes = TestAttributesBuilder::new().with_derived_from(l1(7)).build();

        assert_eq!(
            seal_task(attributes, true).local_safe_origin(),
            Some(LocalSafeOrigin::DerivedFrom(l1(7)))
        );
    }

    /// A sequencer-built block moves no local-safe head at all, which is the outer [`None`] rather
    /// than an unpaired origin.
    #[test]
    fn a_sequenced_block_is_not_a_local_safe_write() {
        let attributes = TestAttributesBuilder::new().with_derived_from(l1(7)).build();

        assert_eq!(seal_task(attributes, false).local_safe_origin(), None);
    }

    /// The Holocene deposits-only retry re-seals a genuinely different block, but not one from a
    /// different L1 origin: `as_deposits_only` copies `derived_from`, so the pairing survives the
    /// fallback. This is the one path where a wrong L1 key would otherwise be possible.
    #[test]
    fn the_deposits_only_fallback_keeps_the_origin() {
        let attributes = TestAttributesBuilder::new()
            .with_derived_from(l1(7))
            .with_transactions(vec![
                Bytes::from_static(&[0x7e, 0x01]), // deposit
                Bytes::from_static(&[0x02, 0x01]), // eip-1559, dropped by the fallback
            ])
            .build();
        let deposits_only = attributes.as_deposits_only();

        assert!(deposits_only.is_deposits_only(), "test setup: the fallback dropped the user tx");
        assert_eq!(
            seal_task(deposits_only, true).local_safe_origin(),
            seal_task(attributes, true).local_safe_origin(),
            "the deposits-only retry must report the same L1 origin as the block it replaces"
        );
    }
}
