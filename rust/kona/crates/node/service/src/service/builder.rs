//! Contains the builder for the [`RollupNode`].

use crate::{
    EngineConfig, NetworkConfig, RollupNode, SequencerConfig, actors::DerivationDelegateClient,
    service::node::L1Config,
};
use alloy_provider::RootProvider;
use kona_engine::{OpEngineClient, l1_provider, l2_provider};
use op_alloy_network::Optimism;
use std::{sync::Arc, time::Duration};
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
    /// The binary release version shared by CLI, metrics and RPC.
    pub version: &'static str,
    /// The rollup configuration.
    pub config: Arc<RollupConfig>,
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
    /// Optional configuration for Derivation Delegate mode.
    /// When present, the node does not run derivation, instead trusting the configured delegate.
    pub derivation_delegate_config: Option<DerivationDelegateConfig>,
    /// The interop dependency set for this chain.
    pub dependency_set: Option<Arc<DependencySet>>,
}

impl RollupNodeBuilder {
    /// Creates a new [`RollupNodeBuilder`] with the given [`RollupConfig`].
    pub const fn new(
        version: &'static str,
        config: Arc<RollupConfig>,
        l1_config_builder: L1ConfigBuilder,
        l2_trust_rpc: bool,
        engine_config: EngineConfig,
        p2p_config: NetworkConfig,
        rpc_config: Option<RpcBuilder>,
    ) -> Self {
        Self {
            version,
            config,
            l1_config_builder,
            l2_trust_rpc,
            engine_config,
            p2p_config,
            rpc_config,
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

    /// Constructs the standard bounded L1 and authenticated L2 providers, then assembles the node.
    pub fn build(self) -> RollupNode {
        let l1 = l1_provider(self.l1_config_builder.rpc_url.clone(), Duration::from_secs(10));
        let l2 = l2_provider(
            self.engine_config.l2_url.clone(),
            self.engine_config.l2_jwt_secret,
            self.engine_config.timeout,
        );
        self.build_with_providers(l1, l2)
    }

    /// Assembles a node using the caller's providers for every L1/L2 consumer, including startup.
    ///
    /// Provider construction does not perform network I/O. [`RollupNode::start`] checks the
    /// Engine API and reads the initial unsafe block signer through these same providers.
    pub fn build_with_providers(self, l1: RootProvider, l2: RootProvider<Optimism>) -> RollupNode {
        let mut l1_beacon = OnlineBeaconClient::new_http(self.l1_config_builder.beacon.to_string());
        if let Some(l1_slot_duration) = self.l1_config_builder.slot_duration_override {
            l1_beacon = l1_beacon.with_l1_slot_duration_override(l1_slot_duration);
        }

        let l1_config = L1Config {
            chain_config: Arc::new(self.l1_config_builder.chain_config),
            trust_rpc: self.l1_config_builder.trust_rpc,
            beacon_client: l1_beacon,
            engine_provider: l1.clone(),
        };

        let rollup_config = self.config;
        let engine_client = OpEngineClient::new(l1, l2.clone(), rollup_config.clone());
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
            version: self.version,
            config: rollup_config,
            l1_config,
            l2_provider: l2,
            l2_block_buffer,
            l2_trust_rpc: self.l2_trust_rpc,
            engine_client,
            mode: self.engine_config.mode,
            rpc_builder: self.rpc_config,
            p2p_config,
            sequencer_config,
            derivation_delegate_provider,
            dependency_set: self.dependency_set,
        }
    }
}
