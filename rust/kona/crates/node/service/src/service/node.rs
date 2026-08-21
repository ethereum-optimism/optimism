//! Contains the [`RollupNode`] implementation.
use crate::{
    ChainController, ChainControllerRequest, ChainControllerRpcActor, ChainControllerRpcRequest,
    ConductorClient, DelayedL1OriginSelectorProvider, DelegateDerivationActor, DerivationActor,
    DerivationActorRequest, DerivationDelegateClient, DerivationError, EngineConfig,
    JsonrpseeServerLauncher, L1OriginSelector, L1WatcherActor, L1WatcherChain, NetworkActor,
    NetworkBuilder, NetworkConfig, NetworkHandler, NodeActor, NodeMode,
    QueuedChainControllerDerivationClient, QueuedDerivationEngineClient, QueuedEngineRpcClient,
    QueuedL1WatcherDerivationClient, QueuedNetworkEngineClient, QueuedSequencerAdminAPIClient,
    QueuedSequencerEngineClient, RpcActor, RpcServerLauncher, SequencerActor, SequencerConfig,
    actors::{BlockStream, QueuedUnsafePayloadGossipClient},
    service::{
        composition::{ComposedChain, L1WatcherPorts},
        spawn::{BoxedNodeActor, IntoBoxedNodeActor, run_actors},
    },
};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::Address;
use alloy_provider::RootProvider;
use jsonrpsee::RpcModule;
use kona_derive::StatefulAttributesBuilder;
use kona_engine::{CrossSafePromoter, Engine, EngineState, OpEngineClient};
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
use kona_safedb::SharedSafeDb;
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

impl L1Config {
    /// Builds the one L1 watcher a host runs, serving every chain whose [`L1WatcherPorts`] are
    /// given.
    ///
    /// This is the single place a watcher is attached to composed chains, and the reason
    /// [`RollupNode::compose`] leaves the watcher out of a chain's actors: the L1 is followed once
    /// per process, not once per chain, so how many watchers exist is the host's decision and this
    /// is where the host states it. A single-chain host passes one chain's ports; a multi-chain
    /// host passes every chain's, and the head and finalized streams are then polled once for all
    /// of them.
    ///
    /// The returned actor is inert until it is stepped by [`run_actors`].
    ///
    /// Returns `impl NodeActor` rather than a named type: the block-stream type produced by
    /// [`BlockStream::new_as_stream`] is `impl Stream`, so the `L1WatcherActor` generic parameter
    /// cannot be written down. Callers only need a `NodeActor` to box.
    ///
    /// # Panics
    ///
    /// Panics if `chains` is empty; a watcher with no chain to serve has nothing to do, and a host
    /// that composed no chain is a configuration bug rather than a runtime condition.
    pub fn l1_watcher(
        &self,
        chains: Vec<L1WatcherPorts>,
    ) -> Result<impl NodeActor<Error = crate::L1WatcherActorError<BlockInfo>> + 'static, String>
    {
        let head_stream = BlockStream::new_as_stream(
            self.engine_provider.clone(),
            BlockNumberOrTag::Latest,
            Duration::from_secs(HEAD_STREAM_POLL_INTERVAL),
        )?;
        let finalized_stream = BlockStream::new_as_stream(
            self.engine_provider.clone(),
            BlockNumberOrTag::Finalized,
            Duration::from_secs(FINALIZED_STREAM_POLL_INTERVAL),
        )?;

        let chains = chains
            .into_iter()
            .map(|ports| {
                let L1WatcherPorts {
                    rollup_config,
                    derivation_actor_request_tx,
                    unsafe_signer_tx,
                    l1_query_rx,
                    l1_head_updates_tx,
                } = ports;

                L1WatcherChain::new(
                    rollup_config,
                    QueuedL1WatcherDerivationClient { derivation_actor_request_tx },
                    unsafe_signer_tx,
                    l1_query_rx,
                    l1_head_updates_tx,
                )
            })
            .collect();

        Ok(L1WatcherActor::new(self.engine_provider.clone(), head_stream, finalized_stream, chains))
    }
}

