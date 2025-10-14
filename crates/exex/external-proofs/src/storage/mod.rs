//! Storage abstractions for external proofs.
mod traits;
pub use traits::{
    BlockStateDiff, OpProofsHashedCursor, OpProofsStorage, OpProofsStorageError,
    OpProofsStorageResult, OpProofsTrieCursor,
};

pub mod in_memory;
pub mod mdbx;
