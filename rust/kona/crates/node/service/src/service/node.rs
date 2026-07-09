//! Contains the [`RollupNode`] implementation.
use crate::{
    ConductorClient, DelayedL1OriginSelectorProvider, DelegateDerivationActor, DerivationActor,
    DerivationActorRequest, DerivationDelegateClient, DerivationError, EngineActor,
    EngineActorRequest, EngineConfig, EngineRpcActor, EngineRpcRequest, InteropMode,
    JsonrpseeServerLauncher, L1OriginSelector, L1WatcherActor, NetworkActor, NetworkBuilder,
    NetworkConfig, NetworkHandler, NodeActor, NodeMode, QueuedDerivationEngineClient,
    QueuedEngineDerivationClient, QueuedEngineRpcClient, QueuedL1WatcherDerivationClient,
    QueuedNetworkEngineClient, QueuedSequencerAdminAPIClient, QueuedSequencerEngineClient,
    RpcActor, RpcServerLauncher, SequencerActor, SequencerConfig,
    actors::{BlockStream, QueuedUnsafePayloadGossipClient},
};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::Address;
use alloy_provider::RootProvider;
use jsonrpsee::RpcModule;
use kona_derive::StatefulAttributesBuilder;
use kona_engine::{Engine, EngineState, OpEngineClient};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_gossip::P2pRpcRequest;
use kona_interop::DependencySet;
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, OnlineBeaconClient, OnlineBlobProvider,
    OnlinePipeline,
};
use kona_rpc::{
    AdminApiServer, AdminRpc, DevEngineApiServer, DevEngineRpc, HealthzApiServer, HealthzRpc,
    L1WatcherQueries, NetworkAdminQuery, OpP2PApiServer, P2pRpc, RollupNodeApiServer, RollupRpc,
    RpcBuilder, WsRPC, WsServer,
};
use op_alloy_network::Optimism;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
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
    /// The interop dependency set for this chain.
    /// Mirrors op-node's `--interop.dependency-set`.
    /// [`StatefulAttributesBuilder`] constructor panics otherwise.
    pub(crate) dependency_set: Option<Arc<DependencySet>>,
}

/// A RollupNode-level derivation actor wrapper.
///
/// This type selects the concrete derivation actor implementation
/// based on `RollupNode` configuration.
///
/// It is not intended to be generic or reusable outside the
/// `RollupNode` wiring logic.
enum ConfiguredDerivationActor {
    Delegate(
        Box<
            DelegateDerivationActor<
                QueuedDerivationEngineClient,
                DerivationDelegateClient,
                AlloyChainProvider,
            >,
        >,
    ),
    Normal(Box<DerivationActor<QueuedDerivationEngineClient, OnlinePipeline>>),
}

#[async_trait::async_trait]
impl NodeActor for ConfiguredDerivationActor
where
    DelegateDerivationActor<
        QueuedDerivationEngineClient,
        DerivationDelegateClient,
        AlloyChainProvider,
    >: NodeActor<Error = DerivationError>,
    DerivationActor<QueuedDerivationEngineClient, OnlinePipeline>:
        NodeActor<Error = DerivationError>,
{
    type Error = DerivationError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        match self {
            Self::Delegate(a) => a.step().await,
            Self::Normal(a) => a.step().await,
        }
    }
}

/// Concrete type of the engine actor used by `RollupNode`.
type ConfiguredEngineActor =
    EngineActor<OpEngineClient<RootProvider, RootProvider<Optimism>>, QueuedEngineDerivationClient>;

/// Concrete type of the engine rpc actor used by `RollupNode`.
type ConfiguredEngineRpcActor =
    EngineRpcActor<OpEngineClient<RootProvider, RootProvider<Optimism>>>;

/// Concrete type of the sequencer actor used by `RollupNode`.
type ConfiguredSequencerActor = SequencerActor<
    StatefulAttributesBuilder<AlloyChainProvider, AlloyL2ChainProvider>,
    ConductorClient,
    L1OriginSelector<DelayedL1OriginSelectorProvider>,
    QueuedSequencerEngineClient,
    QueuedUnsafePayloadGossipClient,
>;

/// Concrete type of the rpc actor used by `RollupNode`.
type ConfiguredRpcActor = RpcActor<JsonrpseeServerLauncher>;

impl RollupNode {
    /// The mode of operation for the node.
    const fn mode(&self) -> NodeMode {
        self.engine_config.mode
    }

