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

    /// An endpoint reported that its failsafe is active. Failsafe on any endpoint is a hard
    /// rejection: a message must never be accepted while any reachable endpoint is in failsafe.
    #[error("interop filter failsafe enabled")]
    FailsafeEnabled,

    /// Not enough endpoints returned a definitive verdict to satisfy the configured quorum.
    /// Produced only by the multi-endpoint aggregator; never fed back into the classifier.
    #[error("interop quorum not reached: {received} definitive responses, {required} required")]
    QuorumNotReached {
        /// Number of definitive verdicts (valid + invalid) collected.
        received: usize,
        /// Number of definitive verdicts required to decide.
        required: usize,
    },

    /// The endpoints that responded definitively disagreed on the verdict.
    /// Produced only by the multi-endpoint aggregator; never fed back into the classifier.
    #[error("interop endpoints disagreed: {valid} valid, {invalid} invalid")]
    Disagreement {
        /// Number of endpoints that returned a valid verdict.
        valid: usize,
        /// Number of endpoints that returned a definitive invalid verdict.
        invalid: usize,
    },

    /// Catch-all variant.
    #[error("interop filter server error: {0}")]
    Other(Box<dyn error::Error + Send + Sync>),
}

impl InteropTxValidatorError {
    /// Returns `true` if this error represents a definitive validation rejection from an
    /// endpoint (i.e. a verdict that counts toward quorum). Transport errors, timeouts, and
    /// the aggregator's own outcomes are not definitive rejections.
    pub const fn is_definitive_invalid(&self) -> bool {
        matches!(self, Self::InvalidEntry(_))
    }

    /// Returns `true` if this error represents an endpoint reporting that its failsafe is active.
    pub const fn is_failsafe(&self) -> bool {
        matches!(self, Self::FailsafeEnabled)
    }

    /// Returns a new instance of [`Other`](Self::Other) error variant.
    pub fn other<E>(err: E) -> Self
    where
        E: error::Error + Send + Sync + 'static,
    {
        Self::Other(Box::new(err))
    }

    /// This function will parse the error code to determine if it matches
    /// one of the known interop filter errors, and return the corresponding
    /// error variant. Otherwise, it returns a generic [`Other`](Self::Other) error.
    pub fn from_json_rpc<E>(err: RpcError<E>) -> Self
    where
        E: error::Error + Send + Sync + 'static,
    {
        // Try to extract error details from the RPC error
        if let Some(error_payload) = err.as_error_resp() {
            let code = error_payload.code as i32;

            // The filter's failsafe rejection is coded -32602, which is overloaded (it also
            // covers generic param/read fallbacks), so disambiguate on the message.
            if error_payload.message.to_ascii_lowercase().contains("failsafe") {
                return Self::FailsafeEnabled;
            }

            // Try to convert the error code to an SuperchainDAError variant
            if let Ok(invalid_entry) = SuperchainDAError::try_from(code) {
                return Self::InvalidEntry(invalid_entry);
            }
        }

        // Default to generic error
        Self::Other(Box::new(err))
    }
}
