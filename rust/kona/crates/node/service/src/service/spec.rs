//! The rules of the chain a [`RollupNode`](crate::RollupNode) follows.

use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use std::sync::Arc;

/// The rules of the chain a rollup node follows: its rollup config, the config of the L1 chain it
/// settles on, and its interop dependency set. They are resolved together and travel together, so
/// every component of the node sees the same chain.
///
/// Each part is behind an [`Arc`], so cloning the spec, or handing one part to a component, is
/// cheap.
#[derive(Debug, Clone)]
pub struct ChainSpec {
    /// The L2 chain's rollup config.
    pub rollup_config: Arc<RollupConfig>,
    /// The config of the L1 chain the L2 chain settles on.
    pub l1_config: Arc<L1ChainConfig>,
    /// The L2 chain's interop dependency set. It decides which upgrade transactions the Lagoon
    /// hardfork's activation block carries; a chain outside any interop cluster has a set
    /// containing only itself.
    pub dependency_set: Arc<DependencySet>,
}
