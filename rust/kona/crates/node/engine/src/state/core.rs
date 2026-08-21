//! The internal state of the engine controller.

use crate::{
    Metrics,
    state::{CrossSafePromotion, CrossSafeSource, LocalSafeHead, LocalSafeOrigin},
};
use alloy_rpc_types_engine::ForkchoiceState;
use kona_protocol::L2BlockInfo;
use serde::{Deserialize, Serialize};

/// The synchronization state of the execution layer across different safety levels.
///
/// Tracks block progression through various stages of verification and finalization,
/// from initial unsafe blocks received via P2P to fully finalized blocks derived from
/// finalized L1 data. Each level represents increasing confidence in the block's validity.
///
/// # Safety Levels
///
/// The state tracks blocks at different safety levels, listed from least to most safe:
///
/// 1. **Unsafe** - Most recent blocks from P2P network (unverified). There is no local/cross split
///    here: a single unsafe head.
/// 2. **Local-safe** - Derived from this chain's L1 data, completed span-batch.
/// 3. **Cross-safe** - Local-safe *and* cross-verified against the other chains' safe dependencies.
///    This is the head reported as `safeBlockHash` in the forkchoice update. It *advances* only
///    through [`EngineSyncState::apply_cross_safe_promotion`]; the only other move it makes is
///    downwards, when [`EngineSyncState::apply_update`] rewinds the local-safe head below it.
/// 4. **Finalized** - Derived from finalized L1 data only.
///
/// See the [OP Stack specifications](https://specs.optimism.io) for detailed safety definitions.
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq)]
pub struct EngineSyncState {
    /// Most recent block found on the P2P network (lowest safety level).
    unsafe_head: L2BlockInfo,
    /// Derived from L1 data as a completed span-batch, but not yet cross-verified.
    local_safe_head: L2BlockInfo,
    /// The L1 block `local_safe_head` was derived from, as recorded by whoever wrote that head.
    ///
    /// Written in the same step as the head it describes, so the two cannot drift apart; see
    /// [`EngineSyncState::local_safe`] for the paired read.
    local_safe_origin: LocalSafeOrigin,
    /// Local-safe and cross-verified to have safe L1 dependencies on every dependency chain.
    cross_safe_head: L2BlockInfo,
    /// Derived from finalized L1 data with only finalized dependencies (highest safety level).
    finalized_head: L2BlockInfo,
    /// Where cross-safe promotions for this engine come from.
    cross_safe_source: CrossSafeSource,
}

impl EngineSyncState {
    /// Returns the current unsafe head.
    pub const fn unsafe_head(&self) -> L2BlockInfo {
        self.unsafe_head
    }

    /// Returns the current local-safe head.
    pub const fn local_safe_head(&self) -> L2BlockInfo {
        self.local_safe_head
    }

    /// Returns the L1 origin recorded for the current local-safe head.
    ///
    /// [`LocalSafeOrigin::Unpaired`] is a real answer — the writer of this head held no L1 key —
    /// and not a placeholder for a value that will arrive later.
    pub const fn local_safe_origin(&self) -> LocalSafeOrigin {
        self.local_safe_origin
    }

    /// Returns the current local-safe head together with its L1 origin.
    ///
    /// This is the atomic read of the pairing: both halves come from the same snapshot of this
    /// state, so a consumer answering "which L1 block was the chain safe at?" cannot observe a head
    /// from one update alongside an origin from another. Reading
    /// [`Self::local_safe_head`] and [`Self::local_safe_origin`] separately off a `Copy` of this
    /// state is equivalent; reading them off a live engine is not.
    pub const fn local_safe(&self) -> LocalSafeHead {
        LocalSafeHead::new(self.local_safe_head, self.local_safe_origin)
    }

    /// Returns the current cross-safe head, i.e. the forkchoice `safeBlockHash`.
    pub const fn cross_safe_head(&self) -> L2BlockInfo {
        self.cross_safe_head
    }

