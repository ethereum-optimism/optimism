use kona_genesis::RollupConfig;
use tokio::sync::oneshot::Sender;

/// A sender for L1 watcher queries.
pub type L1WatcherQuerySender = tokio::sync::mpsc::Sender<L1WatcherQueries>;

/// The inbound queries to the L1 watcher.
#[derive(Debug)]
pub enum L1WatcherQueries {
    /// Get the rollup config from the L1 watcher.
    Config(Sender<RollupConfig>),
}
