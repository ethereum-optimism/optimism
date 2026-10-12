//! Task and its associated types for importing a block that has been started.

mod task;
pub(crate) use task::{BuildSealCoupling, SealTask};

mod error;
pub use error::SealTaskError;

#[cfg(test)]
mod task_test;
