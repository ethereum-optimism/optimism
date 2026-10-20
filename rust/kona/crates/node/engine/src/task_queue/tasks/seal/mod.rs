//! Task and its associated types for importing a block that has been started.

mod task;
pub use task::SealTask;
pub(in crate::task_queue) use task::get_payload;

mod error;
pub use error::SealTaskError;

#[cfg(test)]
mod task_test;
