//! Consolidation Task

mod error;
pub use error::ConsolidateTaskError;

mod operation;
pub use operation::ConsolidateInput;
pub(crate) use operation::ConsolidateTask;
