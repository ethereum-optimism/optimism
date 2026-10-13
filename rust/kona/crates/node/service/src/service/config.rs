//! The configuration of a [`RollupNode`](crate::RollupNode).

use crate::{ChainSpec, NetworkConfig, SequencerConfig};
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
    /// The [`NetworkConfig`].
    pub p2p_config: NetworkConfig,
    /// The RPC server's configuration.
    pub rpc_config: RpcBuilder,
    /// The [`SequencerConfig`] of a node that can sequence; `None` for a node that only follows
    /// the chain.
    ///
    /// A node that can sequence starts with its sequencer stopped, and serves the sequencer's
    /// admin RPC methods, such as `admin_startSequencer`, when the admin API is enabled.
    pub sequencer: Option<SequencerConfig>,
}
