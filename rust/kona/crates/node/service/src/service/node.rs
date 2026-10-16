//! Contains the [`RollupNode`] implementation.
use super::{Supervisor, adapters, error::RollupNodeError, middleware::RpcMetricsLayer, startup};
use crate::{
    ChainSpec, ConductorClient, NetworkConfig, RollupNodeClients, RollupNodeConfig,
    SequencerConfig, service::BufferImportedBlocks, signer,
};
use backon::ExponentialBuilder;
use jsonrpsee::{
    RpcModule,
    server::{
        Server, ServerConfig, ServerHandle,
        middleware::{http::ProxyGetRequestLayer, rpc::RpcServiceBuilder},
    },
};
use kona_derive::StatefulAttributesBuilder;
use kona_node_actors::{
    derivation, discovery, engine, finality, l1, l1_signer_updater, network, origin, sequencer,
};
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, BufferedAlloyL2ChainProvider, OnlineBlobProvider,
    OnlinePipeline,
};
use kona_providers_local::BufferedL2Provider;
use kona_rpc::{
    AdminApiServer, AdminRpc, HealthzApiServer, HealthzRpc, OpP2PApiServer, P2pRpc,
    RollupNodeApiServer, RollupRpc, SequencerAdminApiServer, SequencerAdminRpc,
};
use kona_sources::{BlockSigner, EngineApi, UntrustedRpc};
use std::{
    convert::Infallible, future::Future, net::SocketAddr, pin::Pin, sync::Arc, time::Duration,
};

const DERIVATION_PROVIDER_CACHE_SIZE: usize = 1024;
/// How many recently imported blocks to keep for the local L2 lookups.
///
/// Every lookup served from here is for the block being built on top of, which the engine recorded
/// one step earlier. The size absorbs what lands in between — gossiped unsafe blocks keep
/// arriving, and on a sequencer the two builders ask about different heads.
const IMPORTED_BLOCK_BUFFER_SIZE: usize = 32;
const HEAD_STREAM_POLL_INTERVAL: u64 = 4;
const FINALIZED_STREAM_POLL_INTERVAL: u64 = 60;

/// How long the L1 actor waits before reading L1 again after a failed read.
const L1_RETRY_DELAY: Duration = Duration::from_secs(1);

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
    /// The address the RPC server listens on, or `None` for a node that serves no RPC.
    pub(crate) rpc_addr: Option<SocketAddr>,
    /// The P2P [`NetworkConfig`] for the node, or `None` for a node without P2P.
    pub(crate) p2p_config: Option<NetworkConfig>,
    /// The [`SequencerConfig`] of a node that can sequence.
    pub(crate) sequencer: Option<SequencerConfig>,
    /// The block signer, which a node that can sequence and has P2P signs its blocks with.
    pub(crate) signer: Option<BlockSigner>,
    /// Blocks the engine has imported, shared with the derivation providers so they can be read
    /// locally instead of fetched back from the execution layer.
    pub(crate) l2_block_buffer: BufferedL2Provider,
}

