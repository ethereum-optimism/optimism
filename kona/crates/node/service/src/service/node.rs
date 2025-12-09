//! Contains the [`RollupNode`] implementation.
use crate::{
    ConductorClient, DelayedL1OriginSelectorProvider, DerivationActor, DerivationBuilder,
    DerivationContext, EngineActor, EngineConfig, EngineContext, InteropMode, L1OriginSelector,
    L1WatcherActor, NetworkActor, NetworkBuilder, NetworkConfig, NetworkContext, NodeActor,
    NodeMode, QueuedBlockBuildingClient, QueuedSequencerAdminAPIClient, RpcActor, RpcContext,
    SequencerConfig,
    actors::{
        BlockStream, DerivationInboundChannels, EngineInboundData, NetworkInboundData,
        QueuedUnsafePayloadGossipClient, SequencerActorBuilder,
    },
};
use alloy_eips::BlockNumberOrTag;
use alloy_provider::RootProvider;
use kona_derive::StatefulAttributesBuilder;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_providers_alloy::{AlloyChainProvider, AlloyL2ChainProvider, OnlineBeaconClient};
use kona_rpc::RpcBuilder;
use op_alloy_network::Optimism;
use std::{sync::Arc, time::Duration};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

const DERIVATION_PROVIDER_CACHE_SIZE: usize = 1024;
const HEAD_STREAM_POLL_INTERVAL: u64 = 4;
const FINALIZED_STREAM_POLL_INTERVAL: u64 = 60;

/// The configuration for the L1 chain.
#[derive(Debug, Clone)]
pub struct L1Config {
    /// The L1 chain configuration.
    pub chain_config: Arc<L1ChainConfig>,
    /// Whether to trust the L1 RPC.
    pub trust_rpc: bool,
    /// The L1 beacon client.
    pub beacon_client: OnlineBeaconClient,
    /// The L1 engine provider.
    pub engine_provider: RootProvider,
}

/// The standard implementation of the [RollupNode] service, using the governance approved OP Stack
/// configuration of components.
#[derive(Debug)]
pub struct RollupNode {
    /// The rollup configuration.
    pub(crate) config: Arc<RollupConfig>,
    /// The L1 configuration.
    pub(crate) l1_config: L1Config,
    /// The interop mode for the node.
    pub(crate) interop_mode: InteropMode,
    /// The L2 EL provider.
    pub(crate) l2_provider: RootProvider<Optimism>,
    /// Whether to trust the L2 RPC.
    pub(crate) l2_trust_rpc: bool,
    /// The [`EngineConfig`] for the node.
    pub(crate) engine_config: EngineConfig,
    /// The [`RpcBuilder`] for the node.
    pub(crate) rpc_builder: Option<RpcBuilder>,
    /// The P2P [`NetworkConfig`] for the node.
    pub(crate) p2p_config: NetworkConfig,
    /// The [`SequencerConfig`] for the node.
    pub(crate) sequencer_config: SequencerConfig,
}

impl RollupNode {
    /// The mode of operation for the node.
    const fn mode(&self) -> NodeMode {
        self.engine_config.mode
    }

    /// Returns a derivation builder for the node.
    fn derivation_builder(&self) -> DerivationBuilder {
        DerivationBuilder {
            l1_provider: self.l1_config.engine_provider.clone(),
            l1_trust_rpc: self.l1_config.trust_rpc,
            l1_beacon: self.l1_config.beacon_client.clone(),
            l2_provider: self.l2_provider.clone(),
            l2_trust_rpc: self.l2_trust_rpc,
            rollup_config: self.config.clone(),
            l1_config: self.l1_config.chain_config.clone(),
            interop_mode: self.interop_mode,
        }
    }

    /// Creates a network builder for the node.
    fn network_builder(&self) -> NetworkBuilder {
        NetworkBuilder::from(self.p2p_config.clone())
    }

    /// Returns an engine builder for the node.
    fn engine_config(&self) -> EngineConfig {
        self.engine_config.clone()
    }

