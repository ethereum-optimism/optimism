//! Contains the [`RollupNode`] implementation.
use crate::{
    ConductorClient, DelayedL1OriginSelectorProvider, DelegateDerivationActor,
    DelegateDerivationActorBuilder, DerivationActor, DerivationActorBuilder,
    DerivationDelegateClient, DerivationError, EngineActor, EngineActorBuilder, EngineActorRequest,
    EngineConfig, EngineProcessor, EngineRpcProcessor, InteropMode, L1OriginSelector,
    L1WatcherActor, L1WatcherActorBuilder, NetworkActor, NetworkActorBuilder, NetworkBuilder,
    NetworkConfig, NodeActor, NodeMode, QueuedDerivationEngineClient, QueuedEngineDerivationClient,
    QueuedEngineRpcClient, QueuedL1WatcherDerivationClient, QueuedNetworkEngineClient,
    QueuedSequencerAdminAPIClient, QueuedSequencerEngineClient, RollupBoostAdminApiClient,
    RollupBoostHealthRpcClient, RpcActor, RpcActorBuilder, SequencerActor, SequencerActorBuilder,
    SequencerConfig,
    actors::{BlockStream, QueuedUnsafePayloadGossipClient},
};
use alloy_eips::BlockNumberOrTag;
use alloy_provider::RootProvider;
use kona_derive::StatefulAttributesBuilder;
use kona_engine::{Engine, EngineState, OpEngineClient};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_protocol::L2BlockInfo;
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, OnlineBeaconClient, OnlineBlobProvider,
    OnlinePipeline,
};
use kona_rpc::RpcBuilder;
use op_alloy_network::Optimism;
use std::{ops::Not as _, sync::Arc, time::Duration};
use tokio::sync::{mpsc, watch};
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

/// The standard implementation of the [`RollupNode`] service, using the governance approved OP
/// Stack configuration of components.
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
    /// Optional derivation delegate provider.
    pub(crate) derivation_delegate_provider: Option<DerivationDelegateClient>,
}

/// Builder for [`ConfiguredDerivationActor`].
///
/// Selects between delegated and normal derivation based on `RollupNode` configuration.
enum ConfiguredDerivationActorBuilder {
    Delegate(Box<DelegateDerivationActorBuilder<QueuedDerivationEngineClient>>),
    Normal(Box<DerivationActorBuilder<QueuedDerivationEngineClient, OnlinePipeline>>),
}

/// A RollupNode-level derivation actor wrapper.
///
/// This type selects the concrete derivation actor implementation
/// based on `RollupNode` configuration.
///
/// Both inner variants use `InboundData = ()` because their request channels
/// are pre-created in `RollupNode::start()` to resolve the Engine <-> Derivation
/// circular dependency. If a future variant needs inbound data, this enum
/// would need to be reworked.
enum ConfiguredDerivationActor {
    Delegate(Box<DelegateDerivationActor<QueuedDerivationEngineClient>>),
    Normal(Box<DerivationActor<QueuedDerivationEngineClient, OnlinePipeline>>),
}

#[async_trait::async_trait]
impl NodeActor for ConfiguredDerivationActor {
    type Error = DerivationError;
    type Builder = ConfiguredDerivationActorBuilder;
    type InboundData = ();

    async fn init(builder: Self::Builder) -> Result<(Self::InboundData, Self), Self::Error> {
        match builder {
            ConfiguredDerivationActorBuilder::Delegate(b) => {
                let ((), actor) = DelegateDerivationActor::init(*b).await?;
                Ok(((), Self::Delegate(Box::new(actor))))
            }
            ConfiguredDerivationActorBuilder::Normal(b) => {
                let ((), actor) = DerivationActor::init(*b).await?;
                Ok(((), Self::Normal(Box::new(actor))))
            }
        }
    }

    async fn step(&mut self) -> Result<(), Self::Error> {
        match self {
            Self::Delegate(a) => a.step().await,
            Self::Normal(a) => a.step().await,
        }
    }
}

