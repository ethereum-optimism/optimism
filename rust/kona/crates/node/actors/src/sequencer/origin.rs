//! The L1 origins the sequencer builds on.

use crate::origin::{self, OriginError};
use async_trait::async_trait;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::fmt::Debug;

/// The L1 origins the sequencer builds on.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait Origin: Debug + Send + Sync {
    /// Returns the L1 origin of the child of `parent`.
    async fn next(&self, parent: L2BlockInfo) -> Result<BlockInfo, OriginError>;
}

#[async_trait]
impl Origin for origin::Handle {
    async fn next(&self, parent: L2BlockInfo) -> Result<BlockInfo, OriginError> {
        Self::next(self, parent).await
    }
}
