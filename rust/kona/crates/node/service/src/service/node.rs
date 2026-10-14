//! Contains the [`RollupNode`] implementation.
use super::{Supervisor, adapters, error::RollupNodeError, middleware::RpcMetricsLayer, startup};
use crate::{
    ChainSpec, ConductorClient, L1OriginSelector, NetworkConfig, NodeMode, RollupNodeClients,
    RollupNodeConfig, SequencerConfig, service::BufferImportedBlocks, signer,
};
use backon::ExponentialBuilder;
use jsonrpsee::{
    RpcModule,
    server::{
        Server, ServerConfig, ServerHandle,
        middleware::{http::ProxyGetRequestLayer, rpc::RpcServiceBuilder},
    },
};
use kona_derive::{BlobProviderError, StatefulAttributesBuilder};
use kona_node_actors::{
    derivation, discovery, engine, finality, l1, l1_signer_updater, network, sequencer,
};
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, BufferedAlloyL2ChainProvider, OnlineBlobProvider,
    OnlinePipeline,
};
use kona_providers_local::BufferedL2Provider;
use kona_rpc::{
    AdminApiServer, AdminRpc, HealthzApiServer, HealthzRpc, OpP2PApiServer, P2pRpc,
    RollupNodeApiServer, RollupRpc, RpcBuilder,
};
use kona_sources::{BlockSignerHandler, EngineApi, UntrustedRpc};
use std::{convert::Infallible, future::Future, pin::Pin, sync::Arc, time::Duration};
use tokio::sync::watch;

const DERIVATION_PROVIDER_CACHE_SIZE: usize = 1024;
/// How many recently imported blocks to keep for the local L2 lookups.
///
/// Every lookup served from here is for the block being built on top of, which the engine recorded
/// one step earlier. The size absorbs what lands in between — gossiped unsafe blocks keep
/// arriving, and on a sequencer the two builders ask about different heads.
const IMPORTED_BLOCK_BUFFER_SIZE: usize = 32;
const HEAD_STREAM_POLL_INTERVAL: u64 = 4;

/// How long the L1 actor waits before reading L1 again after a failed read.
const L1_RETRY_DELAY: Duration = Duration::from_secs(1);
const FINALIZED_STREAM_POLL_INTERVAL: u64 = 60;

/// The standard implementation of the [`RollupNode`] service, using the governance approved OP
/// Stack configuration of components.
#[derive(Debug)]
pub struct RollupNode {
    /// The application version reported by the RPC server.
    pub(crate) version: String,
    /// The rules of the chain the node follows.
    pub(crate) chain: ChainSpec,
    /// The clients the node talks to the L1 and L2 chains with.
    pub(crate) clients: RollupNodeClients,
    /// The client for the L2 execution layer's Engine API, built from the clients and the rollup
    /// config.
    pub(crate) engine_api: Arc<EngineApi>,
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
}

/// The lifetime future of whichever derivation actor the node is configured to run.
type DerivationTask = Pin<Box<dyn Future<Output = Result<(), derivation::ActorError>> + Send>>;

/// The lifetime future of the finality actor.
type FinalityTask = Pin<Box<dyn Future<Output = Result<(), finality::ActorError>> + Send>>;

impl RollupNode {
    /// Creates the node from its configuration and the clients it talks to the L1 and L2 chains
    /// with, usually built by [`RollupNodeClients::connect`] from the node's
    /// [`RollupNodeEndpoints`](crate::RollupNodeEndpoints).
    pub fn new(config: RollupNodeConfig, clients: RollupNodeClients) -> Self {
        // The buffer's reorg depth governs its handling of chain events, which nothing feeds it:
        // lookups are keyed by hash, so a reorged-out block is never returned regardless.
        let l2_block_buffer = BufferedL2Provider::new(
            config.chain.rollup_config.clone(),
            IMPORTED_BLOCK_BUFFER_SIZE,
            0,
        );
        let engine_api =
            Arc::new(EngineApi::new(clients.engine.clone(), config.chain.rollup_config.clone()));
        Self {
            version: config.version,
            chain: config.chain,
            clients,
            engine_api,
            mode: config.mode,
            rpc_builder: config.rpc_config,
            p2p_config: config.p2p_config,
            sequencer_config: config.sequencer_config,
            l2_block_buffer,
        }
    }