    /// Returns the current finalized head.
    pub const fn finalized_head(&self) -> L2BlockInfo {
        self.finalized_head
    }

    /// Returns where this engine's cross-safe promotions come from.
    pub const fn cross_safe_source(&self) -> CrossSafeSource {
        self.cross_safe_source
    }

    /// Returns a copy of this state whose cross-safe head is fed by the given source.
    pub(crate) const fn with_cross_safe_source(mut self, source: CrossSafeSource) -> Self {
        self.cross_safe_source = source;
        self
    }

    /// Creates a `ForkchoiceState`
    ///
    /// - `head_block` = `unsafe_head`
    /// - `safe_block` = `cross_safe_head`
    /// - `finalized_block` = `finalized_head`
    ///
    /// If the block info is not yet available, the default values are used.
    pub const fn create_forkchoice_state(&self) -> ForkchoiceState {
        ForkchoiceState {
            head_block_hash: self.unsafe_head.hash(),
            safe_block_hash: self.cross_safe_head.hash(),
            finalized_block_hash: self.finalized_head.hash(),
        }
    }

    /// Applies the update to the provided sync state, using the current state values if the update
    /// is not specified. Returns the new sync state.
    ///
    /// [`EngineSyncStateUpdate`] cannot express a cross-safe move. When this engine's cross-safe
    /// head is fed from local-safe (standalone kona-node), a local-safe advance mints the
    /// corresponding trivial promotion here, so the resulting forkchoice update still carries
    /// local-safe == cross-safe in a single call.
    ///
    /// A local-safe *rewind* is the one case where this moves the cross-safe head without a
    /// promotion: it is held at the rewound head, since cross-safe cannot outrank the local-safe
    /// head it is derived from. See the private `hold_cross_safe_at` helper below.
    ///
    /// Private to this module: callers go through [`EngineState::apply_sync_update`], which
    /// supplies the chain ID from the state that already owns it. Every head this writes is
    /// reported as a chain-labelled metric, so a multi-chain process can tell the series apart.
    fn apply_update(self, chain_id: u64, sync_state_update: EngineSyncStateUpdate) -> Self {
        if let Some(unsafe_head) = sync_state_update.unsafe_head {
            Self::update_block_label_metric(
                chain_id,
                Metrics::UNSAFE_BLOCK_LABEL,
                unsafe_head.block_info.number,
            );
        }
        if let Some(local_safe) = sync_state_update.local_safe_head {
            Self::update_block_label_metric(
                chain_id,
                Metrics::LOCAL_SAFE_BLOCK_LABEL,
                local_safe.head.block_info.number,
            );
        }
        if let Some(finalized_head) = sync_state_update.finalized_head {
            Self::update_block_label_metric(
                chain_id,
                Metrics::FINALIZED_BLOCK_LABEL,
                finalized_head.block_info.number,
            );
        }

        // The pairing is rewritten exactly when the head it describes is, so an origin can never
        // outlive the head it was recorded for: an update that leaves the local-safe head alone
        // carries the previous origin through, and one that moves it — a reset included — replaces
        // the origin with whatever that writer holds, which may be `Unpaired`.
        let local_safe = sync_state_update.local_safe_head.unwrap_or_else(|| self.local_safe());

        let updated = Self {
            unsafe_head: sync_state_update.unsafe_head.unwrap_or(self.unsafe_head),
            local_safe_head: local_safe.head,
            local_safe_origin: local_safe.origin,
            cross_safe_head: self.hold_cross_safe_at(chain_id, local_safe.head),
            finalized_head: sync_state_update.finalized_head.unwrap_or(self.finalized_head),
            cross_safe_source: self.cross_safe_source,
        };

        sync_state_update
            .local_safe_head
            .and_then(|local_safe| updated.cross_safe_source.trivial_promotion(local_safe.head))
            .map_or(updated, |promotion| updated.apply_cross_safe_promotion(chain_id, promotion))
    }

