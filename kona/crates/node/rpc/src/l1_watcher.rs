use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use tokio::sync::oneshot::Sender;

/// The L1 watcher state accessible from RPC queries.
#[derive(Debug, Clone)]
pub struct L1State {
    /// The current L1 block.
    ///
    /// This is the L1 block that the derivation process is last idled at.
    /// This may not be fully derived into L2 data yet.
    /// The safe L2 blocks were produced/included fully from the L1 chain up to _but excluding_
    /// this L1 block. If the node is synced, this matches the `head_l1`, minus the verifier
    /// confirmation distance.
    pub current_l1: Option<BlockInfo>,
    /// The current L1 finalized block.
    ///
    /// This is a legacy sync-status attribute. This is deprecated.
    /// A previous version of the L1 finalization-signal was updated only after the block was
    /// retrieved by number. This attribute just matches `finalized_l1` now.
    pub current_l1_finalized: Option<BlockInfo>,
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

/// A sender for L1 watcher queries.
pub type L1WatcherQuerySender = tokio::sync::mpsc::Sender<L1WatcherQueries>;

/// The inbound queries to the L1 watcher.
#[derive(Debug)]
pub enum L1WatcherQueries {
    /// Get the rollup config from the L1 watcher.
    Config(Sender<RollupConfig>),
    /// Get a complete view of the L1 state.
    L1State(Sender<L1State>),
}
