//! Task and its associated types for finalizing an L2 block.

mod task;
pub use task::FinalizeTask;

mod error;
pub use error::FinalizeTaskError;

mod id;
pub use id::FinalizeBlockId;

#[cfg(test)]
mod task_test;
