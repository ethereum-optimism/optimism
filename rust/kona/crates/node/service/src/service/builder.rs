//! Contains the builder for the [`RollupNode`].

use crate::{
    BlocksClientConfig, EngineConfig, InteropMode, NetworkConfig, RollupNode, SequencerConfig,
    actors::DerivationDelegateClient, service::node::L1Config,
};
use alloy_primitives::Bytes;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
};
use http_body_util::Full;
use op_alloy_network::Optimism;
use std::sync::Arc;
use tower::ServiceBuilder;
use url::Url;

use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_providers_alloy::OnlineBeaconClient;
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

/// The [`L1ConfigBuilder`] is used to construct a [`L1Config`].
#[derive(Debug)]
pub struct L1ConfigBuilder {
    /// The L1 chain configuration.
    pub chain_config: L1ChainConfig,
    /// Whether to trust the L1 RPC.
    pub trust_rpc: bool,
    /// The L1 beacon API.
    pub beacon: Url,
    /// The L1 RPC URL.
    pub rpc_url: Url,
    /// The duration in seconds of an L1 slot. This can be used to hardcode a fixed slot
    /// duration if the l1-beacon's slot configuration is not available.
    pub slot_duration_override: Option<u64>,
}

/// The [`RollupNodeBuilder`] is used to construct a [`RollupNode`] service.
#[derive(Debug)]
pub struct RollupNodeBuilder {
    /// The rollup configuration.
    pub config: RollupConfig,
    /// The L1 chain configuration.
    pub l1_config_builder: L1ConfigBuilder,
    /// Whether to trust the L2 RPC.
    pub l2_trust_rpc: bool,
    /// Engine builder configuration.
    pub engine_config: EngineConfig,
    /// The [`NetworkConfig`].
    pub p2p_config: NetworkConfig,
    /// An RPC Configuration.
    pub rpc_config: Option<RpcBuilder>,
    /// Optional canonical unsafe blocks client configuration.
    pub blocks_client_config: Option<BlocksClientConfig>,
    /// The [`SequencerConfig`].
    pub sequencer_config: Option<SequencerConfig>,
    /// Whether to run the node in interop mode.
    pub interop_mode: InteropMode,
    /// Optional configuration for Derivation Delegate mode.
    /// When present, the node does not run derivation, instead trusting the configured delegate.
    pub derivation_delegate_config: Option<DerivationDelegateConfig>,
    /// The interop dependency set for this chain.
    pub dependency_set: Option<Arc<DependencySet>>,
}

impl RollupNodeBuilder {
    /// Creates a new [`RollupNodeBuilder`] with the given [`RollupConfig`].
    pub fn new(
        config: RollupConfig,
        l1_config_builder: L1ConfigBuilder,
        l2_trust_rpc: bool,
        engine_config: EngineConfig,
        p2p_config: NetworkConfig,
        rpc_config: Option<RpcBuilder>,
    ) -> Self {
        Self {
            config,
            l1_config_builder,
            l2_trust_rpc,
            engine_config,
            p2p_config,
            rpc_config,
            blocks_client_config: None,
            interop_mode: InteropMode::default(),
            sequencer_config: None,
            derivation_delegate_config: None,
            dependency_set: None,
        }
    }

    /// Sets the interop [`DependencySet`] on the [`RollupNodeBuilder`].
    ///
    /// Must be called when the rollup config schedules the Lagoon hardfork.
    /// When not set, the underlying [`kona_derive::StatefulAttributesBuilder`]
    /// constructor panics on an interop-scheduled chain.
    pub fn with_dependency_set(self, dependency_set: Option<Arc<DependencySet>>) -> Self {
        Self { dependency_set, ..self }
    }

    /// Sets the [`EngineConfig`] on the [`RollupNodeBuilder`].
    pub fn with_engine_config(self, engine_config: EngineConfig) -> Self {
        Self { engine_config, ..self }
    }

    /// Sets the [`RpcBuilder`] on the [`RollupNodeBuilder`].
    pub fn with_rpc_config(self, rpc_config: Option<RpcBuilder>) -> Self {
        Self { rpc_config, ..self }
    }

    /// Sets the canonical unsafe blocks client configuration.
    pub fn with_blocks_client_config(
        self,
        blocks_client_config: Option<BlocksClientConfig>,
    ) -> Self {
        Self { blocks_client_config, ..self }
    }

