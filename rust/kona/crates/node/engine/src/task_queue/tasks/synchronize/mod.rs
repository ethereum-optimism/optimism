//! The forkchoice engine update shared by the engine tasks, and its error type.

mod helper;
pub(in crate::task_queue) use helper::synchronize;

mod error;
pub use error::SynchronizeTaskError;