    /// Returns an rpc builder for the node.
    fn rpc_builder(&self) -> Option<RpcBuilder> {
        self.rpc_builder.clone()
    }

    /// Returns the sequencer builder for the node.
    fn create_attributes_builder(
        &self,
    ) -> StatefulAttributesBuilder<AlloyChainProvider, AlloyL2ChainProvider> {
        let l1_derivation_provider = AlloyChainProvider::new_with_trust(
            self.l1_config.engine_provider.clone(),
            DERIVATION_PROVIDER_CACHE_SIZE,
            self.l1_config.trust_rpc,
        );
        let l2_derivation_provider = AlloyL2ChainProvider::new_with_trust(
            self.l2_provider.clone(),
            self.config.clone(),
            DERIVATION_PROVIDER_CACHE_SIZE,
            self.l2_trust_rpc,
        );

        StatefulAttributesBuilder::new(
            self.config.clone(),
            self.l1_config.chain_config.clone(),
            l2_derivation_provider,
            l1_derivation_provider,
        )
    }

    /// Starts the rollup node service.
    ///
    /// The rollup node, in validator mode, listens to two sources of information to sync the L2
    /// chain:
    ///
    /// 1. The data availability layer, with a watcher that listens for new updates. L2 inputs (L2
    ///    transaction batches + deposits) are then derived from the DA layer.
    /// 2. The L2 sequencer, which produces unsafe L2 blocks and sends them to the network over p2p
    ///    gossip.
    ///
    /// From these two sources, the node imports `unsafe` blocks from the L2 sequencer, `safe`
    /// blocks from the L2 derivation pipeline into the L2 execution layer via the Engine API,
    /// and finalizes `safe` blocks that it has derived when L1 finalized block updates are
    /// received.
    ///
    /// In sequencer mode, the node is responsible for producing unsafe L2 blocks and sending them
    /// to the network over p2p gossip. The node also listens for L1 finalized block updates and
    /// finalizes `safe` blocks that it has derived when L1 finalized block updates are
    /// received.
    pub async fn start(&self) -> Result<(), String> {
        // Create a global cancellation token for graceful shutdown of tasks.
        let cancellation = CancellationToken::new();

        // 1. CONFIGURE STATE

        // Create the derivation actor.
        let (
            DerivationInboundChannels {
                derivation_signal_tx,
                l1_head_updates_tx,
                engine_l2_safe_head_tx,
                el_sync_complete_tx,
            },
            derivation,
        ) = DerivationActor::new(self.derivation_builder());

        // Create the engine actor.
        let (
            EngineInboundData {
                attributes_tx,
                build_request_tx,
                finalized_l1_block_tx,
                inbound_queries_tx: engine_rpc,
                reset_request_tx,
                rollup_boost_admin_query_tx: rollup_boost_admin_rpc,
                rollup_boost_health_query_tx: rollup_boost_health_rpc,
                seal_request_tx,
                unsafe_block_tx,
                unsafe_head_rx,
            },
            engine,
        ) = EngineActor::new(self.engine_config());

        // Create the p2p actor.
        let (
            NetworkInboundData {
                signer,
                p2p_rpc: network_rpc,
                gossip_payload_tx,
                admin_rpc: net_admin_rpc,
            },
            network,
        ) = NetworkActor::new(self.network_builder());

        // Create the RPC server actor.
        let rpc = self.rpc_builder().map(RpcActor::new);

        let (sequencer_actor_builder, sequencer_admin_api_client) = if self.mode().is_sequencer() {
            // Create the admin API channel
            let (admin_api_tx, admin_api_rx) = mpsc::channel(1024);

            let cfg = self.sequencer_config.clone();

            let builder = SequencerActorBuilder::new()
                .with_active_status(!cfg.sequencer_stopped)
                .with_recovery_mode_status(cfg.sequencer_recovery_mode)
                .with_rollup_config(self.config.clone())
                .with_admin_api_receiver(admin_api_rx);

            (Some(builder), Some(QueuedSequencerAdminAPIClient::new(admin_api_tx)))
        } else {
            (None, None)
        };

        // Create the L1 Watcher actor

        // A channel to send queries about the state of L1.
        let (l1_query_tx, l1_query_rx) = mpsc::channel(1024);

        let head_stream = BlockStream::new_as_stream(
            self.l1_config.engine_provider.clone(),
            BlockNumberOrTag::Latest,
            Duration::from_secs(HEAD_STREAM_POLL_INTERVAL),
        )?;
        let finalized_stream = BlockStream::new_as_stream(
            self.l1_config.engine_provider.clone(),
            BlockNumberOrTag::Finalized,
            Duration::from_secs(FINALIZED_STREAM_POLL_INTERVAL),
        )?;

        // Create the [`L1WatcherActor`]. Previously known as the DA watcher actor.
        let l1_watcher = L1WatcherActor::new(
            self.config.clone(),
            self.l1_config.engine_provider.clone(),
            l1_query_rx,
            l1_head_updates_tx.clone(),
            finalized_l1_block_tx.clone(),
            signer,
            cancellation.clone(),
            head_stream,
            finalized_stream,
        );

        // 2. CONFIGURE DEPENDENCIES

        let sequencer_actor = sequencer_actor_builder.map_or_else(
            || None,
            |mut builder| {
                let cfg = self.sequencer_config.clone();

                let l1_provider = DelayedL1OriginSelectorProvider::new(
                    self.l1_config.engine_provider.clone(),
                    l1_head_updates_tx.subscribe(),
                    cfg.l1_conf_delay,
                );

                let origin_selector = L1OriginSelector::new(self.config.clone(), l1_provider);

                let block_building_client = QueuedBlockBuildingClient {
                    build_request_tx: build_request_tx.expect(
                        "build_request_tx is None in sequencer mode. This should never happen.",
                    ),
                    reset_request_tx: reset_request_tx.clone(),
                    seal_request_tx: seal_request_tx.expect(
                        "seal_request_tx is None in sequencer mode. This should never happen.",
                    ),
                    unsafe_head_rx: unsafe_head_rx.expect(
                        "unsafe_head_rx is None in sequencer mode. This should never happen.",
                    ),
                };

                // Conditionally add conductor if configured
                if let Some(conductor_url) = cfg.conductor_rpc_url {
                    builder = builder.with_conductor(ConductorClient::new_http(conductor_url));
                }

                Some(
                    builder
                        .with_attributes_builder(self.create_attributes_builder())
                        .with_block_building_client(block_building_client)
                        .with_cancellation_token(cancellation.clone())
                        .with_unsafe_payload_gossip_client(QueuedUnsafePayloadGossipClient::new(
                            gossip_payload_tx.clone(),
                        ))
                        .with_origin_selector(origin_selector)
                        .build()
                        .expect("Failed to build SequencerActor"),
                )
            },
        );

        crate::service::spawn_and_wait!(
            cancellation,
            actors = [
                rpc.map(|r| (
                    r,
                    RpcContext {
                        cancellation: cancellation.clone(),
                        p2p_network: network_rpc,
                        network_admin: net_admin_rpc,
                        sequencer_admin: sequencer_admin_api_client,
                        l1_watcher_queries: l1_query_tx,
                        engine_query: engine_rpc,
                        rollup_boost_admin: rollup_boost_admin_rpc,
                        rollup_boost_health: rollup_boost_health_rpc,
                    }
                )),
                sequencer_actor.map(|s| (s, ())),
                Some((
                    network,
                    NetworkContext { blocks: unsafe_block_tx, cancellation: cancellation.clone() }
                )),
                Some((l1_watcher, ())),
                Some((
                    derivation,
                    DerivationContext {
                        reset_request_tx: reset_request_tx.clone(),
                        derived_attributes_tx: attributes_tx,
                        cancellation: cancellation.clone(),
                    }
                )),
                Some((
                    engine,
                    EngineContext {
                        engine_l2_safe_head_tx,
                        sync_complete_tx: el_sync_complete_tx,
                        derivation_signal_tx,
                        cancellation: cancellation.clone(),
                    }
                )),
            ]
        );
        Ok(())
    }
}
