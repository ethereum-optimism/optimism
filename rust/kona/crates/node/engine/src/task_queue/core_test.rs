//! Tests for [`Engine::reset`] and [`Engine::reset_to`].
//!
//! The two differ in exactly one respect — where the target heads come from — so most of these
//! tests are about proving that difference is real: that `reset_to` never reaches for the RPC
//! walkback, and that `reset` still does.

use super::core::RESET_TO_MAX_ATTEMPTS;
use crate::{
    Engine, EngineResetError, EngineState, L2ForkchoiceState, LocalSafeOrigin,
    test_utils::{MockEngineClient, TestEngineStateBuilder, test_engine_client_builder},
};
use alloy_eips::{BlockId, BlockNumHash, BlockNumberOrTag};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ForkchoiceState, ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum,
};
use kona_genesis::{ChainGenesis, RollupConfig};
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;
use tokio::sync::watch;

/// Block `number`, hashed as a repeated byte so `block(0)` is the all-zero genesis of
/// [`RollupConfig::default`].
fn block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            number,
            hash: B256::repeat_byte(number as u8),
            parent_hash: B256::repeat_byte(number.saturating_sub(1) as u8),
            timestamp: number * 2,
        },
        ..Default::default()
    }
}

/// A client that answers forkchoice updates and nothing else. Every walkback lookup misses, so any
/// call to [`crate::find_starting_forkchoice`] fails — which is what lets these tests tell the two
/// reset flavours apart by outcome alone.
fn walkback_blind_client(cfg: Arc<RollupConfig>) -> Arc<MockEngineClient> {
    Arc::new(
        test_engine_client_builder()
            .with_config(cfg)
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(PayloadStatus::new(
                PayloadStatusEnum::Valid,
                None,
            )))
            .build(),
    )
}

fn engine(state: EngineState) -> Engine<MockEngineClient> {
    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let (len_tx, _len_rx) = watch::channel(0usize);
    Engine::new(state, state_tx, len_tx)
}

fn last_fcu(states: &[ForkchoiceState]) -> ForkchoiceState {
    *states.last().expect("no forkchoice update was issued")
}

/// The headline behaviour: `reset_to` applies the heads it is handed without consulting the chain.
///
/// The client cannot serve a walkback at all, so `reset` — which must discover its start point —
/// fails on it, while `reset_to` succeeds. That difference in outcome is the assertion that no
/// discovery RPC was issued; a `reset_to` that quietly fell back to discovery would fail here too.
#[tokio::test]
async fn reset_to_skips_the_walkback_and_applies_the_given_heads() {
    let cfg = Arc::new(RollupConfig::default());
    let client = walkback_blind_client(cfg.clone());
    let (genesis, b1, b2) = (block(0), block(1), block(2));

    let mut walking = engine(TestEngineStateBuilder::new().build());
    assert!(
        matches!(
            walking.reset(client.clone(), cfg.clone()).await,
            Err(EngineResetError::SyncStart(_))
        ),
        "the client cannot serve a walkback, so a discovering reset must fail on it"
    );
    assert!(
        client.fork_choice_states().await.is_empty(),
        "a reset that never found a start point cannot have issued a forkchoice update"
    );

    let target = L2ForkchoiceState { un_safe: b2, local_safe: b1, finalized: genesis };
    let mut targeted = engine(TestEngineStateBuilder::new().build());
    let landed = targeted
        .reset_to(client.clone(), cfg, target)
        .await
        .expect("a targeted reset needs no walkback");

    assert_eq!(landed, b1, "reset_to returns the local-safe head it was given");
    assert_eq!(targeted.state().sync_state.unsafe_head(), b2);
    assert_eq!(targeted.state().sync_state.local_safe_head(), b1);
    assert_eq!(targeted.state().sync_state.finalized_head(), genesis);

    let fcu = last_fcu(&client.fork_choice_states().await);
    assert_eq!(fcu.head_block_hash, b2.block_info.hash);
    assert_eq!(fcu.finalized_block_hash, genesis.block_info.hash);
}

