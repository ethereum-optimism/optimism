//! Task and its associated types for finalizing an L2 block.

mod operation;
pub(crate) use operation::FinalizeTask;

mod error;
pub use error::FinalizeTaskError;

mod id;
pub use id::FinalizeBlockId;

#[cfg(test)]
mod tests;
