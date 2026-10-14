//! The L1 origins the sequencer builds on.

use crate::origin::{self, OriginError};
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::fmt::Debug;

/// The L1 origins the sequencer builds on.
#[cfg_attr(test, mockall::automock)]
pub trait Origin: Debug + Send + Sync {
    /// Returns the L1 origin of the child of `parent`, without waiting.
    fn next(&self, parent: L2BlockInfo) -> Result<BlockInfo, OriginError>;
}

impl Origin for origin::Handle {
    fn next(&self, parent: L2BlockInfo) -> Result<BlockInfo, OriginError> {
        Self::next(self, parent)
    }
}
