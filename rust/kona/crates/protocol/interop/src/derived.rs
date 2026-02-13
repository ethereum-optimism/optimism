//! Contains derived types for interop.

use alloy_eips::eip1898::BlockNumHash;
use derive_more::Display;
use kona_protocol::BlockInfo;

/// A pair of [`BlockNumHash`]s representing a derivation relationship between two blocks.
///
/// The [`DerivedIdPair`] links a source block (L1) to a derived block (L2) where the derived block
/// is derived from the source block.
///
/// - `source`: The [`BlockNumHash`] of the source (L1) block.
/// - `derived`: The [`BlockNumHash`] of the derived (L2) block.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct DerivedIdPair {
    /// The [`BlockNumHash`] of the source (L1) block.
    pub source: BlockNumHash,
    /// The [`BlockNumHash`] of the derived (L2) block.
    pub derived: BlockNumHash,
}

/// A pair of [`BlockInfo`]s representing a derivation relationship between two blocks.
///
/// The [`DerivedRefPair`] contains full block information for both the source (L1) and
/// derived (L2) blocks, where the derived block is produced from the source block.
///
/// - `source`: The [`BlockInfo`] of the source (L1) block.
/// - `derived`: The [`BlockInfo`] of the derived (L2) block.
// See the interop control flow specification: https://github.com/ethereum-optimism/specs/blob/main/specs/interop/managed-node.md
#[derive(Debug, Clone, Copy, Display, PartialEq, Eq)]
#[display("source: {source}, derived: {derived}")]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct DerivedRefPair {
    /// The [`BlockInfo`] of the source (L1) block.
    pub source: BlockInfo,
    /// The [`BlockInfo`] of the derived (L2) block.
    pub derived: BlockInfo,
}
