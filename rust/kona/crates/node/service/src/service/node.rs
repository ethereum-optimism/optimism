//! Contains the [`RollupNode`] implementation.
use super::{Supervisor, middleware::RpcMetricsLayer};
use crate::{
    BlockStream, ConductorClient, DelayedL1OriginSelectorProvider, DelegateDerivationActor,
    DerivationActor, DerivationActorRequest, DerivationDelegateClient, DerivationError,
    EngineActor, EngineActorRequest, EngineConfig, L1OriginSelector, L1WatcherActor,
    L1WatcherChain, NetworkActor, NetworkBuilder, NetworkConfig, NetworkHandler, NodeActor,
    NodeMode, QueuedDerivationEngineClient, QueuedEngineDerivationClient,
    QueuedL1WatcherDerivationClient, QueuedNetworkEngineClient, QueuedSequencerEngineClient,
    RpcActor, SequencerConfig, service::BufferImportedBlocks, signer,
};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::Address;
use alloy_provider::RootProvider;
use jsonrpsee::{
    RpcModule,
    server::{
        Server, ServerConfig,
        middleware::{http::ProxyGetRequestLayer, rpc::RpcServiceBuilder},
    },
};
use kona_derive::{BlobProviderError, StatefulAttributesBuilder};
use kona_engine::{Engine, EngineClient, EngineState};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_node_actors::{DerivationStatus, L1State, sequencer};
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, BufferedAlloyL2ChainProvider, OnlineBeaconClient,
    OnlineBlobProvider, OnlinePipeline,
};
use kona_providers_local::BufferedL2Provider;
use kona_rpc::{
    AdminApiServer, AdminRpc, HealthzApiServer, HealthzRpc, OpP2PApiServer, P2pRpc,
    RollupNodeApiServer, RollupRpc, RpcBuilder,
};
use kona_sources::BlockSignerHandler;
use op_alloy_network::Optimism;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{ops::Not as _, sync::Arc, time::Duration};
use tokio::sync::{mpsc, watch};
use tokio_util::sync::CancellationToken;

const DERIVATION_PROVIDER_CACHE_SIZE: usize = 1024;
/// How many recently imported blocks to keep for the local L2 lookups.
///
/// Every lookup served from here is for the block being built on top of, which the engine recorded
/// one step earlier. The size absorbs what lands in between — gossiped unsafe blocks keep
/// arriving, and on a sequencer the two builders ask about different heads.
pub(super) const IMPORTED_BLOCK_BUFFER_SIZE: usize = 32;
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
    /// The application version reported by the RPC server.
    pub(crate) version: String,
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
    /// Blocks the engine has imported, shared with the derivation providers so they can be read
    /// locally instead of fetched back from the execution layer.
    pub(crate) l2_block_buffer: BufferedL2Provider,
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
type ConfiguredEngineActor = EngineActor<QueuedEngineDerivationClient>;

impl RollupNode {
    /// The mode of operation for the node.
    const fn mode(&self) -> NodeMode {
        self.engine_config.mode
    }

    /// Creates a network builder for the node whose gossip validation follows
    /// `unsafe_block_signer`.
    fn network_builder(&self, unsafe_block_signer: watch::Receiver<Address>) -> NetworkBuilder {
        NetworkBuilder::from(self.p2p_config.clone()).with_unsafe_block_signer(unsafe_block_signer)
    }

    /// Returns an rpc builder for the node.
    fn rpc_builder(&self) -> Option<RpcBuilder> {
        self.rpc_builder.clone()
    }

    /// Returns the sequencer builder for the node.
    fn create_attributes_builder(
        &self,
    ) -> StatefulAttributesBuilder<AlloyChainProvider, BufferedAlloyL2ChainProvider> {
        let l1_derivation_provider = AlloyChainProvider::new_with_trust(
            self.l1_config.engine_provider.clone(),
            DERIVATION_PROVIDER_CACHE_SIZE,
            self.l1_config.trust_rpc,
        );
        let l2_derivation_provider = BufferedAlloyL2ChainProvider::new(
            self.l2_block_buffer.clone(),
            AlloyL2ChainProvider::new_with_trust(
                self.l2_provider.clone(),
                self.config.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
                self.l2_trust_rpc,
            ),
        );

        StatefulAttributesBuilder::new(
            self.config.clone(),
            self.l1_config.chain_config.clone(),
            l2_derivation_provider,
            l1_derivation_provider,
            self.dependency_set.clone(),
        )
    }

