//! The operations the engine performs.

mod severity;
pub use severity::{EngineTaskError, EngineTaskErrorSeverity};

mod synchronize;
pub(crate) use synchronize::SynchronizeTask;
pub use synchronize::SynchronizeTaskError;

mod insert;
pub(crate) use insert::InsertTask;
pub use insert::InsertTaskError;

mod build;
pub(crate) use build::BuildTask;
pub use build::{BuildTaskError, EngineBuildError};

mod seal;
pub use seal::SealTaskError;
pub(crate) use seal::{BuildSealCoupling, SealTask};

mod consolidate;
pub(crate) use consolidate::ConsolidateTask;
pub use consolidate::{ConsolidateInput, ConsolidateTaskError};

mod finalize;
pub(crate) use finalize::FinalizeTask;
pub use finalize::{FinalizeBlockId, FinalizeTaskError};

mod util;
pub(super) use util::{BuildAndSealError, build_and_seal};
