use async_trait::async_trait;
use libp2p::Multiaddr;
use std::{collections::HashSet, fmt::Debug, time::Duration};

/// Discovery, as the network actor uses it to ban the addresses of peers it disconnects.
#[async_trait]
pub trait Discovery: Debug + Send + Sync {
    /// An error queuing a ban.
    type Error: std::error::Error + Send + Sync + 'static;

    /// Queues a ban of the discovery nodes at any of `addrs` for `duration`.
    async fn ban(&self, addrs: HashSet<Multiaddr>, duration: Duration) -> Result<(), Self::Error>;
}
