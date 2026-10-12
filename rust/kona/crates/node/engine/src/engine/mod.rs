//! The [`Engine`] and the operations it performs.

mod core;
pub use core::{Engine, EngineResetError};

mod operations;
use operations::build_and_seal;
pub(crate) use operations::{
    BuildSealCoupling, BuildTask, ConsolidateTask, FinalizeTask, InsertTask, SealTask,
    SynchronizeTask,
};
pub use operations::{
    BuildTaskError, ConsolidateInput, ConsolidateTaskError, EngineBuildError, EngineTaskError,
    EngineTaskErrorSeverity, FinalizeBlockId, FinalizeTaskError, InsertTaskError, SealTaskError,
    SynchronizeTaskError,
};
