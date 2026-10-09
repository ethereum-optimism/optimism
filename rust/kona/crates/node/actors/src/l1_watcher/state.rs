use kona_protocol::BlockInfo;

/// The latest L1 observations published by the L1 watcher.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub struct State {
    /// The L1 head block ref.
    ///
    /// The head is not guaranteed to build on the other L1 sync status fields,
    /// as the node may be in progress of resetting to adapt to a L1 reorg.
    pub head_l1: Option<BlockInfo>,
    /// The L1 safe head block ref.
    pub safe_l1: Option<BlockInfo>,
    /// The finalized L1 block ref.
    pub finalized_l1: Option<BlockInfo>,
}
