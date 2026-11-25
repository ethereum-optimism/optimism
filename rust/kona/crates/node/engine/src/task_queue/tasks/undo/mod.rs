//! Task and its associated types for deriving again from the finalized head after a cross-chain
//! verifier undid verifications.

mod task;
pub use task::UndoTask;

mod error;
pub use error::UndoTaskError;

#[cfg(test)]
mod task_test;
