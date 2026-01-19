//! Contains the managed node event.

use crate::{BlockReplacement, DerivedRefPair};
use alloc::{format, string::String, vec::Vec};
use derive_more::Constructor;
use kona_protocol::BlockInfo;

/// Event sent by the node to the supervisor to share updates.
///
/// This struct is used to communicate various events that occur within the node.
/// At least one of the fields will be `Some`, and the rest will be `None`.
///
/// See: <https://specs.optimism.io/interop/managed-mode.html#node---supervisor>
#[derive(Debug, Clone, Default, PartialEq, Eq, Constructor)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct ManagedEvent {
    /// This is emitted when the node has determined that it needs a reset.
    /// It tells the supervisor to send the interop_reset event with the
    /// required parameters.
    pub reset: Option<String>,

    /// New L2 unsafe block was processed, updating local-unsafe head.
    pub unsafe_block: Option<BlockInfo>,

    /// Signals that an L2 block is considered local-safe.
    pub derivation_update: Option<DerivedRefPair>,

    /// Emitted when no more L1 Blocks are available.
    /// Ready to take new L1 blocks from supervisor.
    pub exhaust_l1: Option<DerivedRefPair>,

    /// Emitted when a block gets replaced for any reason.
    pub replace_block: Option<BlockReplacement>,

    /// Signals that an L2 block is now local-safe because of the given L1 traversal.
    /// This would be accompanied with [`Self::derivation_update`].
    pub derivation_origin_update: Option<BlockInfo>,
}

impl core::fmt::Display for ManagedEvent {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        let mut parts = Vec::new();
        if let Some(ref reset) = self.reset {
            parts.push(format!("reset: {reset}"));
        }
        if let Some(ref block) = self.unsafe_block {
            parts.push(format!("unsafe_block: {block}"));
        }
        if let Some(ref pair) = self.derivation_update {
            parts.push(format!("derivation_update: {pair}"));
        }
        if let Some(ref pair) = self.exhaust_l1 {
            parts.push(format!("exhaust_l1: {pair}"));
        }
        if let Some(ref replacement) = self.replace_block {
            parts.push(format!("replace_block: {replacement}"));
        }
        if let Some(ref origin) = self.derivation_origin_update {
            parts.push(format!("derivation_origin_update: {origin}"));
        }

        if parts.is_empty() { write!(f, "none") } else { write!(f, "{}", parts.join(", ")) }
    }
}
