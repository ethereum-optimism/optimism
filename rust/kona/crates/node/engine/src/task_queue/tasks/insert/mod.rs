//! Task to insert an unsafe payload into the execution engine.

mod task;
pub(crate) use task::InsertTask;

mod error;
pub use error::InsertTaskError;
