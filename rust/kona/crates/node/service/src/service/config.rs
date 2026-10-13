//! The configuration of a [`RollupNode`](crate::RollupNode).

use crate::{EngineConfig, NetworkConfig, SequencerConfig};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_rpc::RpcBuilder;
use std::sync::Arc;
use url::Url;

/// Configuration for Derivation Delegate mode.
#[derive(Debug, Clone)]
pub struct DerivationDelegateConfig {
    /// The L2 consensus layer RPC URL to delegate derivation to.
    /// This CL must expose the `optimism_syncStatus` RPC endpoint.
    pub l2_cl_url: Url,
}

impl Default for DerivationDelegateConfig {
    fn default() -> Self {
        Self { l2_cl_url: Url::parse("http://localhost:9545").unwrap() }
    }
}

/// How to reach the L1 chain.
#[derive(Debug)]
pub struct L1ClientConfig {
    /// The L1 chain configuration.
    pub chain_config: Arc<L1ChainConfig>,
    /// The L1 beacon API.
    pub beacon: Url,
    /// The L1 RPC URL.
    pub rpc_url: Url,
}

/// The configuration of a [`RollupNode`](crate::RollupNode): what the node does, as plain data.
///
/// It holds no clients; [`RollupNodeClients::connect`](crate::RollupNodeClients::connect) builds
/// the standard ones from it.
#[derive(Debug)]
pub struct RollupNodeConfig {
    /// The application version reported by the RPC server.
    pub version: String,
    /// The rollup configuration, shared by every component of the node.
    pub rollup_config: Arc<RollupConfig>,
    /// How to reach the L1 chain.
    pub l1_client: L1ClientConfig,
    /// How the node reaches the L2 engine, and the mode it runs in.
    pub engine_config: EngineConfig,
    /// The [`NetworkConfig`].
    pub p2p_config: NetworkConfig,
    /// An RPC Configuration.
    pub rpc_config: Option<RpcBuilder>,
    /// The [`SequencerConfig`], used when the engine runs in sequencer mode.
    pub sequencer_config: SequencerConfig,
    /// Optional configuration for Derivation Delegate mode.
    /// When present, the node does not run derivation, instead trusting the configured delegate.
    pub derivation_delegate_config: Option<DerivationDelegateConfig>,
    /// The interop dependency set for this chain.
    ///
    /// Must be set when the rollup config schedules the Lagoon hardfork. When not set, the
    /// underlying [`kona_derive::StatefulAttributesBuilder`] constructor panics on an
    /// interop-scheduled chain.
    pub dependency_set: Option<Arc<DependencySet>>,
}
