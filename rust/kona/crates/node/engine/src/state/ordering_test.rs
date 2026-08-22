//! The head-ordering invariant, across the transitions that stress it.
//!
//! `unsafe >= local-safe >= cross-safe >= finalized` is not enforced in one place. Four separate
//! mechanisms hold it, each guarding a different writer:
//!
//! - [`FinalizeTask`] drops a finality signal whose block is not yet cross-safe, so the finalized
//!   head cannot overtake cross-safe.
//! - [`EngineSyncState::hold_cross_safe_at`] drags the cross-safe head *down* when a writer rewinds
//!   the local-safe head beneath it, because an update cannot express a cross-safe move.
//! - [`EngineSyncState::apply_cross_safe_promotion`] clamps a promotion that runs ahead of
//!   local-safe, and clamps one that falls below finalized.
//!
//! Each has its own unit test. What none of them covers is the *sequence*: a reset rewinds the
//! heads, then a promotion arrives that was decided against the pre-reset chain. That is where the
//! mechanisms interact, and where a regression in one would be masked by another until it wasn't.
//! These tests assert the invariant after every individual transition rather than only at the end,
//! so a violation names the step that introduced it.
//!
//! [`FinalizeTask`]: crate::FinalizeTask

use crate::{
    EngineState, EngineSyncState, EngineSyncStateUpdate, LocalSafeHead, state::CrossSafePromoter,
    test_utils::TestEngineStateBuilder,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use kona_protocol::{BlockInfo, L2BlockInfo};

/// A block at `number`, with a hash that identifies it in a failure message.
fn block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            number,
            hash: B256::repeat_byte(number as u8),
            parent_hash: B256::repeat_byte(number.saturating_sub(1) as u8),
            timestamp: number * 2,
        },
        l1_origin: BlockNumHash::default(),
        seq_num: 0,
    }
}

/// The four head numbers, in the order the invariant requires them to descend.
fn heads(state: &EngineSyncState) -> (u64, u64, u64, u64) {
    (
        state.unsafe_head().block_info.number,
        state.local_safe_head().block_info.number,
        state.cross_safe_head().block_info.number,
        state.finalized_head().block_info.number,
    )
}

/// Asserts `unsafe >= local-safe >= cross-safe >= finalized`, naming the step that broke it.
///
/// Each pair is checked separately so the failure says which boundary went wrong rather than only
/// that the ordering is off somewhere.
fn assert_ordered(state: &EngineSyncState, step: &str) {
    let (u, l, c, f) = heads(state);
    assert!(u >= l, "{step}: unsafe {u} is behind local-safe {l}");
    assert!(l >= c, "{step}: local-safe {l} is behind cross-safe {c}");
    assert!(c >= f, "{step}: cross-safe {c} is behind finalized {f}");
}

/// Applies a reset: the walkback installs all three heads it knows, unpaired, with no promotion.
///
/// This is what [`crate::Engine::reset_to`] does through [`crate::SynchronizeTask`], and the reason
/// it is the interesting half of the sequence: it can move the local-safe head *backwards* while
/// carrying no cross-safe promotion, so the cross-safe head has to be held down by the state
/// transition itself.
fn reset_to(state: &EngineState, target: L2BlockInfo, finalized: L2BlockInfo) -> EngineSyncState {
    state.apply_sync_update(EngineSyncStateUpdate {
        unsafe_head: Some(target),
        local_safe_head: Some(LocalSafeHead::unpaired(target)),
        finalized_head: Some(finalized),
    })
}

/// A caught-up interop chain: every head at 20 except a finalized head two epochs back.
fn caught_up() -> EngineState {
    TestEngineStateBuilder::new()
        .with_unsafe_head(block(20))
        .with_local_safe_head(block(20))
        .with_cross_safe_head(block(20))
        .with_finalized_head(block(10))
        .build()
}

