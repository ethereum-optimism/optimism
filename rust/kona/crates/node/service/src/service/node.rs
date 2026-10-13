//! Contains the [`RollupNode`] implementation.
use super::{Supervisor, adapters, error::RollupNodeError, middleware::RpcMetricsLayer, startup};
use crate::{
    ConductorClient, DelayedL1OriginSelectorProvider, L1OriginSelector, NetworkActor,
    NetworkBuilder, NetworkConfig, NetworkHandler, NodeActor, NodeMode, RollupNodeClients,
    RollupNodeConfig, RpcActor, SequencerConfig, service::BufferImportedBlocks, signer,
};
use alloy_primitives::Address;
use backon::ExponentialBuilder;
use jsonrpsee::{
    RpcModule,
    server::{
        Server, ServerConfig,
        middleware::{http::ProxyGetRequestLayer, rpc::RpcServiceBuilder},
    },
};
use kona_derive::{BlobProviderError, StatefulAttributesBuilder};
use kona_engine::EngineClient;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_node_actors::{derivation, engine, l1, l1_signer_updater, sequencer};
use kona_protocol::BlockInfo;
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, BufferedAlloyL2ChainProvider, OnlineBlobProvider,
    OnlinePipeline,
};
use kona_providers_local::BufferedL2Provider;
use kona_rpc::{
    AdminApiServer, AdminRpc, HealthzApiServer, HealthzRpc, OpP2PApiServer, P2pRpc,
    RollupNodeApiServer, RollupRpc, RpcBuilder,
};
use kona_sources::BlockSignerHandler;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{convert::Infallible, future::Future, pin::Pin, sync::Arc, time::Duration};
use tokio::sync::{mpsc, watch};

const DERIVATION_PROVIDER_CACHE_SIZE: usize = 1024;
/// How many recently imported blocks to keep for the local L2 lookups.
///
/// Every lookup served from here is for the block being built on top of, which the engine recorded
/// one step earlier. The size absorbs what lands in between — gossiped unsafe blocks keep
/// arriving, and on a sequencer the two builders ask about different heads.
const IMPORTED_BLOCK_BUFFER_SIZE: usize = 32;
const HEAD_STREAM_POLL_INTERVAL: u64 = 4;
const FINALIZED_STREAM_POLL_INTERVAL: u64 = 60;

/// The standard implementation of the [`RollupNode`] service, using the governance approved OP
/// Stack configuration of components.
#[derive(Debug)]
pub struct RollupNode {
    /// The application version reported by the RPC server.
    pub(crate) version: String,
    /// The rollup configuration.
    pub(crate) config: Arc<RollupConfig>,
    /// The L1 chain configuration.
    pub(crate) l1_chain_config: Arc<L1ChainConfig>,
    /// The clients the node talks to the L1 and L2 chains with.
    pub(crate) clients: RollupNodeClients,
    /// The client for the L2 execution layer's Engine API, built from the clients and the rollup
    /// config.
    pub(crate) engine_client: Arc<EngineClient>,
    /// The mode of operation for the node.
    pub(crate) mode: NodeMode,
    /// The RPC server's configuration.
    pub(crate) rpc_builder: RpcBuilder,
    /// The P2P [`NetworkConfig`] for the node.
    pub(crate) p2p_config: NetworkConfig,
    /// The [`SequencerConfig`] for the node.
    pub(crate) sequencer_config: SequencerConfig,
    /// Blocks the engine has imported, shared with the derivation providers so they can be read
    /// locally instead of fetched back from the execution layer.
    pub(crate) l2_block_buffer: BufferedL2Provider,
    /// The interop dependency set for this chain.
    /// Mirrors op-node's `--interop.dependency-set`.
    /// [`StatefulAttributesBuilder`] constructor panics otherwise.
    pub(crate) dependency_set: Option<Arc<DependencySet>>,
}

/// The lifetime future of whichever derivation actor the node is configured to run.
type DerivationTask = Pin<Box<dyn Future<Output = Result<(), derivation::ActorError>> + Send>>;

impl RollupNode {
    /// Creates the node from its configuration and the clients it talks to the L1 and L2 chains
    /// with, usually built by [`RollupNodeClients::connect`] from the node's
    /// [`RollupNodeEndpoints`](crate::RollupNodeEndpoints).
    pub fn new(config: RollupNodeConfig, clients: RollupNodeClients) -> Self {
        // The buffer's reorg depth governs its handling of chain events, which nothing feeds it:
        // lookups are keyed by hash, so a reorged-out block is never returned regardless.
        let l2_block_buffer =
            BufferedL2Provider::new(config.rollup_config.clone(), IMPORTED_BLOCK_BUFFER_SIZE, 0);
        let engine_client = Arc::new(EngineClient::new(
            clients.l1.clone(),
            clients.engine.clone(),
            config.rollup_config.clone(),
        ));
        Self {
            version: config.version,
            config: config.rollup_config,
            l1_chain_config: config.l1_chain_config,
            clients,
            engine_client,
            mode: config.mode,
            rpc_builder: config.rpc_config,
            p2p_config: config.p2p_config,
            sequencer_config: config.sequencer_config,
            l2_block_buffer,
            dependency_set: config.dependency_set,
        }
    }

