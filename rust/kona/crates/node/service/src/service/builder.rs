//! Contains the builder for the [`RollupNode`].

use crate::{
    EngineConfig, InteropMode, NetworkBuilder, NetworkConfig, RollupNode, SequencerConfig,
    actors::DerivationDelegateClient,
    service::node::{DapFactory, HandlerFactory, L1Config},
};
use alloy_primitives::{Address, Bytes};
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
};
use core::fmt::Debug;
use http_body_util::Full;
use op_alloy_network::Optimism;
use std::sync::Arc;
use tokio::sync::watch;
use tower::ServiceBuilder;
use url::Url;

use kona_derive::{DataAvailabilityProvider, EthereumDataSource};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_gossip::{BlockHandler, Handler};
use kona_interop::DependencySet;
use kona_providers_alloy::{OnlineBeaconClient, OnlineDataProvider};
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
///
/// Generic over the data-availability provider (`DAP`) and gossip handler
/// (`H`) so binaries can plug in custom implementations via
/// [`Self::with_data_availability_provider`] and [`Self::with_block_handler`].
/// Defaults match upstream's stock behaviour and keep every existing caller
/// compiling unchanged.
pub struct RollupNodeBuilder<DAP = OnlineDataProvider, H = BlockHandler>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + 'static,
    H: Handler + Clone + Send + 'static,
{
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
    /// The [`SequencerConfig`].
    pub sequencer_config: Option<SequencerConfig>,
    /// Whether to run the node in interop mode.
    pub interop_mode: InteropMode,
    /// Optional configuration for Derivation Delegate mode.
    /// When present, the node does not run derivation, instead trusting the configured delegate.
    pub derivation_delegate_config: Option<DerivationDelegateConfig>,
    /// The interop dependency set for this chain.
    pub dependency_set: Option<Arc<DependencySet>>,
    /// Caller-supplied DAP factory. `None` ⇒ `build()` installs the
    /// default `OnlineDataProvider` factory.
    dap_factory: Option<DapFactory<DAP>>,
    /// Caller-supplied gossip-handler factory. `None` ⇒ `build()` installs
    /// the default `BlockHandler` factory.
    handler_factory: Option<HandlerFactory<H>>,
}

impl Debug for RollupNodeBuilder {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("RollupNodeBuilder")
            .field("config", &self.config)
            .field("l1_config_builder", &self.l1_config_builder)
            .field("l2_trust_rpc", &self.l2_trust_rpc)
            .field("engine_config", &self.engine_config)
            .field("p2p_config", &self.p2p_config)
            .field("rpc_config", &self.rpc_config)
            .field("sequencer_config", &self.sequencer_config)
            .field("interop_mode", &self.interop_mode)
            .field("derivation_delegate_config", &self.derivation_delegate_config)
            .field("dependency_set", &self.dependency_set)
            .field("dap_factory_set", &self.dap_factory.is_some())
            .field("handler_factory_set", &self.handler_factory.is_some())
            .finish()
    }
}

impl RollupNodeBuilder<OnlineDataProvider, BlockHandler> {
    /// Creates a new [`RollupNodeBuilder`] with the given [`RollupConfig`].
    /// The returned builder is parameterised on the default
    /// `OnlineDataProvider` and `BlockHandler`; flip those types via
    /// [`Self::with_data_availability_provider`] /
    /// [`Self::with_block_handler`].
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
            interop_mode: InteropMode::default(),
            sequencer_config: None,
            derivation_delegate_config: None,
            dependency_set: None,
            dap_factory: None,
            handler_factory: None,
        }
    }
}

