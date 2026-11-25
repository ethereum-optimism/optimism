//! Task and its associated types for replacing local-safe blocks a cross-chain verifier denied.

mod task;
pub use task::DenyTask;

mod error;
pub use error::DenyTaskError;

#[cfg(test)]
mod task_test;
