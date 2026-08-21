//! Engine State

mod core;
pub use core::{EngineState, EngineSyncState, EngineSyncStateUpdate};

mod local_safe;
pub use local_safe::{LocalSafeHead, LocalSafeOrigin};

mod snapshot;
pub use snapshot::{LocalSafeAtTimestamp, LocalSafeSnapshot};

mod cross_safe;
pub use cross_safe::{CrossSafePromoter, CrossSafePromotion, CrossSafeSource};

// The head-ordering invariant spans every module above, so its tests sit here rather than inside
// any one of them.
#[cfg(test)]
mod ordering_test;
