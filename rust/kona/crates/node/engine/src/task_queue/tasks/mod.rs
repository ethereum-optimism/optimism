//! Tasks to update the engine state.

mod task;
pub use task::{
    EngineTask, EngineTaskError, EngineTaskErrorSeverity, EngineTaskErrors, EngineTaskExt,
};

mod synchronize;
pub use synchronize::SynchronizeTaskError;
pub(super) use synchronize::synchronize;

mod insert;
pub(super) use insert::insert_payload;
pub use insert::{InsertTask, InsertTaskError};

mod build;
pub(super) use build::start_build;
pub use build::{BuildTask, BuildTaskError, EngineBuildError};

mod seal;
pub(super) use seal::get_payload;
pub use seal::{SealTask, SealTaskError};

mod consolidate;
pub use consolidate::{ConsolidateInput, ConsolidateTask, ConsolidateTaskError};

mod finalize;
pub use finalize::{FinalizeBlockId, FinalizeTask, FinalizeTaskError};

mod util;
pub(super) use util::{BuildAndImportError, build_and_import, import_payload};
