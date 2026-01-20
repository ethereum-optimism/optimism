use crate::NodeMode;
use alloy_provider::RootProvider;
use alloy_rpc_types_engine::JwtSecret;
use kona_engine::{
    EngineClientBuilder, EngineClientBuilderError, OpEngineClient, RollupBoostServerArgs,
};
use kona_genesis::RollupConfig;
use op_alloy_network::Optimism;
use std::{sync::Arc, time::Duration};
use url::Url;

/// Configuration for the Engine Actor.
#[derive(Debug, Clone)]
pub struct EngineConfig {
    /// The [`RollupConfig`].
    pub config: Arc<RollupConfig>,

    /// Builder url.
    pub builder_url: Url,
    /// Builder jwt secret.
    pub builder_jwt_secret: JwtSecret,
    /// Builder timeout.
    pub builder_timeout: Duration,

    /// The engine rpc url.
    pub l2_url: Url,
    /// The engine jwt secret.
    pub l2_jwt_secret: JwtSecret,
    /// The l2 timeout.
    pub l2_timeout: Duration,

    /// The L1 rpc url.
    pub l1_url: Url,

    /// The mode of operation for the node.
    /// When the node is in sequencer mode, the engine actor will receive requests to build blocks
    /// from the sequencer actor.
    pub mode: NodeMode,

    /// The rollup boost arguments.
    pub rollup_boost: RollupBoostServerArgs,
}

impl EngineConfig {
    /// Builds and returns the [`OpEngineClient`].
    pub fn build_engine_client(
        self,
    ) -> Result<OpEngineClient<RootProvider, RootProvider<Optimism>>, EngineClientBuilderError>
    {
        EngineClientBuilder {
            builder: self.builder_url.clone(),
            builder_jwt: self.builder_jwt_secret,
            builder_timeout: self.builder_timeout,
            l2: self.l2_url.clone(),
            l2_jwt: self.l2_jwt_secret,
            l2_timeout: self.l2_timeout,
            l1_rpc: self.l1_url.clone(),
            cfg: self.config.clone(),
            rollup_boost: self.rollup_boost,
        }
        .build()
    }
}
