//! Task and its associated types for moving the safe label to a verifier's head.

mod head;
pub use head::VerifierHead;

mod task;
pub use task::PromoteTask;

mod error;
pub use error::PromoteTaskError;

#[cfg(test)]
mod task_test;
