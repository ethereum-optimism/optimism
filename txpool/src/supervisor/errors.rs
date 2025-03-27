//! Error types for the `kona-interop` crate.
// Source: https://github.com/op-rs/kona
// Copyright © 2023 kona contributors Copyright © 2024 Optimism
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
// associated documentation files (the “Software”), to deal in the Software without restriction,
// including without limitation the rights to use, copy, modify, merge, publish, distribute,
// sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or
// substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
// NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
use core::error;
use op_alloy_consensus::interop::SafetyLevel;

/// Derived from op-supervisor
// todo: rm once resolved <https://github.com/ethereum-optimism/optimism/issues/14603>
const UNKNOWN_CHAIN_MSG: &str = "unknown chain: ";
/// Derived from [op-supervisor](https://github.com/ethereum-optimism/optimism/blob/4ba2eb00eafc3d7de2c8ceb6fd83913a8c0a2c0d/op-supervisor/supervisor/backend/backend.go#L479)
// todo: rm once resolved <https://github.com/ethereum-optimism/optimism/issues/14603>
const MINIMUM_SAFETY_MSG: &str = "does not meet the minimum safety";

/// Invalid inbox entry
#[derive(thiserror::Error, Debug)]
pub enum InvalidInboxEntry {
    /// Message does not meet minimum safety level
    #[error("message does not meet min safety level, got: {got}, expected: {expected}")]
    MinimumSafety {
        /// Actual level of the message
        got: SafetyLevel,
        /// Minimum acceptable level that was passed to supervisor
        expected: SafetyLevel,
    },
    /// Invalid chain
    #[error("unsupported chain id: {0}")]
    UnknownChain(u64),
}

impl InvalidInboxEntry {
    /// Parses error message. Returns `None`, if message is not recognized.
    // todo: match on error code instead of message string once resolved <https://github.com/ethereum-optimism/optimism/issues/14603>
    pub fn parse_err_msg(err_msg: &str) -> Option<Self> {
        // Check if it's invalid message call, message example:
        // `failed to check message: failed to check log: unknown chain: 14417`
        if err_msg.contains(UNKNOWN_CHAIN_MSG) {
            if let Ok(chain_id) =
                err_msg.split(' ').next_back().expect("message contains chain id").parse::<u64>()
            {
                return Some(Self::UnknownChain(chain_id))
            }
        // Check if it's `does not meet the minimum safety` error, message example:
        // `message {0x4200000000000000000000000000000000000023 4 1 1728507701 901}
        // (safety level: unsafe) does not meet the minimum safety cross-unsafe"`
        } else if err_msg.contains(MINIMUM_SAFETY_MSG) {
            let message_safety = if err_msg.contains("safety level: safe") {
                SafetyLevel::Safe
            } else if err_msg.contains("safety level: local-safe") {
                SafetyLevel::LocalSafe
            } else if err_msg.contains("safety level: cross-unsafe") {
                SafetyLevel::CrossUnsafe
            } else if err_msg.contains("safety level: unsafe") {
                SafetyLevel::Unsafe
            } else if err_msg.contains("safety level: invalid") {
                SafetyLevel::Invalid
            } else {
                // Unexpected level name
                return None
            };
            let expected_safety = if err_msg.contains("safety finalized") {
                SafetyLevel::Finalized
            } else if err_msg.contains("safety safe") {
                SafetyLevel::Safe
            } else if err_msg.contains("safety local-safe") {
                SafetyLevel::LocalSafe
            } else if err_msg.contains("safety cross-unsafe") {
                SafetyLevel::CrossUnsafe
            } else if err_msg.contains("safety unsafe") {
                SafetyLevel::Unsafe
            } else {
                // Unexpected level name
                return None
            };

            return Some(Self::MinimumSafety { expected: expected_safety, got: message_safety })
        }

        None
    }
}

/// Failures occurring during validation of inbox entries.
#[derive(thiserror::Error, Debug)]
pub enum InteropTxValidatorError {
    /// Error validating interop event.
    #[error(transparent)]
    InvalidInboxEntry(#[from] InvalidInboxEntry),

    /// RPC client failure.
    #[error("supervisor rpc client failure: {0}")]
    RpcClientError(Box<dyn error::Error + Send + Sync>),

    /// Message validation against the Supervisor took longer than allowed.
    #[error("message validation timed out, timeout: {0} secs")]
    ValidationTimeout(u64),

    /// Catch-all variant for other supervisor server errors.
    #[error("unexpected error from supervisor: {0}")]
    SupervisorServerError(Box<dyn error::Error + Send + Sync>),
}

impl InteropTxValidatorError {
    /// Returns a new instance of [`RpcClientError`](Self::RpcClientError) variant.
    pub fn client(err: impl error::Error + Send + Sync + 'static) -> Self {
        Self::RpcClientError(Box::new(err))
    }

    /// Returns a new instance of [`RpcClientError`](Self::RpcClientError) variant.
    pub fn server_unexpected(err: impl error::Error + Send + Sync + 'static) -> Self {
        Self::SupervisorServerError(Box::new(err))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const MIN_SAFETY_CROSS_UNSAFE_ERROR: &str = "message {0x4200000000000000000000000000000000000023 4 1 1728507701 901} (safety level: unsafe) does not meet the minimum safety cross-unsafe";
    const MIN_SAFETY_UNSAFE_ERROR: &str = "message {0x4200000000000000000000000000000000000023 1091637521 4369 0 901} (safety level: invalid) does not meet the minimum safety unsafe";
    const MIN_SAFETY_FINALIZED_ERROR: &str = "message {0x4200000000000000000000000000000000000023 1091600001 215 1170 901} (safety level: safe) does not meet the minimum safety finalized";
    const INVALID_CHAIN: &str =
        "failed to check message: failed to check log: unknown chain: 14417";
    const RANDOM_ERROR: &str = "gibberish error";

    #[test]
    fn test_op_supervisor_error_parsing() {
        assert!(matches!(
            InvalidInboxEntry::parse_err_msg(MIN_SAFETY_CROSS_UNSAFE_ERROR).unwrap(),
            InvalidInboxEntry::MinimumSafety {
                expected: SafetyLevel::CrossUnsafe,
                got: SafetyLevel::Unsafe
            }
        ));

        assert!(matches!(
            InvalidInboxEntry::parse_err_msg(MIN_SAFETY_UNSAFE_ERROR).unwrap(),
            InvalidInboxEntry::MinimumSafety {
                expected: SafetyLevel::Unsafe,
                got: SafetyLevel::Invalid
            }
        ));

        assert!(matches!(
            InvalidInboxEntry::parse_err_msg(MIN_SAFETY_FINALIZED_ERROR).unwrap(),
            InvalidInboxEntry::MinimumSafety {
                expected: SafetyLevel::Finalized,
                got: SafetyLevel::Safe,
            }
        ));

        assert!(matches!(
            InvalidInboxEntry::parse_err_msg(INVALID_CHAIN).unwrap(),
            InvalidInboxEntry::UnknownChain(14417)
        ));

        assert!(InvalidInboxEntry::parse_err_msg(RANDOM_ERROR).is_none());
    }
}
