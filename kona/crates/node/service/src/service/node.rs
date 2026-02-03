//! Contains the [`RollupNode`] implementation.
use crate::{
    ConductorClient, DelayedL1OriginSelectorProvider, DelegateDerivationActor, DerivationActor,
    DerivationDelegateClient, DerivationError, EngineActor, EngineActorRequest, EngineConfig,
    EngineProcessor, EngineRpcProcessor, InteropMode, L1OriginSelector, L1WatcherActor,
    NetworkActor, NetworkBuilder, NetworkConfig, NodeActor, NodeMode, QueuedDerivationEngineClient,
    QueuedEngineDerivationClient, QueuedEngineRpcClient, QueuedL1WatcherDerivationClient,
    QueuedNetworkEngineClient, QueuedSequencerAdminAPIClient, QueuedSequencerEngineClient,
    RollupBoostAdminApiClient, RollupBoostHealthRpcClient, RpcActor, RpcContext, SequencerActor,
    SequencerConfig,
    actors::{BlockStream, NetworkInboundData, QueuedUnsafePayloadGossipClient},
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
    /// Optional derivation delegate provider.
    pub(crate) derivation_delegate_provider: Option<DerivationDelegateClient>,
}

/// A RollupNode-level derivation actor wrapper.
///
/// This type selects the concrete derivation actor implementation
/// based on RollupNode configuration.
///
/// It is not intended to be generic or reusable outside the
/// RollupNode wiring logic.
enum ConfiguredDerivationActor {
    Delegate(Box<DelegateDerivationActor<QueuedDerivationEngineClient>>),
    Normal(Box<DerivationActor<QueuedDerivationEngineClient, OnlinePipeline>>),
}

