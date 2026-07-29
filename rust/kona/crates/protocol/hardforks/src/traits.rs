//! The trait abstraction for a Hardfork.

use alloy_primitives::Bytes;

/// The trait abstraction for a Hardfork.
pub trait Hardfork {
    /// Returns the hardfork upgrade transactions as [`Bytes`].
    fn txs(&self) -> impl Iterator<Item = Bytes> + '_;

    /// Returns the additional gas required by upgrade transactions.
    ///
    /// Starting with Karst, upgrade transactions carry their own gas budget that is
    /// added to the block gas limit at the fork activation block. Pre-Karst forks
    /// return 0 (upgrade txs ran within the system tx gas allowance).
    fn upgrade_gas(&self) -> u64 {
        0
    }
}
