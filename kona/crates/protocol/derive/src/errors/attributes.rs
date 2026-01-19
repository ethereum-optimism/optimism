//! Error types for kona's attributes builder.

use alloc::string::String;
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use thiserror::Error;

/// An [`AttributesBuilder`] Error.
///
/// [`AttributesBuilder`]: crate::traits::AttributesBuilder
#[derive(Error, Clone, Debug, PartialEq, Eq)]
pub enum BuilderError {
    /// Mismatched blocks.
    #[error("Block mismatch. Expected {0:?}, got {1:?}")]
    BlockMismatch(BlockNumHash, BlockNumHash),
    /// Mismatched blocks for the start of an Epoch.
    #[error("Block mismatch on epoch reset. Expected {0:?}, got {1:?}")]
    BlockMismatchEpochReset(BlockNumHash, BlockNumHash, B256),
    /// [`SystemConfig`] update failed.
    ///
    /// [`SystemConfig`]: kona_genesis::SystemConfig
    #[error("System config update failed")]
    SystemConfigUpdate,
    /// Broken time invariant between L2 and L1.
    #[error(
        "Time invariant broken. L1 origin: {0:?} | Next L2 time: {1} | L1 block: {2:?} | L1 timestamp {3:?}"
    )]
    BrokenTimeInvariant(BlockNumHash, u64, BlockNumHash, u64),
    /// Attributes unavailable.
    #[error("Attributes unavailable")]
    AttributesUnavailable,
    /// A custom error.
    #[error("Error in attributes builder: {0}")]
    Custom(String),
}
