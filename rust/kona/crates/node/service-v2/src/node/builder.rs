//! Rollup node service builder.

use crate::{
    engine::EngineConfig,
    network::NetworkConfig,
    node::{InteropMode, RollupNode, config::L1Config},
    unsafe_chain::SequencerConfig,
};
use alloy_primitives::Bytes;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
};
use http_body_util::Full;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_providers_alloy::OnlineBeaconClient;
use kona_rpc::RpcBuilder;
use op_alloy_network::Optimism;
use std::sync::Arc;
use tower::ServiceBuilder;
use url::Url;

/// Configuration for derivation delegation.
#[derive(Debug, Clone)]
pub struct DerivationDelegateConfig {
    /// Consensus-layer RPC endpoint to follow for safe/finalized progress.
    pub l2_cl_url: Url,
}

impl Default for DerivationDelegateConfig {
    fn default() -> Self {
        Self { l2_cl_url: Url::parse("http://localhost:9545").expect("static URL is valid") }
    }
}

/// Configuration used to construct shared L1 access.
#[derive(Debug)]
pub struct L1ConfigBuilder {
    /// L1 chain configuration.
    pub chain_config: L1ChainConfig,
    /// Whether provider responses may be trusted without verification.
    pub trust_rpc: bool,
    /// L1 beacon endpoint.
    pub beacon: Url,
    /// L1 execution endpoint.
    pub rpc_url: Url,
    /// Optional fixed L1 slot duration.
    pub slot_duration_override: Option<u64>,
}

/// Builder for the service-oriented rollup node.
#[derive(Debug)]
pub struct RollupNodeBuilder {
    config: RollupConfig,
    l1: L1ConfigBuilder,
    l2_trust_rpc: bool,
    engine: EngineConfig,
    network: NetworkConfig,
    rpc: Option<RpcBuilder>,
    sequencer: Option<SequencerConfig>,
    interop_mode: InteropMode,
    delegate: Option<DerivationDelegateConfig>,
    dependency_set: Option<Arc<DependencySet>>,
}

impl RollupNodeBuilder {
    /// Creates a node builder.
    pub fn new(
        config: RollupConfig,
        l1: L1ConfigBuilder,
        l2_trust_rpc: bool,
        engine: EngineConfig,
        network: NetworkConfig,
        rpc: Option<RpcBuilder>,
    ) -> Self {
        Self {
            config,
            l1,
            l2_trust_rpc,
            engine,
            network,
            rpc,
            sequencer: None,
            interop_mode: InteropMode::default(),
            delegate: None,
            dependency_set: None,
        }
    }

    /// Sets the interop dependency set.
    pub fn with_dependency_set(self, dependency_set: Option<Arc<DependencySet>>) -> Self {
        Self { dependency_set, ..self }
    }

    /// Replaces engine configuration.
    pub fn with_engine_config(self, engine: EngineConfig) -> Self {
        Self { engine, ..self }
    }

    /// Replaces RPC configuration.
    pub fn with_rpc_config(self, rpc: Option<RpcBuilder>) -> Self {
        Self { rpc, ..self }
    }

    /// Configures local sequencing capability.
    pub fn with_sequencer_config(self, sequencer: SequencerConfig) -> Self {
        Self { sequencer: Some(sequencer), ..self }
    }

    /// Configures derivation delegation.
    pub fn with_derivation_delegate_config(
        self,
        delegate: Option<DerivationDelegateConfig>,
    ) -> Self {
        Self { delegate, ..self }
    }

    /// Assembles the node without spawning any task.
    pub fn build(self) -> RollupNode {
        let mut beacon = OnlineBeaconClient::new_http(self.l1.beacon.to_string());
        if let Some(slot_duration) = self.l1.slot_duration_override {
            beacon = beacon.with_l1_slot_duration_override(slot_duration);
        }
        let l1 = L1Config {
            chain_config: Arc::new(self.l1.chain_config),
            trust_rpc: self.l1.trust_rpc,
            beacon_client: beacon,
            provider: RootProvider::new_http(self.l1.rpc_url),
        };

        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
        let service = ServiceBuilder::new()
            .layer(AuthLayer::new(self.engine.l2_jwt_secret))
            .service(hyper_client);
        let transport = HyperClient::with_service(service);
        let http = Http::with_client(transport, self.engine.l2_url.clone());
        let l2_provider = RootProvider::<Optimism>::new(RpcClient::new(http, false));

        RollupNode {
            config: Arc::new(self.config),
            l1,
            l2_provider,
            l2_trust_rpc: self.l2_trust_rpc,
            engine_config: self.engine,
            network_config: self.network,
            rpc_config: self.rpc,
            sequencer_config: self.sequencer.unwrap_or_default(),
            interop_mode: self.interop_mode,
            delegate_config: self.delegate,
            dependency_set: self.dependency_set,
        }
    }
}
