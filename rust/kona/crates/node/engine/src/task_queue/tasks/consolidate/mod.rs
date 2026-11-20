//! Consolidation Task

mod el_sync_progress;
pub use el_sync_progress::DEFAULT_EL_SYNC_STALL_TIMEOUT;

mod error;
pub use error::ConsolidateTaskError;

mod task;
pub use task::{ConsolidateInput, ConsolidateTask};
