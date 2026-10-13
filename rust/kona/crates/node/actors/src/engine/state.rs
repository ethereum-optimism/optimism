//! The engine state that the engine actor publishes.

use alloy_rpc_types_engine::ForkchoiceState;
use kona_engine::Metrics;
use kona_protocol::{BlockInfo, L2BlockInfo};

/// The engine's view of the L2 chain: its heads at each safety level.
///
/// Tracks block progression through various stages of verification and finalization,
/// from initial unsafe blocks received via P2P to fully finalized blocks derived from
/// finalized L1 data. Each level represents increasing confidence in the block's validity.
///
/// # Safety Levels
///
/// The state tracks blocks at different safety levels, listed from least to most safe:
///
/// 1. **Unsafe** - Most recent blocks from P2P network (unverified)
/// 2. **Local-safe** - Derived from L1 data, completed span-batch
/// 3. **Safe** - Cross-verified with safe L1 dependencies
/// 4. **Finalized** - Derived from finalized L1 data only
///
/// See the [OP Stack specifications](https://specs.optimism.io) for detailed safety definitions.
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq)]
pub struct EngineState {
    /// Most recent block found on the P2P network (lowest safety level).
    unsafe_head: L2BlockInfo,
    /// Derived from L1 data as a completed span-batch, but not yet cross-verified.
    local_safe_head: L2BlockInfo,
    /// Derived from L1 data and cross-verified to have safe L1 dependencies.
    safe_head: L2BlockInfo,
    /// Derived from finalized L1 data with only finalized dependencies (highest safety level).
    finalized_head: L2BlockInfo,
    /// The L1 block the safe head was derived from, if a consolidation made it safe. Unknown after
    /// a reset, or once a delegated safe head replaces it.
    safe_head_source: Option<BlockInfo>,
}

impl EngineState {
    /// Returns the current unsafe head.
    pub const fn unsafe_head(&self) -> L2BlockInfo {
        self.unsafe_head
    }

    /// Returns the current local safe head.
    pub const fn local_safe_head(&self) -> L2BlockInfo {
        self.local_safe_head
    }

    /// Returns the current safe head.
    pub const fn safe_head(&self) -> L2BlockInfo {
        self.safe_head
    }

    /// Returns the current finalized head.
    pub const fn finalized_head(&self) -> L2BlockInfo {
        self.finalized_head
    }

    /// Returns the L1 block the safe head was derived from, if known.
    pub const fn safe_head_source(&self) -> Option<BlockInfo> {
        self.safe_head_source
    }

    /// Returns the state with the given source for its safe head.
    #[must_use]
    pub const fn with_safe_head_source(self, safe_head_source: Option<BlockInfo>) -> Self {
        Self { safe_head_source, ..self }
    }

    /// Creates a `ForkchoiceState`
    ///
    /// - `head_block` = `unsafe_head`
    /// - `safe_block` = `safe_head`
    /// - `finalized_block` = `finalized_head`
    ///
    /// If the block info is not yet available, the default values are used.
    pub const fn create_forkchoice_state(&self) -> ForkchoiceState {
        ForkchoiceState {
            head_block_hash: self.unsafe_head.hash(),
            safe_block_hash: self.safe_head.hash(),
            finalized_block_hash: self.finalized_head.hash(),
        }
    }

    /// Returns whether any head is set, as it is once the initial forkchoice state is sent.
    pub fn has_heads(&self) -> bool {
        [self.unsafe_head, self.local_safe_head, self.safe_head, self.finalized_head] !=
            [L2BlockInfo::default(); 4]
    }

    /// Applies the update to the heads, keeping each head the update does not specify. Returns the
    /// new state, which forgets the safe head's source if the safe head changes.
    #[must_use]
    pub fn apply_update(self, update: HeadsUpdate) -> Self {
        if let Some(unsafe_head) = update.unsafe_head {
            Self::update_block_label_metric(
                Metrics::UNSAFE_BLOCK_LABEL,
                unsafe_head.block_info.number,
            );
        }
        if let Some(local_safe_head) = update.local_safe_head {
            Self::update_block_label_metric(
                Metrics::LOCAL_SAFE_BLOCK_LABEL,
                local_safe_head.block_info.number,
            );
        }
        if let Some(safe_head) = update.safe_head {
            Self::update_block_label_metric(Metrics::SAFE_BLOCK_LABEL, safe_head.block_info.number);
        }
        if let Some(finalized_head) = update.finalized_head {
            Self::update_block_label_metric(
                Metrics::FINALIZED_BLOCK_LABEL,
                finalized_head.block_info.number,
            );
        }

        Self {
            unsafe_head: update.unsafe_head.unwrap_or(self.unsafe_head),
            local_safe_head: update.local_safe_head.unwrap_or(self.local_safe_head),
            safe_head: update.safe_head.unwrap_or(self.safe_head),
            finalized_head: update.finalized_head.unwrap_or(self.finalized_head),
            safe_head_source: if update.safe_head.is_some_and(|head| head != self.safe_head) {
                None
            } else {
                self.safe_head_source
            },
        }
    }

