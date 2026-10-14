use crate::l1::{self, Child, HandleError};
use async_trait::async_trait;
use kona_protocol::BlockInfo;
use std::fmt::Debug;

/// The L1 chain, as the derivation actor follows it.
#[async_trait]
pub trait L1: Debug + Send + Sync {
    /// Waits until L1 has a canonical child of `block`, or `block` is reorged out, and returns
    /// which. Cancelling the wait loses nothing.
    async fn child_of(&self, block: BlockInfo) -> Result<Child, HandleError>;

    /// Returns the canonical child of `block` if L1 has one, [`Child::Reorged`] if `block` is
    /// reorged out, or `None` if L1 cannot tell yet.
    async fn try_child_of(&self, block: BlockInfo) -> Result<Option<Child>, HandleError>;
}

#[async_trait]
impl L1 for l1::Handle {
    async fn child_of(&self, block: BlockInfo) -> Result<Child, HandleError> {
        Self::child_of(self, block).await
    }

    async fn try_child_of(&self, block: BlockInfo) -> Result<Option<Child>, HandleError> {
        Self::try_child_of(self, block, 0).await
    }
}
