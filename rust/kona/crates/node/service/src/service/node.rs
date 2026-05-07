//! Contains the [`RollupNode`] implementation.
use crate::{
    ConductorClient, DelayedL1OriginSelectorProvider, DelegateDerivationActor, DerivationActor,
    DerivationDelegateClient, DerivationError, EngineActor, EngineActorRequest, EngineConfig,
    EngineProcessor, EngineRpcProcessor, InteropMode, L1OriginSelector, L1WatcherActor,
    NetworkActor, NetworkBuilder, NetworkConfig, NodeActor, NodeMode, QueuedDerivationEngineClient,
    QueuedEngineDerivationClient, QueuedEngineRpcClient, QueuedL1WatcherDerivationClient,
    QueuedNetworkEngineClient, QueuedSequencerAdminAPIClient, QueuedSequencerEngineClient,
    RpcActor, RpcContext, SequencerActor, SequencerConfig,
    actors::{BlockStream, NetworkInboundData, QueuedUnsafePayloadGossipClient},
};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::Address;
use alloy_provider::RootProvider;
use core::fmt::Debug;
use kona_derive::{DataAvailabilityProvider, StatefulAttributesBuilder};
use kona_engine::{Engine, EngineState, OpEngineClient};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_gossip::{BlockHandler, Handler};
use kona_interop::DependencySet;
use kona_protocol::L2BlockInfo;
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, OnlineBeaconClient, OnlineBlobProvider,
    OnlineDataProvider, OnlinePipeline,
};
use kona_rpc::RpcBuilder;
use op_alloy_network::Optimism;
use std::{ops::Not as _, sync::Arc, time::Duration};
use tokio::sync::{mpsc, watch};
use tokio_util::sync::CancellationToken;

/// Factory closure that builds the `DataAvailabilityProvider` once the L1
/// chain provider and blob provider have been constructed inside
/// [`RollupNode::start`]. Always installed by [`RollupNodeBuilder::build`]
/// — the default factory builds an [`OnlineDataProvider`]; the custom
/// factory is supplied via
/// [`RollupNodeBuilder::with_data_availability_provider`].
pub(super) type DapFactory<DAP> = Box<
    dyn FnOnce(
            Arc<RollupConfig>,
            AlloyChainProvider,
            OnlineBlobProvider<OnlineBeaconClient>,
        ) -> DAP
        + Send,
>;

/// Factory closure that builds the gossip [`Handler`] paired with its
/// `watch::Sender<Address>`. The default factory delegates to
/// [`NetworkBuilder::default_handler`] which produces a stock
/// [`BlockHandler`]; the custom factory is supplied via
/// [`RollupNodeBuilder::with_block_handler`] and just returns the
/// caller-supplied `(H, sender)` pair.
pub(super) type HandlerFactory<H> =
    Box<dyn FnOnce(&NetworkBuilder) -> (H, watch::Sender<Address>) + Send>;

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

/// The standard implementation of the [`RollupNode`] service, using the
/// governance-approved OP Stack configuration of components.
///
/// Generic over the data-availability provider (`DAP`) and gossip handler
/// (`H`) so downstream binaries can plug in custom implementations — e.g.
/// PSO Chain's SRA-aware DA filter and unsafe-block-signer verifier.
/// Defaults keep every existing caller compiling unchanged.
pub struct RollupNode<DAP = OnlineDataProvider, H = BlockHandler>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + 'static,
    H: Handler + Clone + Send + 'static,
{
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
    /// DAP factory installed by [`RollupNodeBuilder::build`]. Always
    /// `Some` at construction; consumed (taken via `Option::take`) on the
    /// first call to [`Self::start`].
    pub(crate) dap_factory: Option<DapFactory<DAP>>,
    /// Handler factory installed by [`RollupNodeBuilder::build`]. Always
    /// `Some` at construction; consumed on the first call to
    /// [`Self::start`]. The factory receives the [`NetworkBuilder`] so
    /// the default-handler variant can construct a [`BlockHandler`] from
    /// the same rollup config / signer address as upstream.
    pub(crate) handler_factory: Option<HandlerFactory<H>>,
}

// `RollupNode` cannot derive `Debug` because `DapFactory<DAP>` is a
// `Box<dyn FnOnce ...>` which is not `Debug`. Provide a manual impl that
// skips the closure.
impl<DAP, H> Debug for RollupNode<DAP, H>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + 'static,
    H: Handler + Clone + Send + 'static + Debug,
{
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("RollupNode")
            .field("config", &self.config)
            .field("l1_config", &self.l1_config)
            .field("interop_mode", &self.interop_mode)
            .field("l2_trust_rpc", &self.l2_trust_rpc)
            .field("engine_config", &self.engine_config)
            .field("p2p_config", &self.p2p_config)
            .field("sequencer_config", &self.sequencer_config)
            .field("dependency_set", &self.dependency_set)
            .field("dap_factory_set", &self.dap_factory.is_some())
            .field("handler_factory_set", &self.handler_factory.is_some())
            .finish_non_exhaustive()
    }
}

