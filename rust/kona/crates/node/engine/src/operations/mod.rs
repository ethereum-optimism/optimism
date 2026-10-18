//! Execution operations. The caller owns serialization, contexts, publication and retries.

mod error;
pub use error::{
    BuildError, ConsolidateError, FinalizeError, ForkchoiceError, InsertError, ResetError,
};

mod build;
pub use build::{get_payload, start_build};
mod insert;
pub use insert::{ImportMode, insert_payload};
mod forkchoice;
pub use forkchoice::{ForkchoiceStatus, synchronize};
mod consolidate;
pub use consolidate::{Consolidation, consolidate, follow_safe};
mod finalize;
pub use finalize::{FinalizationSource, finalize};

#[cfg(test)]
mod tests;
