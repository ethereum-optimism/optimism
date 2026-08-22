//! Tests for the L1 origin a [`ConsolidateTask`] pairs with the local-safe head it writes.
//!
//! [`ConsolidateTask`]: super::ConsolidateTask

use crate::{
    LocalSafeOrigin, task_queue::tasks::consolidate::ConsolidateInput,
    test_utils::TestAttributesBuilder,
};
use kona_protocol::{BlockInfo, L2BlockInfo};

fn l1(number: u64) -> BlockInfo {
    BlockInfo { number, ..Default::default() }
}

/// The ordinary derivation path: the attributes name their L1 origin and it travels with them, so
/// the head this input writes is paired with the L1 block it was actually derived from.
#[test]
fn derived_attributes_pair_with_their_own_l1_origin() {
    let input =
        ConsolidateInput::from(TestAttributesBuilder::new().with_derived_from(l1(7)).build());

    assert_eq!(input.local_safe_origin(), LocalSafeOrigin::DerivedFrom(l1(7)));
}

/// Derivation delegation injects a bare block with no attributes, so there is no L1 origin to pair
/// with it. It has to be recorded as unpaired rather than inheriting whatever origin the previous
/// local-safe head carried.
#[test]
fn a_delegated_block_is_unpaired() {
    let input = ConsolidateInput::from(L2BlockInfo::default());

    assert_eq!(input.local_safe_origin(), LocalSafeOrigin::Unpaired);
}

/// `derived_from` is itself optional on the attributes, so even the derivation path can arrive
/// without an origin. That is the absent case too, not a defaulted L1 block.
#[test]
fn attributes_without_an_origin_are_unpaired() {
    let input = ConsolidateInput::from(TestAttributesBuilder::new().build());

    assert_eq!(input.local_safe_origin(), LocalSafeOrigin::Unpaired);
    assert_ne!(input.local_safe_origin(), LocalSafeOrigin::DerivedFrom(BlockInfo::default()));
}

