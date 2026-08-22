//! Tasks to update the engine state.

mod task;
pub use task::{
    EngineTask, EngineTaskError, EngineTaskErrorSeverity, EngineTaskErrors, EngineTaskExt,
};

mod synchronize;
pub use synchronize::{SynchronizeTask, SynchronizeTaskError};

mod insert;
pub use insert::{InsertTask, InsertTaskError};

mod commit;
pub use commit::{CommitBlockError, CommitTask, CommitTaskError};

mod build;
pub use build::{BuildTask, BuildTaskError, EngineBuildError};

mod seal;
pub use seal::{BuildSealCoupling, SealTask, SealTaskError};

mod consolidate;
pub use consolidate::{ConsolidateInput, ConsolidateTask, ConsolidateTaskError};

mod finalize;
pub use finalize::{FinalizeBlockId, FinalizeTask, FinalizeTaskError};

mod promote;
pub use promote::{PromoteCrossSafeTask, PromoteCrossSafeTaskError};

mod util;
pub(super) use util::{BuildAndSealError, build_and_seal};