    /// Creates a network builder for the node.
    fn network_builder(&self) -> NetworkBuilder {
        NetworkBuilder::from(self.p2p_config.clone())
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
            self.dependency_set.clone(),
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
                self.dependency_set.clone(),
            ),
            InteropMode::Indexed => OnlinePipeline::new_indexed(
                self.config.clone(),
                self.l1_config.chain_config.clone(),
                OnlineBlobProvider::init(self.l1_config.beacon_client.clone()).await,
                l1_derivation_provider,
                l2_derivation_provider,
                self.dependency_set.clone(),
            ),
        }
    }

    /// Builds both engine actors. They share a single [`kona_engine::EngineClient`] and a watch
    /// over the engine queue length / state, but otherwise run as independent peers.
    ///
    /// The non-rpc actor handles state-mutating requests (build, reset, seal, safe-signal
    /// consolidation, etc); the rpc actor handles read-only queries.
    fn build_engine_actors(
        &self,
        engine_request_rx: mpsc::Receiver<EngineActorRequest>,
        engine_rpc_request_rx: mpsc::Receiver<EngineRpcRequest>,
        derivation_actor_request_tx: mpsc::Sender<DerivationActorRequest>,
        unsafe_head_tx: watch::Sender<L2BlockInfo>,
    ) -> (ConfiguredEngineActor, ConfiguredEngineRpcActor) {
        // Engine-internal watches; not visible outside this helper.
        let engine_state = EngineState::default();
        let (engine_state_tx, engine_state_rx) = watch::channel(engine_state);
        let (engine_queue_length_tx, engine_queue_length_rx) = watch::channel(0);
        let engine = Engine::new(engine_state, engine_state_tx, engine_queue_length_tx);

        let engine_client = Arc::new(self.engine_config.clone().build_engine_client());

        // unsafe_head_tx is only meaningful in sequencer mode; validators ignore it.
        let unsafe_head_tx_opt = self.mode().is_sequencer().then_some(unsafe_head_tx);

        let actor = EngineActor::new(
            engine_client.clone(),
            self.config.clone(),
            QueuedEngineDerivationClient::new(derivation_actor_request_tx),
            engine,
            unsafe_head_tx_opt,
            engine_request_rx,
        );

        let rpc_actor = EngineRpcActor::new(
            engine_client,
            self.config.clone(),
            engine_state_rx,
            engine_queue_length_rx,
            engine_rpc_request_rx,
        );

        (actor, rpc_actor)
    }

    /// Selects between the standard and delegate derivation actor implementations and constructs
    /// the chosen one.
    async fn build_derivation_actor(
        &self,
        engine_actor_request_tx: mpsc::Sender<EngineActorRequest>,
        derivation_actor_request_rx: mpsc::Receiver<DerivationActorRequest>,
    ) -> ConfiguredDerivationActor {
        if let Some(provider) = self.derivation_delegate_provider.clone() {
            // L1 Provider for sanity checking Derivation Delegation
            let l1_provider = AlloyChainProvider::new(
                self.l1_config.engine_provider.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            );
            ConfiguredDerivationActor::Delegate(Box::new(DelegateDerivationActor::new(
                QueuedDerivationEngineClient { engine_actor_request_tx },
                derivation_actor_request_rx,
                provider,
                l1_provider,
            )))
        } else {
            ConfiguredDerivationActor::Normal(Box::new(DerivationActor::<_, OnlinePipeline>::new(
                QueuedDerivationEngineClient { engine_actor_request_tx },
                derivation_actor_request_rx,
                self.create_pipeline().await,
            )))
        }
    }

    /// Builds the L1 watcher actor along with its head and finalized block streams.
    ///
    /// Unlike the other `build_*` helpers, this one returns `impl NodeActor` rather than a named
    /// type alias: the block-stream type produced by [`BlockStream::new_as_stream`] is
    /// `impl Stream`, so the resulting `L1WatcherActor` generic parameter cannot be written down.
    /// Using `impl Trait` here is intentional; the macro consumer only requires `NodeActor`.
    fn build_l1_watcher(
        &self,
        derivation_actor_request_tx: mpsc::Sender<DerivationActorRequest>,
        signer_tx: mpsc::Sender<Address>,
        l1_query_rx: mpsc::Receiver<L1WatcherQueries>,
        l1_head_updates_tx: watch::Sender<Option<BlockInfo>>,
    ) -> Result<impl NodeActor<Error = crate::L1WatcherActorError<BlockInfo>> + 'static, String>
    {
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

        Ok(L1WatcherActor::new(
            self.config.clone(),
            self.l1_config.engine_provider.clone(),
            l1_query_rx,
            l1_head_updates_tx,
            QueuedL1WatcherDerivationClient { derivation_actor_request_tx },
            signer_tx,
            head_stream,
            finalized_stream,
        ))
    }

    /// Builds the sequencer actor when the node is in sequencer mode; otherwise returns `None`.
    fn build_sequencer(
        &self,
        engine_actor_request_tx: mpsc::Sender<EngineActorRequest>,
        gossip_payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
        unsafe_head_rx: watch::Receiver<L2BlockInfo>,
        l1_head_updates_rx: watch::Receiver<Option<BlockInfo>>,
        sequencer_admin_api_rx: mpsc::Receiver<crate::SequencerAdminQuery>,
    ) -> Option<ConfiguredSequencerActor> {
        if !self.mode().is_sequencer() {
            return None;
        }

        let delayed_l1_provider = DelayedL1OriginSelectorProvider::new(
            self.l1_config.engine_provider.clone(),
            l1_head_updates_rx,
            self.sequencer_config.l1_conf_delay,
        );
        let delayed_origin_selector =
            L1OriginSelector::new(self.config.clone(), delayed_l1_provider);

        let conductor =
            self.sequencer_config.conductor_rpc_url.clone().map(ConductorClient::new_http);

        let sequencer_engine_client =
            QueuedSequencerEngineClient { engine_actor_request_tx, unsafe_head_rx };

        let queued_gossip_client = QueuedUnsafePayloadGossipClient::new(gossip_payload_tx);

        Some(SequencerActor::new(
            sequencer_admin_api_rx,
            self.create_attributes_builder(),
            conductor,
            sequencer_engine_client,
            self.sequencer_config.sequencer_stopped.not(),
            self.sequencer_config.sequencer_recovery_mode,
            delayed_origin_selector,
            self.config.clone(),
            queued_gossip_client,
        ))
    }

    /// Assembles the JSON-RPC module set, performs the initial server launch, and returns the
    /// configured [`RpcActor`]. Returns `Ok(None)` when no [`RpcBuilder`] is configured.
    async fn build_rpc_actor(
        &self,
        engine_rpc_request_tx: mpsc::Sender<EngineRpcRequest>,
        sequencer_admin_client: Option<QueuedSequencerAdminAPIClient>,
        p2p_rpc_tx: mpsc::Sender<P2pRpcRequest>,
        network_admin_tx: mpsc::Sender<NetworkAdminQuery>,
        l1_watcher_queries_tx: mpsc::Sender<L1WatcherQueries>,
    ) -> Result<Option<ConfiguredRpcActor>, String> {
        let Some(config) = self.rpc_builder() else {
            return Ok(None);
        };

        let engine_rpc_client = QueuedEngineRpcClient::new(engine_rpc_request_tx);

        let mut modules = RpcModule::new(());
        modules
            .merge(HealthzApiServer::into_rpc(HealthzRpc {}))
            .map_err(|e| format!("Failed to register healthz module: {e:?}"))?;
        modules
            .merge(P2pRpc::new(p2p_rpc_tx).into_rpc())
            .map_err(|e| format!("Failed to register p2p module: {e:?}"))?;
        modules
            .merge(AdminRpc::new(sequencer_admin_client, network_admin_tx).into_rpc())
            .map_err(|e| format!("Failed to register admin module: {e:?}"))?;
        modules
            .merge(RollupRpc::new(engine_rpc_client.clone(), l1_watcher_queries_tx).into_rpc())
            .map_err(|e| format!("Failed to register rollup module: {e:?}"))?;
        if config.dev_enabled() {
            modules
                .merge(DevEngineRpc::new(engine_rpc_client.clone()).into_rpc())
                .map_err(|e| format!("Failed to register dev engine module: {e:?}"))?;
        }
        if config.ws_enabled() {
            modules
                .merge(WsRPC::new(engine_rpc_client.clone()).into_rpc())
                .map_err(|e| format!("Failed to register ws module: {e:?}"))?;
        }

        let restarts_remaining = config.restart_count();
        let launcher = JsonrpseeServerLauncher::new(config);
        let handle = launcher
            .launch(modules.clone())
            .await
            .map_err(|e: std::io::Error| format!("Failed to launch rpc server: {e:?}"))?;

        Ok(Some(RpcActor::new(launcher, modules, handle, restarts_remaining)))
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
    ///
    /// ## Shutdown
    ///
    /// Shutdown is unordered: when any actor exits (success, error, or panic) or an OS signal is
    /// received, the umbrella cancellation token fires and all peer actors observe it on their
    /// next `select!`. Actors may log channel-closed errors while peers are torn down
    /// concurrently; this is expected and not a sign of an unclean exit.
    pub async fn start(&self) -> Result<(), String> {
        // Single umbrella cancellation token owned by the spawn_and_wait! macro.
        let cancellation = CancellationToken::new();

        // ─── cross-actor channels ───────────────────────────────────────────────────────────
        // actor request channels
        let (derivation_actor_request_tx, derivation_actor_request_rx) =
            mpsc::channel::<DerivationActorRequest>(1024);
        let (engine_actor_request_tx, engine_actor_request_rx) =
            mpsc::channel::<EngineActorRequest>(1024);
        let (engine_rpc_request_tx, engine_rpc_request_rx) =
            mpsc::channel::<EngineRpcRequest>(1024);
        let (l1_query_tx, l1_query_rx) = mpsc::channel::<L1WatcherQueries>(1024);
        let (sequencer_admin_api_tx, sequencer_admin_api_rx) = mpsc::channel(1024);
        // Network actor inbound channels
        let (signer_tx, signer_rx) = mpsc::channel::<Address>(16);
        let (p2p_rpc_tx, p2p_rpc_rx) = mpsc::channel::<P2pRpcRequest>(1024);
        let (network_admin_tx, network_admin_rx) = mpsc::channel::<NetworkAdminQuery>(1024);
        let (gossip_payload_tx, gossip_payload_rx) =
            mpsc::channel::<OpExecutionPayloadEnvelope>(256);
        // watch channels
        let (unsafe_head_tx, unsafe_head_rx) = watch::channel(L2BlockInfo::default());
        let (l1_head_updates_tx, l1_head_updates_rx) = watch::channel::<Option<BlockInfo>>(None);

        // ─── actor construction ─────────────────────────────────────────────────────────────
        let (engine_actor, engine_rpc_actor) = self.build_engine_actors(
            engine_actor_request_rx,
            engine_rpc_request_rx,
            derivation_actor_request_tx.clone(),
            unsafe_head_tx,
        );

        let derivation = self
            .build_derivation_actor(engine_actor_request_tx.clone(), derivation_actor_request_rx)
            .await;

        // Build and start the libp2p swarm upstream of `NetworkActor::new` so the constructor
        // stays sync.
        let handler: NetworkHandler = self
            .network_builder()
            .build()
            .map_err(|e| format!("Failed to build network: {e:?}"))?
            .start()
            .await
            .map_err(|e| format!("Failed to start network: {e:?}"))?;

        let network = NetworkActor::new(
            QueuedNetworkEngineClient { engine_actor_request_tx: engine_actor_request_tx.clone() },
            handler,
            signer_rx,
            p2p_rpc_rx,
            network_admin_rx,
            gossip_payload_rx,
        );

        let l1_watcher = self.build_l1_watcher(
            derivation_actor_request_tx,
            signer_tx,
            l1_query_rx,
            l1_head_updates_tx,
        )?;

        let sequencer_actor = self.build_sequencer(
            engine_actor_request_tx,
            gossip_payload_tx,
            unsafe_head_rx,
            l1_head_updates_rx,
            sequencer_admin_api_rx,
        );
        let sequencer_admin_client = sequencer_actor
            .is_some()
            .then(|| QueuedSequencerAdminAPIClient::new(sequencer_admin_api_tx));

        let rpc = self
            .build_rpc_actor(
                engine_rpc_request_tx,
                sequencer_admin_client,
                p2p_rpc_tx,
                network_admin_tx,
                l1_query_tx,
            )
            .await?;

        crate::service::spawn_and_wait!(
            cancellation,
            actors = [
                rpc,
                sequencer_actor,
                Some(network),
                Some(l1_watcher),
                Some(derivation),
                Some(engine_actor),
                Some(engine_rpc_actor),
            ]
        );
        Ok(())
    }
}
