//! Task and its associated types for sealing a sequenced block.

mod task;
pub use task::SealTask;
pub(in crate::task_queue) use task::get_payload;

mod error;
pub use error::SealTaskError;

#[cfg(test)]
mod task_test;