impl RollupNode {
    /// The mode of operation for the node.
    const fn mode(&self) -> NodeMode {
        self.engine_config.mode
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

    async fn create_pipeline(&self) -> OnlinePipeline {
        // Create the caching L1/L2 EL providers for derivation.
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

        match self.interop_mode {
            InteropMode::Polled => OnlinePipeline::new_polled(
                self.config.clone(),
                self.l1_config.chain_config.clone(),
                OnlineBlobProvider::init(self.l1_config.beacon_client.clone()).await,
                l1_derivation_provider,
                l2_derivation_provider,
            ),
            InteropMode::Indexed => OnlinePipeline::new_indexed(
                self.config.clone(),
                self.l1_config.chain_config.clone(),
                OnlineBlobProvider::init(self.l1_config.beacon_client.clone()).await,
                l1_derivation_provider,
                l2_derivation_provider,
            ),
        }
    }

    /// Helper function to assemble the [`EngineActorBuilder`] since there are many structs created
    /// that are not relevant to other actors or logic.
    /// Note: ignoring complex type warning. This type only pertains to this function, so it is
    /// better to have the full type here than have to piece it together from multiple type defs.
    #[allow(clippy::type_complexity)]
    fn create_engine_actor_builder(
        &self,
        engine_request_rx: mpsc::Receiver<EngineActorRequest>,
        derivation_client: QueuedEngineDerivationClient,
        unsafe_head_tx: watch::Sender<L2BlockInfo>,
    ) -> Result<
        EngineActorBuilder<
            EngineProcessor<
                OpEngineClient<RootProvider, RootProvider<Optimism>>,
                QueuedEngineDerivationClient,
            >,
            EngineRpcProcessor<OpEngineClient<RootProvider, RootProvider<Optimism>>>,
        >,
        String,
    > {
        let engine_state = EngineState::default();
        let (engine_state_tx, engine_state_rx) = watch::channel(engine_state);
        let (engine_queue_length_tx, engine_queue_length_rx) = watch::channel(0);
        let engine = Engine::new(engine_state, engine_state_tx, engine_queue_length_tx);

        let engine_client = Arc::new(self.engine_config().build_engine_client().map_err(|e| {
            error!(target: "service", error = ?e, "engine client build failed");
            format!("Engine client build failed: {e:?}")
        })?);

        let engine_processor = EngineProcessor::new(
            engine_client.clone(),
            self.config.clone(),
            derivation_client,
            engine,
            self.mode().is_sequencer().then_some(unsafe_head_tx),
        );

        let engine_rpc_processor = EngineRpcProcessor::new(
            engine_client.clone(),
            engine_client.rollup_boost.clone(),
            self.config.clone(),
            engine_state_rx,
            engine_queue_length_rx,
        );

        Ok(EngineActorBuilder {
            inbound_request_rx: engine_request_rx,
            engine_receiver: engine_processor,
            rpc_receiver: engine_rpc_processor,
        })
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

        // Pre-create channels for the Engine <-> Derivation circular dependency.
        let (derivation_actor_request_tx, derivation_actor_request_rx) = mpsc::channel(1024);
        let (engine_actor_request_tx, engine_actor_request_rx) = mpsc::channel(1024);
        let (unsafe_head_tx, unsafe_head_rx) = watch::channel(L2BlockInfo::default());

        // Init the network actor.
        let (network_inbound, network) = NetworkActor::init(NetworkActorBuilder {
            engine_client: QueuedNetworkEngineClient {
                engine_actor_request_tx: engine_actor_request_tx.clone(),
            },
            network_builder: self.network_builder(),
        })
        .await
        .map_err(|e| format!("Network actor init failed: {e:?}"))?;

        // Create L1 block streams for the L1 watcher.
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

        // Init the L1 watcher actor.
        let (l1_inbound, l1_watcher) = L1WatcherActor::init(L1WatcherActorBuilder {
            rollup_config: self.config.clone(),
            l1_provider: self.l1_config.engine_provider.clone(),
            derivation_client: QueuedL1WatcherDerivationClient {
                derivation_actor_request_tx: derivation_actor_request_tx.clone(),
            },
            block_signer_sender: network_inbound.signer.clone(),
            head_stream,
            finalized_stream,
        })
        .await
        .map_err(|e| format!("L1 watcher actor init failed: {e:?}"))?;

        // Init the engine actor.
        let engine_builder = self.create_engine_actor_builder(
            engine_actor_request_rx,
            QueuedEngineDerivationClient::new(derivation_actor_request_tx.clone()),
            unsafe_head_tx,
        )?;
        let (_, engine) = EngineActor::init(engine_builder)
            .await
            .map_err(|e| format!("Engine actor init failed: {e:?}"))?;

        // Select the concrete derivation actor implementation based on
        // RollupNode configuration.
        let derivation_builder = if let Some(provider) = self.derivation_delegate_provider.clone() {
            let l1_provider = AlloyChainProvider::new(
                self.l1_config.engine_provider.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            );
            ConfiguredDerivationActorBuilder::Delegate(Box::new(DelegateDerivationActorBuilder {
                engine_client: QueuedDerivationEngineClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                inbound_request_rx: derivation_actor_request_rx,
                derivation_delegate_provider: provider,
                l1_provider,
            }))
        } else {
            ConfiguredDerivationActorBuilder::Normal(Box::new(DerivationActorBuilder {
                engine_client: QueuedDerivationEngineClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                inbound_request_rx: derivation_actor_request_rx,
                pipeline: self.create_pipeline().await,
            }))
        };
        let (_, derivation) = ConfiguredDerivationActor::init(derivation_builder)
            .await
            .map_err(|e| format!("Derivation actor init failed: {e:?}"))?;

        // Create delayed L1 origin selector for the sequencer.
        let delayed_l1_provider = DelayedL1OriginSelectorProvider::new(
            self.l1_config.engine_provider.clone(),
            l1_inbound.l1_head_updates_rx.clone(),
            self.sequencer_config.l1_conf_delay,
        );
        let delayed_origin_selector =
            L1OriginSelector::new(self.config.clone(), delayed_l1_provider);

        // Conditionally add conductor if configured
        let conductor =
            self.sequencer_config.conductor_rpc_url.clone().map(ConductorClient::new_http);

        // Create the sequencer if needed
        let (sequencer, sequencer_admin_client) = if self.mode().is_sequencer() {
            let sequencer_engine_client = QueuedSequencerEngineClient {
                engine_actor_request_tx: engine_actor_request_tx.clone(),
                unsafe_head_rx,
            };

            let queued_gossip_client =
                QueuedUnsafePayloadGossipClient::new(network_inbound.gossip_payload_tx.clone());

            let (seq_inbound, seq_actor) = SequencerActor::init(SequencerActorBuilder {
                attributes_builder: self.create_attributes_builder(),
                conductor,
                is_active: self.sequencer_config.sequencer_stopped.not(),
                in_recovery_mode: self.sequencer_config.sequencer_recovery_mode,
                origin_selector: delayed_origin_selector,
                rollup_config: self.config.clone(),
                engine_client: sequencer_engine_client,
                unsafe_payload_gossip_client: queued_gossip_client,
            })
            .await
            .map_err(|e| format!("Sequencer actor init failed: {e:?}"))?;

            (Some(seq_actor), Some(QueuedSequencerAdminAPIClient::new(seq_inbound.admin_api_tx)))
        } else {
            (None, None)
        };

        // Create the RPC server actor.
        let rpc = if let Some(b) = self.rpc_builder() {
            let (_, rpc_actor) = RpcActor::init(RpcActorBuilder {
                config: b,
                engine_rpc_client: QueuedEngineRpcClient::new(engine_actor_request_tx.clone()),
                rollup_boost_admin_rpc_client: RollupBoostAdminApiClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                rollup_boost_health_rpc_client: RollupBoostHealthRpcClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                sequencer_admin_rpc_client: sequencer_admin_client,
                p2p_network: network_inbound.p2p_rpc.clone(),
                network_admin: network_inbound.admin_rpc.clone(),
                l1_watcher_queries: l1_inbound.query_tx.clone(),
            })
            .await
            .map_err(|e| format!("RPC actor init failed: {e:?}"))?;
            Some(rpc_actor)
        } else {
            None
        };

        crate::service::spawn_and_wait!(
            cancellation,
            actors =
                [rpc, sequencer, Some(network), Some(l1_watcher), Some(derivation), Some(engine),]
        );
        Ok(())
    }
}
