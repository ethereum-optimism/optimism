//! Shared metric label keys and label-value constructors.
//!
//! Every kona crate that emits per-chain metrics tags its series with the same key, so the key
//! and the value constructor live here rather than being respelled in each crate's `Metrics`
//! type. A single definition is what makes cross-crate aggregation queries safe to write.

use alloc::{string::ToString, sync::Arc};

/// The label key identifying which L2 chain a metric series belongs to.
///
/// A multi-chain process (lokahi) runs one kona stack per chain in a single metrics registry, so
/// without this dimension the per-chain series collapse into one.
pub const CHAIN_ID_LABEL: &str = "chain_id";

/// Builds the [`CHAIN_ID_LABEL`] value for `chain_id`.
///
/// Returns an [`Arc<str>`] because the label is attached to every emit on a long-lived component:
/// callers construct it once and clone the handle per emit instead of formatting the chain ID
/// again each time.
pub fn chain_id_label(chain_id: u64) -> Arc<str> {
    Arc::from(chain_id.to_string())
}
