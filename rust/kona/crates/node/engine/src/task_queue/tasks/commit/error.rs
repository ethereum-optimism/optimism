//! Contains the error types for the [`CommitTask`](crate::CommitTask).

use crate::{EngineTaskError, InsertTaskError, task_queue::tasks::task::EngineTaskErrorSeverity};

/// Why a commit was refused, as answered to the caller that requested it.
///
/// This is the channel payload, not the task's own error: a refused commit is a normal answer to
/// the requester, and the task that delivered it has done its job.
#[derive(Debug, thiserror::Error)]
pub enum CommitBlockError {
    /// The insert failed: the execution layer rejected the payload, could not be reached, or the
    /// canonicalizing forkchoice update failed.
    #[error(transparent)]
    Insert(#[from] InsertTaskError),
    /// The payload does not descend from the local-safe head, so making it the unsafe head would
    /// rewind the chain under a head derived from L1.
    ///
    /// op-node's `CommitBlock` has no counterpart to this check; kona refuses the write rather
    /// than corrupting its head ordering, and tells the caller so instead of dropping the payload
    /// silently the way the gossip path does.
    #[error("the payload does not descend from the local-safe head")]
    DoesNotDescendFromLocalSafe,
}

/// An error that occurs when running the [`CommitTask`](crate::CommitTask).
///
/// Uninhabited: the commit's outcome — success or [`CommitBlockError`] — travels to the requester
/// over the task's channel, and a requester that went away before hearing it (an RPC caller that
/// disconnected) is logged rather than escalated, because failing the task would either halt the
/// node over a dead client or retry a send that can never succeed.
#[derive(Debug, thiserror::Error)]
pub enum CommitTaskError {}

impl EngineTaskError for CommitTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        unreachable!("CommitTaskError is uninhabited: no value of it can exist to be asked")
    }
}
