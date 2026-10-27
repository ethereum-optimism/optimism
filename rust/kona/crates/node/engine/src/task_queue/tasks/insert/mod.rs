//! Task to insert an unsafe payload into the execution engine.

mod task;
pub use task::InsertTask;
pub(in crate::task_queue) use task::insert_payload;

mod error;
pub use error::InsertTaskError;
