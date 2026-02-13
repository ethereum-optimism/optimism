//! Contains the block replacement type.Add commentMore actions

use alloy_primitives::B256;
use derive_more::Display;
use kona_protocol::BlockInfo;

/// Represents a [`BlockReplacement`] event where one block replaces another.
#[derive(Debug, Clone, Copy, Display, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
#[display("replacement: {replacement}, invalidated: {invalidated}")]
pub struct BlockReplacement<T = BlockInfo> {
    /// The block that replaces the invalidated block
    pub replacement: T,
    /// Hash of the block being invalidated and replaced
    pub invalidated: B256,
}

impl<T> BlockReplacement<T> {
    /// Creates a new [`BlockReplacement`].
    pub const fn new(replacement: T, invalidated: B256) -> Self {
        Self { replacement, invalidated }
    }
}
