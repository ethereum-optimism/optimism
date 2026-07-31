//! Per-chain interop configuration parsed from the chain TOML's `[interop]` block.

use crate::ChainDependency;
use alloc::collections::BTreeMap;
use alloy_primitives::ChainId;

/// Per-chain interop configuration.
///
/// Chain TOMLs in `superchain-registry` may declare an `[interop]` section listing the
/// chain ids that participate in the same interop cluster. Every member of a cluster is
/// expected to declare a matching set; the registry build script validates this and
/// builds one [`crate::DependencySet`] per cluster from the union of equal entries.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[allow(clippy::zero_sized_map_values)]
pub struct InteropConfig {
    /// Chain ids this chain depends on (interop cluster membership).
    /// Keys are L2 chain ids; values are reserved per-dependency config (currently empty).
    #[cfg_attr(feature = "serde", serde(default))]
    pub dependencies: BTreeMap<ChainId, ChainDependency>,
}