/// A reset mints no cross-safe promotion, so on an interop engine — where cross-safe is a head in
/// its own right — it cannot move that head forward, even when the heads it installs run well
/// ahead of where cross-safe sits. (Standalone, cross-safe *is* local-safe and follows it.)
#[tokio::test]
async fn reset_to_does_not_move_the_cross_safe_head_forward() {
    let cfg = Arc::new(RollupConfig::default());
    let client = walkback_blind_client(cfg.clone());
    let (genesis, b3) = (block(0), block(3));

    // An interop engine: cross-safe moves only on externally minted promotions, so absence of
    // promotion is what holds the value here.
    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let (len_tx, _len_rx) = watch::channel(0usize);
    let (mut targeted, _promoter) = Engine::<MockEngineClient>::with_external_cross_safe(
        TestEngineStateBuilder::new().build(),
        state_tx,
        len_tx,
    );

    assert_eq!(targeted.state().sync_state.cross_safe_head(), genesis, "test setup");

    let target = L2ForkchoiceState { un_safe: b3, local_safe: b3, finalized: genesis };
    targeted.reset_to(client.clone(), cfg, target).await.expect("targeted reset");

    assert_eq!(targeted.state().sync_state.local_safe_head(), b3);
    assert_eq!(
        targeted.state().sync_state.cross_safe_head(),
        genesis,
        "the reset advanced local-safe but must leave cross-safe where the last promotion put it"
    );
    assert_eq!(
        last_fcu(&client.fork_choice_states().await).safe_block_hash,
        genesis.block_info.hash,
        "the forkchoice safe label follows cross-safe, not the freshly reset local-safe head"
    );
}

/// A reset that rewinds must leave the heads in a state the execution layer will accept:
/// `unsafe >= local-safe >= cross-safe >= finalized`.
///
/// The interop engine is the case that needs this: its cross-safe head is held by the last
/// promotion, so nothing about the reset itself pulls it back. The bound comes from the state
/// transition, not from `reset_to` — this test exercises it end-to-end through the reset path
/// rather than asserting where it lives.
#[tokio::test]
async fn rewinding_reset_leaves_the_heads_ordered() {
    let cfg = Arc::new(RollupConfig::default());
    let client = walkback_blind_client(cfg.clone());
    let (genesis, b1, b3) = (block(0), block(1), block(3));

    let (state_tx, _state_rx) = watch::channel(EngineState::default());
    let (len_tx, _len_rx) = watch::channel(0usize);
    let (mut targeted, _promoter) = Engine::<MockEngineClient>::with_external_cross_safe(
        TestEngineStateBuilder::new()
            .with_unsafe_head(b3)
            .with_local_safe_head(b3)
            .with_cross_safe_head(b3)
            .with_finalized_head(genesis)
            .build(),
        state_tx,
        len_tx,
    );
    assert_eq!(targeted.state().sync_state.cross_safe_head(), b3, "test setup");

    // Rewind every head well behind where the engine currently sits.
    let target = L2ForkchoiceState { un_safe: b1, local_safe: b1, finalized: genesis };
    targeted.reset_to(client.clone(), cfg, target).await.expect("targeted reset");

    let sync = targeted.state().sync_state;
    assert_eq!(sync.unsafe_head(), b1);
    assert_eq!(sync.local_safe_head(), b1);
    assert_eq!(
        sync.cross_safe_head(),
        b1,
        "cross-safe must come down with the rewind rather than stay stranded at b3"
    );
    assert!(
        sync.unsafe_head().block_info.number >= sync.local_safe_head().block_info.number &&
            sync.local_safe_head().block_info.number >= sync.cross_safe_head().block_info.number &&
            sync.cross_safe_head().block_info.number >= sync.finalized_head().block_info.number,
        "heads left out of order after a rewinding reset: {sync:?}"
    );

    let fcu = last_fcu(&client.fork_choice_states().await);
    assert_eq!(fcu.head_block_hash, b1.block_info.hash);
    assert_eq!(
        fcu.safe_block_hash, b1.block_info.hash,
        "the forkchoice safe label must name a block at or behind the rewound head"
    );
}

