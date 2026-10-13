use async_trait::async_trait;
use kona_protocol::BlockInfo;
use std::fmt::Debug;

/// The L1 chain, as the derivation actor waits on it.
#[async_trait]
pub trait L1: Debug + Send + Sync {
    /// Waits for a canonical L1 head other than the last one returned, and returns it. Returns
    /// `None` once no more heads will arrive. Cancelling the wait loses no head.
    async fn next_head(&mut self) -> Option<BlockInfo>;
}
