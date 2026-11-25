//! Tests for the [`PromoteTask`].

use crate::{
    EngineState, EngineSyncStateUpdate, ExecutableEngineTask, PromoteTask, SafeLabel, VerifierHead,
    test_utils::{TestEngineStateBuilder, test_engine_client, test_l2_block, test_rpc_block},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;

const FINALIZED: u64 = 5;
const SAFE: u64 = 10;
const LOCAL_SAFE: u64 = 15;
const UNSAFE: u64 = 20;

/// The chain the engine has, as `L2BlockInfo`s and the blocks the execution layer serves.
struct Chain {
    blocks: Vec<op_alloy_consensus::OpBlock>,
    genesis: kona_genesis::ChainGenesis,
}

impl Chain {
    fn new() -> Self {
        let mut blocks: Vec<op_alloy_consensus::OpBlock> = Vec::new();
        for number in 0..=UNSAFE {
            let parent = blocks.last().map_or(B256::ZERO, |block| block.header.hash_slow());
            blocks.push(test_l2_block(number, parent, 0));
        }
        let genesis = kona_genesis::ChainGenesis {
            l2: BlockNumHash { number: 0, hash: blocks[0].header.hash_slow() },
            ..Default::default()
        };
        Self { blocks, genesis }
    }

    fn info(&self, number: u64) -> L2BlockInfo {
        L2BlockInfo::from_block_and_genesis(&self.blocks[number as usize], &self.genesis).unwrap()
    }

    fn rpc_block(&self, number: u64) -> crate::test_utils::TestRpcBlock {
        test_rpc_block(&self.blocks[number as usize])
    }

    /// An engine whose safe label a verifier moves, with the labels at the constants above.
    fn verified_engine_state(&self) -> EngineState {
        TestEngineStateBuilder::new()
            .with_safe_label(SafeLabel::Verified { activation_timestamp: 0 })
            .with_unsafe_head(self.info(UNSAFE))
            .with_local_safe_head(self.info(LOCAL_SAFE))
            .with_safe_head(self.info(SAFE))
            .with_finalized_head(self.info(FINALIZED))
            .build()
    }
}

/// A rollup config with genesis at L2 time zero and two-second blocks, as [`test_l2_block`] dates
/// them.
fn cfg() -> Arc<RollupConfig> {
    Arc::new(RollupConfig { block_time: 2, ..Default::default() })
}

fn verified(block: L2BlockInfo) -> VerifierHead {
    VerifierHead::Verified { block: block.block_info.id(), timestamp: block.block_info.timestamp }
}

/// A verified head the engine has not derived to yet holds the safe label where it is, without a
/// lookup or forkchoice update: it never falls back to local-safe.
#[tokio::test]
async fn verified_head_ahead_of_local_safe_holds_the_safe_label() {
    let chain = Chain::new();
    let mut state = chain.verified_engine_state();
    let state_before = state;
    let cfg = cfg();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    let task = PromoteTask::new(Arc::new(client), cfg, verified(chain.info(LOCAL_SAFE + 1)));

    task.execute(&mut state).await.unwrap();

    assert_eq!(state, state_before);
    l1.assert_finished();
    l2.assert_finished();
}

/// A verified head that is the engine's canonical block, at or below local-safe, becomes the
/// safe label, whether it raises or lowers it.
#[rstest::rstest]
#[case::raises(12)]
#[case::lowers(8)]
#[case::at_local_safe_without_lookup(LOCAL_SAFE)]
#[tokio::test]
async fn canonical_verified_head_moves_the_safe_label(#[case] number: u64) {
    let chain = Chain::new();
    let mut state = chain.verified_engine_state();
    let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
        safe_head: Some(chain.info(number)),
        ..Default::default()
    });
    let cfg = cfg();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    if number != LOCAL_SAFE {
        l2.expect_block_by_number(number, Some(chain.rpc_block(number)));
    }
    l2.expect_forkchoice_updated(expected.create_forkchoice_state(), PayloadStatusEnum::Valid);
    let task = PromoteTask::new(Arc::new(client), cfg, verified(chain.info(number)));

    task.execute(&mut state).await.unwrap();

    assert_eq!(state.sync_state.safe_head(), chain.info(number));
    assert_eq!(state.sync_state.local_safe_head(), chain.info(LOCAL_SAFE));
    l1.assert_finished();
    l2.assert_finished();
}

/// A verified head that is not the engine's canonical block at its height holds the safe label.
#[tokio::test]
async fn non_canonical_verified_head_holds_the_safe_label() {
    let chain = Chain::new();
    let mut state = chain.verified_engine_state();
    let state_before = state;
    let cfg = cfg();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_block_by_number(12, Some(chain.rpc_block(12)));
    let other_block = BlockNumHash { number: 12, hash: B256::repeat_byte(0xee) };
    let task = PromoteTask::new(
        Arc::new(client),
        cfg,
        VerifierHead::Verified { block: other_block, timestamp: 24 },
    );

    task.execute(&mut state).await.unwrap();

    assert_eq!(state, state_before);
    l1.assert_finished();
    l2.assert_finished();
}