/// A RollupNode-level derivation actor wrapper.
///
/// This type selects the concrete derivation actor implementation
/// based on `RollupNode` configuration. Generic over `DAP` so the
/// `Normal` variant can carry an `OnlinePipeline<DAP>` with the
/// caller-supplied data-availability provider.
///
/// It is not intended to be generic or reusable outside the
/// `RollupNode` wiring logic.
enum ConfiguredDerivationActor<DAP = OnlineDataProvider>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + 'static,
{
    Delegate(Box<DelegateDerivationActor<QueuedDerivationEngineClient>>),
    Normal(Box<DerivationActor<QueuedDerivationEngineClient, OnlinePipeline<DAP>>>),
}

#[async_trait::async_trait]
impl<DAP> NodeActor for ConfiguredDerivationActor<DAP>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + 'static,
    DelegateDerivationActor<QueuedDerivationEngineClient>:
        NodeActor<StartData = (), Error = DerivationError>,
    DerivationActor<QueuedDerivationEngineClient, OnlinePipeline<DAP>>:
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

impl<DAP, H> RollupNode<DAP, H>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + Clone + 'static,
    H: Handler + Clone + Send + 'static,
{
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
            self.dependency_set.clone(),
        )
    }

    /// Builds the derivation pipeline. The DAP is always constructed via
    /// `self.dap_factory` — for the default `RollupNode<OnlineDataProvider>`
    /// the factory was installed automatically by [`RollupNodeBuilder::build`];
    /// for the override path it was installed by
    /// [`RollupNodeBuilder::with_data_availability_provider`].
    async fn create_pipeline(&mut self) -> OnlinePipeline<DAP> {
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
        let blob_provider = OnlineBlobProvider::init(self.l1_config.beacon_client.clone()).await;

        let factory = self.dap_factory.take().expect(
            "RollupNode constructed without a DAP factory — this is a kona builder bug; \
             RollupNodeBuilder::build always installs a default OnlineDataProvider factory.",
        );
        let dap = factory(self.config.clone(), l1_derivation_provider.clone(), blob_provider);

        match self.interop_mode {
            InteropMode::Polled => OnlinePipeline::<DAP>::new_polled_with_dap(
                self.config.clone(),
                self.l1_config.chain_config.clone(),
                l1_derivation_provider,
                l2_derivation_provider,
                dap,
                self.dependency_set.clone(),
            ),
            InteropMode::Indexed => OnlinePipeline::<DAP>::new_indexed_with_dap(
                self.config.clone(),
                self.l1_config.chain_config.clone(),
                l1_derivation_provider,
                l2_derivation_provider,
                dap,
                self.dependency_set.clone(),
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

        let engine_client = Arc::new(self.engine_config().build_engine_client());

        let engine_processor = EngineProcessor::new(
            engine_client.clone(),
            self.config.clone(),
            derivation_client,
            engine,
            self.mode().is_sequencer().then_some(unsafe_head_tx),
        );

        let engine_rpc_processor = EngineRpcProcessor::new(
            engine_client,
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
    pub async fn start(&mut self) -> Result<(), String> {
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
        let derivation: ConfiguredDerivationActor<DAP> = if let Some(provider) =
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
            ConfiguredDerivationActor::Normal(Box::new(
                DerivationActor::<_, OnlinePipeline<DAP>>::new(
                    QueuedDerivationEngineClient {
                        engine_actor_request_tx: engine_actor_request_tx.clone(),
                    },
                    cancellation.clone(),
                    derivation_actor_request_rx,
                    self.create_pipeline().await,
                ),
            ))
        };

        // Create the p2p actor. The handler factory always produces an
        // `(H, watch::Sender<Address>)` pair — for the default-H path it
        // delegates to `NetworkBuilder::default_handler()` which mirrors
        // upstream's behaviour exactly; for the custom-H path it just
        // returns the caller-supplied pair installed via
        // `RollupNodeBuilder::with_block_handler`.
        let net_engine_client = QueuedNetworkEngineClient {
            engine_actor_request_tx: engine_actor_request_tx.clone(),
        };
        let net_builder = self.network_builder();
        let handler_factory = self.handler_factory.take().expect(
            "RollupNode constructed without a handler factory — this is a kona builder bug; \
             RollupNodeBuilder::build always installs a default BlockHandler factory.",
        );
        let (handler, signer_tx) = handler_factory(&net_builder);
        let (
            NetworkInboundData {
                signer,
                p2p_rpc: network_rpc,
                gossip_payload_tx,
                admin_rpc: net_admin_rpc,
            },
            network,
        ) = NetworkActor::<_, H>::with_handler(
            net_engine_client,
            cancellation.clone(),
            net_builder,
            handler,
            signer_tx,
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
