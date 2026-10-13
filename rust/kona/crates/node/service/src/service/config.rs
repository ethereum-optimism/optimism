//! The configuration of a [`RollupNode`](crate::RollupNode).

use crate::{ChainSpec, NetworkConfig, NodeMode, SequencerConfig};
use kona_rpc::RpcBuilder;

/// The configuration of a [`RollupNode`](crate::RollupNode): what the node does, as plain data.
///
/// It says nothing about where the node's L1 and L2 endpoints are: the node talks to them through
/// the [`RollupNodeClients`](crate::RollupNodeClients) it is given.
#[derive(Debug)]
pub struct RollupNodeConfig {
    /// The application version reported by the RPC server.
    pub version: String,
    /// The rules of the chain the node follows, shared by every component of the node.
    pub chain: ChainSpec,
    /// The mode the node runs in.
    pub mode: NodeMode,
    /// The [`NetworkConfig`].
    pub p2p_config: NetworkConfig,
    /// The RPC server's configuration.
    pub rpc_config: RpcBuilder,
    /// The [`SequencerConfig`], used when the node runs in sequencer mode.
    pub sequencer_config: SequencerConfig,
}
