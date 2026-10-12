//! Task to insert an unsafe payload into the execution engine.

mod operation;
pub(crate) use operation::InsertTask;

mod error;
pub use error::InsertTaskError;
