//! Forkchoice synchronization shared by engine tasks and reset.

mod helper;
pub(in crate::task_queue) use helper::synchronize;

mod error;
pub use error::SynchronizeTaskError;