    /// Returns the cross-safe head held at `local_safe_head`, which the caller is about to install.
    ///
    /// [`EngineSyncStateUpdate`] cannot express a cross-safe move, so
    /// [`Self::apply_update`] carries the cross-safe head through — but the local-safe head it is
    /// carried past can move *backwards*, and cross-safe is local-safe *and* cross-verified, so it
    /// cannot outrank it. Three writers rewind local-safe:
    ///
    /// - [`Engine::reset`] installs the [`find_starting_forkchoice`] walkback point, deliberately
    ///   at least a sequencing window behind the unsafe head.
    /// - [`Engine::reset_to`] installs whatever heads its caller hands it, which may be behind the
    ///   engine's current ones.
    /// - `ConsolidateTask::reconcile_to_local_safe_head` installs the injected local-safe block as
    ///   both the local-safe and the unsafe head.
    ///
    /// Neither carries a promotion, and under [`CrossSafeSource::Promoted`] no trivial promotion is
    /// minted either, so this is the only thing that can hold the cross-safe head down. Left alone
    /// it would sit above both new heads and [`Self::create_forkchoice_state`] would report
    /// `safeBlockHash` ahead of `headBlockHash` — the `INVALID_FORK_CHOICE_STATE` rejection that
    /// costs a full engine reset.
    ///
    /// Only the local-safe head is compared. The unsafe head is the local-safe head's own upper
    /// bound, so a state where it alone drops below cross-safe already violates
    /// `unsafe >= local-safe`, which no writer produces; guarding it here would clamp at the point
    /// of damage rather than maintain the invariant at its source.
    ///
    /// [`Engine::reset`]: crate::Engine::reset
    /// [`Engine::reset_to`]: crate::Engine::reset_to
    /// [`find_starting_forkchoice`]: crate::find_starting_forkchoice
    fn hold_cross_safe_at(&self, chain_id: u64, local_safe_head: L2BlockInfo) -> L2BlockInfo {
        if self.cross_safe_head.block_info.number <= local_safe_head.block_info.number {
            return self.cross_safe_head;
        }

        // Reported rather than warned about: unlike the clamps in
        // `Self::apply_cross_safe_promotion`, this is not a disagreement with the promotion source.
        // A reset walkback or a consolidation rewind is ordinary node operation and would fire this
        // on every interop reset, so a `warn!` here would be noise that devalues the promotion
        // clamp's warnings — but the cross-safe head moving backwards is a real safety-level
        // regression, so it belongs in the log at default verbosity rather than behind `debug!`.
        info!(
            target: "engine",
            cross_safe = self.cross_safe_head.block_info.number,
            local_safe = local_safe_head.block_info.number,
            "Local-safe head rewound below the cross-safe head; holding cross-safe at local-safe"
        );
        Self::update_block_label_metric(
            chain_id,
            Metrics::CROSS_SAFE_BLOCK_LABEL,
            local_safe_head.block_info.number,
        );

        local_safe_head
    }

