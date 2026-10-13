//! The configuration of a [`RollupNode`](crate::RollupNode).

use crate::{NetworkConfig, NodeMode, SequencerConfig};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_rpc::RpcBuilder;
use std::sync::Arc;

/// The configuration of a [`RollupNode`](crate::RollupNode): what the node does, as plain data.
///
/// It says nothing about where the node's L1 and L2 endpoints are: the node talks to them through
/// the [`RollupNodeClients`](crate::RollupNodeClients) it is given.
#[derive(Debug)]
pub struct RollupNodeConfig {
    /// The application version reported by the RPC server.
    pub version: String,
    /// The rollup configuration, shared by every component of the node.
    pub rollup_config: Arc<RollupConfig>,
    /// The L1 chain configuration.
    pub l1_chain_config: Arc<L1ChainConfig>,
    /// The mode the node runs in.
    pub mode: NodeMode,
    /// The [`NetworkConfig`].
    pub p2p_config: NetworkConfig,
    /// An RPC Configuration.
    pub rpc_config: Option<RpcBuilder>,
    /// The [`SequencerConfig`], used when the node runs in sequencer mode.
    pub sequencer_config: SequencerConfig,
    /// The interop dependency set for this chain.
    ///
    /// Must be set when the rollup config schedules the Lagoon hardfork. When not set, the
    /// underlying [`kona_derive::StatefulAttributesBuilder`] constructor panics on an
    /// interop-scheduled chain.
    pub dependency_set: Option<Arc<DependencySet>>,
}