/// The consolidation deny gate: a denied block that matches the derived attributes must be
/// reorged out and rebuilt, never adopted as local-safe — the mirror of op-node's consolidation
/// deny check (`op-node/rollup/attributes/attributes.go:241-256`).
mod deny {
    use crate::{
        ConsolidateTask, ConsolidateTaskError, EngineState, EngineSyncStateUpdate, EngineTaskExt,
        task_queue::tasks::consolidate::ConsolidateInput,
        test_utils::{StaticDenyList, TestAttributesBuilder, test_engine_client_builder},
    };
    use alloy_eips::eip1898::BlockNumberOrTag;
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header};
    use kona_genesis::{ChainGenesis, RollupConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
    use op_alloy_rpc_types::Transaction as OpTransaction;
    use std::sync::Arc;

    /// Derived attributes for block 1 on the default test parent.
    fn attributes() -> OpAttributesWithParent {
        TestAttributesBuilder::new().with_derived_from(super::l1(7)).build()
    }

    /// The canonical block at the attributes' height, matching them field for field.
    fn matching_block(attributes: &OpAttributesWithParent) -> Block<OpTransaction> {
        let payload = &attributes.attributes().payload_attributes;
        let inner = alloy_consensus::Header {
            parent_hash: attributes.parent.block_info.hash,
            number: attributes.block_number(),
            timestamp: payload.timestamp,
            mix_hash: payload.prev_randao,
            beneficiary: payload.suggested_fee_recipient,
            gas_limit: attributes.attributes().gas_limit.expect("test attributes carry one"),
            parent_beacon_block_root: payload.parent_beacon_block_root,
            ..Default::default()
        };
        let hash = inner.hash_slow();
        Block {
            header: Header { hash, inner, total_difficulty: None, size: None },
            uncles: Vec::new(),
            transactions: BlockTransactions::Full(Vec::new()),
            withdrawals: None,
        }
    }

    /// A config whose genesis is the matching block, so adopting it needs no L1-info transaction.
    fn cfg(block: &Block<OpTransaction>) -> Arc<RollupConfig> {
        Arc::new(RollupConfig {
            genesis: ChainGenesis {
                l2: alloy_eips::BlockNumHash {
                    number: block.header.number,
                    hash: block.header.hash,
                },
                ..Default::default()
            },
            ..Default::default()
        })
    }

    /// A state whose unsafe head is ahead of local-safe, so the task consolidates rather than
    /// building fresh.
    fn state_with_unsafe_ahead() -> EngineState {
        let mut state = EngineState::default();
        state.sync_state = state.apply_sync_update(EngineSyncStateUpdate {
            unsafe_head: Some(L2BlockInfo {
                block_info: BlockInfo { number: 2, ..Default::default() },
                ..Default::default()
            }),
            ..Default::default()
        });
        state
    }

    /// A consolidate task over a mock EL that serves the given canonical block and nothing else:
    /// the build path is left unmocked on purpose, so an attempted reorg is observable as a
    /// build failure rather than an adoption.
    fn task(
        block: Block<OpTransaction>,
        deny: Arc<StaticDenyList>,
    ) -> ConsolidateTask<crate::test_utils::MockEngineClient> {
        let attributes = attributes();
        let cfg = cfg(&block);
        let client = test_engine_client_builder()
            .with_config(cfg.clone())
            .with_l2_block_by_label(BlockNumberOrTag::Number(attributes.block_number()), block)
            .build();
        ConsolidateTask::new(Arc::new(client), cfg, ConsolidateInput::from(attributes), Some(deny))
    }

    /// The control: an undenied matching block is adopted as local-safe without any rebuild.
    #[tokio::test]
    async fn an_undenied_matching_block_is_adopted() {
        let block = matching_block(&attributes());
        let block_hash = block.header.hash;
        let deny = StaticDenyList::denying([]);
        let mut state = state_with_unsafe_ahead();

        task(block, deny.clone()).execute(&mut state).await.expect("consolidation adopts it");

        assert_eq!(deny.queries(), vec![(1, block_hash)]);
        assert_eq!(state.sync_state.local_safe_head().block_info.hash, block_hash);
    }

    /// A denied matching block is not adopted: the task takes the build path to reorg it out —
    /// observable here as the unmocked build failing — and the local-safe head stays put.
    #[tokio::test]
    async fn a_denied_matching_block_is_reorged_not_adopted() {
        let block = matching_block(&attributes());
        let block_hash = block.header.hash;
        let deny = StaticDenyList::denying([(1, block_hash)]);
        let mut state = state_with_unsafe_ahead();

        let err = task(block, deny.clone())
            .execute(&mut state)
            .await
            .expect_err("the denied block must not be adopted");
        assert!(matches!(err, ConsolidateTaskError::BuildTaskFailed(_)), "{err:?}");
        assert_ne!(state.sync_state.local_safe_head().block_info.hash, block_hash);
    }

    /// A deny list that cannot be read fails CLOSED here, unlike the seal-time check: without an
    /// answer the block can be neither promoted nor reorged, so the task stalls with a temporary
    /// error and is retried — op-node's posture at `attributes.go:241-247` ("Fail closed: without
    /// a deny-list result we cannot promote the block, and we must not reorg either").
    #[tokio::test]
    async fn an_unreadable_deny_list_stalls_consolidation() {
        let block = matching_block(&attributes());
        let block_hash = block.header.hash;
        let deny = StaticDenyList::unreadable("the store is down");
        let mut state = state_with_unsafe_ahead();

        let err =
            task(block, deny).execute(&mut state).await.expect_err("no deny answer, no adoption");
        assert!(matches!(err, ConsolidateTaskError::DenyListUnavailable), "{err:?}");
        assert_ne!(state.sync_state.local_safe_head().block_info.hash, block_hash);
    }
}
