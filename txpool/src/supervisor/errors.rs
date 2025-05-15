use alloy_json_rpc::RpcError;
use core::error;
use op_alloy_rpc_types::InvalidInboxEntry;

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
