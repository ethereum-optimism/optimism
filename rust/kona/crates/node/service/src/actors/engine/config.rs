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

    /// The mode of operation for the node.
    /// When the node is in sequencer mode, the engine actor will receive requests to build blocks
    /// from the sequencer actor.
    pub mode: NodeMode,

    /// Authenticated L2 engine provider. Each request must be bounded by a deadline; see
    /// [`EngineClient::rpc_client`].
    pub provider: RootProvider<Optimism>,
}

impl EngineConfig {
    /// Builds the [`EngineClient`], which reads L1 through the node's shared L1 client.
    pub fn build_engine_client(self, l1: AlloyChainProvider) -> EngineClient {
        EngineClient::new(l1, self.provider, self.config)
    }
}
