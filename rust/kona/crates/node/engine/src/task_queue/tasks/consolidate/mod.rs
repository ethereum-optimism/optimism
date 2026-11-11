//! Consolidation Task

mod delegated_rejection;
pub use delegated_rejection::DelegatedRejection;

mod delegated_safe_heads;
pub use delegated_safe_heads::{DelegatedSafeHeads, InconsistentSafeHeadsError};

mod error;
pub use error::ConsolidateTaskError;

mod task;
pub use task::{ConsolidateInput, ConsolidateTask};
