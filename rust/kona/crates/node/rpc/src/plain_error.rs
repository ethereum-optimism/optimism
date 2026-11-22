//! The error op-node's server sends for a plain Go error.

use jsonrpsee::types::ErrorObject;

/// The JSON-RPC code op-node's server assigns to a plain Go error, one without its own code.
pub(crate) const PLAIN_ERROR_CODE: i32 = -32000;

/// An error with [`PLAIN_ERROR_CODE`] and `message`, as op-node sends a plain Go error.
pub(crate) fn plain_error(message: impl Into<String>) -> ErrorObject<'static> {
    ErrorObject::owned(PLAIN_ERROR_CODE, message.into(), None::<()>)
}