/// The lifetime future of whichever derivation actor the node is configured to run.
type DerivationTask =
    Pin<Box<dyn Future<Output = Result<(), derivation::untrusted::ActorError>> + Send>>;

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
            Arc::new(EngineApi::new(clients.l2.clone(), config.chain.rollup_config.clone()));
        Self {
            version: config.version,
            chain: config.chain,
            clients,
            engine_api,
            rpc_addr: config.rpc_addr,
            p2p_config: config.p2p_config,
            sequencer: config.sequencer,
            signer: config.signer,
            l2_block_buffer,
        }
    }

    /// Starts the rollup node service.
    ///
    /// The rollup node listens to two sources of information to sync the L2 chain:
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
    /// A node configured with a [`SequencerConfig`] can also produce unsafe L2 blocks and send
    /// them to the network over p2p gossip. Its sequencer starts stopped: the operator starts it
    /// with `admin_startSequencer`. A node without one does not serve the sequencer's admin
    /// methods.
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
    /// - [`RollupNodeError::NoBlockSigner`] if the node can sequence and has P2P, but no block
    ///   signer;
    /// - [`RollupNodeError::ListenAddressUnavailable`] if a P2P listen address cannot be bound;
    /// - [`RollupNodeError::JwtRejected`] if the engine rejects the node's JWT;
    /// - [`RollupNodeError::EngineCapabilityExchange`] if the capability exchange that checks the
    ///   JWT secret still fails after the retries;
    /// - [`RollupNodeError::L2Heads`] if the execution layer's heads, which the engine starts from,
    ///   cannot be read;
    /// - [`RollupNodeError::L1FinalizedBlock`] if the finalized L1 block cannot be read;
    /// - [`RollupNodeError::UnsafeBlockSigner`] if the unsafe block signer, which a node with P2P
    ///   checks gossiped blocks against, cannot be read from L1.
    ///
    /// Any other component that fails to start is a [`RollupNodeError::Startup`]. Once the actors
    /// run, the node stops when the first of them exits: this returns `Ok(())` if it exited
    /// cleanly, and [`RollupNodeError::Stopped`] if it failed.
    pub async fn start(&self) -> Result<(), RollupNodeError> {
        // ─── startup checks ─────────────────────────────────────────────────────────────────
        let block_signer = self.block_signer()?;
        if let Some(p2p_config) = &self.p2p_config {
            startup::check_listen_ports(p2p_config)?;
        }
        startup::validate_engine_jwt(&self.engine_api, ExponentialBuilder::default()).await?;
        // The engine starts from the execution layer's heads, so that its state, which the RPC
        // serves, is never empty. The read fails fast, as the execution layer just answered.
        let engine_state =
            engine::execution_layer_state(&self.chain.rollup_config, &self.engine_api)
                .await
                .map_err(RollupNodeError::L2Heads)?;

        // ─── cross-actor channels ───────────────────────────────────────────────────────────
        // The L1 actor and the startup read share the L1 client: the read fails fast, and the actor
        // retries until it can read L1.
        let l1_rpc = UntrustedRpc::new(self.clients.l1.clone());
        let l1_finalized =
            l1::finalized_block(&l1_rpc).await.map_err(RollupNodeError::L1FinalizedBlock)?;
        let l1_builder = l1::Builder::new(l1_rpc.retrying(L1_RETRY_DELAY), l1_finalized);
        let l1 = l1_builder.handle();
        // actor request channels
        let engine_builder = engine::Builder::new(engine_state);
        let engine = engine_builder.handle();
        let sequencer = self.sequencer.as_ref().map(|config| (config, sequencer::Builder::new()));
        let sequencer_admin_rpc = sequencer.as_ref().map(|(config, builder)| {
            SequencerAdminRpc::new(builder.handle(), config.conductor_rpc_url.is_some())
        });

        // ─── derivation and finality ────────────────────────────────────────────────────────
        // A node that delegates derivation follows the delegate's safe and finalized heads, so it
        // neither derives from L1 nor runs the finality actor.
        let (safety, finality, derivation_actor, finality_actor): (
            adapters::Safety,
            adapters::Finality,
            DerivationTask,
            Option<FinalityTask>,
        ) = if let Some(provider) = self.clients.derivation_delegate.clone() {
            // L1 Provider for sanity checking Derivation Delegation
            let l1_provider =
                AlloyChainProvider::new(self.clients.l1.clone(), DERIVATION_PROVIDER_CACHE_SIZE);
            let builder = derivation::trusted::Builder::new();
            let safety = adapters::Safety::Trusted(Box::new(builder.handle()));
            let finality = adapters::Finality::Trusted(Box::new(builder.handle()));
            let actor = builder.build(provider, l1_provider);
            // The actor runs until it is dropped: its output, `Infallible`, never exists.
            (safety, finality, Box::pin(async move { match actor.await {} }), None)
        } else {
            // Create the caching L1/L2 EL providers for derivation. The pipeline reads L1
            // heights from the L1 actor.
            let l1_derivation_provider = derivation::L1ChainProvider::new(
                l1.clone(),
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
            let blob_provider = OnlineBlobProvider::init(self.clients.l1_beacon.clone())
                .await
                .map_err(|error| {
                    RollupNodeError::Startup(format!(
                        "Failed to initialize L1 blob provider: {error}"
                    ))
                })?;
            let pipeline = OnlinePipeline::new_polled(
                self.chain.rollup_config.clone(),
                self.chain.l1_config.clone(),
                blob_provider,
                l1_derivation_provider,
                l2_derivation_provider,
                Some(self.chain.dependency_set.clone()),
            );
            let builder = derivation::untrusted::Builder::new();
            let safety = adapters::Safety::Untrusted(builder.handle());
            let actor = builder.build(pipeline, l1.clone(), self.clients.l2.clone());
            let finality_builder = finality::Builder::new();
            let finality = adapters::Finality::Untrusted(finality_builder.handle());
            let finality_actor = finality_builder.build(adapters::L1::new(l1.clone()));
            (safety, finality, Box::pin(actor), Some(Box::pin(finality_actor)))
        };

        // ─── engine ─────────────────────────────────────────────────────────────────────────
        let engine_actor = engine_builder.build(
            self.engine_api.clone(),
            l1.clone(),
            self.chain.rollup_config.clone(),
            Arc::new(BufferImportedBlocks::new(self.l2_block_buffer.clone())),
            safety.clone(),
            finality,
        );

        // ─── network ────────────────────────────────────────────────────────────────────────
        // A node without P2P runs no network, discovery or block signer actor, serves no P2P RPC,
        // and does not sign the blocks its sequencer builds, having no peers to gossip them to.
        let mut p2p_actors = None;
        let mut p2p_rpc = None;
        let mut sequencer_signer = None;
        if let Some(p2p_config) = self.p2p_config.clone() {
            // The network accepts gossiped blocks signed by the unsafe block signer that L1 names.
            let initial_signer = l1_signer_updater::signer_at_l1_head(
                &self.clients.l1,
                self.chain.rollup_config.l1_system_config_address,
            )
            .await
            .map_err(RollupNodeError::UnsafeBlockSigner)?;
            let signer_updater_builder = l1_signer_updater::Builder::new(initial_signer);
            let signer_rx = signer_updater_builder.handle();
            let signer_updater = signer_updater_builder.build(
                self.clients.l1.clone(),
                self.chain.rollup_config.l1_system_config_address,
                adapters::L1::new(l1.clone()),
            );

            // Start the sequencer's block signer before the network, so a misconfigured or
            // unreachable remote signer fails before the node binds its p2p ports.
            let block_signer = match block_signer {
                Some(signer) => Some(signer.clone().start().await.map_err(|e| {
                    RollupNodeError::Startup(format!("Failed to start block signer: {e}"))
                })?),
                None => None,
            };

            let (gossip, discovery_driver) = p2p_config
                .start(signer_rx.clone())
                .await
                .map_err(|e| RollupNodeError::Startup(format!("Failed to start network: {e}")))?;
            let network_builder = network::Builder::new(gossip);
            let network = network_builder.handle();
            // A node without discovery runs no discovery actor, and bans no discovery nodes.
            let discovery_builder = discovery::Builder::new();
            let discovery = discovery_driver.is_some().then(|| discovery_builder.handle());
            let discovery_actor = discovery_driver
                .map(|driver| discovery_builder.build(driver, adapters::Network(network.clone())));
            let network_actor = network_builder.build(
                adapters::Engine::new(engine.clone()),
                adapters::Discovery(discovery.clone()),
            );
            p2p_rpc = Some(P2pRpc::new(
                network.state(),
                network.clone(),
                discovery,
                self.chain.rollup_config.l2_chain_id.id(),
            ));

            // Unsafe payloads to gossip flow from the sequencer to the signer actor and on to the
            // network actor. While signing or publishing stalls, the sequencer pauses block
            // production.
            let signer_actor;
            (sequencer_signer, signer_actor) = block_signer
                .map(|block_signer| {
                    let signer_builder = signer::Builder::new();
                    let handle = signer_builder.handle();
                    let actor = signer_builder.build(
                        block_signer,
                        self.chain.rollup_config.l2_chain_id.id(),
                        signer_rx,
                        adapters::Network(network),
                    );
                    (handle, actor)
                })
                .unzip();
            // Boxed, so that the actors do not sit in `start`'s frame while it awaits.
            p2p_actors = Some(P2pActors {
                network: Box::pin(network_actor),
                discovery: discovery_actor.map(Box::pin),
                signer: signer_actor.map(Box::pin),
                signer_updater: Box::pin(signer_updater),
            });
        }

        // ─── L1 watcher ─────────────────────────────────────────────────────────────────────
        let l1_actor = l1_builder.build(l1::Config {
            head_interval: Duration::from_secs(HEAD_STREAM_POLL_INTERVAL),
            finalized_interval: Duration::from_secs(FINALIZED_STREAM_POLL_INTERVAL),
        });

        // ─── sequencer ──────────────────────────────────────────────────────────────────────
        // The origin actor keeps the sequencer's next L1 origin ready.
        let sequencer_actors = sequencer.map(|(config, builder)| {
            let origin_builder =
                origin::Builder::new(self.chain.rollup_config.clone(), config.l1_conf_delay);
            let origin = origin_builder.handle();
            let origin_actor = origin_builder.build(l1.clone(), engine.state());
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
            let attributes_builder = StatefulAttributesBuilder::new(
                self.chain.rollup_config.clone(),
                self.chain.l1_config.clone(),
                l2_derivation_provider,
                l1_derivation_provider,
                Some(self.chain.dependency_set.clone()),
            );
            let sequencer_actor = builder.build(
                attributes_builder,
                adapters::Engine::new(engine.clone()),
                origin,
                self.chain.rollup_config.clone(),
                config.conductor_rpc_url.clone().map(|url| {
                    ConductorClient::new(kona_sources::conductor_rpc(
                        url,
                        config.conductor_rpc_timeout,
                    ))
                }),
                adapters::Signer(sequencer_signer),
            );
            (sequencer_actor, origin_actor)
        });

        // ─── RPC ────────────────────────────────────────────────────────────────────────────
        // A node run without an RPC address serves no RPC.
        let rpc = match self.rpc_addr {
            None => None,
            Some(rpc_addr) => {
                let mut modules = RpcModule::new(());
                modules
                    .merge(HealthzApiServer::into_rpc(HealthzRpc::new(self.version.clone())))
                    .map_err(|e| {
                        RollupNodeError::Startup(format!(
                            "Failed to register healthz module: {e:?}"
                        ))
                    })?;
                if let Some(p2p_rpc) = p2p_rpc {
                    modules.merge(p2p_rpc.into_rpc()).map_err(|e| {
                        RollupNodeError::Startup(format!("Failed to register p2p module: {e:?}"))
                    })?;
                }
                modules.merge(AdminRpc::new(engine.clone()).into_rpc()).map_err(|e| {
                    RollupNodeError::Startup(format!("Failed to register admin module: {e:?}"))
                })?;
                // A node that cannot sequence does not serve the sequencer's methods, so calls to
                // them fail with "method not found".
                if let Some(sequencer_admin_rpc) = sequencer_admin_rpc {
                    modules.merge(sequencer_admin_rpc.into_rpc()).map_err(|e| {
                        RollupNodeError::Startup(format!(
                            "Failed to register sequencer admin module: {e:?}"
                        ))
                    })?;
                }
                modules
                    .merge(
                        RollupRpc::new(
                            self.version.clone(),
                            self.chain.rollup_config.clone(),
                            engine.state(),
                            self.clients.l2.clone(),
                            l1,
                            safety,
                        )
                        .into_rpc(),
                    )
                    .map_err(|e| {
                        RollupNodeError::Startup(format!("Failed to register rollup module: {e:?}"))
                    })?;

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
                        ServerConfig::builder()
                            .max_response_body_size(max_response_body_size)
                            .build(),
                    )
                    .set_http_middleware(middleware)
                    .set_rpc_middleware(rpc_middleware)
                    .build(rpc_addr)
                    .await
                    .map_err(|e: std::io::Error| {
                        RollupNodeError::Startup(format!("Failed to launch rpc server: {e:?}"))
                    })?;
                if let Ok(addr) = server.local_addr() {
                    info!(target: "rpc", addr = ?addr, "RPC server bound to address");
                } else {
                    error!(target: "rpc", "Failed to get local address for RPC server");
                }
                Some(supervise_rpc_server(server.start(modules)))
            }
        };

        let mut supervisor = Supervisor::new();
        if let Some(rpc) = rpc {
            supervisor.spawn("rpc", rpc);
        }
        if let Some((sequencer, origin)) = sequencer_actors {
            supervisor.spawn("sequencer", sequencer);
            supervisor.spawn("origin", origin);
        }
        if let Some(P2pActors { network, discovery, signer, signer_updater }) = p2p_actors {
            supervisor.spawn("network", network);
            if let Some(discovery) = discovery {
                supervisor.spawn("discovery", discovery);
            }
            if let Some(signer) = signer {
                supervisor.spawn("signer", signer);
            }
            supervisor.spawn("l1_signer_updater", async move {
                signer_updater.await;
                Ok::<(), Infallible>(())
            });
        }
        // The L1 actor runs until it is dropped: its output, `Infallible`, never exists.
        supervisor.spawn("l1", async move { Err::<(), _>(l1_actor.await) });
        supervisor.spawn("derivation", derivation_actor);
        if let Some(finality) = finality_actor {
            supervisor.spawn("finality", finality);
        }
        supervisor.spawn("engine", engine_actor);
        supervisor.wait().await.map_err(RollupNodeError::Stopped)
    }
}

impl RollupNode {
    /// Returns the block signer the node signs its blocks with. Only a node that can sequence and
    /// has P2P signs blocks, to gossip them, and it needs a signer to.
    fn block_signer(&self) -> Result<Option<&BlockSigner>, RollupNodeError> {
        if self.sequencer.is_none() || self.p2p_config.is_none() {
            return Ok(None);
        }
        self.signer.as_ref().map(Some).ok_or(RollupNodeError::NoBlockSigner)
    }
}

/// The lifetime futures of the actors that run only with P2P.
struct P2pActors<Network, Discovery, Signer, SignerUpdater> {
    network: Network,
    /// The discovery actor, which runs only with discovery.
    discovery: Option<Discovery>,
    /// The block signer actor, which runs only on a node that can sequence.
    signer: Option<Signer>,
    /// Keeps the unsafe block signer that the network and the block signer check against current.
    signer_updater: SignerUpdater,
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