    async fn create_pipeline(&self) -> Result<OnlinePipeline, BlobProviderError> {
        // Create the caching L1/L2 EL providers for derivation.
        let l1_derivation_provider = AlloyChainProvider::new_with_trust(
            self.l1_config.engine_provider.clone(),
            DERIVATION_PROVIDER_CACHE_SIZE,
            self.l1_config.trust_rpc,
        );
        let l2_derivation_provider = BufferedAlloyL2ChainProvider::new(
            self.l2_block_buffer.clone(),
            AlloyL2ChainProvider::new_with_trust(
                self.l2_provider.clone(),
                self.config.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
                self.l2_trust_rpc,
            ),
        );

        Ok(OnlinePipeline::new_polled(
            self.config.clone(),
            self.l1_config.chain_config.clone(),
            OnlineBlobProvider::init(self.l1_config.beacon_client.clone()).await?,
            l1_derivation_provider,
            l2_derivation_provider,
            self.dependency_set.clone(),
        ))
    }

    /// Builds the engine actor and returns the shared client and state watch used by RPC.
    fn build_engine_actor(
        &self,
        engine_request_rx: mpsc::Receiver<EngineActorRequest>,
        derivation_actor_request_tx: mpsc::Sender<DerivationActorRequest>,
        unsafe_head_tx: watch::Sender<L2BlockInfo>,
    ) -> (ConfiguredEngineActor, EngineClient, watch::Receiver<EngineState>) {
        // Share engine state with RPC without routing reads through the actor.
        let engine_state = EngineState::default();
        let (engine_state_tx, engine_state_rx) = watch::channel(engine_state);
        let engine = Engine::new(engine_state, engine_state_tx);

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
            Arc::new(BufferImportedBlocks::new(self.l2_block_buffer.clone())),
        );

