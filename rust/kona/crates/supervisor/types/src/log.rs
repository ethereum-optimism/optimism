use crate::message::ExecutingMessage;
use alloy_primitives::B256;

/// A reference entry representing a log observed in an L2 receipt.
///
/// This struct does **not** store the actual log content. Instead:
/// - `index` is the index of the log.
/// - `hash` is the hash of the log, which uniquely identifies the log entry and can be used for
///   lookups or comparisons.
/// - `executing_message` is present if the log represents an `ExecutingMessage` emitted by the
///   `CrossL2Inbox` contract.
///
/// This is the unit persisted by the log indexer into the database for later validation.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Log {
    /// The index of the log.
    pub index: u32,
    /// The hash of the log, derived from the log address and payload.
    pub hash: B256,
    /// The parsed message, if the log matches an `ExecutingMessage` event.
    pub executing_message: Option<ExecutingMessage>,
}