    /// Updates a block label metric, keyed by the label.
    #[inline]
    fn update_block_label_metric(label: &'static str, number: u64) {
        metrics::gauge!(Metrics::BLOCK_LABELS, "label" => label).set(number as f64);
    }
}

/// Specifies how to update the heads of the [`EngineState`].
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq)]
pub struct HeadsUpdate {
    /// Most recent block found on the p2p network
    pub unsafe_head: Option<L2BlockInfo>,
    /// Derived from L1, and known to be a completed span-batch,
    /// but not cross-verified yet.
    pub local_safe_head: Option<L2BlockInfo>,
    /// Derived from L1 and cross-verified to have cross-safe dependencies.
    pub safe_head: Option<L2BlockInfo>,
    /// Derived from finalized L1 data,
    /// and cross-verified to only have finalized dependencies.
    pub finalized_head: Option<L2BlockInfo>,
}

#[cfg(test)]
mod test {
    use super::*;
    use kona_protocol::BlockInfo;
    use metrics_exporter_prometheus::PrometheusBuilder;
    use rstest::rstest;

    impl EngineState {
        /// Set the unsafe head.
        pub fn set_unsafe_head(&mut self, unsafe_head: L2BlockInfo) {
            *self = self
                .apply_update(HeadsUpdate { unsafe_head: Some(unsafe_head), ..Default::default() });
        }

        /// Set the local safe head.
        pub fn set_local_safe_head(&mut self, local_safe_head: L2BlockInfo) {
            *self = self.apply_update(HeadsUpdate {
                local_safe_head: Some(local_safe_head),
                ..Default::default()
            });
        }

        /// Set the safe head.
        pub fn set_safe_head(&mut self, safe_head: L2BlockInfo) {
            *self =
                self.apply_update(HeadsUpdate { safe_head: Some(safe_head), ..Default::default() });
        }

        /// Set the finalized head.
        pub fn set_finalized_head(&mut self, finalized_head: L2BlockInfo) {
            *self = self.apply_update(HeadsUpdate {
                finalized_head: Some(finalized_head),
                ..Default::default()
            });
        }
    }

    /// The safe head's source lasts until the safe head changes.
    #[test]
    fn safe_head_source_lasts_until_the_safe_head_changes() {
        let safe_head = |number| L2BlockInfo {
            block_info: BlockInfo { number, ..Default::default() },
            ..Default::default()
        };
        let source = Some(BlockInfo { number: 7, ..Default::default() });
        let state = EngineState::default()
            .apply_update(HeadsUpdate { safe_head: Some(safe_head(1)), ..Default::default() })
            .with_safe_head_source(source);

        let unchanged = state.apply_update(HeadsUpdate {
            unsafe_head: Some(safe_head(2)),
            safe_head: Some(safe_head(1)),
            ..Default::default()
        });
        assert_eq!(unchanged.safe_head_source(), source);
        let changed =
            state.apply_update(HeadsUpdate { safe_head: Some(safe_head(2)), ..Default::default() });
        assert_eq!(changed.safe_head_source(), None);
    }

    #[rstest]
    #[case::set_unsafe(EngineState::set_unsafe_head, Metrics::UNSAFE_BLOCK_LABEL, 1)]
    #[case::set_local_safe(EngineState::set_local_safe_head, Metrics::LOCAL_SAFE_BLOCK_LABEL, 3)]
    #[case::set_safe_head(EngineState::set_safe_head, Metrics::SAFE_BLOCK_LABEL, 4)]
    #[case::set_finalized_head(EngineState::set_finalized_head, Metrics::FINALIZED_BLOCK_LABEL, 5)]
    fn test_chain_label_metrics(
        #[case] set_fn: impl Fn(&mut EngineState, L2BlockInfo),
        #[case] label_name: &str,
        #[case] number: u64,
    ) {
        let handle = PrometheusBuilder::new().install_recorder().unwrap();
        Metrics::init();

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
