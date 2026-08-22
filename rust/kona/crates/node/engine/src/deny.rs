//! The super-authority deny list, as the engine consults it.
//!
//! When a cross-chain verifier invalidates a block, the block is recorded on a deny list and the
//! chain is rewound off it. Derivation then rebuilds the block from the same L1 batch data — and
//! reproduces it byte for byte, because the build is deterministic. The deny list is what breaks
//! that loop: a derived payload whose hash is denied is not inserted, and a deposits-only
//! replacement is built at the same height instead, exactly as op-node does when its
//! `SuperAuthority` denies a payload (`op-node/rollup/engine/payload_process.go`).
//!
//! The engine does not own the list. It is written by whoever decides invalidity — in lokahi, the
//! interop verifier's invalidated-output archive doubles as it — and the engine only reads it,
//! through this trait. A node composed without one (`None` everywhere a
//! [`SharedDenyList`] is optional) behaves exactly as before: nothing is ever denied.
//!
//! ## Error postures are per call site, not global
//!
//! op-node's deny list is consulted at three points with two different failure postures, and the
//! split is deliberate (each is documented at its kona call site):
//!
//! - **Payload insertion fails open** (`payload_process.go:62`): a read error is logged at error
//!   level and the payload proceeds. A wedged engine is worse than looping invalidation until the
//!   store heals.
//! - **Unsafe ingestion fails open** (`engine_controller.go:772`): same reasoning, always logged.
//! - **Consolidation fails closed** (`op-node/rollup/attributes/attributes.go:241`): without a
//!   deny-list answer a block must be neither promoted nor reorged, so the caller stalls and
//!   retries.

use alloy_primitives::B256;
use std::{fmt::Debug, sync::Arc};

/// A read error from the deny list.
///
/// Carries the underlying store's rendering rather than a typed cause: the engine's only decision
/// on an error is the per-call-site posture above, which does not depend on why the read failed.
#[derive(Debug, thiserror::Error, Clone, PartialEq, Eq)]
#[error("deny list read failed: {0}")]
pub struct DenyListReadError(pub String);

/// The deny list of blocks a super authority has invalidated.
///
/// Implementations answer from durable storage: a denial must survive the restart that a rewind
/// mid-crash forces, or the rebuilt block is re-adopted on the way back up.
pub trait DenyList: Debug + Send + Sync {
    /// Whether the block at `number` with `hash` has been denied.
    fn is_denied(&self, number: u64, hash: B256) -> Result<bool, DenyListReadError>;

    /// The highest denied block height, or [`None`] when nothing is denied.
    ///
    /// Backs the unsafe-ingestion gate: unsafe payloads are refused from the moment a block is
    /// denied until the finalized head passes the highest denied height, so unsafe sync cannot
    /// re-adopt the invalidated branch (op-node's `unsafeDenyGatingActive`,
    /// `op-node/rollup/engine/engine_controller.go:776`).
    fn max_denied_height(&self) -> Result<Option<u64>, DenyListReadError>;
}

/// A shared handle to the deny list.
pub type SharedDenyList = Arc<dyn DenyList>;
