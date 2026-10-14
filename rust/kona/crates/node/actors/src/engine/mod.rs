//! Engine actor and handle.

mod actor;
pub use actor::Builder;

mod attributes;

mod block_sink;
pub use block_sink::{ImportedBlockSink, NoopBlockSink};

mod error;
pub use error::{
    ActorError, BuildTaskError, ConsolidateTaskError, EngineBuildError, EngineClientError,
    EngineClientResult, EngineResetError, EngineTaskError, EngineTaskErrorSeverity,
    FinalizeTaskError, InsertTaskError, SealTaskError, SynchronizeTaskError,
};

mod handle;
pub use crate::capacity::{Capacity, InvalidCapacity};
pub use handle::{Consolidation, Handle};

mod input;
pub use input::FinalizeBlockId;

mod state;
pub use state::{EngineState, HeadsUpdate};

mod sync;
pub use sync::SyncStartError;

#[cfg(test)]
mod test_utils;
