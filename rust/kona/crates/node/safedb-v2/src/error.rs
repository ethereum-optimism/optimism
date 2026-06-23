//! Error type for the safe-head database.

/// Errors returned by [`SafeDbV2`](crate::SafeDbV2) implementations.
#[derive(Debug, thiserror::Error)]
pub enum SafeDbError {
    /// No matching entry was found.
    #[error("not found")]
    NotFound,
    /// A stored key/value pair did not match the expected layout.
    #[error("invalid db entry")]
    InvalidEntry,
    /// The database has been closed.
    #[error("safe db closed")]
    Closed,
    /// The safe-head database is not enabled on this host.
    #[error("safe head database not enabled")]
    NotEnabled,
    /// The requested L2 target is above the latest recorded safe head, or the database is empty.
    ///
    /// This condition is transient: the target may become available as derivation advances.
    #[error("l1 at safe head not found")]
    L1AtSafeHeadNotFound,
    /// The requested L2 target predates recorded history.
    ///
    /// This condition is permanent: the relevant records have been pruned or were never recorded.
    #[error("l1 at safe head history unavailable")]
    L1AtSafeHeadUnavailable,
    /// An error originating from the underlying key/value store.
    #[cfg(feature = "rocksdb")]
    #[error(transparent)]
    Backend(#[from] rocksdb::Error),
}
