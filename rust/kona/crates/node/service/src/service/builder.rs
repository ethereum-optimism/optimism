//! Contains the builder for the [`RollupNode`].

use crate::{EngineConfig, NetworkConfig, RollupNode, SequencerConfig, service::node::L1Config};
use std::sync::Arc;
use url::Url;

use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_providers_alloy::{AlloyChainProvider, OnlineBeaconClient};
use kona_providers_local::BufferedL2Provider;
use kona_rpc::RpcBuilder;
use kona_safedb::SafeDb;

/// The [`L1ConfigBuilder`] is used to construct a [`L1Config`].
#[derive(Debug)]
pub struct L1ConfigBuilder {
    /// The L1 chain configuration.
    pub chain_config: L1ChainConfig,
    /// The node's L1 client, shared by every component that reads L1.
    ///
    /// The node does not bound its L1 requests itself, so this client's transport must: a stalled
    /// endpoint would otherwise hang L1 traversal, the sequencer's origin preparation and the
    /// unsafe block signer refresh. kona-node wraps its transport with
    /// [`kona_engine::with_deadline`].
    pub provider: AlloyChainProvider,
    /// The L1 beacon API.
    pub beacon: Url,
    /// The duration in seconds of an L1 slot. This can be used to hardcode a fixed slot
    /// duration if the l1-beacon's slot configuration is not available.
    pub slot_duration_override: Option<u64>,
}

/// The [`RollupNodeBuilder`] is used to construct a [`RollupNode`] service.
#[derive(Debug)]
pub struct RollupNodeBuilder {
    /// The application version reported by the RPC server.
    pub version: String,
    /// The rollup configuration.
    pub config: RollupConfig,
    /// The L1 chain configuration.
    pub l1_config_builder: L1ConfigBuilder,
    /// Engine builder configuration.
    pub engine_config: EngineConfig,
    /// The [`NetworkConfig`].
    pub p2p_config: NetworkConfig,
    /// An RPC Configuration.
    pub rpc_config: Option<RpcBuilder>,
    /// The [`SequencerConfig`].
    pub sequencer_config: Option<SequencerConfig>,
    /// The L2 CL RPC to delegate derivation to. It must expose `optimism_syncStatus`.
    /// When present, the node does not run derivation, instead trusting the delegate.
    pub derivation_delegate_url: Option<Url>,
    /// The interop dependency set for this chain.
    pub dependency_set: Option<Arc<DependencySet>>,
    /// Records the safe head reached as of each L1 block when configured.
    pub safe_db: Option<Arc<dyn SafeDb>>,
    /// Number of L1 confirmations required before deriving L2 blocks.
    pub verifier_l1_confs: u64,
}

impl RollupNodeBuilder {
    /// Creates a new [`RollupNodeBuilder`] with the application version and [`RollupConfig`].
    pub const fn new(
        version: String,
        config: RollupConfig,
        l1_config_builder: L1ConfigBuilder,
        engine_config: EngineConfig,
        p2p_config: NetworkConfig,
        rpc_config: Option<RpcBuilder>,
    ) -> Self {
        Self {
            version,
            config,
            l1_config_builder,
            engine_config,
            p2p_config,
            rpc_config,
            sequencer_config: None,
            derivation_delegate_url: None,
            dependency_set: None,
            safe_db: None,
            verifier_l1_confs: 0,
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

    /// Sets the safe head database that records derivation progress per L1 block.
    pub fn with_safe_db(self, safe_db: Option<Arc<dyn SafeDb>>) -> Self {
        Self { safe_db, ..self }
    }

    /// Sets the L1 confirmation depth for derivation in both node modes.
    pub fn with_verifier_l1_confs(self, verifier_l1_confs: u64) -> Self {
        Self { verifier_l1_confs, ..self }
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

    /// Sets the L2 CL RPC to delegate derivation to, trusting it for safe head updates.
    pub fn with_derivation_delegate_url(self, derivation_delegate_url: Option<Url>) -> Self {
        Self { derivation_delegate_url, ..self }
    }

    /// Assembles the [`RollupNode`] service.
    pub fn build(self) -> RollupNode {
        let mut l1_beacon = OnlineBeaconClient::new_http(self.l1_config_builder.beacon.to_string());
        if let Some(l1_slot_duration) = self.l1_config_builder.slot_duration_override {
            l1_beacon = l1_beacon.with_l1_slot_duration_override(l1_slot_duration);
        }

        let l1_config = L1Config {
            chain_config: Arc::new(self.l1_config_builder.chain_config),
            beacon_client: l1_beacon,
            provider: self.l1_config_builder.provider,
        };

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

        RollupNode {
            version: self.version,
            config: rollup_config,
            l1_config,
            l2_block_buffer,
            engine_config: self.engine_config,
            rpc_builder: self.rpc_config,
            p2p_config,
            sequencer_config,
            derivation_delegate_url: self.derivation_delegate_url,
            dependency_set: self.dependency_set,
            safe_db: self.safe_db,
            verifier_l1_confs: self.verifier_l1_confs,
        }
    }
}
