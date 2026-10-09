use async_trait::async_trait;
use kona_protocol::BlockInfo;
use std::fmt::Debug;

/// Receives L1 observations for derivation.
#[cfg_attr(test, mockall::automock(type Error = std::io::Error;))]
#[async_trait]
pub trait Derivation: Debug + Send + Sync {
    /// An error delivering an observation.
    type Error: std::error::Error + Send + Sync + 'static;

    /// Queues a finalized L1 block.
    ///
    /// Success means the observation was queued, not that it was processed.
    async fn send_finalized_l1_block(&self, block: BlockInfo) -> Result<(), Self::Error>;

    /// Queues the latest L1 head.
    ///
    /// Success means the observation was queued, not that it was processed.
    async fn send_new_l1_head(&self, block: BlockInfo) -> Result<(), Self::Error>;
}
