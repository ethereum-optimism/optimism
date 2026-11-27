//! The task that runs the block-building steps of the `opstack` RPC namespace.

mod task;
pub use task::OpStackTask;

mod error;
pub use error::OpenBlockError;

#[cfg(test)]
mod task_test;
