use crate::NodeMode;
use alloy_provider::RootProvider;
use kona_engine::EngineClient;
use kona_genesis::RollupConfig;
use kona_providers_alloy::AlloyChainProvider;
use op_alloy_network::Optimism;
use std::sync::Arc;

/// Configuration for the Engine Actor.
#[derive(Debug, Clone)]
pub struct EngineConfig {
    /// The [`RollupConfig`].
    pub config: Arc<RollupConfig>,

    /// The node's authenticated L2 engine connection, shared by the engine actor and by every
    /// derivation provider that reads L2.
    ///
    /// kona-node builds it once with [`EngineClient::rpc_client`], so each request has a deadline
    /// and is recorded in the engine request metrics and logs.
    pub provider: RootProvider<Optimism>,

    /// The mode of operation for the node.
    /// When the node is in sequencer mode, the engine actor will receive requests to build blocks
    /// from the sequencer actor.
    pub mode: NodeMode,
}

impl EngineConfig {
    /// Builds the [`EngineClient`], which reads L1 through the node's shared L1 client.
    pub fn build_engine_client(self, l1: AlloyChainProvider) -> EngineClient {
        EngineClient::new(l1, self.provider, self.config)
    }
}
