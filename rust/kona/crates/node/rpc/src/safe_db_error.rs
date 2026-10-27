//! The error shape op-node's clients expect from the safe head database RPCs.

use jsonrpsee::types::ErrorObject;
use kona_safedb::SafeDbError;

/// The JSON-RPC error code op-node's server assigns to a plain error.
pub(crate) const SAFE_DB_ERROR_CODE: i32 = -32000;

/// Maps a database error the way op-node reports it: a not-found answer is the bare message, so
/// that op-node's rollup client recognizes it, and every other failure names the operation.
pub(crate) fn safe_db_error(
    error: SafeDbError,
    operation: std::fmt::Arguments<'_>,
) -> ErrorObject<'static> {
    let message = match error {
        SafeDbError::NotFound => error.to_string(),
        e => format!("{operation}: {e}"),
    };
    ErrorObject::owned(SAFE_DB_ERROR_CODE, message, None::<()>)
}
