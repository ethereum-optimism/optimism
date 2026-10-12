//! The [`Engine`] and the operations it performs.

mod core;
pub use core::{Engine, EngineResetError};

mod tasks;
use tasks::build_and_seal;
pub(crate) use tasks::{
    BuildSealCoupling, BuildTask, ConsolidateTask, FinalizeTask, InsertTask, SealTask,
    SynchronizeTask,
};
pub use tasks::{
    BuildTaskError, ConsolidateInput, ConsolidateTaskError, EngineBuildError, EngineTaskError,
    EngineTaskErrorSeverity, FinalizeBlockId, FinalizeTaskError, InsertTaskError, SealTaskError,
    SynchronizeTaskError,
};
