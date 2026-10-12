//! Consolidation Task

mod error;
pub use error::ConsolidateTaskError;

mod task;
pub use task::ConsolidateInput;
pub(crate) use task::ConsolidateTask;