/// The standard implementation of the [`RollupNode`] service, using the governance approved OP
/// Stack configuration of components.
#[derive(Debug)]
pub struct RollupNode {
    /// The rollup configuration.
    pub(crate) config: Arc<RollupConfig>,
    /// The L1 configuration.
    pub(crate) l1_config: L1Config,
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
    /// Whether this chain's cross-safe head is fed by an external cross-chain verifier.
    ///
    /// `false` is the standalone default, under which every local-safe advance is trivially
    /// cross-safe. `true` hands the resulting [`CrossSafePromoter`] back to the host through
    /// [`ComposedChain::cross_safe_promoter`], and the cross-safe head then moves only on the
    /// promotions that promoter mints — so a host that sets this and never promotes deliberately
    /// holds the chain's `safeBlockHash` where it is.
    pub(crate) external_cross_safe: bool,
    /// The safe-head database this chain's controller records local-safe advances into.
    ///
    /// [`kona_safedb::DisabledDatabase`] for a host that does not need the history: its writes
    /// are no-ops, so nothing branches on whether recording is on.
    pub(crate) safe_db: SharedSafeDb,
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

/// Concrete type of the chain controller used by `RollupNode`.
type ConfiguredChainController = ChainController<
    OpEngineClient<RootProvider, RootProvider<Optimism>>,
    QueuedChainControllerDerivationClient,
>;

/// Concrete type of the engine rpc actor used by `RollupNode`.
type ConfiguredChainControllerRpcActor =
    ChainControllerRpcActor<OpEngineClient<RootProvider, RootProvider<Optimism>>>;

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
            self.config.l2_chain_id.id(),
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
            self.config.l2_chain_id.id(),
        );
        let l2_derivation_provider = AlloyL2ChainProvider::new_with_trust(
            self.l2_provider.clone(),
            self.config.clone(),
            DERIVATION_PROVIDER_CACHE_SIZE,
            self.l2_trust_rpc,
        );

        OnlinePipeline::new_polled(
            self.config.clone(),
            self.l1_config.chain_config.clone(),
            OnlineBlobProvider::init(self.l1_config.beacon_client.clone())
                .await
                .with_chain_id(self.config.l2_chain_id.id()),
            l1_derivation_provider,
            l2_derivation_provider,
            self.dependency_set.clone(),
        )
    }

    /// Builds the chain controller and its read-only RPC peer. They share a single
    /// [`kona_engine::EngineClient`] and a watch over the engine queue length / state, but
    /// otherwise run as independent peers.
    ///
    /// The controller handles state-mutating requests (build, reset, seal, local-safe-signal
    /// consolidation, etc); the RPC peer handles read-only queries.
    fn build_chain_controller_actors(
        &self,
        controller_request_rx: mpsc::Receiver<ChainControllerRequest>,
        controller_rpc_request_rx: mpsc::Receiver<ChainControllerRpcRequest>,
        derivation_actor_request_tx: mpsc::Sender<DerivationActorRequest>,
        unsafe_head_tx: watch::Sender<L2BlockInfo>,
    ) -> (ConfiguredChainController, ConfiguredChainControllerRpcActor, Option<CrossSafePromoter>)
    {
        // Engine-internal watches; not visible outside this helper.
        let engine_state =
            EngineState { chain_id: self.config.l2_chain_id.id(), ..Default::default() };
        let (engine_state_tx, engine_state_rx) = watch::channel(engine_state);
        let (engine_queue_length_tx, engine_queue_length_rx) = watch::channel(0);
        // At most one promoter per engine, and only when a verifier is there to hold it: a chain
        // composed without one keeps the trivial local-safe feed, so this is the single switch
        // between standalone and interop cross-safety.
        let (engine, cross_safe_promoter) = if self.external_cross_safe {
            let (engine, promoter) = Engine::with_external_cross_safe(
                engine_state,
                engine_state_tx,
                engine_queue_length_tx,
            );
            (engine, Some(promoter))
        } else {
            (Engine::new(engine_state, engine_state_tx, engine_queue_length_tx), None)
        };

        let engine_client = Arc::new(self.engine_config.clone().build_engine_client());

        // unsafe_head_tx is only meaningful in sequencer mode; validators ignore it.
        let unsafe_head_tx_opt = self.mode().is_sequencer().then_some(unsafe_head_tx);

        let actor = ChainController::new(
            engine_client.clone(),
            self.config.clone(),
            QueuedChainControllerDerivationClient::new(derivation_actor_request_tx),
            engine,
            unsafe_head_tx_opt,
            controller_request_rx,
            self.safe_db.clone(),
        );

        let rpc_actor = ChainControllerRpcActor::new(
            engine_client,
            self.config.clone(),
            engine_state_rx,
            engine_queue_length_rx,
            controller_rpc_request_rx,
        );

        (actor, rpc_actor, cross_safe_promoter)
    }

    /// Selects between the standard and delegate derivation actor implementations and constructs
    /// the chosen one.
    async fn build_derivation_actor(
        &self,
        controller_request_tx: mpsc::Sender<ChainControllerRequest>,
        derivation_actor_request_rx: mpsc::Receiver<DerivationActorRequest>,
    ) -> ConfiguredDerivationActor {
        if let Some(provider) = self.derivation_delegate_provider.clone() {
            // L1 Provider for sanity checking Derivation Delegation
            let l1_provider = AlloyChainProvider::new(
                self.l1_config.engine_provider.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
                self.config.l2_chain_id.id(),
            );
            ConfiguredDerivationActor::Delegate(Box::new(DelegateDerivationActor::new(
                QueuedDerivationEngineClient { controller_request_tx },
                derivation_actor_request_rx,
                provider,
                l1_provider,
            )))
        } else {
            ConfiguredDerivationActor::Normal(Box::new(DerivationActor::<_, OnlinePipeline>::new(
                QueuedDerivationEngineClient { controller_request_tx },
                derivation_actor_request_rx,
                self.create_pipeline().await,
            )))
        }
    }

    /// Builds the sequencer actor when the node is in sequencer mode; otherwise returns `None`.
    fn build_sequencer(
        &self,
        controller_request_tx: mpsc::Sender<ChainControllerRequest>,
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
            QueuedSequencerEngineClient { controller_request_tx, unsafe_head_rx };

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
        controller_rpc_request_tx: mpsc::Sender<ChainControllerRpcRequest>,
        sequencer_admin_client: Option<QueuedSequencerAdminAPIClient>,
        p2p_rpc_tx: mpsc::Sender<P2pRpcRequest>,
        network_admin_tx: mpsc::Sender<NetworkAdminQuery>,
        l1_watcher_queries_tx: mpsc::Sender<L1WatcherQueries>,
    ) -> Result<Option<ConfiguredRpcActor>, String> {
        let Some(config) = self.rpc_builder() else {
            return Ok(None);
        };

        let engine_rpc_client = QueuedEngineRpcClient::new(controller_rpc_request_tx);

        let mut modules = RpcModule::new(());
        modules
            .merge(HealthzApiServer::into_rpc(HealthzRpc {}))
            .map_err(|e| format!("Failed to register healthz module: {e:?}"))?;
        modules
            .merge(P2pRpc::new(p2p_rpc_tx, self.config.l2_chain_id.id()).into_rpc())
            .map_err(|e| format!("Failed to register p2p module: {e:?}"))?;
        merge_admin_module(
            &mut modules,
            config.enable_admin(),
            sequencer_admin_client,
            network_admin_tx,
            self.config.l2_chain_id.id(),
        )?;
        modules
            .merge(
                RollupRpc::new(
                    engine_rpc_client.clone(),
                    l1_watcher_queries_tx,
                    self.config.l2_chain_id.id(),
                )
                .with_dependency_set(self.dependency_set.clone())
                .into_rpc(),
            )
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

    /// Composes this chain's actor group: the single-chain composition entry point.
    ///
    /// Every host that runs a chain goes through here, so there is exactly one definition of how a
    /// chain is wired. [`Self::start`] is the single-chain host on top of it; a multi-chain host
    /// calls this once per configured chain and runs all of the resulting actors together. A second
    /// hand-copied composition would silently drift from this one, so there isn't one.
    ///
    /// What this builds:
    ///
    /// - the [`ChainController`] and its read-only RPC peer,
    /// - the derivation actor (delegating or self-deriving, per configuration),
    /// - the P2P network actor, with its libp2p swarm already started,
    /// - the sequencer actor, in sequencer mode only,
    /// - the JSON-RPC actor, with its server already launched, when an [`RpcBuilder`] is
    ///   configured.
    ///
    /// The one actor it does *not* build is the L1 watcher: a multi-chain host runs a single
    /// watcher across all of its chains, so composition hands back that chain's
    /// [`L1WatcherPorts`] and lets the host decide how many watchers to attach to them.
    ///
    /// The returned actors are inert until they are stepped by [`run_actors`].
    pub async fn compose(&self) -> Result<ComposedChain, String> {
        // ─── cross-actor channels ───────────────────────────────────────────────────────────
        // actor request channels
        let (derivation_actor_request_tx, derivation_actor_request_rx) =
            mpsc::channel::<DerivationActorRequest>(1024);
        let (controller_request_tx, controller_request_rx) =
            mpsc::channel::<ChainControllerRequest>(1024);
        let (controller_rpc_request_tx, controller_rpc_request_rx) =
            mpsc::channel::<ChainControllerRpcRequest>(1024);
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
        let (chain_controller, chain_controller_rpc_actor, cross_safe_promoter) = self
            .build_chain_controller_actors(
                controller_request_rx,
                controller_rpc_request_rx,
                derivation_actor_request_tx.clone(),
                unsafe_head_tx,
            );

        let derivation = self
            .build_derivation_actor(controller_request_tx.clone(), derivation_actor_request_rx)
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
            QueuedNetworkEngineClient { controller_request_tx: controller_request_tx.clone() },
            handler,
            signer_rx,
            p2p_rpc_rx,
            network_admin_rx,
            gossip_payload_rx,
        );

        let sequencer_actor = self.build_sequencer(
            controller_request_tx.clone(),
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
                controller_rpc_request_tx.clone(),
                sequencer_admin_client,
                p2p_rpc_tx,
                network_admin_tx,
                l1_query_tx.clone(),
            )
            .await?;

        // Spawn order is not significant: every actor is spawned before any of them is awaited.
        let mut actors: Vec<BoxedNodeActor> = Vec::with_capacity(6);
        actors.extend(rpc.map(IntoBoxedNodeActor::boxed));
        actors.extend(sequencer_actor.map(IntoBoxedNodeActor::boxed));
        actors.push(network.boxed());
        actors.push(derivation.boxed());
        actors.push(chain_controller.boxed());
        actors.push(chain_controller_rpc_actor.boxed());

        Ok(ComposedChain {
            actors,
            l1_watcher_ports: L1WatcherPorts {
                rollup_config: self.config.clone(),
                derivation_actor_request_tx,
                unsafe_signer_tx: signer_tx,
                l1_query_rx,
                l1_head_updates_tx,
            },
            controller_request_tx,
            controller_rpc_request_tx,
            l1_query_tx,
            cross_safe_promoter,
        })
    }

    /// Starts the rollup node service: the single-chain host over [`Self::compose`].
    ///
    /// Composes this chain, attaches an L1 watcher that follows the L1 for it alone, and runs
    /// every actor until one of them exits or an OS shutdown signal arrives. Returns the first
    /// fatal actor error, if any.
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
    /// Shutdown is unordered and process-wide; see [`run_actors`].
    pub async fn start(&self) -> Result<(), String> {
        // Single umbrella cancellation token owned by `run_actors`.
        let cancellation = CancellationToken::new();

        let ComposedChain { mut actors, l1_watcher_ports, .. } = self.compose().await?;
        actors.push(self.l1_config.l1_watcher(vec![l1_watcher_ports])?.boxed());

        run_actors(cancellation, actors).await
    }
}

