use alloy_json_rpc::RpcError;
use core::error;
use op_alloy_rpc_types::SuperchainDAError;

/// Failures occurring during validation of inbox entries.
#[derive(thiserror::Error, Debug)]
pub enum InteropTxValidatorError {
    /// Inbox entry validation against the interop filter took longer than allowed.
    #[error("inbox entry validation timed out, timeout: {0} secs")]
    Timeout(u64),

    /// Message does not satisfy validation requirements
    #[error(transparent)]
    InvalidEntry(#[from] SuperchainDAError),

    /// The interop filter rejected the request because failsafe is enabled. Distinct from a
    /// transport/server error so it is counted as a failsafe rejection, not an infra failure.
    #[error("interop filter failsafe is enabled")]
    FailsafeEnabled,

    /// Catch-all variant.
    #[error("interop filter server error: {0}")]
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

    /// Maps a JSON-RPC error code to a known interop filter error variant, if recognized.
    /// Returns `None` for unknown codes (the caller falls back to [`Other`](Self::Other)).
    ///
    /// Failsafe shares the [`SuperchainDAError`] code space (`-320602`) but is an operational
    /// state, not a data-availability rejection, so it maps to its own variant.
    fn from_error_code(code: i32) -> Option<Self> {
        match SuperchainDAError::try_from(code) {
            Ok(SuperchainDAError::Failsafe) => Some(Self::FailsafeEnabled),
            Ok(invalid_entry) => Some(Self::InvalidEntry(invalid_entry)),
            Err(_) => None,
        }
    }

    /// This function will parse the error code to determine if it matches
    /// one of the known interop filter errors, and return the corresponding
    /// error variant. Otherwise, it returns a generic [`Other`](Self::Other) error.
    pub fn from_json_rpc<E>(err: RpcError<E>) -> Self
    where
        E: error::Error + Send + Sync + 'static,
    {
        // Try to extract a known error variant from the RPC error code.
        if let Some(error_payload) = err.as_error_resp() &&
            let Some(known) = Self::from_error_code(error_payload.code as i32)
        {
            return known;
        }

        // Default to generic error
        Self::Other(Box::new(err))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn failsafe_code_maps_to_failsafe_variant() {
        assert!(matches!(
            InteropTxValidatorError::from_error_code(SuperchainDAError::Failsafe as i32),
            Some(InteropTxValidatorError::FailsafeEnabled)
        ));
    }

    #[test]
    fn da_error_codes_map_to_invalid_entry() {
        assert!(matches!(
            InteropTxValidatorError::from_error_code(-320600),
            Some(InteropTxValidatorError::InvalidEntry(SuperchainDAError::ConflictingData))
        ));
    }

    #[test]
    fn unknown_code_is_unrecognized() {
        assert!(InteropTxValidatorError::from_error_code(-32000).is_none());
    }
}