    /// Applies a cross-safe promotion. This is the single entry point through which the cross-safe
    /// head — and therefore the forkchoice `safeBlockHash` — moves.
    ///
    /// The target is clamped into the band the safety definition allows:
    ///
    /// - **At or below local-safe.** Cross-safe is local-safe *and* cross-verified, so a promotion
    ///   naming a block this engine has not derived locally yet is held at the local-safe head.
    ///   Unclamped, [`Self::create_forkchoice_state`] would report `safeBlockHash` ahead of
    ///   `headBlockHash`; the EL rejects that with `INVALID_FORK_CHOICE_STATE`, which arrives as
    ///   [`SynchronizeTaskError::InvalidForkchoiceState`] and costs a full engine reset.
    /// - **At or above finalized.** Promotions may move the cross-safe head backwards, which is how
    ///   an interop rewind unwinds a decision, but finalization is irreversible, so a target below
    ///   the finalized head is held there.
    ///
    /// Both clamps warn rather than absorbing the disagreement silently: either one means the
    /// promotion source and this engine disagree about the chain, which is a verifier bug worth
    /// diagnosing. Clamping keeps the invariant without a fatal error — the same treatment
    /// `EngineController.resolveVerifiedAsSafe` gives a super-authority head that runs ahead of
    /// local-safe in op-node.
    ///
    /// [`SynchronizeTaskError::InvalidForkchoiceState`]: crate::SynchronizeTaskError::InvalidForkchoiceState
    pub fn apply_cross_safe_promotion(self, chain_id: u64, promotion: CrossSafePromotion) -> Self {
        let target = promotion.target();

        let target = if target.block_info.number > self.local_safe_head.block_info.number {
            warn!(
                target: "engine",
                promotion = target.block_info.number,
                local_safe = self.local_safe_head.block_info.number,
                "Cross-safe promotion is ahead of the local-safe head; holding at local-safe"
            );
            self.local_safe_head
        } else {
            target
        };

        let cross_safe_head = if target.block_info.number < self.finalized_head.block_info.number {
            warn!(
                target: "engine",
                promotion = target.block_info.number,
                finalized = self.finalized_head.block_info.number,
                "Cross-safe promotion is below the finalized head; holding at finalized"
            );
            self.finalized_head
        } else {
            target
        };

        Self::update_block_label_metric(
            chain_id,
            Metrics::CROSS_SAFE_BLOCK_LABEL,
            cross_safe_head.block_info.number,
        );

        Self { cross_safe_head, ..self }
    }

    /// Updates a block label metric, keyed by the chain ID and the label.
    #[inline]
    fn update_block_label_metric(chain_id: u64, label: &'static str, number: u64) {
        kona_macros::set!(
            gauge,
            Metrics::BLOCK_LABELS,
            number as f64,
            "label" => label,
            Metrics::CHAIN_ID_LABEL => chain_id.to_string()
        );
    }
}

/// Specifies how to update the sync state of the engine.
///
/// There is deliberately no cross-safe field: the cross-safe head advances only through
/// [`EngineSyncState::apply_cross_safe_promotion`], so an ordinary head writer cannot advance it.
/// A writer that rewinds the local-safe head does drag it down, because cross-safe cannot outrank
/// local-safe — see [`EngineSyncState::apply_update`].
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct EngineSyncStateUpdate {
    /// Most recent block found on the p2p network
    pub unsafe_head: Option<L2BlockInfo>,
    /// Derived from L1, and known to be a completed span-batch, but not cross-verified yet,
    /// paired with the L1 block it was derived from.
    ///
    /// The pairing is part of the head rather than a field beside it, so a writer cannot advance
    /// the local-safe head while leaving a previous L1 origin in place. Writers that hold no L1
    /// key say so with [`LocalSafeHead::unpaired`].
    pub local_safe_head: Option<LocalSafeHead>,
    /// Derived from finalized L1 data,
    /// and cross-verified to only have finalized dependencies.
    pub finalized_head: Option<L2BlockInfo>,
}

impl EngineSyncStateUpdate {
    /// An update that changes nothing.
    pub const NONE: Self = Self { unsafe_head: None, local_safe_head: None, finalized_head: None };
}

/// The chain state viewed by the engine controller.
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq)]
pub struct EngineState {
    /// The L2 chain ID this engine drives. Emitted as a label on every engine metric so that a
    /// multi-chain process can tell the per-chain series apart.
    pub chain_id: u64,

    /// The sync state of the engine.
    pub sync_state: EngineSyncState,

    /// Whether or not the EL has finished syncing.
    pub el_sync_finished: bool,

