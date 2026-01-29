//! The trait abstraction for a Hardfork.

use alloy_primitives::Bytes;

/// The trait abstraction for a Hardfork.
pub trait Hardfork {
    /// Returns the hardfork upgrade transactions as [`Bytes`].
    fn txs(&self) -> impl Iterator<Item = Bytes> + '_;
}
