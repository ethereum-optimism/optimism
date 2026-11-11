use crate::{EngineState, EngineSyncStateUpdate};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, b256};
use kona_protocol::{BlockInfo, L2BlockInfo};

/// Builder for creating test `EngineState` instances with sensible defaults.
///
/// An unset local-safe head equals the safe head, as on every chain before interop; tests of the
/// local-safe and safe labels apart set it with [`Self::with_local_safe_head`].
#[derive(Debug)]
pub struct TestEngineStateBuilder {
    unsafe_head: L2BlockInfo,
    local_safe_head: Option<L2BlockInfo>,
    safe_head: Option<L2BlockInfo>,
    finalized_head: Option<L2BlockInfo>,
    el_applied_last_forkchoice: bool,
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
            safe_head: None,
            finalized_head: None,
            el_applied_last_forkchoice: true,
            el_sync_finished: true,
        }
    }

    /// Sets the unsafe head
    pub const fn with_unsafe_head(mut self, block: L2BlockInfo) -> Self {
        self.unsafe_head = block;
        self
    }

    /// Sets the local-safe head, which otherwise follows the safe head
    pub const fn with_local_safe_head(mut self, block: L2BlockInfo) -> Self {
        self.local_safe_head = Some(block);
        self
    }

    /// Sets the safe head
    pub const fn with_safe_head(mut self, block: L2BlockInfo) -> Self {
        self.safe_head = Some(block);
        self
    }

    /// Sets the finalized head
    pub const fn with_finalized_head(mut self, block: L2BlockInfo) -> Self {
        self.finalized_head = Some(block);
        self
    }

    /// Sets whether EL sync is finished. An execution layer still syncing has not applied the last
    /// forkchoice either, so this sets both flags.
    #[allow(dead_code)]
    pub const fn with_el_sync_finished(mut self, finished: bool) -> Self {
        self.el_applied_last_forkchoice = finished;
        self.el_sync_finished = finished;
        self
    }

    /// Sets whether the execution layer applied the last forkchoice update.
    #[allow(dead_code)]
    pub const fn with_el_applied_last_forkchoice(mut self, applied: bool) -> Self {
        self.el_applied_last_forkchoice = applied;
        self
    }

    /// Builds the `EngineState`
    pub fn build(self) -> EngineState {
        let mut state = EngineState::default();

        // Set unsafe head (required)
        state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(self.unsafe_head),
            local_safe_head: Some(
                self.local_safe_head.or(self.safe_head).unwrap_or(self.unsafe_head),
            ),
            safe_head: Some(self.safe_head.unwrap_or(self.unsafe_head)),
            finalized_head: Some(self.finalized_head.unwrap_or(self.unsafe_head)),
        });

        state.el_applied_last_forkchoice = self.el_applied_last_forkchoice;
        state.el_sync_finished = self.el_sync_finished;
        state
    }
}

impl Default for TestEngineStateBuilder {
    fn default() -> Self {
        Self::new()
    }
}
