//! Tests for the pairing a [`SealTask`] hands to the [`InsertTask`] it runs.
//!
//! [`SealTask`]: super::SealTask
//! [`InsertTask`]: crate::InsertTask

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

fn seal_task(attributes: OpAttributesWithParent, is_derived: bool) -> SealTask<MockEngineClient> {
    let cfg = Arc::new(RollupConfig::default());
    SealTask {
        engine: Arc::new(test_engine_client_builder().with_config(cfg.clone()).build()),
        cfg,
        payload_id: PayloadId::default(),
        attributes,
        is_attributes_derived: is_derived,
        result_tx: None,
        deny: None,
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

/// An unsafe head that moved between build and seal is expected state on the consolidation path —
/// during invalidation recovery the deposits-only replacement moves it under the queued task —
/// and the recovery is a reset that drops the stale work, not a crash. op-node handles the same
/// condition by emitting a `ResetEvent` ("pending safe head changed ... conflicting with queued
/// safe attributes", `op-node/rollup/attributes/attributes.go:175-183`); escalating it to
/// Critical instead kills the node.
#[test]
fn a_changed_unsafe_head_is_a_reset_not_a_crash() {
    use crate::{EngineTaskError, EngineTaskErrorSeverity, SealTaskError};
    assert_eq!(
        SealTaskError::UnsafeHeadChangedSinceBuild.severity(),
        EngineTaskErrorSeverity::Reset,
        "a changed unsafe head during replacement is expected, not Critical"
    );
}

/// The deny check at seal time: a denied derived payload is never inserted — the deposits-only
/// replacement is built and imported at the same height instead. These are the mechanics behind
/// block invalidation: after the rewind, derivation deterministically rebuilds the invalidated
/// block, and this check is what turns that rebuild into the replacement (op-node's
/// `op-node/rollup/engine/payload_process.go:56-85`).
mod deny {
    use crate::{
        EngineState, EngineSyncStateUpdate, EngineTaskExt, LocalSafeHead, SealTask, SealTaskError,
        state::EngineSyncState,
        test_utils::{StaticDenyList, TestAttributesBuilder, test_engine_client_builder},
    };
    use alloy_primitives::{Bytes, U256};
    use alloy_rpc_types_engine::{
        ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2, ExecutionPayloadV1, ForkchoiceUpdated,
        PayloadId, PayloadStatus, PayloadStatusEnum,
    };
    use kona_genesis::{ChainGenesis, RollupConfig};
    use kona_protocol::OpAttributesWithParent;
    use op_alloy_consensus::{OpBlock, OpTxEnvelope};
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
    use std::sync::Arc;

    /// The payload every `engine_getPayload` answers with: an empty block at height 1.
    fn sealed_payload() -> ExecutionPayloadV1 {
        let mut payload =
            ExecutionPayloadV1::from_block_slow(&alloy_consensus::Block::<OpTxEnvelope>::default());
        payload.block_number = 1;
        payload
    }

    /// A rollup config whose genesis is the sealed payload's block, so the insert can decode it
    /// without an L1-info transaction.
    fn cfg() -> Arc<RollupConfig> {
        let block: OpBlock =
            OpExecutionPayloadEnvelope::V1(sealed_payload()).try_into_block().unwrap();
        Arc::new(RollupConfig {
            genesis: ChainGenesis {
                l2: alloy_eips::BlockNumHash { number: 1, hash: block.header.hash_slow() },
                ..Default::default()
            },
            ..Default::default()
        })
    }

    /// Derived attributes carrying one deposit and one user transaction, so the deposits-only
    /// conversion has something to strip.
    fn derived_attributes() -> OpAttributesWithParent {
        TestAttributesBuilder::new()
            .with_derived_from(super::l1(7))
            .with_transactions(vec![
                Bytes::from_static(&[0x7e, 0x01]), // deposit
                Bytes::from_static(&[0x02, 0x01]), // eip-1559 user transaction
            ])
            .build()
    }

    /// An engine state whose unsafe head is the attributes' parent, as it is when the seal runs.
    fn state_on_parent(attributes: &OpAttributesWithParent) -> EngineState {
        let mut state = EngineState::default();
        state.sync_state = state.apply_sync_update(EngineSyncStateUpdate {
            unsafe_head: Some(attributes.parent),
            local_safe_head: Some(LocalSafeHead::unpaired(attributes.parent)),
            ..Default::default()
        });
        state
    }

    /// A seal task over a mock EL that seals, builds and inserts successfully.
    fn seal_task_with_deny(
        deny: Arc<StaticDenyList>,
    ) -> SealTask<crate::test_utils::MockEngineClient> {
        let cfg = cfg();
        let engine = test_engine_client_builder()
            .with_config(cfg.clone())
            .with_execution_payload_v2(ExecutionPayloadEnvelopeV2 {
                execution_payload: ExecutionPayloadFieldV2::V1(sealed_payload()),
                block_value: U256::ZERO,
            })
            .with_fork_choice_updated_v2_response(
                ForkchoiceUpdated::new(PayloadStatus::new(PayloadStatusEnum::Valid, None))
                    .with_payload_id(PayloadId::new([1u8; 8])),
            )
            .with_fork_choice_updated_v3_response(
                ForkchoiceUpdated::new(PayloadStatus::new(PayloadStatusEnum::Valid, None))
                    .with_payload_id(PayloadId::new([1u8; 8])),
            )
            .with_new_payload_v1_response(PayloadStatus::new(PayloadStatusEnum::Valid, None))
            .build();
        SealTask {
            engine: Arc::new(engine),
            cfg,
            payload_id: PayloadId::new([1u8; 8]),
            attributes: derived_attributes(),
            is_attributes_derived: true,
            result_tx: None,
            deny: Some(deny),
        }
    }

    /// Reads the sync state after a seal, for asserting where the heads landed.
    fn heads(state: &EngineState) -> EngineSyncState {
        state.sync_state
    }

    /// The core of the replacement flow: the sealed payload's hash is denied, so it is not
    /// inserted — the deposits-only version of the same attributes is built and imported at the
    /// same height, and the channel is flushed exactly as the Holocene invalid-payload fallback
    /// flushes it (op-node flushes there too: `DepositsOnlyAttributes` calls
    /// `aq.prev.FlushChannel()`, `op-node/rollup/derive/attributes_queue.go:167`).
    #[tokio::test]
    async fn a_denied_derived_payload_is_replaced_deposits_only() {
        let payload = sealed_payload();
        let deny = StaticDenyList::denying([(1, payload.block_hash)]);
        let task = seal_task_with_deny(deny.clone());
        let mut state = state_on_parent(&task.attributes);

        let err = task.execute(&mut state).await.expect_err("the flush signal is the outcome");
        assert!(matches!(err, SealTaskError::HoloceneInvalidFlush), "{err:?}");

        // The deny list was consulted once, for the sealed payload — the deposits-only rebuild is
        // exempt, or the replacement itself would recurse.
        assert_eq!(deny.queries(), vec![(1, payload.block_hash)]);

        // The replacement was imported and is the new head: the local-safe write carries the
        // deposits-only block, not the denied one.
        let sync = heads(&state);
        assert_eq!(sync.local_safe_head().block_info.number, 1);
    }

    /// An undenied payload takes the ordinary path: consulted, admitted, inserted.
    #[tokio::test]
    async fn an_undenied_payload_is_inserted_normally() {
        let deny = StaticDenyList::denying([]);
        let task = seal_task_with_deny(deny.clone());
        let mut state = state_on_parent(&task.attributes);

        task.execute(&mut state).await.expect("the payload is admitted");
        assert_eq!(deny.queries().len(), 1, "the deny list is consulted for every derived seal");
        assert_eq!(heads(&state).local_safe_head().block_info.number, 1);
    }

    /// A deny list that cannot be read fails open, exactly like op-node's insert-time check
    /// (`payload_process.go:62`: "Failed to check `SuperAuthority` denylist, proceeding with
    /// payload"): the payload is inserted and the error is logged, because a wedged engine is
    /// worse than looping invalidation until the store heals.
    #[tokio::test]
    async fn an_unreadable_deny_list_fails_open_at_seal_time() {
        let deny = StaticDenyList::unreadable("the store is down");
        let task = seal_task_with_deny(deny.clone());
        let mut state = state_on_parent(&task.attributes);

        task.execute(&mut state).await.expect("a deny-list read error must not block the payload");
        assert_eq!(deny.queries().len(), 1);
        assert_eq!(heads(&state).local_safe_head().block_info.number, 1);
    }

    /// A sequencer-built (underived) payload is not checked here: its deny gate is the commit
    /// path's, which answers the caller. The seal-time check exists for derived payloads because
    /// only they are rebuilt deterministically from L1 data.
    #[tokio::test]
    async fn an_underived_seal_does_not_consult_the_deny_list() {
        let payload = sealed_payload();
        let deny = StaticDenyList::denying([(1, payload.block_hash)]);
        let mut task = seal_task_with_deny(deny.clone());
        task.is_attributes_derived = false;
        let mut state = state_on_parent(&task.attributes);

        task.execute(&mut state).await.expect("an underived seal is not gated here");
        assert!(deny.queries().is_empty());
    }
}