impl<DAP, H> RollupNodeBuilder<DAP, H>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + Clone + 'static,
    H: Handler + Clone + Send + 'static,
{
    /// Sets the interop [`DependencySet`] on the [`RollupNodeBuilder`].
    ///
    /// Must be called when the rollup config schedules the Interop hardfork.
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

    /// Replace the default `OnlineDataProvider` with a caller-supplied DAP.
    /// The factory closure runs inside [`RollupNode::start`] once the L1
    /// chain provider and blob provider have been constructed.
    ///
    /// Flips the builder's `DAP` type parameter, so the resulting
    /// [`RollupNode`] is `RollupNode<NewDAP, H>`.
    pub fn with_data_availability_provider<NewDAP, F>(
        self,
        factory: F,
    ) -> RollupNodeBuilder<NewDAP, H>
    where
        NewDAP: DataAvailabilityProvider + Debug + Send + Sync + Clone + 'static,
        F: FnOnce(
                Arc<RollupConfig>,
                kona_providers_alloy::AlloyChainProvider,
                kona_providers_alloy::OnlineBlobProvider<OnlineBeaconClient>,
            ) -> NewDAP
            + Send
            + 'static,
    {
        RollupNodeBuilder {
            config: self.config,
            l1_config_builder: self.l1_config_builder,
            l2_trust_rpc: self.l2_trust_rpc,
            engine_config: self.engine_config,
            p2p_config: self.p2p_config,
            rpc_config: self.rpc_config,
            sequencer_config: self.sequencer_config,
            interop_mode: self.interop_mode,
            derivation_delegate_config: self.derivation_delegate_config,
            dependency_set: self.dependency_set,
            dap_factory: Some(Box::new(factory)),
            handler_factory: self.handler_factory,
        }
    }

    /// Replace the default `BlockHandler` with a caller-supplied
    /// [`Handler`] impl plus its paired `watch::Sender<Address>` (the
    /// sender is retained on the node so SystemConfig-driven signer
    /// updates still flow even when the custom handler ignores them).
    ///
    /// Flips the builder's `H` type parameter, so the resulting
    /// [`RollupNode`] is `RollupNode<DAP, NewH>`.
    pub fn with_block_handler<NewH>(
        self,
        handler: NewH,
        signer_tx: watch::Sender<Address>,
    ) -> RollupNodeBuilder<DAP, NewH>
    where
        NewH: Handler + Clone + Send + 'static,
    {
        RollupNodeBuilder {
            config: self.config,
            l1_config_builder: self.l1_config_builder,
            l2_trust_rpc: self.l2_trust_rpc,
            engine_config: self.engine_config,
            p2p_config: self.p2p_config,
            rpc_config: self.rpc_config,
            sequencer_config: self.sequencer_config,
            interop_mode: self.interop_mode,
            derivation_delegate_config: self.derivation_delegate_config,
            dependency_set: self.dependency_set,
            dap_factory: self.dap_factory,
            handler_factory: Some(Box::new(move |_nb: &NetworkBuilder| (handler, signer_tx))),
        }
    }
}

// =========================================================================
// build() — only on the default-types builder, since the default DAP /
// handler factory closures it installs return concrete OnlineDataProvider /
// BlockHandler. For custom-typed builders, call `build_with_overrides`.
// =========================================================================

impl RollupNodeBuilder<OnlineDataProvider, BlockHandler> {
    /// Assembles the [`RollupNode`] service with default DAP / handler.
    ///
    /// ## Panics
    ///
    /// Panics if any of the L1 / L2 / engine URLs / JWT secret / P2P
    /// config is not set.
    pub fn build(mut self) -> RollupNode<OnlineDataProvider, BlockHandler> {
        // Install the default DAP factory if none has been set.
        if self.dap_factory.is_none() {
            self.dap_factory = Some(Box::new(|cfg, chain_provider, blob_provider| {
                EthereumDataSource::new_from_parts(chain_provider, blob_provider, &cfg)
            }));
        }
        // Install the default handler factory if none has been set.
        if self.handler_factory.is_none() {
            self.handler_factory =
                Some(Box::new(|nb: &NetworkBuilder| nb.default_handler()));
        }
        self.build_inner()
    }
}

impl<DAP, H> RollupNodeBuilder<DAP, H>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + Clone + 'static,
    H: Handler + Clone + Send + 'static,
{
    /// Assembles the [`RollupNode`] service. Both factories must have been
    /// installed; on a builder created via [`Self::new`] without flipping
    /// the type parameters, [`Self::build`] handles this automatically.
    /// On a builder with custom DAP / handler types, the corresponding
    /// `with_*` setters install the factories.
    pub fn build_with_overrides(self) -> RollupNode<DAP, H> {
        assert!(
            self.dap_factory.is_some(),
            "build_with_overrides called without with_data_availability_provider — \
             custom-typed builder must install a DAP factory before building.",
        );
        assert!(
            self.handler_factory.is_some(),
            "build_with_overrides called without with_block_handler — \
             custom-typed builder must install a handler factory before building.",
        );
        self.build_inner()
    }

    fn build_inner(self) -> RollupNode<DAP, H> {
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
            p2p_config,
            sequencer_config,
            derivation_delegate_provider,
            dependency_set: self.dependency_set,
            dap_factory: self.dap_factory,
            handler_factory: self.handler_factory,
        }
    }
}