        (actor, engine_client.as_ref().clone(), engine_state_rx)
    }

    /// Selects between the standard and delegate derivation actor implementations and constructs
    /// the chosen one.
    async fn build_derivation_actor(
        &self,
        engine_actor_request_tx: mpsc::Sender<EngineActorRequest>,
        derivation_actor_request_rx: mpsc::Receiver<DerivationActorRequest>,
    ) -> Result<(ConfiguredDerivationActor, watch::Receiver<DerivationStatus>), String> {
        if let Some(provider) = self.derivation_delegate_provider.clone() {
            // L1 Provider for sanity checking Derivation Delegation
            let l1_provider = AlloyChainProvider::new(
                self.l1_config.engine_provider.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            );
            let actor = DelegateDerivationActor::new(
                QueuedDerivationEngineClient { engine_actor_request_tx },
                derivation_actor_request_rx,
                provider,
                l1_provider,
            );
            let status = actor.state_receiver();
            Ok((ConfiguredDerivationActor::Delegate(Box::new(actor)), status))
        } else {
            let pipeline = self
                .create_pipeline()
                .await
                .map_err(|error| format!("Failed to initialize L1 blob provider: {error}"))?;
            let actor = DerivationActor::<_, OnlinePipeline>::new(
                QueuedDerivationEngineClient { engine_actor_request_tx },
                derivation_actor_request_rx,
                pipeline,
            );
            let status = actor.state_receiver();
            Ok((ConfiguredDerivationActor::Normal(Box::new(actor)), status))
        }
    }

    /// Builds the L1 watcher actor with independent head, safe, and finalized block streams.
    ///
    /// Unlike the other `build_*` helpers, this one returns `impl NodeActor` rather than a named
    /// type alias: the block-stream type produced by [`BlockStream::new_as_stream`] is
    /// `impl Stream`, so the resulting `L1WatcherActor` generic parameter cannot be written down.
    /// Using `impl Trait` here is intentional; the lifetime adapter only requires `NodeActor`.
    fn build_l1_watcher(
        &self,
        derivation_actor_request_tx: mpsc::Sender<DerivationActorRequest>,
        signer_tx: watch::Sender<Address>,
    ) -> Result<
        (
            impl NodeActor<Error = crate::L1WatcherActorError<BlockInfo>> + 'static,
            watch::Receiver<L1State>,
        ),
        String,
    > {
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

        let safe_stream = BlockStream::new_as_stream(
            self.l1_config.engine_provider.clone(),
            BlockNumberOrTag::Safe,
            Duration::from_secs(HEAD_STREAM_POLL_INTERVAL),
        )?;

        let chain = L1WatcherChain::new(
            self.config.clone(),
            QueuedL1WatcherDerivationClient { derivation_actor_request_tx },
            signer_tx,
        );

        let actor = L1WatcherActor::new(
            self.l1_config.engine_provider.clone(),
            head_stream,
            finalized_stream,
            safe_stream,
            vec![chain],
        );
        let state = actor.state_receiver();
        Ok((actor, state))
    }

    /// Starts the signing backend when the node is in sequencer mode; otherwise returns `None`.
    ///
    /// A sequencer must have a block signer: its blocks are gossiped only once signed.
    async fn build_signer(&self) -> Result<Option<BlockSignerHandler>, String> {
        if !self.mode().is_sequencer() {
            if self.p2p_config.gossip_signer.is_some() {
                warn!(target: "rollup_node", "Ignoring the configured block signer: only a sequencer signs blocks");
            }
            return Ok(None);
        }
        let Some(signer) = self.p2p_config.gossip_signer.clone() else {
            return Err("Sequencer mode requires a block signer: set --p2p.sequencer.key, \
                 --p2p.sequencer.key.path, or --p2p.signer.endpoint with --p2p.signer.address"
                .into());
        };
        let signer =
            signer.start().await.map_err(|e| format!("Failed to start block signer: {e}"))?;
        Ok(Some(signer))
    }

    /// Wires the sequencer dependencies and returns its lifetime future.
    fn build_sequencer(
        &self,
        engine_actor_request_tx: mpsc::Sender<EngineActorRequest>,
        signer: signer::Handle,
        unsafe_head_rx: watch::Receiver<L2BlockInfo>,
        l1_state: watch::Receiver<L1State>,
        builder: sequencer::Builder<ConductorClient>,
        cancellation: CancellationToken,
    ) -> impl Future<Output = Result<(), sequencer::ActorError>> + Send + 'static {
        let delayed_l1_provider = DelayedL1OriginSelectorProvider::new(
            self.l1_config.engine_provider.clone(),
            l1_state,
            self.sequencer_config.l1_conf_delay,
        );
        let delayed_origin_selector =
            L1OriginSelector::new(self.config.clone(), delayed_l1_provider);

        let sequencer_engine_client =
            QueuedSequencerEngineClient { engine_actor_request_tx, unsafe_head_rx };

        builder.build(
            self.create_attributes_builder(),
            sequencer_engine_client,
            delayed_origin_selector,
            self.config.clone(),
            signer,
            cancellation,
        )
    }

    /// Assembles the JSON-RPC module set, performs the initial server launch, and returns the
    /// configured [`RpcActor`]. Returns `Ok(None)` when no [`RpcBuilder`] is configured.
    async fn build_rpc_actor(
        &self,
        l2_query_client: EngineClient,
        engine_state_rx: watch::Receiver<EngineState>,
        admin_rpc: AdminRpc,
        p2p_rpc: P2pRpc,
        l1_state: watch::Receiver<L1State>,
        derivation_status: watch::Receiver<DerivationStatus>,
    ) -> Result<Option<RpcActor>, String> {
        let Some(config) = self.rpc_builder() else {
            return Ok(None);
        };

        let mut modules = RpcModule::new(());
        modules
            .merge(HealthzApiServer::into_rpc(HealthzRpc::new(self.version.clone())))
            .map_err(|e| format!("Failed to register healthz module: {e:?}"))?;
        modules
            .merge(p2p_rpc.into_rpc())
            .map_err(|e| format!("Failed to register p2p module: {e:?}"))?;
        // The admin API is opt-in via `--rpc.enable-admin`, matching op-node.
        if config.enable_admin() {
            modules
                .merge(admin_rpc.into_rpc())
                .map_err(|e| format!("Failed to register admin module: {e:?}"))?;
        }
        modules
            .merge(
                RollupRpc::new(
                    self.version.clone(),
                    self.config.clone(),
                    engine_state_rx,
                    l2_query_client,
                    l1_state,
                    derivation_status,
                )
                .into_rpc(),
            )
            .map_err(|e| format!("Failed to register rollup module: {e:?}"))?;

        let middleware = tower::ServiceBuilder::new()
            .layer(
                ProxyGetRequestLayer::new([("/healthz", "healthz")])
                    .expect("Critical: Failed to build GET method proxy"),
            )
            .timeout(Duration::from_secs(2));
        let max_response_body_size = jsonrpsee::core::TEN_MB_SIZE_BYTES;
        let rpc_middleware = RpcServiceBuilder::new()
            .layer(RpcMetricsLayer::new(modules.method_names(), max_response_body_size));
        let server = Server::builder()
            .set_config(
                ServerConfig::builder().max_response_body_size(max_response_body_size).build(),
            )
            .set_http_middleware(middleware)
            .set_rpc_middleware(rpc_middleware)
            .build(config.socket)
            .await
            .map_err(|e: std::io::Error| format!("Failed to launch rpc server: {e:?}"))?;

        if let Ok(addr) = server.local_addr() {
            info!(target: "rpc", addr = ?addr, "RPC server bound to address");
        } else {
            error!(target: "rpc", "Failed to get local address for RPC server");
        }

        Ok(Some(RpcActor::new(server.start(modules))))
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
    /// Shutdown is unordered: when any actor exits (success, error, or panic),
    /// the umbrella cancellation token fires and all peer actors observe it on their
    /// next `select!`. Actors may log channel-closed errors while peers are torn down
    /// concurrently; this is expected and not a sign of an unclean exit.
    ///
    /// Dropping this future aborts the remaining actors. Callers are responsible for handling
    /// OS shutdown signals.
    pub async fn start(&self) -> Result<(), String> {
        // Single umbrella cancellation token shared by the supervisor and actor lifetimes.
        let cancellation = CancellationToken::new();

        // ─── cross-actor channels ───────────────────────────────────────────────────────────
        // actor request channels
        let (derivation_actor_request_tx, derivation_actor_request_rx) =
            mpsc::channel::<DerivationActorRequest>(1024);
        let (engine_actor_request_tx, engine_actor_request_rx) =
            mpsc::channel::<EngineActorRequest>(1024);
        let sequencer_builder = if self.mode().is_sequencer() {
            Some(sequencer::Builder::new(
                sequencer::Capacity::try_from(1024).map_err(|error| error.to_string())?,
                self.sequencer_config.conductor_rpc_url.clone().map(ConductorClient::new_http),
                self.sequencer_config.sequencer_stopped.not(),
                self.sequencer_config.sequencer_recovery_mode,
            ))
        } else {
            None
        };
        let sequencer_admin = sequencer_builder.as_ref().map(sequencer::Builder::handle);
        // Network actor inbound channels
        let (gossip_command_tx, gossip_command_rx) = mpsc::channel(1024);
        let (admin_payload_tx, admin_payload_rx) =
            mpsc::channel::<OpExecutionPayloadEnvelope>(1024);
        // Unsafe payloads to gossip flow from the sequencer to the signer actor and on to the
        // network actor. While signing stalls, a full sequencer queue pauses block production.
        let signer_builder = signer::Builder::new(
            signer::Capacity::try_from(32).map_err(|error| error.to_string())?,
        );
        let signer_handle = signer_builder.handle();
        let (signed_payload_tx, signed_payload_rx) = mpsc::channel::<signer::Payload>(16);
        // watch channels
        let (unsafe_head_tx, unsafe_head_rx) = watch::channel(L2BlockInfo::default());
        // The unsafe block signer: the L1 watcher keeps it current from `SystemConfig`, starting
        // from the value read at startup.
        let (signer_tx, signer_rx) = watch::channel(self.p2p_config.unsafe_block_signer);

        // ─── actor construction ─────────────────────────────────────────────────────────────
        let (engine_actor, l2_query_client, engine_state_rx) = self.build_engine_actor(
            engine_actor_request_rx,
            derivation_actor_request_tx.clone(),
            unsafe_head_tx,
        );

        let (derivation, derivation_status) = self
            .build_derivation_actor(engine_actor_request_tx.clone(), derivation_actor_request_rx)
            .await?;

        // Start the block signer before the network, so a misconfigured or unreachable remote
        // signer fails before the node binds its p2p ports.
        let signer_actor = self.build_signer().await?.map(|signer| {
            signer_builder.build(
                signer,
                self.config.l2_chain_id.id(),
                signer_rx.clone(),
                signed_payload_tx,
                cancellation.clone(),
            )
        });

        // Build and start the libp2p swarm upstream of `NetworkActor::new` so the constructor
        // stays sync.
        let handler: NetworkHandler = self
            .network_builder(signer_rx)
            .build()
            .map_err(|e| format!("Failed to build network: {e:?}"))?
            .start()
            .await
            .map_err(|e| format!("Failed to start network: {e:?}"))?;

        let discovery = handler.discovery.clone();
        let network = NetworkActor::new(
            QueuedNetworkEngineClient { engine_actor_request_tx: engine_actor_request_tx.clone() },
            handler,
            gossip_command_rx,
            admin_payload_rx,
            signed_payload_rx,
        );

        let p2p_rpc = P2pRpc::new(network.gossip_query_handle(), discovery, gossip_command_tx);

        let (l1_watcher, l1_state) =
            self.build_l1_watcher(derivation_actor_request_tx, signer_tx)?;

        let sequencer_actor = sequencer_builder.map(|builder| {
            self.build_sequencer(
                engine_actor_request_tx.clone(),
                signer_handle,
                unsafe_head_rx,
                l1_state.clone(),
                builder,
                cancellation.clone(),
            )
        });

        let admin_rpc = AdminRpc::new(sequencer_admin, engine_actor_request_tx, admin_payload_tx);
        let rpc = self
            .build_rpc_actor(
                l2_query_client,
                engine_state_rx,
                admin_rpc,
                p2p_rpc,
                l1_state,
                derivation_status,
            )
            .await?;

        let mut supervisor = Supervisor::new(cancellation.clone());
        if let Some(rpc) = rpc {
            supervisor.spawn("rpc", run_node_actor(rpc, cancellation.clone()));
        }
        if let Some(sequencer) = sequencer_actor {
            supervisor.spawn("sequencer", sequencer);
        }
        if let Some(signer) = signer_actor {
            supervisor.spawn("signer", signer);
        }
        supervisor.spawn("network", run_node_actor(network, cancellation.clone()));
        supervisor.spawn("l1", run_node_actor(l1_watcher, cancellation.clone()));
        supervisor.spawn("derivation", run_node_actor(derivation, cancellation.clone()));
        supervisor.spawn("engine", run_node_actor(engine_actor, cancellation));
        supervisor.wait().await
    }
}

