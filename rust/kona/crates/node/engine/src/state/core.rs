//! The internal state of the engine controller.

use crate::{
    Metrics,
    state::{CrossSafePromotion, CrossSafeSource},
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
///    This is the head reported as `safeBlockHash` in the forkchoice update, and it moves only
///    through [`EngineSyncState::apply_cross_safe_promotion`].
/// 4. **Finalized** - Derived from finalized L1 data only.
///
/// See the [OP Stack specifications](https://specs.optimism.io) for detailed safety definitions.
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq)]
pub struct EngineSyncState {
    /// Most recent block found on the P2P network (lowest safety level).
    unsafe_head: L2BlockInfo,
    /// Derived from L1 data as a completed span-batch, but not yet cross-verified.
    local_safe_head: L2BlockInfo,
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
    pub fn apply_update(self, sync_state_update: EngineSyncStateUpdate) -> Self {
        if let Some(unsafe_head) = sync_state_update.unsafe_head {
            Self::update_block_label_metric(
                Metrics::UNSAFE_BLOCK_LABEL,
                unsafe_head.block_info.number,
            );
        }
        if let Some(local_safe_head) = sync_state_update.local_safe_head {
            Self::update_block_label_metric(
                Metrics::LOCAL_SAFE_BLOCK_LABEL,
                local_safe_head.block_info.number,
            );
        }
        if let Some(finalized_head) = sync_state_update.finalized_head {
            Self::update_block_label_metric(
                Metrics::FINALIZED_BLOCK_LABEL,
                finalized_head.block_info.number,
            );
        }

        let updated = Self {
            unsafe_head: sync_state_update.unsafe_head.unwrap_or(self.unsafe_head),
            local_safe_head: sync_state_update.local_safe_head.unwrap_or(self.local_safe_head),
            cross_safe_head: self.cross_safe_head,
            finalized_head: sync_state_update.finalized_head.unwrap_or(self.finalized_head),
            cross_safe_source: self.cross_safe_source,
        };

        match sync_state_update
            .local_safe_head
            .and_then(|head| updated.cross_safe_source.trivial_promotion(head))
        {
            Some(promotion) => updated.apply_cross_safe_promotion(promotion),
            None => updated,
        }
    }

    /// Applies a cross-safe promotion. This is the single entry point through which the cross-safe
    /// head — and therefore the forkchoice `safeBlockHash` — moves.
    ///
    /// Promotions may move the cross-safe head backwards, which is how an interop rewind unwinds a
    /// decision, but never below the finalized head: finalization is irreversible, so a promotion
    /// targeting a lower block is clamped to the finalized head.
    pub fn apply_cross_safe_promotion(self, promotion: CrossSafePromotion) -> Self {
        let target = promotion.target();
        let cross_safe_head = if target.block_info.number < self.finalized_head.block_info.number {
            self.finalized_head
        } else {
            target
        };

        Self::update_block_label_metric(
            Metrics::CROSS_SAFE_BLOCK_LABEL,
            cross_safe_head.block_info.number,
        );

        Self { cross_safe_head, ..self }
    }

    /// Updates a block label metric, keyed by the label.
    #[inline]
    fn update_block_label_metric(label: &'static str, number: u64) {
        kona_macros::set!(gauge, Metrics::BLOCK_LABELS, "label", label, number as f64);
    }
}

/// Specifies how to update the sync state of the engine.
///
/// There is deliberately no cross-safe field: the cross-safe head moves only through
/// [`EngineSyncState::apply_cross_safe_promotion`], so an ordinary head writer cannot advance it.
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct EngineSyncStateUpdate {
    /// Most recent block found on the p2p network
    pub unsafe_head: Option<L2BlockInfo>,
    /// Derived from L1, and known to be a completed span-batch,
    /// but not cross-verified yet.
    pub local_safe_head: Option<L2BlockInfo>,
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
    /// The sync state of the engine.
    pub sync_state: EngineSyncState,

    /// Whether or not the EL has finished syncing.
    pub el_sync_finished: bool,

    /// Track when the rollup node changes the forkchoice to restore previous
    /// known unsafe chain. e.g. Unsafe Reorg caused by Invalid span batch.
    /// This update does not retry except engine returns non-input error
    /// because engine may forgot backupUnsafeHead or backupUnsafeHead is not part
    /// of the chain.
    pub need_fcu_call_backup_unsafe_reorg: bool,
}

impl EngineState {
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
    use kona_protocol::BlockInfo;
    use metrics_exporter_prometheus::PrometheusBuilder;
    use rstest::rstest;

    impl EngineState {
        /// Set the unsafe head.
        pub fn set_unsafe_head(&mut self, unsafe_head: L2BlockInfo) {
            self.sync_state.apply_update(EngineSyncStateUpdate {
                unsafe_head: Some(unsafe_head),
                ..Default::default()
            });
        }

        /// Set the local safe head.
        pub fn set_local_safe_head(&mut self, local_safe_head: L2BlockInfo) {
            self.sync_state.apply_update(EngineSyncStateUpdate {
                local_safe_head: Some(local_safe_head),
                ..Default::default()
            });
        }

        /// Promote the cross-safe head.
        pub fn set_cross_safe_head(&mut self, cross_safe_head: L2BlockInfo) {
            self.sync_state
                .apply_cross_safe_promotion(CrossSafePromoter::new().promote(cross_safe_head));
        }

        /// Set the finalized head.
        pub fn set_finalized_head(&mut self, finalized_head: L2BlockInfo) {
            self.sync_state.apply_update(EngineSyncStateUpdate {
                finalized_head: Some(finalized_head),
                ..Default::default()
            });
        }
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
        let handle = PrometheusBuilder::new().install_recorder().unwrap();
        crate::Metrics::init();

        let mut state = EngineState::default();
        set_fn(
            &mut state,
            L2BlockInfo {
                block_info: BlockInfo { number, ..Default::default() },
                ..Default::default()
            },
        );

        assert!(handle.render().contains(
            format!("kona_node_block_labels{{label=\"{label_name}\"}} {number}").as_str()
        ));
    }
}
