//! The [`Engine`] and the operations it performs.

mod core;
pub use core::{Engine, EngineResetError};

mod operations;
use operations::BuildSealCoupling;
pub use operations::{
    BuildTaskError, ConsolidateInput, ConsolidateTaskError, EngineBuildError, EngineTaskError,
    EngineTaskErrorSeverity, FinalizeBlockId, FinalizeTaskError, InsertTaskError, SealTaskError,
    SynchronizeTaskError,
};