/// The retry ceiling is the deliberate difference from [`Engine::reset`]'s unbounded loop: with
/// the heads fixed, every attempt puts the identical forkchoice update on the wire, so retrying
/// past the ceiling could only spin.
#[tokio::test]
async fn reset_to_gives_up_after_a_bounded_number_of_attempts() {
    let cfg = Arc::new(RollupConfig::default());
    // Every forkchoice update comes back with a status the engine treats as a temporary failure,
    // so the retry path is taken on each attempt and never succeeds.
    let client = Arc::new(
        test_engine_client_builder()
            .with_config(cfg.clone())
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(PayloadStatus::new(
                PayloadStatusEnum::Invalid { validation_error: String::new() },
                None,
            )))
            .build(),
    );
    let (genesis, b1, b2) = (block(0), block(1), block(2));

    let mut targeted = engine(TestEngineStateBuilder::new().build());
    let target = L2ForkchoiceState { un_safe: b2, local_safe: b1, finalized: genesis };
    let err = targeted
        .reset_to(client.clone(), cfg, target)
        .await
        .expect_err("a forkchoice update that never succeeds must surface as an error");

    assert!(
        matches!(err, EngineResetError::Forkchoice(_)),
        "the caller needs the forkchoice error back, not a sync-start error: {err:?}"
    );
    assert_eq!(
        client.fork_choice_states().await.len(),
        RESET_TO_MAX_ATTEMPTS,
        "reset_to must stop at its ceiling rather than resend the same heads forever"
    );
}

/// `reset` is unchanged: it still discovers its own start point and installs that, ignoring
/// wherever the engine happened to be.
#[tokio::test]
async fn reset_still_discovers_its_start_point_by_walkback() {
    // The walkback reads blocks back as *consensus* blocks, which recompute their hash rather than
    // trusting the RPC `hash` field, so the rollup genesis has to name the hash the header
    // actually hashes to.
    let header = alloy_consensus::Header { number: 0, ..Default::default() };
    let genesis_hash = header.hash_slow();
    let cfg = Arc::new(RollupConfig {
        genesis: ChainGenesis {
            l2: BlockNumHash { number: 0, hash: genesis_hash },
            ..Default::default()
        },
        ..Default::default()
    });
    let genesis = L2BlockInfo {
        block_info: BlockInfo {
            number: 0,
            hash: genesis_hash,
            parent_hash: B256::ZERO,
            timestamp: 0,
        },
        ..Default::default()
    };

    // A chain whose only block is genesis: the walkback loads it as the unsafe head, finds its L1
    // origin canonical, and stops immediately because the local-safe cursor is at genesis.
    let genesis_block: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> =
        alloy_rpc_types_eth::Block {
            header: alloy_rpc_types_eth::Header {
                hash: genesis_hash,
                inner: header,
                ..Default::default()
            },
            ..Default::default()
        };
    let client = Arc::new(
        test_engine_client_builder()
            .with_config(cfg.clone())
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(PayloadStatus::new(
                PayloadStatusEnum::Valid,
                None,
            )))
            .with_l2_block(BlockId::Number(BlockNumberOrTag::Latest), genesis_block.clone())
            .with_l2_block(BlockId::Number(0.into()), genesis_block)
            .with_l1_block(BlockId::Hash(B256::ZERO.into()), Default::default())
            .build(),
    );

    // The engine sits far ahead of what the chain can justify.
    let mut walking = engine(
        TestEngineStateBuilder::new()
            .with_unsafe_head(block(5))
            .with_local_safe_head(block(5))
            .with_finalized_head(genesis)
            .build(),
    );

    let landed = walking.reset(client.clone(), cfg).await.expect("walkback reset");

    assert_eq!(
        landed, genesis,
        "reset must land on the head the walkback discovered, not the one the engine held"
    );
    assert_eq!(walking.state().sync_state.unsafe_head(), genesis);
    assert_eq!(
        last_fcu(&client.fork_choice_states().await).head_block_hash,
        genesis_hash,
        "the forkchoice update must name the discovered head"
    );
}

