use alloy_json_rpc::RpcError;
use core::error;
use op_alloy_rpc_types::SuperchainDAError;

/// Dedicated JSON-RPC error code emitted by the op-interop-filter server when its failsafe is
/// active. Detection relies on this code; the filter must emit it (a prerequisite server change).
const FAILSAFE_ENABLED_CODE: i32 = -320602;

/// Standard JSON-RPC "internal error" code. Treated as a non-response (transport-class), not a
/// definitive rejection, because it signals a filter-internal failure rather than a verdict.
const JSON_RPC_INTERNAL_ERROR_CODE: i32 = -32603;

/// Filter codes meaning "this node does not have the data yet" (it is out of sync), as opposed to
/// a verdict that the entry is invalid. These are soft, non-response failures: a single out-of-sync
/// node is ignored as long as the quorum is met by other nodes.
///
/// `FUTURE_DATA_CODE` is also a [`SuperchainDAError`] variant, so this check must run before the
/// `SuperchainDAError` mapping; `UNINITIALIZED_CODE` is not in that enum and needs the raw check.
const FUTURE_DATA_CODE: i32 = -321401;
const UNINITIALIZED_CODE: i32 = -320400;

/// Failures occurring during validation of inbox entries.
#[derive(thiserror::Error, Debug)]
pub enum InteropTxValidatorError {
    /// Inbox entry validation against the interop filter took longer than allowed.
    #[error("inbox entry validation timed out, timeout: {0} secs")]
    Timeout(u64),

    /// Message does not satisfy validation requirements
    #[error(transparent)]
    InvalidEntry(#[from] SuperchainDAError),

    /// The filter returned a JSON-RPC error response rejecting the access list that is not one of
    /// the known [`SuperchainDAError`] codes (e.g. a generic `-32602` "failed to parse access
    /// entry"). This is still a definitive rejection — only transport failures are non-responses.
    /// The original code and message are preserved so the real reason propagates to the caller.
    #[error("interop filter rejected access list (code {code}): {message}")]
    Rejected {
        /// The JSON-RPC error code returned by the filter.
        code: i32,
        /// The JSON-RPC error message returned by the filter.
        message: String,
    },

    /// An endpoint reported that its failsafe is active. Failsafe on any endpoint is a hard
    /// rejection: a message must never be accepted while any reachable endpoint is in failsafe.
    #[error("interop filter failsafe enabled")]
    FailsafeEnabled,

    /// An endpoint does not yet have the data needed to decide (it is out of sync). This is a soft
    /// failure, not a verdict: it counts as a non-response, so a single out-of-sync node is
    /// ignored as long as the quorum is met by other nodes. It is neither a definitive invalid nor
    /// a failsafe.
    #[error("interop filter data not yet available (code {code})")]
    DataUnavailable {
        /// The JSON-RPC error code indicating the data is not yet available.
        code: i32,
    },

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
        matches!(self, Self::InvalidEntry(_) | Self::Rejected { .. })
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

    /// Classifies a JSON-RPC error from the interop filter.
    ///
    /// Any error *response* from the filter is a definitive rejection verdict — the filter
    /// evaluated the request and refused it — except a JSON-RPC internal error
    /// (`-32603`), which signals a filter-internal failure rather than a verdict. Only
    /// transport-level failures (no error response at all) are non-responses.
    ///
    /// - failsafe code (`-320602`) → [`FailsafeEnabled`](Self::FailsafeEnabled)
    /// - data-not-yet-available code (`-321401` / `-320400`) →
    ///   [`DataUnavailable`](Self::DataUnavailable) (soft, non-response)
    /// - known [`SuperchainDAError`] code → [`InvalidEntry`](Self::InvalidEntry)
    /// - internal error (`-32603`) → [`Other`](Self::Other) (non-response)
    /// - any other error response → [`Rejected`](Self::Rejected), preserving code + message
    /// - no error response (transport) → [`Other`](Self::Other) (non-response)
    pub fn from_json_rpc<E>(err: RpcError<E>) -> Self
    where
        E: error::Error + Send + Sync + 'static,
    {
        let Some(error_payload) = err.as_error_resp() else {
            // No error response: a transport-level failure, treated as a non-response.
            return Self::Other(Box::new(err));
        };
        let code = error_payload.code as i32;

        // The filter emits a dedicated code for failsafe rejections. Detect by code only;
        // never match on the message text.
        if code == FAILSAFE_ENABLED_CODE {
            return Self::FailsafeEnabled;
        }

        // An out-of-sync node that does not have the data yet is a soft, non-response failure.
        // Must precede the `SuperchainDAError` mapping because `FUTURE_DATA_CODE` is one of its
        // variants but should be treated as soft, not as a definitive invalid.
        if code == FUTURE_DATA_CODE || code == UNINITIALIZED_CODE {
            return Self::DataUnavailable { code };
        }

        // Known structured rejection codes map to the typed variant.
        if let Ok(invalid_entry) = SuperchainDAError::try_from(code) {
            return Self::InvalidEntry(invalid_entry);
        }

        // A JSON-RPC internal error is a filter-internal failure, not a verdict — non-response.
        if code == JSON_RPC_INTERNAL_ERROR_CODE {
            return Self::Other(Box::new(err));
        }

        // Any other error response is a definitive rejection; preserve the real code + message.
        Self::Rejected { code, message: error_payload.message.to_string() }
    }
}