/// Before the verifier is active the safe label follows local-safe, and an anchor the engine has
/// not derived to yet is bounded by local-safe.
#[rstest::rstest]
#[case::pre_activation(VerifierHead::PreActivation)]
#[case::anchor_ahead_of_local_safe(VerifierHead::Anchor { timestamp: (LOCAL_SAFE + 3) * 2 })]
#[tokio::test]
async fn unverified_head_moves_the_safe_label_to_local_safe(#[case] head: VerifierHead) {
    let chain = Chain::new();
    let mut state = chain.verified_engine_state();
    let expected = state.sync_state.apply_update(EngineSyncStateUpdate {
        safe_head: Some(chain.info(LOCAL_SAFE)),
        ..Default::default()
    });
    let cfg = cfg();
    let (client, l1, l2) = test_engine_client(cfg.clone());
    l2.expect_forkchoice_updated(expected.create_forkchoice_state(), PayloadStatusEnum::Valid);
    let task = PromoteTask::new(Arc::new(client), cfg, head);

    task.execute(&mut state).await.unwrap();

    assert_eq!(state.sync_state.safe_head(), chain.info(LOCAL_SAFE));
    l1.assert_finished();
    l2.assert_finished();
}

/// With a verifier moving the safe label, derivation moving local-safe ahead leaves the safe label
/// behind; without one, the safe label moves with local-safe.
#[rstest::rstest]
#[case::verified(SafeLabel::Verified { activation_timestamp: 0 }, SAFE)]
#[case::local_safe(SafeLabel::LocalSafe, LOCAL_SAFE + 1)]
fn local_safe_progress_moves_the_safe_label_only_without_a_verifier(
    #[case] safe_label: SafeLabel,
    #[case] expected_safe: u64,
) {
    let chain = Chain::new();
    let state = TestEngineStateBuilder::new()
        .with_safe_label(safe_label)
        .with_unsafe_head(chain.info(UNSAFE))
        .with_local_safe_head(chain.info(LOCAL_SAFE))
        .with_safe_head(chain.info(if matches!(safe_label, SafeLabel::Verified { .. }) {
            SAFE
        } else {
            LOCAL_SAFE
        }))
        .with_finalized_head(chain.info(FINALIZED))
        .build();

    let derived = state.sync_state.apply_update(EngineSyncStateUpdate {
        local_safe_head: Some(chain.info(LOCAL_SAFE + 1)),
        ..Default::default()
    });

    assert_eq!(derived.local_safe_head(), chain.info(LOCAL_SAFE + 1));
    assert_eq!(derived.safe_head(), chain.info(expected_safe));
}

/// Before the first reset a verified safe label is empty. The reset raises it to the recovered
/// finalized head, not to the recovered local-safe head.
#[test]
fn first_reset_starts_a_verified_safe_label_at_finality() {
    let chain = Chain::new();
    let empty = crate::EngineSyncState::new(SafeLabel::Verified { activation_timestamp: 0 });

    let reset = empty.apply_update(EngineSyncStateUpdate {
        unsafe_head: Some(chain.info(UNSAFE)),
        local_safe_head: Some(chain.info(LOCAL_SAFE)),
        safe_head: None,
        finalized_head: Some(chain.info(0)),
    });

    assert_eq!(reset.safe_head(), chain.info(0));
}

/// While local-safe is before the verifier's activation, the safe label is the local-safe head,
/// also across a reset that moves local-safe back. From activation on, local-safe progress leaves
/// it at the last block before activation, where the verifier's anchor is.
#[test]
fn verified_safe_label_follows_local_safe_before_activation() {
    let chain = Chain::new();
    let first_active = LOCAL_SAFE + 1;
    let activation_timestamp = chain.info(first_active).block_info.timestamp;
    let mut state = crate::EngineSyncState::new(SafeLabel::Verified { activation_timestamp });
    let local_safe = |number| EngineSyncStateUpdate {
        local_safe_head: Some(chain.info(number)),
        ..Default::default()
    };

    for number in 1..first_active {
        state = state.apply_update(local_safe(number));
        assert_eq!(state.safe_head(), chain.info(number));
    }
    state = state.apply_update(local_safe(FINALIZED));
    assert_eq!(state.safe_head(), chain.info(FINALIZED), "a reset before activation");
    for number in FINALIZED + 1..=UNSAFE {
        state = state.apply_update(local_safe(number));
    }

    assert_eq!(state.local_safe_head(), chain.info(UNSAFE));
    assert_eq!(state.safe_head(), chain.info(LOCAL_SAFE));
}
