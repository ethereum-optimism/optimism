//! Fully constructed node dependencies.

use alloy_provider::RootProvider;
use kona_genesis::L1ChainConfig;
use kona_providers_alloy::OnlineBeaconClient;
use std::sync::Arc;

/// L1 providers and chain configuration shared by node services.
#[derive(Debug, Clone)]
pub struct L1Config {
    /// L1 consensus configuration.
    pub chain_config: Arc<L1ChainConfig>,
    /// Whether L1 RPC results may be trusted.
    pub trust_rpc: bool,
    /// L1 beacon API client.
    pub beacon_client: OnlineBeaconClient,
    /// L1 execution provider.
    pub provider: RootProvider,
}