/// Adapts an existing step-based actor into a cancellable lifetime future.
async fn run_node_actor<A: NodeActor>(
    mut actor: A,
    cancellation: CancellationToken,
) -> Result<(), A::Error> {
    loop {
        tokio::select! {
            biased;
            _ = cancellation.cancelled() => return Ok(()),
            result = actor.step() => result?,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        future::pending,
        sync::{
            Arc,
            atomic::{AtomicUsize, Ordering},
        },
    };
    use tokio::sync::oneshot;

    struct StepActor {
        calls: Arc<AtomicUsize>,
        fail_after: Option<usize>,
        started: Option<oneshot::Sender<()>>,
    }

    #[async_trait::async_trait]
    impl NodeActor for StepActor {
        type Error = &'static str;

        async fn step(&mut self) -> Result<(), Self::Error> {
            let calls = self.calls.fetch_add(1, Ordering::Relaxed) + 1;
            if let Some(started) = self.started.take() {
                let _ = started.send(());
            }
            match self.fail_after {
                Some(limit) if calls >= limit => Err("step failed"),
                Some(_) => Ok(()),
                None => pending().await,
            }
        }
    }

    #[tokio::test]
    async fn adapter_repeats_steps_until_error() {
        let calls = Arc::new(AtomicUsize::new(0));
        let actor = StepActor { calls: calls.clone(), fail_after: Some(3), started: None };

        let result = run_node_actor(actor, CancellationToken::new()).await;

        assert_eq!(result, Err("step failed"));
        assert_eq!(calls.load(Ordering::Relaxed), 3);
    }

    #[tokio::test]
    async fn adapter_prioritizes_cancellation_before_stepping() {
        let cancellation = CancellationToken::new();
        cancellation.cancel();
        let calls = Arc::new(AtomicUsize::new(0));
        let actor = StepActor { calls: calls.clone(), fail_after: Some(1), started: None };

        assert_eq!(run_node_actor(actor, cancellation).await, Ok(()));
        assert_eq!(calls.load(Ordering::Relaxed), 0);
    }

    #[tokio::test]
    async fn successful_lifetime_cancels_pending_step_actor() {
        let cancellation = CancellationToken::new();
        let mut supervisor = Supervisor::new(cancellation.clone());
        let (started_tx, started_rx) = oneshot::channel();
        let actor = StepActor {
            calls: Arc::new(AtomicUsize::new(0)),
            fail_after: None,
            started: Some(started_tx),
        };
        supervisor.spawn("stepping", run_node_actor(actor, cancellation.clone()));
        supervisor.spawn("lifetime", async move {
            started_rx.await.unwrap();
            Ok::<(), std::io::Error>(())
        });

        assert_eq!(supervisor.wait().await, Ok(()));
        assert!(cancellation.is_cancelled());
    }
}
