//! Engine actor and handle.

mod actor;
pub use actor::Builder;

mod block_sink;
pub use block_sink::{ImportedBlockSink, NoopBlockSink};

mod error;
pub use error::{
    ActorError, BuildTaskError, ConsolidateTaskError, EngineBuildError, EngineClientError,
    EngineClientResult, EngineResetError, EngineTaskError, EngineTaskErrorSeverity,
    FinalizeTaskError, InsertTaskError, SealTaskError, SynchronizeTaskError,
};

mod derivation;
pub use derivation::{Derived, Finality, Safe, Safety};

mod handle;
pub use handle::Handle;

mod state;
pub use state::{EngineState, HeadsUpdate};

mod sync;
pub use sync::{SyncStartError, execution_layer_state};

#[cfg(test)]
mod test_utils;