#[async_trait::async_trait]
impl NodeActor for ConfiguredDerivationActor
where
    DelegateDerivationActor<QueuedDerivationEngineClient>:
        NodeActor<StartData = (), Error = DerivationError>,
    DerivationActor<QueuedDerivationEngineClient, OnlinePipeline>:
        NodeActor<StartData = (), Error = DerivationError>,
{
    type StartData = ();
    type Error = DerivationError;

    async fn start(self, ctx: ()) -> Result<(), Self::Error> {
        match self {
            Self::Delegate(a) => a.start(ctx).await,
            Self::Normal(a) => a.start(ctx).await,
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

    /// Helper function to assemble the [`EngineActor`] since there are many structs created that
    /// are not relevant to other actors or logic.
    /// Note: ignoring complex type warning. This type only pertains to this function, so it is
    /// better to have the full type here than have to piece it together from multiple type defs.
    #[allow(clippy::type_complexity)]
    fn create_engine_actor(
        &self,
        cancellation_token: CancellationToken,
        engine_request_rx: mpsc::Receiver<EngineActorRequest>,
        derivation_client: QueuedEngineDerivationClient,
        unsafe_head_tx: watch::Sender<L2BlockInfo>,
    ) -> Result<
        EngineActor<
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
            if self.mode().is_sequencer() { Some(unsafe_head_tx) } else { None },
        );

        let engine_rpc_processor = EngineRpcProcessor::new(
            engine_client.clone(),
            engine_client.rollup_boost.clone(),
            self.config.clone(),
            engine_state_rx,
            engine_queue_length_rx,
        );

        Ok(EngineActor::new(
            cancellation_token,
            engine_request_rx,
            engine_processor,
            engine_rpc_processor,
        ))
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

        let (derivation_actor_request_tx, derivation_actor_request_rx) = mpsc::channel(1024);

        let (engine_actor_request_tx, engine_actor_request_rx) = mpsc::channel(1024);
        let (unsafe_head_tx, unsafe_head_rx) = watch::channel(L2BlockInfo::default());

        let engine_actor = self.create_engine_actor(
            cancellation.clone(),
            engine_actor_request_rx,
            QueuedEngineDerivationClient::new(derivation_actor_request_tx.clone()),
            unsafe_head_tx,
        )?;

        // Select the concrete derivation actor implementation based on
        // RollupNode configuration.
        let derivation: ConfiguredDerivationActor = if let Some(provider) =
            self.derivation_delegate_provider.clone()
        {
            // L1 Provider for sanity checking Derivation Delegation
            let l1_provider = AlloyChainProvider::new(
                self.l1_config.engine_provider.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            );
            ConfiguredDerivationActor::Delegate(Box::new(DelegateDerivationActor::<_>::new(
                QueuedDerivationEngineClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                cancellation.clone(),
                derivation_actor_request_rx,
                provider,
                l1_provider,
            )))
        } else {
            ConfiguredDerivationActor::Normal(Box::new(DerivationActor::<_, OnlinePipeline>::new(
                QueuedDerivationEngineClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                cancellation.clone(),
                derivation_actor_request_rx,
                self.create_pipeline().await,
            )))
        };

        // Create the p2p actor.
        let (
            NetworkInboundData {
                signer,
                p2p_rpc: network_rpc,
                gossip_payload_tx,
                admin_rpc: net_admin_rpc,
            },
            network,
        ) = NetworkActor::new(
            QueuedNetworkEngineClient { engine_actor_request_tx: engine_actor_request_tx.clone() },
            cancellation.clone(),
            self.network_builder(),
        );

        let (l1_head_updates_tx, l1_head_updates_rx) = watch::channel(None);
        let delayed_l1_provider = DelayedL1OriginSelectorProvider::new(
            self.l1_config.engine_provider.clone(),
            l1_head_updates_rx,
            self.sequencer_config.l1_conf_delay,
        );

        let delayed_origin_selector =
            L1OriginSelector::new(self.config.clone(), delayed_l1_provider);

        // Conditionally add conductor if configured
        let conductor =
            self.sequencer_config.conductor_rpc_url.clone().map(ConductorClient::new_http);

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
            QueuedL1WatcherDerivationClient { derivation_actor_request_tx },
            signer,
            cancellation.clone(),
            head_stream,
            finalized_stream,
        );

        // Create the sequencer if needed
        let (sequencer_actor, sequencer_admin_client) = if self.mode().is_sequencer() {
            let sequencer_engine_client = QueuedSequencerEngineClient {
                engine_actor_request_tx: engine_actor_request_tx.clone(),
                unsafe_head_rx,
            };

            // Create the admin API channel
            let (sequencer_admin_api_tx, sequencer_admin_api_rx) = mpsc::channel(1024);
            let queued_gossip_client =
                QueuedUnsafePayloadGossipClient::new(gossip_payload_tx.clone());

            (
                Some(SequencerActor {
                    admin_api_rx: sequencer_admin_api_rx,
                    attributes_builder: self.create_attributes_builder(),
                    cancellation_token: cancellation.clone(),
                    conductor,
                    engine_client: sequencer_engine_client,
                    is_active: self.sequencer_config.sequencer_stopped.not(),
                    in_recovery_mode: self.sequencer_config.sequencer_recovery_mode,
                    origin_selector: delayed_origin_selector,
                    rollup_config: self.config.clone(),
                    unsafe_payload_gossip_client: queued_gossip_client,
                }),
                Some(QueuedSequencerAdminAPIClient::new(sequencer_admin_api_tx)),
            )
        } else {
            (None, None)
        };

        // Create the RPC server actor.
        let rpc = self.rpc_builder().map(|b| {
            RpcActor::new(
                b,
                QueuedEngineRpcClient::new(engine_actor_request_tx.clone()),
                RollupBoostAdminApiClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                RollupBoostHealthRpcClient {
                    engine_actor_request_tx: engine_actor_request_tx.clone(),
                },
                sequencer_admin_client,
            )
        });

        crate::service::spawn_and_wait!(
            cancellation,
            actors = [
                rpc.map(|r| (
                    r,
                    RpcContext {
                        cancellation: cancellation.clone(),
                        p2p_network: network_rpc,
                        network_admin: net_admin_rpc,
                        l1_watcher_queries: l1_query_tx,
                    }
                )),
                sequencer_actor.map(|s| (s, ())),
                Some((network, ())),
                Some((l1_watcher, ())),
                Some((derivation, ())),
                Some((engine_actor, ())),
            ]
        );
        Ok(())
    }
}
