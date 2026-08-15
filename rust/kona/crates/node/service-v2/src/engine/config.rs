//! Execution-engine service configuration.

use crate::node::NodeMode;
use alloy_provider::RootProvider;
use alloy_rpc_types_engine::JwtSecret;
use kona_engine::{EngineClientBuilder, OpEngineClient};
use kona_genesis::RollupConfig;
use op_alloy_network::Optimism;
use std::sync::Arc;
use url::Url;

/// Configuration required to connect the engine service to an execution layer.
#[derive(Debug, Clone)]
pub struct EngineConfig {
    /// Rollup protocol configuration.
    pub config: Arc<RollupConfig>,
    /// Authenticated Engine API URL.
    pub l2_url: Url,
    /// Engine API JWT secret.
    pub l2_jwt_secret: JwtSecret,
    /// L1 execution RPC URL used during startup reconciliation.
    pub l1_url: Url,
    /// Node operating mode.
    pub mode: NodeMode,
}

impl EngineConfig {
    /// Builds the raw Engine API client owned by [`super::EngineService`].
    pub fn build_client(self) -> OpEngineClient<RootProvider, RootProvider<Optimism>> {
        EngineClientBuilder {
            l2: self.l2_url,
            l2_jwt: self.l2_jwt_secret,
            l1_rpc: self.l1_url,
            cfg: self.config,
        }
        .build()
    }
}
