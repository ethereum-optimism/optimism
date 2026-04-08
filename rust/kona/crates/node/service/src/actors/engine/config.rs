use crate::NodeMode;
use alloy_provider::RootProvider;
use alloy_rpc_types_engine::JwtSecret;
use kona_engine::{EngineClientBuilder, OpEngineClient};
use kona_genesis::RollupConfig;
use op_alloy_network::Optimism;
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

    /// The L1 rpc url.
    pub l1_url: Url,

    /// The mode of operation for the node.
    /// When the node is in sequencer mode, the engine actor will receive requests to build blocks
    /// from the sequencer actor.
    pub mode: NodeMode,
}

impl EngineConfig {
    /// Builds and returns the [`OpEngineClient`].
    pub fn build_engine_client(self) -> OpEngineClient<RootProvider, RootProvider<Optimism>> {
        EngineClientBuilder {
            l2: self.l2_url,
            l2_jwt: self.l2_jwt_secret,
            l1_rpc: self.l1_url,
            cfg: self.config,
        }
        .build()
    }
}