    /// Whether a forkchoice update has been dispatched to the execution layer yet.
    ///
    /// The initial forkchoice update has to be emitted even when it carries no change, so
    /// [`crate::SynchronizeTask`] only skips a no-op update once this is set.
    ///
    /// It is tracked explicitly rather than inferred from the sync state differing from its
    /// default, because that inference is defeated by any non-head field: an engine built by
    /// [`Engine::with_external_cross_safe`] carries [`CrossSafeSource::Promoted`] from birth, so
    /// its sync state is unequal to the default before anything has happened, and it would
    /// silently skip the very first forkchoice update.
    ///
    /// [`Engine::with_external_cross_safe`]: crate::Engine::with_external_cross_safe
    /// [`CrossSafeSource::Promoted`]: crate::CrossSafeSource::Promoted
    pub forkchoice_emitted: bool,

    /// Track when the rollup node changes the forkchoice to restore previous
    /// known unsafe chain. e.g. Unsafe Reorg caused by Invalid span batch.
    /// This update does not retry except engine returns non-input error
    /// because engine may forgot backupUnsafeHead or backupUnsafeHead is not part
    /// of the chain.
    pub need_fcu_call_backup_unsafe_reorg: bool,
}

impl EngineState {
    /// Applies `update` to this state's sync state, returning the new sync state.
    ///
    /// [`EngineState`] owns both the sync state and the chain ID that its metrics are labelled
    /// with, so callers never pass the chain ID across the API boundary themselves.
    pub fn apply_sync_update(&self, update: EngineSyncStateUpdate) -> EngineSyncState {
        self.sync_state.apply_update(self.chain_id, update)
    }

    /// Applies `promotion` to this state's sync state, returning the new sync state.
    ///
    /// The counterpart of [`Self::apply_sync_update`] for the one entry point through which the
    /// cross-safe head moves: [`EngineState`] owns the chain ID its metrics are labelled with, so
    /// callers do not pass it across the API boundary themselves.
    pub fn apply_cross_safe_promotion(&self, promotion: CrossSafePromotion) -> EngineSyncState {
        self.sync_state.apply_cross_safe_promotion(self.chain_id, promotion)
    }

