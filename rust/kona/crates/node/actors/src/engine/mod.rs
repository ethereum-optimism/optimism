//! Engine actor and handle.

mod actor;
pub use actor::Builder;

mod error;
pub use error::{
    ActorError, BuildTaskError, ConsolidateTaskError, EngineBuildError, EngineClientError,
    EngineClientResult, EngineResetError, EngineTaskError, EngineTaskErrorSeverity,
    FinalizeTaskError, InsertTaskError, SealTaskError, SynchronizeTaskError,
};

mod handle;
pub use handle::{Consolidation, Handle};

mod input;
pub use input::FinalizeBlockId;

mod state;
pub use state::{EngineState, HeadsUpdate};
