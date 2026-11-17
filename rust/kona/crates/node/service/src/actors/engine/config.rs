use crate::NodeMode;
use alloy_rpc_types_engine::JwtSecret;
use kona_engine::{EngineClient, EngineClientBuilder};
use kona_genesis::RollupConfig;
use kona_providers_alloy::AlloyChainProvider;
use std::sync::Arc;
use url::Url;

/// Configuration for the Engine Actor.
#[derive(Debug, Clone)]
pub struct EngineConfig {
    /// The [`RollupConfig`].
    pub config: Arc<RollupConfig>,

    /// The engine rpc url.
    pub l2_url: Url,
    /// The engine jwt secret.
    pub l2_jwt_secret: JwtSecret,

    /// The mode of operation for the node.
    /// When the node is in sequencer mode, the engine actor will receive requests to build blocks
    /// from the sequencer actor.
    pub mode: NodeMode,
}

impl EngineConfig {
    /// Builds the [`EngineClient`], which reads L1 through the node's shared L1 client.
    pub fn build_engine_client(self, l1: AlloyChainProvider) -> EngineClient {
        EngineClientBuilder { l2: self.l2_url, l2_jwt: self.l2_jwt_secret, l1, cfg: self.config }
            .build()
    }
}
