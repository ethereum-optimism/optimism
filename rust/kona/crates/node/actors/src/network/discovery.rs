use libp2p::Multiaddr;
use std::{collections::HashSet, fmt::Debug, time::Duration};
use tokio::sync::mpsc::error::TrySendError;

/// Discovery, as the network actor uses it to ban the addresses of peers it disconnects.
pub trait Discovery: Debug + Send + Sync {
    /// Queues a ban of the discovery nodes at any of `addrs` for `duration`, without waiting, so
    /// that a busy discovery service never stops the network.
    ///
    /// Success means the ban was queued, not that it was applied. Fails with
    /// [`TrySendError::Full`] while discovery's queue is full, and with [`TrySendError::Closed`]
    /// once discovery has stopped.
    fn try_ban(
        &self,
        addrs: HashSet<Multiaddr>,
        duration: Duration,
    ) -> Result<(), TrySendError<()>>;
}
