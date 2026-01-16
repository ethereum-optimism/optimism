//! The internal state of the engine controller.

use crate::Metrics;
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
/// 1. **Unsafe** - Most recent blocks from P2P network (unverified)
/// 2. **Cross-unsafe** - Unsafe blocks with cross-layer verification
/// 3. **Local-safe** - Derived from L1 data, completed span-batch
/// 4. **Safe** - Cross-verified with safe L1 dependencies
/// 5. **Finalized** - Derived from finalized L1 data only
///
/// See the [OP Stack specifications](https://specs.optimism.io) for detailed safety definitions.
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq)]
pub struct EngineSyncState {
    /// Most recent block found on the P2P network (lowest safety level).
    unsafe_head: L2BlockInfo,
    /// Cross-verified unsafe head (equal to unsafe_head pre-interop).
    cross_unsafe_head: L2BlockInfo,
    /// Derived from L1 data as a completed span-batch, but not yet cross-verified.
    local_safe_head: L2BlockInfo,
    /// Derived from L1 data and cross-verified to have safe L1 dependencies.
    safe_head: L2BlockInfo,
    /// Derived from finalized L1 data with only finalized dependencies (highest safety level).
    finalized_head: L2BlockInfo,
}

impl EngineSyncState {
    /// Returns the current unsafe head.
    pub const fn unsafe_head(&self) -> L2BlockInfo {
        self.unsafe_head
    }

    /// Returns the current cross-verified unsafe head.
    pub const fn cross_unsafe_head(&self) -> L2BlockInfo {
        self.cross_unsafe_head
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

    /// Applies the update to the provided sync state, using the current state values if the update
    /// is not specified. Returns the new sync state.
    pub fn apply_update(self, sync_state_update: EngineSyncStateUpdate) -> Self {
        if let Some(unsafe_head) = sync_state_update.unsafe_head {
            Self::update_block_label_metric(
                Metrics::UNSAFE_BLOCK_LABEL,
                unsafe_head.block_info.number,
            );
        }
        if let Some(cross_unsafe_head) = sync_state_update.cross_unsafe_head {
            Self::update_block_label_metric(
                Metrics::CROSS_UNSAFE_BLOCK_LABEL,
                cross_unsafe_head.block_info.number,
            );
        }
        if let Some(local_safe_head) = sync_state_update.local_safe_head {
            Self::update_block_label_metric(
                Metrics::LOCAL_SAFE_BLOCK_LABEL,
                local_safe_head.block_info.number,
            );
        }
        if let Some(safe_head) = sync_state_update.safe_head {
            Self::update_block_label_metric(Metrics::SAFE_BLOCK_LABEL, safe_head.block_info.number);
        }
        if let Some(finalized_head) = sync_state_update.finalized_head {
            Self::update_block_label_metric(
                Metrics::FINALIZED_BLOCK_LABEL,
                finalized_head.block_info.number,
            );
        }

        Self {
            unsafe_head: sync_state_update.unsafe_head.unwrap_or(self.unsafe_head),
            cross_unsafe_head: sync_state_update
                .cross_unsafe_head
                .unwrap_or(self.cross_unsafe_head),
            local_safe_head: sync_state_update.local_safe_head.unwrap_or(self.local_safe_head),
            safe_head: sync_state_update.safe_head.unwrap_or(self.safe_head),
            finalized_head: sync_state_update.finalized_head.unwrap_or(self.finalized_head),
        }
    }

    /// Updates a block label metric, keyed by the label.
    #[inline]
    fn update_block_label_metric(label: &'static str, number: u64) {
        kona_macros::set!(gauge, Metrics::BLOCK_LABELS, "label", label, number as f64);
    }
}

/// Specifies how to update the sync state of the engine.
#[derive(Default, Debug, Copy, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct EngineSyncStateUpdate {
    /// Most recent block found on the p2p network
    pub unsafe_head: Option<L2BlockInfo>,
    /// Cross-verified unsafe head, always equal to the unsafe head pre-interop
    pub cross_unsafe_head: Option<L2BlockInfo>,
    /// Derived from L1, and known to be a completed span-batch,
    /// but not cross-verified yet.
    pub local_safe_head: Option<L2BlockInfo>,
    /// Derived from L1 and cross-verified to have cross-safe dependencies.
    pub safe_head: Option<L2BlockInfo>,
    /// Derived from finalized L1 data,
    /// and cross-verified to only have finalized dependencies.
    pub finalized_head: Option<L2BlockInfo>,
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
    /// is ahead of the safe head. When the two are equal, consolidation isn't
    /// required and the [`crate::BuildTask`] can be used to build the block.
    ///
    /// [Consolidation]: https://specs.optimism.io/protocol/derivation.html#l1-consolidation-payload-attributes-matching
    pub fn needs_consolidation(&self) -> bool {
        self.sync_state.safe_head() != self.sync_state.unsafe_head()
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::Metrics;
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

        /// Set the cross-verified unsafe head.
        pub fn set_cross_unsafe_head(&mut self, cross_unsafe_head: L2BlockInfo) {
            self.sync_state.apply_update(EngineSyncStateUpdate {
                cross_unsafe_head: Some(cross_unsafe_head),
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

        /// Set the safe head.
        pub fn set_safe_head(&mut self, safe_head: L2BlockInfo) {
            self.sync_state.apply_update(EngineSyncStateUpdate {
                safe_head: Some(safe_head),
                ..Default::default()
            });
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
    #[case::set_cross_unsafe(
        EngineState::set_cross_unsafe_head,
        Metrics::CROSS_UNSAFE_BLOCK_LABEL,
        2
    )]
    #[case::set_local_safe(EngineState::set_local_safe_head, Metrics::LOCAL_SAFE_BLOCK_LABEL, 3)]
    #[case::set_safe_head(EngineState::set_safe_head, Metrics::SAFE_BLOCK_LABEL, 4)]
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
