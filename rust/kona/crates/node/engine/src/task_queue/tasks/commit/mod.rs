//! Task to commit an externally built payload and answer the caller that asked.

mod task;
pub use task::CommitTask;

mod error;
pub use error::{CommitBlockError, CommitTaskError};