    /// The mode of operation for the node.
    const fn mode(&self) -> NodeMode {
        self.mode
    }

    /// Creates a network builder for the node whose gossip validation follows
    /// `unsafe_block_signer`.
    fn network_builder(&self, unsafe_block_signer: watch::Receiver<Address>) -> NetworkBuilder {
        NetworkBuilder::from_config(self.p2p_config.clone(), unsafe_block_signer)
    }

    /// Returns an rpc builder for the node.
    fn rpc_builder(&self) -> RpcBuilder {
        self.rpc_builder.clone()
    }

    /// Returns the sequencer builder for the node.
    fn create_attributes_builder(
        &self,
    ) -> StatefulAttributesBuilder<AlloyChainProvider, BufferedAlloyL2ChainProvider> {
        let l1_derivation_provider =
            AlloyChainProvider::new(self.clients.l1.clone(), DERIVATION_PROVIDER_CACHE_SIZE);
        let l2_derivation_provider = BufferedAlloyL2ChainProvider::new(
            self.l2_block_buffer.clone(),
            AlloyL2ChainProvider::new(
                self.clients.l2.clone(),
                self.config.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            ),
        );

        StatefulAttributesBuilder::new(
            self.config.clone(),
            self.l1_chain_config.clone(),
            l2_derivation_provider,
            l1_derivation_provider,
            self.dependency_set.clone(),
        )
    }

    async fn create_pipeline(&self) -> Result<OnlinePipeline, BlobProviderError> {
        // Create the caching L1/L2 EL providers for derivation.
        let l1_derivation_provider =
            AlloyChainProvider::new(self.clients.l1.clone(), DERIVATION_PROVIDER_CACHE_SIZE);
        let l2_derivation_provider = BufferedAlloyL2ChainProvider::new(
            self.l2_block_buffer.clone(),
            AlloyL2ChainProvider::new(
                self.clients.l2.clone(),
                self.config.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            ),
        );

        Ok(OnlinePipeline::new_polled(
            self.config.clone(),
            self.l1_chain_config.clone(),
            OnlineBlobProvider::init(self.clients.l1_beacon.clone()).await?,
            l1_derivation_provider,
            l2_derivation_provider,
            self.dependency_set.clone(),
        ))
    }

