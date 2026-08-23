//! Contains the builder for the [`RollupNode`].

use crate::{
    EngineConfig, NetworkConfig, RollupNode, SequencerConfig, SharedRpcServerLauncher,
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

use kona_engine::SharedDenyList;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_providers_alloy::OnlineBeaconClient;
use kona_providers_local::BufferedL2Provider;
use kona_rpc::RpcBuilder;
use kona_safedb::{DisabledDatabase, SharedSafeDb};

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
    /// How often, in seconds, to poll the L1 for epoch updates (finalized-block changes).
    ///
    /// op-node's `--l1.epoch-poll-interval`. `None` keeps
    /// [`L1Config::DEFAULT_L1_EPOCH_POLL_INTERVAL`].
    pub l1_epoch_poll_interval: Option<u64>,
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
    /// The launcher this chain's RPC module set is handed to, when the host supplies one.
    pub rpc_launcher: Option<SharedRpcServerLauncher>,
    /// The [`SequencerConfig`].
    pub sequencer_config: Option<SequencerConfig>,
    /// Optional configuration for Derivation Delegate mode.
    /// When present, the node does not run derivation, instead trusting the configured delegate.
    pub derivation_delegate_config: Option<DerivationDelegateConfig>,
    /// The interop dependency set for this chain.
    pub dependency_set: Option<Arc<DependencySet>>,
    /// Whether this chain's cross-safe head is fed by an external cross-chain verifier.
    pub external_cross_safe: bool,
    /// The safe-head database the chain controller records local-safe advances into.
    pub safe_db: SharedSafeDb,
    /// The super-authority deny list the engine consults, when the node runs under one.
    pub deny_list: Option<SharedDenyList>,
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
            rpc_launcher: None,
            sequencer_config: None,
            derivation_delegate_config: None,
            dependency_set: None,
            external_cross_safe: false,
            safe_db: Arc::new(DisabledDatabase),
            deny_list: None,
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

    /// Feeds this chain's cross-safe head from an external cross-chain verifier.
    ///
    /// Set by an interop host, which then takes the chain's
    /// [`CrossSafePromoter`](kona_engine::CrossSafePromoter) out of the
    /// [`ComposedChain`](crate::ComposedChain) and is thereafter the only writer of that head.
    /// Left unset, the cross-safe head trivially follows local-safe, which is standalone
    /// kona-node's behaviour and stays byte-identical.
    pub fn with_external_cross_safe(self, external_cross_safe: bool) -> Self {
        Self { external_cross_safe, ..self }
    }

    /// Sets the safe-head database the chain controller records local-safe advances into.
    ///
    /// Needed by a host that has to answer *which L1 block made an L2 block safe* for a block
    /// behind the local-safe head — the live engine state holds that pairing only for the head
    /// itself. Left unset, the recording writes are no-ops.
    pub fn with_safe_db(self, safe_db: SharedSafeDb) -> Self {
        Self { safe_db, ..self }
    }

    /// Sets the super-authority deny list on the [`RollupNodeBuilder`].
    ///
    /// The engine consults it before adopting or inserting blocks, and it is what turns a
    /// post-invalidation rebuild into the deposits-only replacement. A node built without one
    /// denies nothing.
    pub fn with_deny_list(self, deny_list: Option<SharedDenyList>) -> Self {
        Self { deny_list, ..self }
    }

    /// Sets the [`EngineConfig`] on the [`RollupNodeBuilder`].
    pub fn with_engine_config(self, engine_config: EngineConfig) -> Self {
        Self { engine_config, ..self }
    }

    /// Sets the [`RpcBuilder`] on the [`RollupNodeBuilder`].
    pub fn with_rpc_config(self, rpc_config: Option<RpcBuilder>) -> Self {
        Self { rpc_config, ..self }
    }

    /// Hands this chain's RPC module set to `rpc_launcher` instead of binding it to
    /// [`RpcBuilder::socket`].
    ///
    /// Set by a multi-chain host, which serves every chain's module set from one socket and routes
    /// to them by chain id: N chains each binding their own socket is N addresses for one process,
    /// and there is only one address for a caller to have been told about. The
    /// [`RpcBuilder`] still decides *whether* an RPC is built and which
    /// namespaces it carries; only where it is served changes.
    ///
    /// Left unset, the module set is bound to its own socket by
    /// [`JsonrpseeServerLauncher`](crate::JsonrpseeServerLauncher), which is standalone
    /// kona-node's behaviour and stays byte-identical.
    pub fn with_rpc_launcher(self, rpc_launcher: SharedRpcServerLauncher) -> Self {
        Self { rpc_launcher: Some(rpc_launcher), ..self }
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
        let mut l1_beacon = OnlineBeaconClient::new_http(self.l1_config_builder.beacon.to_string())
            .with_chain_id(self.config.l2_chain_id.id());
        if let Some(l1_slot_duration) = self.l1_config_builder.slot_duration_override {
            l1_beacon = l1_beacon.with_l1_slot_duration_override(l1_slot_duration);
        }

        let l1_config = L1Config {
            chain_config: Arc::new(self.l1_config_builder.chain_config),
            trust_rpc: self.l1_config_builder.trust_rpc,
            beacon_client: l1_beacon,
            engine_provider: RootProvider::new_http(self.l1_config_builder.rpc_url.clone()),
            l1_epoch_poll_interval: self
                .l1_config_builder
                .l1_epoch_poll_interval
                .map_or(L1Config::DEFAULT_L1_EPOCH_POLL_INTERVAL, std::time::Duration::from_secs),
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
        let sequencer_config = self.sequencer_config.unwrap_or_default();

        let derivation_delegate_provider = self.derivation_delegate_config.as_ref().map(|config| {
            DerivationDelegateClient::new(config.l2_cl_url.clone()).expect(
                "Failed to create Derivation Delegate provider despite config being present",
            )
        });

        RollupNode {
            config: rollup_config,
            l1_config,
            l2_provider,
            l2_block_buffer,
            l2_trust_rpc: self.l2_trust_rpc,
            engine_config: self.engine_config,
            rpc_builder: self.rpc_config,
            rpc_launcher: self.rpc_launcher,
            p2p_config,
            sequencer_config,
            derivation_delegate_provider,
            dependency_set: self.dependency_set,
            external_cross_safe: self.external_cross_safe,
            safe_db: self.safe_db,
            deny_list: self.deny_list,
        }
    }
}