    /// Returns if consolidation is needed.
    ///
    /// [Consolidation] is only performed by a rollup node when the unsafe head
    /// is ahead of the local-safe head. When the two are equal, consolidation isn't
    /// required and the [`crate::BuildTask`] can be used to build the block.
    ///
    /// [Consolidation]: https://specs.optimism.io/protocol/derivation.html#l1-consolidation-payload-attributes-matching
    pub fn needs_consolidation(&self) -> bool {
        self.sync_state.local_safe_head() != self.sync_state.unsafe_head()
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::{Metrics, state::CrossSafePromoter};
    use alloy_primitives::B256;
    use kona_protocol::BlockInfo;
    use metrics_exporter_prometheus::PrometheusBuilder;
    use rstest::rstest;

    impl EngineState {
        /// Set the unsafe head.
        pub fn set_unsafe_head(&mut self, unsafe_head: L2BlockInfo) {
            self.apply_sync_update(EngineSyncStateUpdate {
                unsafe_head: Some(unsafe_head),
                ..Default::default()
            });
        }

        /// Set the local safe head.
        pub fn set_local_safe_head(&mut self, local_safe_head: L2BlockInfo) {
            self.apply_sync_update(EngineSyncStateUpdate {
                local_safe_head: Some(LocalSafeHead::unpaired(local_safe_head)),
                ..Default::default()
            });
        }

        /// Promote the cross-safe head.
        ///
        /// A promotion is held at the local-safe head, so the target has to be local-safe first —
        /// cross-safe is local-safe *and* cross-verified. Unlike the other setters here, this one
        /// has to keep the resulting state, because the clamp reads `local_safe_head` back.
        pub fn set_cross_safe_head(&mut self, cross_safe_head: L2BlockInfo) {
            self.sync_state = self.apply_sync_update(EngineSyncStateUpdate {
                local_safe_head: Some(LocalSafeHead::unpaired(cross_safe_head)),
                ..Default::default()
            });
            self.sync_state =
                self.apply_cross_safe_promotion(CrossSafePromoter::new().promote(cross_safe_head));
        }

        /// Set the finalized head.
        pub fn set_finalized_head(&mut self, finalized_head: L2BlockInfo) {
            self.apply_sync_update(EngineSyncStateUpdate {
                finalized_head: Some(finalized_head),
                ..Default::default()
            });
        }
    }

    /// The chain id the state-level tests label their metrics with.
    const TEST_CHAIN_ID: u64 = 10;

    fn l1(number: u64) -> BlockInfo {
        BlockInfo { number, hash: B256::repeat_byte(number as u8), ..Default::default() }
    }

    fn l2(number: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                number,
                hash: B256::repeat_byte(number as u8),
                ..Default::default()
            },
            ..Default::default()
        }
    }

    /// The read the interop query needs: one call, both halves, from one snapshot.
    #[test]
    fn local_safe_reads_the_head_and_its_origin_together() {
        let state = EngineSyncState::default().apply_update(
            TEST_CHAIN_ID,
            EngineSyncStateUpdate {
                local_safe_head: Some(LocalSafeHead::derived_from(l2(4), l1(2))),
                ..EngineSyncStateUpdate::NONE
            },
        );

        assert_eq!(state.local_safe(), LocalSafeHead::derived_from(l2(4), l1(2)));
        assert_eq!(state.local_safe().head, state.local_safe_head());
        assert_eq!(state.local_safe().origin, state.local_safe_origin());
        assert_eq!(state.local_safe().derived_from_l1(), Some(l1(2)));
    }

    /// A fresh state has a local-safe head and no origin for it, and says so.
    #[test]
    fn a_default_state_is_unpaired() {
        let state = EngineSyncState::default();

        assert_eq!(state.local_safe_origin(), LocalSafeOrigin::Unpaired);
        assert_eq!(state.local_safe().derived_from_l1(), None);
    }

    /// The origin belongs to the head it was written with. An update that does not touch the head
    /// must not disturb it either.
    #[test]
    fn an_update_that_leaves_the_head_alone_carries_the_origin_through() {
        let paired = EngineSyncState::default().apply_update(
            TEST_CHAIN_ID,
            EngineSyncStateUpdate {
                local_safe_head: Some(LocalSafeHead::derived_from(l2(4), l1(2))),
                ..EngineSyncStateUpdate::NONE
            },
        );

        let after = paired.apply_update(
            TEST_CHAIN_ID,
            EngineSyncStateUpdate {
                unsafe_head: Some(l2(9)),
                finalized_head: Some(l2(1)),
                ..EngineSyncStateUpdate::NONE
            },
        );

        assert_eq!(after.local_safe(), paired.local_safe());
    }

    /// Moving the head rewrites the pairing, so an origin can never describe a head that has since
    /// moved on.
    #[test]
    fn moving_the_head_rewrites_the_origin() {
        let state = EngineSyncState::default()
            .apply_update(
                TEST_CHAIN_ID,
                EngineSyncStateUpdate {
                    local_safe_head: Some(LocalSafeHead::derived_from(l2(4), l1(2))),
                    ..EngineSyncStateUpdate::NONE
                },
            )
            .apply_update(
                TEST_CHAIN_ID,
                EngineSyncStateUpdate {
                    local_safe_head: Some(LocalSafeHead::derived_from(l2(5), l1(3))),
                    ..EngineSyncStateUpdate::NONE
                },
            );

        assert_eq!(state.local_safe(), LocalSafeHead::derived_from(l2(5), l1(3)));
    }

    /// An unpaired write is not a no-op: it *invalidates* the pairing it replaces. This is what
    /// keeps a reset, or the derivation-delegation path, from leaving an L1 key behind that
    /// describes a head the engine is no longer on.
    #[test]
    fn an_unpaired_write_invalidates_the_previous_pairing() {
        let state = EngineSyncState::default()
            .apply_update(
                TEST_CHAIN_ID,
                EngineSyncStateUpdate {
                    local_safe_head: Some(LocalSafeHead::derived_from(l2(4), l1(2))),
                    ..EngineSyncStateUpdate::NONE
                },
            )
            .apply_update(
                TEST_CHAIN_ID,
                EngineSyncStateUpdate {
                    local_safe_head: Some(LocalSafeHead::unpaired(l2(2))),
                    ..EngineSyncStateUpdate::NONE
                },
            );

        assert_eq!(state.local_safe_head(), l2(2));
        assert_eq!(
            state.local_safe_origin(),
            LocalSafeOrigin::Unpaired,
            "a rewind to a head with no known origin must not keep the old one"
        );
        assert_eq!(state.local_safe().derived_from_l1(), None);
    }

    /// A local-safe rewind drags the cross-safe head down with it. The pairing has to survive that
    /// path intact, since it runs through a different branch of `apply_update`.
    #[test]
    fn a_rewind_that_holds_cross_safe_still_records_the_origin() {
        let state = EngineSyncState::default()
            .apply_update(
                TEST_CHAIN_ID,
                EngineSyncStateUpdate {
                    local_safe_head: Some(LocalSafeHead::derived_from(l2(9), l1(4))),
                    ..EngineSyncStateUpdate::NONE
                },
            )
            .apply_update(
                TEST_CHAIN_ID,
                EngineSyncStateUpdate {
                    local_safe_head: Some(LocalSafeHead::derived_from(l2(3), l1(1))),
                    ..EngineSyncStateUpdate::NONE
                },
            );

        assert_eq!(state.cross_safe_head(), l2(3), "cross-safe cannot outrank local-safe");
        assert_eq!(state.local_safe(), LocalSafeHead::derived_from(l2(3), l1(1)));
    }

    #[rstest]
    #[case::set_unsafe(EngineState::set_unsafe_head, Metrics::UNSAFE_BLOCK_LABEL, 1)]
    #[case::set_local_safe(EngineState::set_local_safe_head, Metrics::LOCAL_SAFE_BLOCK_LABEL, 3)]
    #[case::set_cross_safe_head(
        EngineState::set_cross_safe_head,
        Metrics::CROSS_SAFE_BLOCK_LABEL,
        4
    )]
    #[case::set_finalized_head(EngineState::set_finalized_head, Metrics::FINALIZED_BLOCK_LABEL, 5)]
    #[cfg(feature = "metrics")]
    fn test_chain_label_metrics(
        #[case] set_fn: impl Fn(&mut EngineState, L2BlockInfo),
        #[case] label_name: &str,
        #[case] number: u64,
    ) {
        const CHAIN_ID: u64 = 10;

        // A local recorder keeps the rstest cases independent; a global one can only be
        // installed once per process, so all but the first case would fail.
        let recorder = PrometheusBuilder::new().build_recorder();
        let handle = recorder.handle();

        metrics::with_local_recorder(&recorder, || {
            crate::Metrics::init(CHAIN_ID);

            let mut state = EngineState { chain_id: CHAIN_ID, ..Default::default() };
            set_fn(
                &mut state,
                L2BlockInfo {
                    block_info: BlockInfo { number, ..Default::default() },
                    ..Default::default()
                },
            );
        });

        let rendered = handle.render();
        // The line is selected by the label under test, not by being the first block-labels line:
        // a setter may write more than one head — promoting the cross-safe head first advances
        // local-safe, since cross-safe is local-safe *and* cross-verified — and the exporter does
        // not guarantee the order it renders a metric's series in. Label ordering within a line is
        // not guaranteed either, so the labels are matched individually rather than as one string.
        let prefix = format!("{}{{", Metrics::BLOCK_LABELS);
        let line = rendered
            .lines()
            .find(|line| {
                line.starts_with(&prefix) && line.contains(&format!("label=\"{label_name}\""))
            })
            .unwrap_or_else(|| panic!("no {label_name} block label was rendered in:\n{rendered}"));

        assert!(line.contains(&format!("{}=\"{CHAIN_ID}\"", Metrics::CHAIN_ID_LABEL)), "{line}");
        assert!(line.ends_with(&format!(" {number}")), "{line}");
    }
}