    /// Wires the engine dependencies and returns its lifetime future.
    fn build_engine(
        &self,
        builder: engine::Builder,
        client: Arc<EngineClient>,
    ) -> impl Future<Output = Result<(), engine::ActorError>> + Send + 'static {
        builder.build(
            client,
            self.config.clone(),
            Arc::new(BufferImportedBlocks::new(self.l2_block_buffer.clone())),
        )
    }

    /// Selects between the standard and delegate derivation actors and builds the chosen one.
    async fn build_derivation(
        &self,
        builder: derivation::Builder,
        engine: engine::Handle,
        head: watch::Receiver<BlockInfo>,
        finalized: watch::Receiver<BlockInfo>,
    ) -> Result<DerivationTask, String> {
        if let Some(provider) = self.clients.derivation_delegate.clone() {
            // L1 Provider for sanity checking Derivation Delegation
            let l1_provider =
                AlloyChainProvider::new(self.clients.l1.clone(), DERIVATION_PROVIDER_CACHE_SIZE);
            Ok(Box::pin(builder.build_delegated(adapters::Engine(engine), provider, l1_provider)))
        } else {
            let pipeline = self
                .create_pipeline()
                .await
                .map_err(|error| format!("Failed to initialize L1 blob provider: {error}"))?;
            Ok(Box::pin(builder.build(adapters::Engine(engine), pipeline, head, finalized)))
        }
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
        engine: engine::Handle,
        signer: signer::Handle,
        l1_head: watch::Receiver<BlockInfo>,
        builder: sequencer::Builder,
    ) -> impl Future<Output = Result<(), sequencer::ActorError>> + Send + 'static {
        let delayed_l1_provider = DelayedL1OriginSelectorProvider::new(
            self.clients.l1.clone(),
            l1_head,
            self.sequencer_config.l1_conf_delay,
        );
        let delayed_origin_selector =
            L1OriginSelector::new(self.config.clone(), delayed_l1_provider);

        builder.build(
            self.create_attributes_builder(),
            adapters::Engine(engine),
            delayed_origin_selector,
            self.config.clone(),
            self.sequencer_config.conductor_rpc_url.clone().map(ConductorClient::new_http),
            adapters::Signer(signer),
        )
    }

    /// Assembles the JSON-RPC module set, performs the initial server launch, and returns the
    /// configured [`RpcActor`].
    #[allow(clippy::too_many_arguments)]
    async fn build_rpc_actor(
        &self,
        l2_query_client: EngineClient,
        engine_state: watch::Receiver<engine::EngineState>,
        admin_rpc: AdminRpc,
        p2p_rpc: P2pRpc,
        head_l1: watch::Receiver<BlockInfo>,
        safe_l1: watch::Receiver<BlockInfo>,
        finalized_l1: watch::Receiver<BlockInfo>,
        derivation_status: watch::Receiver<derivation::DerivationStatus>,
    ) -> Result<RpcActor, String> {
        let config = self.rpc_builder();

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
                    engine_state,
                    l2_query_client,
                    head_l1,
                    safe_l1,
                    finalized_l1,
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

        Ok(RpcActor::new(server.start(modules)))
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
    /// the supervisor aborts the remaining actors. Actors may log channel-closed errors
    /// while peers are torn down concurrently.
    ///
    /// Dropping this future aborts the remaining actors. Callers are responsible for handling
    /// OS shutdown signals.
    ///
    /// ## Errors
    ///
    /// Before starting any actor, this fails with:
    /// - [`RollupNodeError::ListenAddressUnavailable`] if a P2P listen address cannot be bound;
    /// - [`RollupNodeError::JwtRejected`] if the engine rejects the node's JWT;
    /// - [`RollupNodeError::EngineCapabilityExchange`] if the capability exchange that checks the
    ///   JWT secret still fails after the retries;
    /// - [`RollupNodeError::UnsafeBlockSigner`] if the unsafe block signer cannot be read from L1.
    ///
    /// Any other component that fails to start is a [`RollupNodeError::Startup`]. Once the actors
    /// run, the node stops when the first of them exits: this returns `Ok(())` if it exited
    /// cleanly, and [`RollupNodeError::Stopped`] if it failed.
    pub async fn start(&self) -> Result<(), RollupNodeError> {
        // ─── startup checks ─────────────────────────────────────────────────────────────────
        startup::check_listen_ports(&self.p2p_config)?;
        startup::validate_engine_jwt(&self.engine_client, ExponentialBuilder::default()).await?;

        // ─── cross-actor channels ───────────────────────────────────────────────────────────
        let head_builder = l1::Builder::new(BlockInfo::default());
        let safe_builder = l1::Builder::new(BlockInfo::default());
        let finalized_builder = l1::Builder::new(BlockInfo::default());
        let head = head_builder.handle();
        let safe = safe_builder.handle();
        let finalized = finalized_builder.handle();
        let initial_signer = l1_signer_updater::signer_at_l1_head(
            &self.clients.l1,
            self.config.l1_system_config_address,
        )
        .await
        .map_err(RollupNodeError::UnsafeBlockSigner)?;
        let signer_updater_builder = l1_signer_updater::Builder::new(initial_signer);
        let signer_rx = signer_updater_builder.handle();
        // actor request channels
        let derivation_builder = derivation::Builder::new();
        let derivation = derivation_builder.handle();
        let engine_builder = engine::Builder::new(
            engine::Capacity::try_from(1024)
                .map_err(|error| RollupNodeError::Startup(error.to_string()))?,
        );
        let engine = engine_builder.handle();
        let sequencer_builder = if self.mode().is_sequencer() {
            Some(sequencer::Builder::new(
                sequencer::Capacity::try_from(1024)
                    .map_err(|error| RollupNodeError::Startup(error.to_string()))?,
            ))
        } else {
            None
        };
        let sequencer_handle = sequencer_builder.as_ref().map(sequencer::Builder::handle);
        // Network actor inbound channels
        let (gossip_command_tx, gossip_command_rx) = mpsc::channel(1024);
        let (admin_payload_tx, admin_payload_rx) =
            mpsc::channel::<OpExecutionPayloadEnvelope>(1024);
        // Unsafe payloads to gossip flow from the sequencer to the signer actor and on to the
        // network actor. While signing stalls, a full sequencer queue pauses block production.
        let signer_builder = signer::Builder::new(
            signer::Capacity::try_from(32)
                .map_err(|error| RollupNodeError::Startup(error.to_string()))?,
        );
        let signer_handle = signer_builder.handle();
        let (signed_payload_tx, signed_payload_rx) = mpsc::channel::<signer::Payload>(16);

        // ─── actor construction ─────────────────────────────────────────────────────────────
        let engine_client = self.engine_client.clone();
        let l2_query_client = engine_client.as_ref().clone();
        let engine_actor = self.build_engine(engine_builder, engine_client);

        let derivation_actor = self
            .build_derivation(derivation_builder, engine.clone(), head.clone(), finalized.clone())
            .await
            .map_err(RollupNodeError::Startup)?;

        // Start the block signer before the network, so a misconfigured or unreachable remote
        // signer fails before the node binds its p2p ports.
        let signer_actor =
            self.build_signer().await.map_err(RollupNodeError::Startup)?.map(|signer| {
                signer_builder.build(
                    signer,
                    self.config.l2_chain_id.id(),
                    signer_rx.clone(),
                    signed_payload_tx,
                )
            });

        // Build and start the libp2p swarm upstream of `NetworkActor::new` so the constructor
        // stays sync.
        let handler: NetworkHandler = self
            .network_builder(signer_rx)
            .build()
            .map_err(|e| RollupNodeError::Startup(format!("Failed to build network: {e:?}")))?
            .start()
            .await
            .map_err(|e| RollupNodeError::Startup(format!("Failed to start network: {e:?}")))?;

        let discovery = handler.discovery.clone();
        let network = NetworkActor::new(
            adapters::Engine(engine.clone()),
            handler,
            gossip_command_rx,
            admin_payload_rx,
            signed_payload_rx,
        );

        let p2p_rpc = P2pRpc::new(network.gossip_query_handle(), discovery, gossip_command_tx);

        let signer_updater = signer_updater_builder.build(
            self.clients.l1.clone(),
            self.config.l1_system_config_address,
            head.clone(),
        );
        let l1_head = head_builder.build(
            self.clients.l1.clone(),
            l1::Tag::Head,
            Duration::from_secs(HEAD_STREAM_POLL_INTERVAL),
        );
        let l1_safe = safe_builder.build(
            self.clients.l1.clone(),
            l1::Tag::Safe,
            Duration::from_secs(HEAD_STREAM_POLL_INTERVAL),
        );
        let l1_finalized = finalized_builder.build(
            self.clients.l1.clone(),
            l1::Tag::Finalized,
            Duration::from_secs(FINALIZED_STREAM_POLL_INTERVAL),
        );

        let sequencer_actor = sequencer_builder.map(|builder| {
            self.build_sequencer(engine.clone(), signer_handle, head.clone(), builder)
        });

        let engine_state = engine.state();
        let admin_rpc = AdminRpc::new(
            sequencer_handle,
            self.sequencer_config.conductor_rpc_url.is_some(),
            engine,
            admin_payload_tx,
        );
        let rpc = self
            .build_rpc_actor(
                l2_query_client,
                engine_state,
                admin_rpc,
                p2p_rpc,
                head,
                safe,
                finalized,
                derivation.status(),
            )
            .await
            .map_err(RollupNodeError::Startup)?;

        let mut supervisor = Supervisor::new();
        supervisor.spawn("rpc", run_node_actor(rpc));
        if let Some(sequencer) = sequencer_actor {
            supervisor.spawn("sequencer", sequencer);
        }
        if let Some(signer) = signer_actor {
            supervisor.spawn("signer", signer);
        }
        supervisor.spawn("network", run_node_actor(network));
        supervisor.spawn("l1_head", l1_head);
        supervisor.spawn("l1_safe", l1_safe);
        supervisor.spawn("l1_finalized", l1_finalized);
        supervisor.spawn("l1_signer_updater", async move {
            signer_updater.await;
            Ok::<(), Infallible>(())
        });
        supervisor.spawn("derivation", derivation_actor);
        supervisor.spawn("engine", engine_actor);
        supervisor.wait().await.map_err(RollupNodeError::Stopped)
    }
}

/// Adapts an existing step-based actor into a lifetime future.
async fn run_node_actor<A: NodeActor>(mut actor: A) -> Result<(), A::Error> {
    loop {
        actor.step().await?;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    };

    struct StepActor {
        calls: Arc<AtomicUsize>,
        fail_after: usize,
    }

    #[async_trait::async_trait]
    impl NodeActor for StepActor {
        type Error = &'static str;

        async fn step(&mut self) -> Result<(), Self::Error> {
            let calls = self.calls.fetch_add(1, Ordering::Relaxed) + 1;
            if calls >= self.fail_after { Err("step failed") } else { Ok(()) }
        }
    }

    #[tokio::test]
    async fn adapter_repeats_steps_until_error() {
        let calls = Arc::new(AtomicUsize::new(0));
        let actor = StepActor { calls: calls.clone(), fail_after: 3 };
        assert_eq!(run_node_actor(actor).await, Err("step failed"));
        assert_eq!(calls.load(Ordering::Relaxed), 3);
    }
}