    /// The mode of operation for the node.
    const fn mode(&self) -> NodeMode {
        self.mode
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
                self.chain.rollup_config.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            ),
        );

        StatefulAttributesBuilder::new(
            self.chain.rollup_config.clone(),
            self.chain.l1_config.clone(),
            l2_derivation_provider,
            l1_derivation_provider,
            Some(self.chain.dependency_set.clone()),
        )
    }

    /// Creates the derivation pipeline, which reads L1 heights from the L1 actor's `l1` handle.
    async fn create_pipeline(
        &self,
        l1: l1::Handle,
    ) -> Result<OnlinePipeline<derivation::L1ChainProvider>, BlobProviderError> {
        // Create the caching L1/L2 EL providers for derivation.
        let l1_derivation_provider = derivation::L1ChainProvider::new(
            l1,
            AlloyChainProvider::new(self.clients.l1.clone(), DERIVATION_PROVIDER_CACHE_SIZE),
        );
        let l2_derivation_provider = BufferedAlloyL2ChainProvider::new(
            self.l2_block_buffer.clone(),
            AlloyL2ChainProvider::new(
                self.clients.l2.clone(),
                self.chain.rollup_config.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            ),
        );

        Ok(OnlinePipeline::new_polled(
            self.chain.rollup_config.clone(),
            self.chain.l1_config.clone(),
            OnlineBlobProvider::init(self.clients.l1_beacon.clone()).await?,
            l1_derivation_provider,
            l2_derivation_provider,
            Some(self.chain.dependency_set.clone()),
        ))
    }

    /// Wires the engine dependencies and returns its lifetime future.
    fn build_engine(
        &self,
        builder: engine::Builder,
        client: Arc<EngineApi>,
    ) -> impl Future<Output = Result<(), engine::ActorError>> + Send + 'static {
        builder.build(
            client,
            self.clients.l1.clone(),
            self.chain.rollup_config.clone(),
            Arc::new(BufferImportedBlocks::new(self.l2_block_buffer.clone())),
        )
    }

    /// Selects between the standard and delegate derivation actors and builds the chosen one.
    async fn build_derivation(
        &self,
        builder: derivation::Builder,
        engine: engine::Handle,
        l1: l1::Handle,
    ) -> Result<DerivationTask, String> {
        if let Some(provider) = self.clients.derivation_delegate.clone() {
            // L1 Provider for sanity checking Derivation Delegation
            let l1_provider =
                AlloyChainProvider::new(self.clients.l1.clone(), DERIVATION_PROVIDER_CACHE_SIZE);
            Ok(Box::pin(builder.build_delegated(
                adapters::Engine::new(engine),
                provider,
                l1_provider,
            )))
        } else {
            let pipeline = self
                .create_pipeline(l1.clone())
                .await
                .map_err(|error| format!("Failed to initialize L1 blob provider: {error}"))?;
            Ok(Box::pin(builder.build(adapters::Engine::new(engine), pipeline, l1)))
        }
    }

    /// Builds the finality actor when the node derives from L1; otherwise returns `None`, since
    /// delegated derivation finalizes the delegate's blocks.
    fn build_finality(&self, engine: engine::Handle, l1: l1::Handle) -> Option<FinalityTask> {
        self.clients.derivation_delegate.is_none().then(|| {
            let (engine_state, resets) = (engine.state(), engine.resets());
            Box::pin(finality::build(
                adapters::Engine::new(engine),
                engine_state,
                resets,
                adapters::L1::new(l1),
            )) as FinalityTask
        })
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
        l1: l1::Handle,
        builder: sequencer::Builder,
    ) -> impl Future<Output = Result<(), sequencer::ActorError>> + Send + 'static {
        let origin_selector = L1OriginSelector::new(
            self.chain.rollup_config.clone(),
            l1,
            self.sequencer_config.l1_conf_delay,
        );

        builder.build(
            self.create_attributes_builder(),
            adapters::Engine::new(engine),
            origin_selector,
            self.chain.rollup_config.clone(),
            self.sequencer_config.conductor_rpc_url.clone().map(ConductorClient::new_http),
            adapters::Signer(signer),
        )
    }

    /// Assembles the JSON-RPC module set, starts the server, and returns its lifetime future.
    #[allow(clippy::too_many_arguments)]
    async fn build_rpc(
        &self,
        l2_query_client: EngineApi,
        engine_state: watch::Receiver<engine::EngineState>,
        admin_rpc: AdminRpc,
        p2p_rpc: P2pRpc,
        l1: l1::Handle,
        derivation_status: watch::Receiver<derivation::DerivationStatus>,
    ) -> Result<impl Future<Output = Result<(), &'static str>> + Send + 'static, String> {
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
                    self.chain.rollup_config.clone(),
                    engine_state,
                    l2_query_client,
                    l1,
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

        Ok(supervise_rpc_server(server.start(modules)))
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
    /// - [`RollupNodeError::L1FinalizedBlock`] if the finalized L1 block cannot be read;
    /// - [`RollupNodeError::UnsafeBlockSigner`] if the unsafe block signer cannot be read from L1.
    ///
    /// Any other component that fails to start is a [`RollupNodeError::Startup`]. Once the actors
    /// run, the node stops when the first of them exits: this returns `Ok(())` if it exited
    /// cleanly, and [`RollupNodeError::Stopped`] if it failed.
    pub async fn start(&self) -> Result<(), RollupNodeError> {
        // ─── startup checks ─────────────────────────────────────────────────────────────────
        startup::check_listen_ports(&self.p2p_config)?;
        startup::validate_engine_jwt(&self.engine_api, ExponentialBuilder::default()).await?;

        // ─── cross-actor channels ───────────────────────────────────────────────────────────
        // The L1 actor and the startup read share the L1 client: the read fails fast, and the actor
        // retries until it can read L1.
        let l1_rpc = UntrustedRpc::new(self.clients.l1.clone());
        let l1_finalized =
            l1::finalized_block(&l1_rpc).await.map_err(RollupNodeError::L1FinalizedBlock)?;
        let l1_builder = l1::Builder::new(l1_rpc.retrying(L1_RETRY_DELAY), l1_finalized);
        let l1 = l1_builder.handle();
        let initial_signer = l1_signer_updater::signer_at_l1_head(
            &self.clients.l1,
            self.chain.rollup_config.l1_system_config_address,
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
        let discovery_builder = discovery::Builder::new(
            discovery::Capacity::try_from(1024)
                .map_err(|error| RollupNodeError::Startup(error.to_string()))?,
        );
        let discovery = discovery_builder.handle();
        // Unsafe payloads to gossip flow from the sequencer to the signer actor and on to the
        // network actor. While signing stalls, a full sequencer queue pauses block production.
        let signer_builder = signer::Builder::new(
            signer::Capacity::try_from(32)
                .map_err(|error| RollupNodeError::Startup(error.to_string()))?,
        );
        let signer_handle = signer_builder.handle();

        // ─── actor construction ─────────────────────────────────────────────────────────────
        let engine_api = self.engine_api.clone();
        let l2_query_client = engine_api.as_ref().clone();
        let engine_actor = self.build_engine(engine_builder, engine_api);

        let derivation_actor = self
            .build_derivation(derivation_builder, engine.clone(), l1.clone())
            .await
            .map_err(RollupNodeError::Startup)?;
        let finality_actor = self.build_finality(engine.clone(), l1.clone());

        // Start the block signer before the network, so a misconfigured or unreachable remote
        // signer fails before the node binds its p2p ports.
        let block_signer = self.build_signer().await.map_err(RollupNodeError::Startup)?;

        let (gossip, discovery_driver) = self
            .p2p_config
            .clone()
            .start(signer_rx.clone())
            .await
            .map_err(|e| RollupNodeError::Startup(format!("Failed to start network: {e}")))?;
        let network_builder = network::Builder::new(
            network::Capacity::try_from(1024)
                .map_err(|error| RollupNodeError::Startup(error.to_string()))?,
            gossip,
        );
        let network = network_builder.handle();
        let discovery_actor =
            discovery_builder.build(discovery_driver, adapters::Network(network.clone()));
        let network_actor = network_builder
            .build(adapters::Engine::new(engine.clone()), adapters::Discovery(discovery.clone()));
        let signer_actor = block_signer.map(|signer| {
            signer_builder.build(
                signer,
                self.chain.rollup_config.l2_chain_id.id(),
                signer_rx,
                adapters::Network(network.clone()),
            )
        });

        let p2p_rpc = P2pRpc::new(
            network.state(),
            network,
            discovery,
            self.chain.rollup_config.l2_chain_id.id(),
        );

        let signer_updater = signer_updater_builder.build(
            self.clients.l1.clone(),
            self.chain.rollup_config.l1_system_config_address,
            adapters::L1::new(l1.clone()),
        );
        let l1_actor = l1_builder.build(l1::Config {
            head_interval: Duration::from_secs(HEAD_STREAM_POLL_INTERVAL),
            finalized_interval: Duration::from_secs(FINALIZED_STREAM_POLL_INTERVAL),
        });

        let sequencer_actor = sequencer_builder.map(|builder| {
            self.build_sequencer(engine.clone(), signer_handle, l1.clone(), builder)
        });

        let engine_state = engine.state();
        let admin_rpc = AdminRpc::new(
            sequencer_handle,
            self.sequencer_config.conductor_rpc_url.is_some(),
            engine,
        );
        let rpc = self
            .build_rpc(l2_query_client, engine_state, admin_rpc, p2p_rpc, l1, derivation.status())
            .await
            .map_err(RollupNodeError::Startup)?;

        let mut supervisor = Supervisor::new();
        supervisor.spawn("rpc", rpc);
        if let Some(sequencer) = sequencer_actor {
            supervisor.spawn("sequencer", sequencer);
        }
        if let Some(signer) = signer_actor {
            supervisor.spawn("signer", signer);
        }
        supervisor.spawn("network", network_actor);
        supervisor.spawn("discovery", discovery_actor);
        // The L1 actor runs until it is dropped: its output, `Infallible`, never exists.
        supervisor.spawn("l1", async move { Err::<(), _>(l1_actor.await) });
        supervisor.spawn("l1_signer_updater", async move {
            signer_updater.await;
            Ok::<(), Infallible>(())
        });
        supervisor.spawn("derivation", derivation_actor);
        if let Some(finality) = finality_actor {
            supervisor.spawn("finality", finality);
        }
        supervisor.spawn("engine", engine_actor);
        supervisor.wait().await.map_err(RollupNodeError::Stopped)
    }
}

/// Returns the RPC server's lifetime future, which fails once the server stops.
///
/// The future owns the server's handle, so dropping it stops the server.
async fn supervise_rpc_server(server: ServerHandle) -> Result<(), &'static str> {
    server.stopped().await;
    Err("server stopped unexpectedly")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn rpc_server_stopping_is_fatal() {
        let server = Server::builder().build("127.0.0.1:0").await.unwrap();
        let handle = server.start(RpcModule::new(()));
        let mut lifetime = std::pin::pin!(supervise_rpc_server(handle.clone()));
        tokio::select! {
            biased;
            result = &mut lifetime => panic!("ended while the server runs: {result:?}"),
            () = std::future::ready(()) => {}
        }

        handle.stop().unwrap();
        assert_eq!(lifetime.await, Err("server stopped unexpectedly"));
    }
}
