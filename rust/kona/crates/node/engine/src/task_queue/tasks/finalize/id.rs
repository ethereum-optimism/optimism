//! Identifier for the L2 block that a [`crate::FinalizeTask`] should finalize.

use alloy_eips::BlockNumHash;

/// Identifies the L2 block to finalize.
///
/// Two semantic regimes drive finalization in kona-node:
///
/// - [`FinalizeBlockId::ByNumber`] is used when the engine's own canonical chain is the
///   authoritative source for the block at a given height (e.g. local L1-finality driven by
///   `L2Finalizer`). The number is sufficient because there is only one chain to consult.
/// - [`FinalizeBlockId::ByHash`] is used when an upstream supplies `(number, hash)` and the engine
///   must finalize the specific block identified by hash. If the engine does not have that hash,
///   the task must error rather than silently finalize the engine's canonical block at the same
///   height.
///
/// EL finalization is irreversible, so finalizing the wrong block is unrecoverable. Forcing every
/// caller to pick a variant prevents accidental hash-loss at the boundary.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FinalizeBlockId {
    /// Finalize whatever block the engine has at the given L2 block number.
    ByNumber(u64),
    /// Finalize the block matching the given `(number, hash)` pair. The task fails if the engine
    /// does not have that hash.
    ByHash(BlockNumHash),
}

impl FinalizeBlockId {
    /// Returns the L2 block number identified by this id.
    pub const fn number(&self) -> u64 {
        match self {
            Self::ByNumber(n) => *n,
            Self::ByHash(id) => id.number,
        }
    }
}
