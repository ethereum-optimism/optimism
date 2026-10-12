//! The operations the engine performs.

mod severity;
pub use severity::{EngineTaskError, EngineTaskErrorSeverity};

mod synchronize;
pub use synchronize::SynchronizeTaskError;

mod insert;
pub use insert::InsertTaskError;

mod build;
pub use build::{BuildTaskError, EngineBuildError};

mod seal;
pub(super) use seal::BuildSealCoupling;
pub use seal::SealTaskError;

mod consolidate;
pub use consolidate::{ConsolidateInput, ConsolidateTaskError};

mod finalize;
pub use finalize::{FinalizeBlockId, FinalizeTaskError};

mod util;
pub(super) use util::BuildAndSealError;