/// Registers the admin API namespace on `modules`, but only when `enable_admin` is set.
///
/// The admin API is opt-in via `--rpc.enable-admin`, matching op-node's admin namespace.
fn merge_admin_module(
    modules: &mut RpcModule<()>,
    enable_admin: bool,
    sequencer_admin_client: Option<QueuedSequencerAdminAPIClient>,
    network_admin_tx: mpsc::Sender<NetworkAdminQuery>,
    chain_id: u64,
) -> Result<(), String> {
    if enable_admin {
        modules
            .merge(AdminRpc::new(sequencer_admin_client, network_admin_tx, chain_id).into_rpc())
            .map_err(|e| format!("Failed to register admin module: {e:?}"))?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn admin_method_names(enable_admin: bool) -> Vec<String> {
        let mut modules = RpcModule::new(());
        let (network_admin_tx, _rx) = mpsc::channel(1);
        merge_admin_module(&mut modules, enable_admin, None, network_admin_tx, 10)
            .expect("admin module registration");
        modules.method_names().map(ToString::to_string).collect()
    }

    #[test]
    fn admin_module_registered_only_when_enabled() {
        // Without `--rpc.enable-admin`, no admin methods are exposed.
        assert!(
            admin_method_names(false).is_empty(),
            "admin namespace must not be registered when disabled"
        );

        // With it enabled, the sequencer-control and payload-injection methods the acceptance
        // suite relies on are present.
        let enabled = admin_method_names(true);
        for method in ["admin_postUnsafePayload", "admin_startSequencer", "admin_stopSequencer"] {
            assert!(enabled.iter().any(|m| m == method), "missing {method}");
        }
    }
}
