//! Task and its associated types for the forkchoice engine update.

mod task;
pub use task::SynchronizeTask;

mod error;
pub use error::SynchronizeTaskError;
