//! The configuration of a [`RollupNode`](crate::RollupNode).

use crate::{ChainSpec, NetworkConfig, SequencerConfig};
use std::net::SocketAddr;

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
    /// The [`NetworkConfig`], or `None` for a node without P2P, which neither receives nor
    /// gossips blocks.
    pub p2p_config: Option<NetworkConfig>,
    /// The address the RPC server listens on, or `None` for a node that serves no RPC. It serves
    /// the admin API, so keep it off untrusted networks.
    pub rpc_addr: Option<SocketAddr>,
    /// The [`SequencerConfig`] of a node that can sequence; `None` for a node that only follows
    /// the chain.
    ///
    /// A node that can sequence starts with its sequencer stopped, and serves the sequencer's
    /// admin RPC methods, such as `admin_startSequencer`.
    pub sequencer: Option<SequencerConfig>,
}