    /// Appends the [`SequencerConfig`] to the builder.
    pub fn with_sequencer_config(self, sequencer_config: SequencerConfig) -> Self {
        Self { sequencer_config: Some(sequencer_config), ..self }
    }

    /// Sets the Derivation Delegate configuration, trusting the configured delegate for safe head
    /// updates.
    pub fn with_derivation_delegate_config(
        self,
        derivation_delegate_config: Option<DerivationDelegateConfig>,
    ) -> Self {
        Self { derivation_delegate_config, ..self }
    }

    /// Assembles the [`RollupNode`] service.
    ///
    /// ## Panics
    ///
    /// Panics if:
    /// - The L1 provider RPC URL is not set.
    /// - The L1 beacon API URL is not set.
    /// - The L2 provider RPC URL is not set.
    /// - The L2 engine URL is not set.
    /// - The jwt secret is not set.
    /// - The P2P config is not set.
    pub fn build(self) -> RollupNode {
        let mut l1_beacon = OnlineBeaconClient::new_http(self.l1_config_builder.beacon.to_string());
        if let Some(l1_slot_duration) = self.l1_config_builder.slot_duration_override {
            l1_beacon = l1_beacon.with_l1_slot_duration_override(l1_slot_duration);
        }

        let l1_config = L1Config {
            chain_config: Arc::new(self.l1_config_builder.chain_config),
            trust_rpc: self.l1_config_builder.trust_rpc,
            beacon_client: l1_beacon,
            engine_provider: RootProvider::new_http(self.l1_config_builder.rpc_url.clone()),
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

        let p2p_config = self.p2p_config;
        let sequencer_config = self.sequencer_config.unwrap_or_default();

        let derivation_delegate_provider = self.derivation_delegate_config.as_ref().map(|config| {
            DerivationDelegateClient::new(config.l2_cl_url.clone()).expect(
                "Failed to create Derivation Delegate provider despite config being present",
            )
        });

        RollupNode {
            config: rollup_config,
            l1_config,
            interop_mode: self.interop_mode,
            l2_provider,
            l2_trust_rpc: self.l2_trust_rpc,
            engine_config: self.engine_config,
            rpc_builder: self.rpc_config,
            blocks_client_config: self.blocks_client_config,
            p2p_config,
            sequencer_config,
            derivation_delegate_provider,
            dependency_set: self.dependency_set,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::Address;
    use alloy_rpc_types_engine::JwtSecret;
    use discv5::enr::CombinedKey;
    use kona_disc::LocalNode;
    use libp2p::{Multiaddr, multiaddr::Protocol};
    use std::net::{IpAddr, Ipv4Addr};

    fn test_builder() -> RollupNodeBuilder {
        let rollup_config = RollupConfig::default();
        let CombinedKey::Secp256k1(signing_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let discovery = LocalNode::new(signing_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 0, 0);
        let mut gossip = Multiaddr::from(IpAddr::V4(Ipv4Addr::LOCALHOST));
        gossip.push(Protocol::Tcp(0));

        RollupNodeBuilder::new(
            rollup_config.clone(),
            L1ConfigBuilder {
                chain_config: L1ChainConfig::default(),
                trust_rpc: true,
                beacon: Url::parse("http://localhost:5052").unwrap(),
                rpc_url: Url::parse("http://localhost:8545").unwrap(),
                slot_duration_override: None,
            },
            true,
            EngineConfig {
                config: Arc::new(rollup_config.clone()),
                l2_url: Url::parse("http://localhost:8551").unwrap(),
                l2_jwt_secret: JwtSecret::random(),
                l1_url: Url::parse("http://localhost:8545").unwrap(),
                mode: crate::NodeMode::Validator,
            },
            NetworkConfig::new(rollup_config, discovery, gossip, Address::ZERO),
            None,
        )
    }

    #[test]
    fn blocks_client_config_is_disabled_by_default_and_propagated_when_set() {
        assert!(test_builder().build().blocks_client_config.is_none());

        let config = BlocksClientConfig::new(
            Url::parse("ws://sequencer.example:8548").expect("valid blocks endpoint"),
        );
        let node = test_builder().with_blocks_client_config(Some(config.clone())).build();
        assert_eq!(node.blocks_client_config, Some(config));
    }
}
