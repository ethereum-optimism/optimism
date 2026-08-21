use crate::{CrossSafePromoter, EngineState, EngineSyncStateUpdate, LocalSafeHead, LocalSafeOrigin};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, b256};
use kona_protocol::{BlockInfo, L2BlockInfo};

/// Builder for creating test `EngineState` instances with sensible defaults
#[derive(Debug)]
pub struct TestEngineStateBuilder {
    unsafe_head: L2BlockInfo,
    local_safe_head: Option<L2BlockInfo>,
    local_safe_origin: LocalSafeOrigin,
    cross_safe_head: Option<L2BlockInfo>,
    finalized_head: Option<L2BlockInfo>,
    el_sync_finished: bool,
}

impl TestEngineStateBuilder {
    /// Creates a new builder with default values.
    /// Default: all heads set to genesis block (block 0)
    pub fn new() -> Self {
        let genesis = L2BlockInfo {
            block_info: BlockInfo {
                number: 0,
                hash: b256!("0000000000000000000000000000000000000000000000000000000000000000"),
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            l1_origin: BlockNumHash::default(),
            seq_num: 0,
        };

        Self {
            unsafe_head: genesis,
            local_safe_head: None,
            local_safe_origin: LocalSafeOrigin::Unpaired,
            cross_safe_head: None,
            finalized_head: None,
            el_sync_finished: true,
        }
    }

    /// Sets the unsafe head
    pub const fn with_unsafe_head(mut self, block: L2BlockInfo) -> Self {
        self.unsafe_head = block;
        self
    }

    /// Sets the local-safe head
    pub const fn with_local_safe_head(mut self, block: L2BlockInfo) -> Self {
        self.local_safe_head = Some(block);
        self
    }

    /// Sets the L1 origin recorded for the local-safe head. Defaults to
    /// [`LocalSafeOrigin::Unpaired`].
    pub const fn with_local_safe_origin(mut self, origin: LocalSafeOrigin) -> Self {
        self.local_safe_origin = origin;
        self
    }

    /// Sets the cross-safe head
    pub const fn with_cross_safe_head(mut self, block: L2BlockInfo) -> Self {
        self.cross_safe_head = Some(block);
        self
    }

    /// Sets the finalized head
    pub const fn with_finalized_head(mut self, block: L2BlockInfo) -> Self {
        self.finalized_head = Some(block);
        self
    }

    /// Sets whether EL sync is finished
    #[allow(dead_code)]
    pub const fn with_el_sync_finished(mut self, finished: bool) -> Self {
        self.el_sync_finished = finished;
        self
    }

    /// Builds the `EngineState`
    pub fn build(self) -> EngineState {
        let mut state = EngineState::default();

        // Set unsafe head (required)
        state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(self.unsafe_head),
            local_safe_head: Some(LocalSafeHead::new(
                self.local_safe_head.unwrap_or(self.unsafe_head),
                self.local_safe_origin,
            )),
            finalized_head: Some(self.finalized_head.unwrap_or(self.unsafe_head)),
        });
        // Cross-safe defaults to local-safe, matching the standalone trivial feed. Tests that
        // want cross-safe to lag set it explicitly.
        state.sync_state =
            state.sync_state.apply_cross_safe_promotion(CrossSafePromoter::new().promote(
                self.cross_safe_head.or(self.local_safe_head).unwrap_or(self.unsafe_head),
            ));

        state.el_sync_finished = self.el_sync_finished;
        state
    }
}

impl Default for TestEngineStateBuilder {
    fn default() -> Self {
        Self::new()
    }
}
