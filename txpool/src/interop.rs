//! Additional support for pooled transactions with [`TransactionInterop`]

/// Helper trait that allows attaching a [`TransactionInterop`].
pub trait MaybeInteropTransaction {
    /// Attach a [`TransactionInterop`].
    fn set_interop(&self, interop: TransactionInterop);

    /// Get attached [`TransactionInterop`] if any.
    fn interop(&self) -> Option<TransactionInterop>;

    /// Helper that sets the interop and returns the instance again
    fn with_interop(self, interop: TransactionInterop) -> Self
    where
        Self: Sized,
    {
        self.set_interop(interop);
        self
    }
}

/// Helper to keep track of cross transaction interop validity
#[derive(Debug, Clone, Default, Eq, PartialEq)]
pub struct TransactionInterop {
    /// Unix timestamp until which tx if considered valid by supervisor.
    ///
    /// If None - tx is not validated, it should be automatically revalidated by interop tracker.
    pub timeout: u64,
}

impl TransactionInterop {
    /// Checks if provided timestamp fits into tx validation window
    pub fn is_valid(&self, timestamp: u64) -> bool {
        timestamp < self.timeout
    }

    /// Checks if transaction needs revalidation based on offset
    pub fn is_stale(&self, timestamp: u64, offset: u64) -> bool {
        timestamp + offset > self.timeout
    }
}
