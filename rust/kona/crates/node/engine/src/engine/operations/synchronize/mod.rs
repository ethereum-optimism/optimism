//! Task and its associated types for the forkchoice engine update.

mod operation;
pub(crate) use operation::SynchronizeTask;

mod error;
pub use error::SynchronizeTaskError;
