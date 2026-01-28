//! Additional support for estimating the data availability size of transactions.

/// Helper trait that allows attaching an estimated data availability size.
pub trait DataAvailabilitySized {
    /// Get the estimated data availability size of the transaction.
    ///
    /// Note: it is expected that this value will be cached internally.
    fn estimated_da_size(&self) -> u64;
}
