//! The operation and its associated types for importing a block that has been started.

mod operation;
pub(in crate::engine) use operation::BuildSealCoupling;

mod error;
pub use error::SealTaskError;

#[cfg(test)]
mod tests;
