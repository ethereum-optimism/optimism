//! Task and its associated types for importing a block that has been started.

mod operation;
pub(crate) use operation::{BuildSealCoupling, SealTask};

mod error;
pub use error::SealTaskError;

#[cfg(test)]
mod tests;
