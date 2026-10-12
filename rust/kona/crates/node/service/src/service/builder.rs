//! Contains the builder for the [`RollupNode`].

use crate::{
    DerivationDelegateClient, EngineConfig, NetworkConfig, RollupNode, SequencerConfig,
    service::node::L1Config,
};
use alloy_primitives::Bytes;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
    reqwest,
};
use http_body_util::Full;
use op_alloy_network::Optimism;
use std::{sync::Arc, time::Duration};
use tower::ServiceBuilder;
use url::Url;

use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_providers_alloy::OnlineBeaconClient;
use kona_providers_local::BufferedL2Provider;
use kona_rpc::RpcBuilder;

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
    pub chain_config: L1ChainConfig,
    /// Whether to trust the L1 RPC.
    pub trust_rpc: bool,
    /// The L1 beacon API.
    pub beacon: Url,
    /// The L1 RPC URL.
    pub rpc_url: Url,
}

/// The configuration of a [`RollupNode`], which [`Self::build`] assembles the node from.
#[derive(Debug)]
pub struct RollupNodeConfig {
    /// The application version reported by the RPC server.
    pub version: String,
    /// The rollup configuration.
    pub config: RollupConfig,
    /// How to reach the L1 chain.
    pub l1_client: L1ClientConfig,
    /// Whether to trust the L2 RPC.
    pub l2_trust_rpc: bool,
    /// Engine builder configuration.
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

impl RollupNodeConfig {
    /// Assembles the [`RollupNode`] service.
    ///
    /// ## Panics
    ///
    /// Panics if:
    /// - The L1 HTTP client cannot be initialized.
    /// - The derivation delegate client cannot be created from its configuration.
    pub fn build(self) -> RollupNode {
        let l1_beacon = OnlineBeaconClient::new_http(self.l1_client.beacon.to_string());

        let l1_client = reqwest::Client::builder()
            .timeout(Duration::from_secs(10))
            .build()
            .expect("l1 http client initialization");
        let l1_rpc_client =
            RpcClient::new_http_with_client(l1_client, self.l1_client.rpc_url.clone());
        let l1_config = L1Config {
            chain_config: Arc::new(self.l1_client.chain_config),
            trust_rpc: self.l1_client.trust_rpc,
            beacon_client: l1_beacon,
            engine_provider: RootProvider::new(l1_rpc_client),
        };

        let jwt_secret = self.engine_config.l2_jwt_secret;
        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();

        let auth_layer = AuthLayer::new(jwt_secret);
        let service = ServiceBuilder::new().layer(auth_layer).service(hyper_client);

        let layer_transport = HyperClient::with_service(service);
        let http_hyper = Http::with_client(layer_transport, self.engine_config.l2_url.clone());
        let rpc_client = RpcClient::new(http_hyper, false);
        let l2_provider = RootProvider::<Optimism>::new(rpc_client);

        let rollup_config = Arc::new(self.config);
        // The buffer's reorg depth governs its handling of chain events, which nothing feeds it:
        // lookups are keyed by hash, so a reorged-out block is never returned regardless.
        let l2_block_buffer = BufferedL2Provider::new(
            rollup_config.clone(),
            super::node::IMPORTED_BLOCK_BUFFER_SIZE,
            0,
        );

        let p2p_config = self.p2p_config;

        let derivation_delegate_provider = self.derivation_delegate_config.as_ref().map(|config| {
            DerivationDelegateClient::new(config.l2_cl_url.clone()).expect(
                "Failed to create Derivation Delegate provider despite config being present",
            )
        });

        RollupNode {
            version: self.version,
            config: rollup_config,
            l1_config,
            l2_provider,
            l2_block_buffer,
            l2_trust_rpc: self.l2_trust_rpc,
            engine_config: self.engine_config,
            rpc_builder: self.rpc_config,
            p2p_config,
            sequencer_config: self.sequencer_config,
            derivation_delegate_provider,
            dependency_set: self.dependency_set,
        }
    }
}
