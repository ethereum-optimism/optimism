use alloy_json_rpc::RpcError;
use core::error;
use derive_more;

/// Supervisor protocol error codes.
///
/// Specs: <https://specs.optimism.io/interop/supervisor.html#protocol-specific-error-codes>
#[derive(thiserror::Error, Debug, Clone, Copy, PartialEq, Eq, derive_more::TryFrom)]
#[repr(i64)]
#[try_from(repr)]
pub enum InvalidInboxEntry {
    // -3204XX DEADLINE_EXCEEDED errors
    /// Happens when a chain database is not initialized yet.
    #[error("chain database is not initialized")]
    UninitializedChainDatabase = -320400,

    // -3205XX NOT_FOUND errors
    /// Happens when we try to retrieve data that is not available (pruned).
    /// It may also happen if we erroneously skip data, that was not considered a conflict, if the
    /// DB is corrupted.
    #[error("data was skipped or pruned and is not available")]
    SkippedData = -320500,

    /// Happens when a chain is unknown, not in the dependency set.
    #[error("unsupported chain id")]
    UnknownChain = -320501,

    // -3206XX ALREADY_EXISTS errors
    /// Happens when we know for sure that there is different canonical data.
    #[error("conflicting data exists in the database")]
    ConflictingData = -320600,

    /// Happens when data is accepted as compatible, but did not change anything.
    /// This happens when a node is deriving an L2 block we already know of being
    /// derived from the given source,
    /// but without path to skip forward to newer source blocks without doing the known
    /// derivation work first.
    #[error("data is already known and didn't change anything")]
    IneffectiveData = -320601,

    // -3209XX FAILED_PRECONDITION errors
    /// Happens when you try to add data to the DB, but it does not actually fit onto
    /// the latest data.
    /// (by being too old or new).
    #[error("data is out of order (too old or new)")]
    OutOfOrder = -320900,

    /// Happens when we know for sure that a replacement block is needed before progress
    /// can be made.
    #[error("waiting for replacement block before progress can be made")]
    AwaitingReplacement = -320901,

    // -3211XX OUT_OF_RANGE errors
    /// Happens when data is accessed, but access is not allowed, because of a limited
    /// scope.
    /// E.g. when limiting scope to L2 blocks derived from a specific subset of the L1
    /// chain.
    #[error("data access not allowed due to limited scope")]
    OutOfScope = -321100,

    // -3212XX UNIMPLEMENTED errors
    /// Happens when you try to get the previous block of the first block.
    /// E.g. when trying to determine the previous source block for the first L1 block
    /// in the database.
    #[error("cannot get parent of first block in database")]
    NoParentForFirstBlock = -321200,

    // -3214XX UNAVAILABLE errors
    /// Happens when data is just not yet available.
    #[error("data is not yet available (from the future)")]
    FutureData = -321401,

    // -3215XX DATA_LOSS errors
    /// Happens when we search the DB, know the data may be there, but is not (e.g.
    /// different revision).
    #[error("data may exist but was not found (possibly different revision)")]
    MissedData = -321500,

    /// Happens when the underlying DB has some I/O issue.
    #[error("underlying database has I/O issues or is corrupted")]
    DataCorruption = -321501,
}

/// Failures occurring during validation of inbox entries.
#[derive(thiserror::Error, Debug)]
pub enum InteropTxValidatorError {
    /// Inbox entry validation against the Supervisor took longer than allowed.
    #[error("inbox entry validation timed out, timeout: {0} secs")]
    Timeout(u64),

    /// Message does not satisfy validation requirements
    #[error(transparent)]
    InvalidEntry(#[from] InvalidInboxEntry),

    /// Catch-all variant.
    #[error("supervisor server error: {0}")]
    Other(Box<dyn error::Error + Send + Sync>),
}

impl InteropTxValidatorError {
    /// Returns a new instance of [`Other`](Self::Other) error variant.
    pub fn other<E>(err: E) -> Self
    where
        E: error::Error + Send + Sync + 'static,
    {
        Self::Other(Box::new(err))
    }

    /// This function will parse the error code to determine if it matches
    /// one of the known Supervisor errors, and return the corresponding
    /// error variant. Otherwise, it returns a generic [`Other`](Self::Other) error.
    pub fn from_json_rpc<E>(err: RpcError<E>) -> Self
    where
        E: error::Error + Send + Sync + 'static,
    {
        // Try to extract error details from the RPC error
        if let Some(error_payload) = err.as_error_resp() {
            let code = error_payload.code;

            // Try to convert the error code to an InvalidInboxEntry variant
            if let Ok(invalid_entry) = InvalidInboxEntry::try_from(code) {
                return Self::InvalidEntry(invalid_entry);
            }
        }

        // Default to generic error
        Self::Other(Box::new(err))
    }
}