/// The sequence the individual unit tests miss: reset the heads back, then apply a promotion that
/// was decided against the chain as it stood *before* the reset.
///
/// The promotion is not merely stale, it is impossible — it names a block the engine has just
/// disowned. The clamp has to notice that and hold at local-safe, and the invariant has to survive
/// both steps, not just the second.
#[test]
fn a_promotion_decided_before_a_reset_cannot_break_the_ordering() {
    let mut state = caught_up();
    assert_ordered(&state.sync_state, "caught up");

    // A reset walks the heads back to 12; nothing promotes, so cross-safe must be dragged down.
    state.sync_state = reset_to(&state, block(12), block(10));
    assert_ordered(&state.sync_state, "after reset to 12");
    assert_eq!(
        state.sync_state.cross_safe_head().block_info.number,
        12,
        "the reset must drag cross-safe down to local-safe, not leave it at 20"
    );

    // The verifier's in-flight round finishes and promotes 18 — decided against the pre-reset
    // chain, and now ahead of a local-safe head that has moved backwards underneath it.
    state.sync_state =
        state.apply_cross_safe_promotion(CrossSafePromoter::new().promote(block(18)));
    assert_ordered(&state.sync_state, "after a stale promotion to 18");
    assert_eq!(
        state.sync_state.cross_safe_head().block_info.number,
        12,
        "a promotion ahead of local-safe must be held at local-safe"
    );
}

/// The other end of the same sequence: a reset deep enough to land beneath the finalized head.
///
/// Finalization is irreversible, so the finalized head is the floor no rewind may cross. A
/// promotion arriving afterwards must not be able to place cross-safe below it either.
#[test]
fn a_promotion_below_the_finalized_head_cannot_break_the_ordering() {
    let mut state = caught_up();

    // Reset to 12, then promote to 5 — below the finalized head at 10.
    state.sync_state = reset_to(&state, block(12), block(10));
    assert_ordered(&state.sync_state, "after reset to 12");

    state.sync_state = state.apply_cross_safe_promotion(CrossSafePromoter::new().promote(block(5)));
    assert_ordered(&state.sync_state, "after a promotion to 5, below finalized 10");
    assert_eq!(
        state.sync_state.cross_safe_head().block_info.number,
        10,
        "a promotion below finalized must be held at finalized"
    );
}

/// Reset, promote, reset again, promote again — the interleaving a chain reorging under a running
/// verifier actually produces.
///
/// A single reset-then-promote pair can pass while the mechanisms disagree about which head is
/// authoritative; repeating the pair at different depths is what exposes that.
#[test]
fn repeated_resets_and_promotions_preserve_the_ordering() {
    let mut state = caught_up();

    for (reset_to_number, promote_to_number) in
        [(18u64, 20u64), (14, 18), (11, 13), (10, 10), (16, 15)]
    {
        state.sync_state = reset_to(&state, block(reset_to_number), block(10));
        assert_ordered(&state.sync_state, &format!("after reset to {reset_to_number}"));

        state.sync_state = state
            .apply_cross_safe_promotion(CrossSafePromoter::new().promote(block(promote_to_number)));
        assert_ordered(
            &state.sync_state,
            &format!("after reset to {reset_to_number} then promotion to {promote_to_number}"),
        );

        // Whatever the clamps decided, cross-safe may never exceed the local-safe head the reset
        // installed — that is the property the promotion path exists to preserve.
        assert!(
            state.sync_state.cross_safe_head().block_info.number <= reset_to_number.max(10),
            "cross-safe {} escaped the reset target {reset_to_number}",
            state.sync_state.cross_safe_head().block_info.number
        );
    }
}

/// A promotion to exactly the local-safe head is the ordinary case and must be applied verbatim,
/// so the clamps above are not quietly holding cross-safe back in normal operation.
#[test]
fn a_promotion_at_the_local_safe_head_is_applied_unchanged() {
    let mut state = caught_up();
    state.sync_state = reset_to(&state, block(12), block(10));

    state.sync_state =
        state.apply_cross_safe_promotion(CrossSafePromoter::new().promote(block(12)));
    assert_ordered(&state.sync_state, "after promoting to local-safe");
    assert_eq!(state.sync_state.cross_safe_head().block_info.number, 12);
}
