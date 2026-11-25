//! Tasks to update the engine state.

mod task;
pub use task::{
    EngineTask, EngineTaskError, EngineTaskErrorSeverity, EngineTaskErrors, ExecutableEngineTask,
};

mod synchronize;
pub use synchronize::SynchronizeTaskError;
pub(super) use synchronize::{synchronize, synchronize_unconditionally};

mod insert;
pub use insert::{InsertTask, InsertTaskError};
pub(super) use insert::{insert_payload, new_payload};

mod build;
pub(super) use build::start_build;
pub use build::{BuildTask, BuildTaskError, EngineBuildError};

mod seal;
pub(super) use seal::get_payload;
pub use seal::{SealTask, SealTaskError};

mod canonicalize;
pub use canonicalize::{CanonicalizeTask, CanonicalizeTaskError};

mod consolidate;
pub use consolidate::{ConsolidateInput, ConsolidateTask, ConsolidateTaskError};

mod finalize;
pub use finalize::{FinalizeBlockId, FinalizeTask, FinalizeTaskError};

mod promote;
pub use promote::{PromoteTask, PromoteTaskError, VerifierHead};

mod replace;
pub use replace::ReplaceDeniedError;
pub(super) use replace::replace_denied;

mod deny;
pub use deny::{DenyTask, DenyTaskError};

mod util;
pub(super) use util::{BuildAndImportError, build_and_import};