/// An execution layer that cannot answer the walkback yet — unreachable, still starting up — is
/// retried rather than surfaced: op-node re-runs a failed `FindL2Heads` through its step backoff
/// forever (`op-node/rollup/engine/engine_controller.go:1423-1432`,
/// `op-node/rollup/driver/step_scheduling_deriver.go:97-109`), while returning the error here
/// killed the requesting actor — and with it the node — over a transient condition. The client
/// below fails its first two L2 block reads with a transport error and then serves the chain; the
/// reset must ride the failures out and land on the discovered head.
#[tokio::test]
async fn reset_retries_the_walkback_while_the_execution_layer_is_unreachable() {
    let header = alloy_consensus::Header { number: 0, ..Default::default() };
    let genesis_hash = header.hash_slow();
    let cfg = Arc::new(RollupConfig {
        genesis: ChainGenesis {
            l2: BlockNumHash { number: 0, hash: genesis_hash },
            ..Default::default()
        },
        ..Default::default()
    });
    let genesis = L2BlockInfo {
        block_info: BlockInfo {
            number: 0,
            hash: genesis_hash,
            parent_hash: B256::ZERO,
            timestamp: 0,
        },
        ..Default::default()
    };

    let genesis_block: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> =
        alloy_rpc_types_eth::Block {
            header: alloy_rpc_types_eth::Header {
                hash: genesis_hash,
                inner: header,
                ..Default::default()
            },
            ..Default::default()
        };
    let client = Arc::new(
        test_engine_client_builder()
            .with_config(cfg.clone())
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::new(PayloadStatus::new(
                PayloadStatusEnum::Valid,
                None,
            )))
            .with_l2_block(BlockId::Number(BlockNumberOrTag::Latest), genesis_block.clone())
            .with_l2_block(BlockId::Number(0.into()), genesis_block)
            .with_l1_block(BlockId::Hash(B256::ZERO.into()), Default::default())
            .with_l2_block_transport_failures(2)
            .build(),
    );

    let mut walking = engine(TestEngineStateBuilder::new().build());
    let landed = walking
        .reset(client.clone(), cfg)
        .await
        .expect("the reset must retry through the transport failures rather than surface them");

    assert_eq!(landed, genesis, "the reset must land on the discovered head after retrying");
}

/// A reset mutates the engine state outside `drain`, so it has to publish the state watch itself.
/// Every reader served through the watch — the RPC actor's queries, and the interop verifier's
/// observations through them — would otherwise keep seeing the heads from before the reset until
/// some unrelated task happened to succeed. That staleness is what held the interop verifier on an
/// invalidated block after its rewind: the chain was off the block, but the watch still answered
/// with it, and every round failed on the mismatch. (op-supernode's readers take the chain
/// container's live state under lock, so they cannot lag a rewind at all.)
#[tokio::test]
async fn reset_to_publishes_the_state_watch() {
    let cfg = Arc::new(RollupConfig::default());
    let client = walkback_blind_client(cfg.clone());
    let (genesis, b1, b2, b3) = (block(0), block(1), block(2), block(3));

    let initial = TestEngineStateBuilder::new()
        .with_unsafe_head(b3)
        .with_local_safe_head(b3)
        .with_finalized_head(genesis)
        .build();
    let (state_tx, state_rx) = watch::channel(initial);
    let (len_tx, _len_rx) = watch::channel(0usize);
    let mut targeted = Engine::new(initial, state_tx, len_tx);

    let target = L2ForkchoiceState { un_safe: b2, local_safe: b1, finalized: genesis };
    targeted.reset_to(client, cfg, target).await.expect("targeted reset");

    let published = *state_rx.borrow();
    assert_eq!(
        published.sync_state.local_safe_head(),
        b1,
        "the watch must reflect the reset heads, not the heads from before it"
    );
    assert_eq!(published.sync_state, targeted.state().sync_state);
}

/// A reset installs a walkback point found by traversing the L2 chain, not one derived from L1, so
/// it has no L1 key to pair with the head it writes — and the pairing it supersedes describes a
/// head the engine is no longer on. Recording the new head as unpaired is what invalidates it: a
/// consumer asking "which L1 block was the chain safe at?" gets the absent answer rather than a
/// stale one that reads like a real origin.
#[tokio::test]
async fn reset_to_invalidates_the_local_safe_pairing() {
    let cfg = Arc::new(RollupConfig::default());
    let client = walkback_blind_client(cfg.clone());
    let (genesis, b1, b2) = (block(0), block(1), block(2));

    let mut targeted = engine(
        TestEngineStateBuilder::new()
            .with_unsafe_head(b2)
            .with_local_safe_head(b2)
            .with_local_safe_origin(LocalSafeOrigin::DerivedFrom(BlockInfo {
                number: 9,
                ..Default::default()
            }))
            .build(),
    );
    assert!(
        targeted.state().sync_state.local_safe_origin().is_paired(),
        "test setup: the engine starts with a pairing to invalidate"
    );

    let target = L2ForkchoiceState { un_safe: b1, local_safe: b1, finalized: genesis };
    targeted.reset_to(client, cfg, target).await.expect("targeted reset");

    assert_eq!(targeted.state().sync_state.local_safe_head(), b1);
    assert_eq!(
        targeted.state().sync_state.local_safe_origin(),
        LocalSafeOrigin::Unpaired,
        "the reset must not leave behind an L1 origin for a head it walked back from"
    );
    assert_eq!(targeted.state().sync_state.local_safe().derived_from_l1(), None);
}
