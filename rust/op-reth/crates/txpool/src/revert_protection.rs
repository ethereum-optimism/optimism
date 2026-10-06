//! Support for revert-protected pool transactions, as submitted via `eth_sendBundle`.

/// Helper trait that lets the payload builder tell whether a pool transaction is revert
/// protected.
///
/// A revert-protected transaction is only committed to a block when its execution succeeds. If
/// it reverts, the builder leaves it out of the block instead of including it and charging the
/// sender gas. The transaction stays in the pool and is retried in later blocks until its
/// conditional (see [`crate::conditional::MaybeConditionalTransaction`]) expires.
///
/// Like the conditional, the flag lives on the pool transaction only: a transaction that is
/// reorged out and reinjected into the pool from the block comes back without it.
pub trait MaybeRevertProtectedTransaction {
    /// Marks the transaction as revert protected (or not).
    fn set_revert_protected(&mut self, revert_protected: bool);

    /// Returns `true` if the transaction must not be included in a block when it reverts.
    fn is_revert_protected(&self) -> bool;

    /// Helper that sets the revert protection flag and returns the instance again.
    fn with_revert_protected(mut self, revert_protected: bool) -> Self
    where
        Self: Sized,
    {
        self.set_revert_protected(revert_protected);
        self
    }
}
