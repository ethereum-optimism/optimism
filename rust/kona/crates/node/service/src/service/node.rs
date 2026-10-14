//! Contains the [`RollupNode`] implementation.
use super::{Supervisor, adapters, error::RollupNodeError, middleware::RpcMetricsLayer, startup};
use crate::{
    ChainSpec, ConductorClient, DelayedL1OriginSelectorProvider, L1OriginSelector, NetworkConfig,
    NodeActor, RollupNodeClients, RollupNodeConfig, RpcActor, SequencerConfig,
    service::BufferImportedBlocks, signer,
};
use backon::ExponentialBuilder;
use jsonrpsee::{
    RpcModule,
    server::{
        Server, ServerConfig,
        middleware::{http::ProxyGetRequestLayer, rpc::RpcServiceBuilder},
    },
};
use kona_derive::StatefulAttributesBuilder;
use kona_engine::EngineClient;
use kona_node_actors::{
    derivation, discovery, engine, finality, l1, l1_signer_updater, network, sequencer,
};
use kona_protocol::BlockInfo;
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, BufferedAlloyL2ChainProvider, OnlineBlobProvider,
    OnlinePipeline,
};
use kona_providers_local::BufferedL2Provider;
use kona_rpc::{
    AdminApiServer, AdminRpc, HealthzApiServer, HealthzRpc, OpP2PApiServer, P2pRpc,
    RollupNodeApiServer, RollupRpc, SequencerAdminApiServer, SequencerAdminRpc,
};
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
    pub(crate) engine_client: Arc<EngineClient>,
    /// The address the RPC server listens on.
    pub(crate) rpc_addr: SocketAddr,
    /// The P2P [`NetworkConfig`] for the node.
    pub(crate) p2p_config: NetworkConfig,
    /// The [`SequencerConfig`] of a node that can sequence.
    pub(crate) sequencer: Option<SequencerConfig>,
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
        let engine_client = Arc::new(EngineClient::new(
            clients.l1.clone(),
            clients.engine.clone(),
            config.chain.rollup_config.clone(),
        ));
        Self {
            version: config.version,
            chain: config.chain,
            clients,
            engine_client,
            rpc_addr: config.rpc_addr,
            p2p_config: config.p2p_config,
            sequencer: config.sequencer,
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
            self.chain.rollup_config.l1_system_config_address,
        )
        .await
        .map_err(RollupNodeError::UnsafeBlockSigner)?;
        let signer_updater_builder = l1_signer_updater::Builder::new(initial_signer);
        let signer_rx = signer_updater_builder.handle();
        // actor request channels
        let derivation_builder = derivation::Builder::new();
        let derivation = derivation_builder.handle();
        let engine_builder = engine::Builder::new();
        let engine = engine_builder.handle();
        let sequencer = self.sequencer.as_ref().map(|config| (config, sequencer::Builder::new()));
        let sequencer_admin_rpc = sequencer.as_ref().map(|(config, builder)| {
            SequencerAdminRpc::new(builder.handle(), config.conductor_rpc_url.is_some())
        });
        let discovery_builder = discovery::Builder::new();
        let discovery = discovery_builder.handle();
        // Unsafe payloads to gossip flow from the sequencer to the signer actor and on to the
        // network actor. While signing or publishing stalls, the sequencer pauses block production.
        let signer_builder = signer::Builder::new();
        let signer_handle = signer_builder.handle();

        // ─── engine ─────────────────────────────────────────────────────────────────────────
        let l2_query_client = self.engine_client.as_ref().clone();
        let engine_actor = engine_builder.build(
            self.engine_client.clone(),
            self.chain.rollup_config.clone(),
            Arc::new(BufferImportedBlocks::new(self.l2_block_buffer.clone())),
        );

        // ─── derivation and finality ────────────────────────────────────────────────────────
        // A node that delegates derivation follows the delegate's safe and finalized heads, so it
        // neither derives from L1 nor finalizes.
        let (derivation_actor, finality_actor): (DerivationTask, Option<FinalityTask>) =
            if let Some(provider) = self.clients.derivation_delegate.clone() {
                // L1 Provider for sanity checking Derivation Delegation
                let l1_provider = AlloyChainProvider::new(
                    self.clients.l1.clone(),
                    DERIVATION_PROVIDER_CACHE_SIZE,
                );
                let derivation = derivation_builder.build_delegated(
                    adapters::Engine::new(engine.clone()),
                    provider,
                    l1_provider,
                );
                (Box::pin(derivation), None)
            } else {
                // Create the caching L1/L2 EL providers for derivation.
                let l1_derivation_provider = AlloyChainProvider::new(
                    self.clients.l1.clone(),
                    DERIVATION_PROVIDER_CACHE_SIZE,
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
                let derivation = derivation_builder.build(
                    adapters::Engine::new(engine.clone()),
                    pipeline,
                    head.clone(),
                );
                let (engine_state, resets) = (engine.state(), engine.resets());
                let finality = finality::build(
                    adapters::Engine::new(engine.clone()),
                    engine_state,
                    resets,
                    finalized.clone(),
                );
                (Box::pin(derivation), Some(Box::pin(finality)))
            };

        // ─── block signer ───────────────────────────────────────────────────────────────────
        // Start the sequencer's block signer before the network, so a misconfigured or
        // unreachable remote signer fails before the node binds its p2p ports.
        let block_signer = match &self.sequencer {
            Some(config) => Some(config.signer.clone().start().await.map_err(|e| {
                RollupNodeError::Startup(format!("Failed to start block signer: {e}"))
            })?),
            None => None,
        };

        // ─── network ────────────────────────────────────────────────────────────────────────
        let (gossip, discovery_driver) = self
            .p2p_config
            .clone()
            .start(signer_rx.clone())
            .await
            .map_err(|e| RollupNodeError::Startup(format!("Failed to start network: {e}")))?;
        let network_builder = network::Builder::new(gossip);
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

        // ─── L1 watchers ────────────────────────────────────────────────────────────────────
        let signer_updater = signer_updater_builder.build(
            self.clients.l1.clone(),
            self.chain.rollup_config.l1_system_config_address,
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

        // ─── sequencer ──────────────────────────────────────────────────────────────────────
        let sequencer_actor = sequencer.map(|(config, builder)| {
            let delayed_l1_provider = DelayedL1OriginSelectorProvider::new(
                self.clients.l1.clone(),
                head.clone(),
                config.l1_conf_delay,
            );
            let origin_selector =
                L1OriginSelector::new(self.chain.rollup_config.clone(), delayed_l1_provider);
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
            builder.build(
                attributes_builder,
                adapters::Engine::new(engine.clone()),
                origin_selector,
                self.chain.rollup_config.clone(),
                config.conductor_rpc_url.clone().map(ConductorClient::new_http),
                adapters::Signer(signer_handle),
            )
        });

        // ─── RPC ────────────────────────────────────────────────────────────────────────────
        let mut modules = RpcModule::new(());
        modules.merge(HealthzApiServer::into_rpc(HealthzRpc::new(self.version.clone()))).map_err(
            |e| RollupNodeError::Startup(format!("Failed to register healthz module: {e:?}")),
        )?;
        modules.merge(p2p_rpc.into_rpc()).map_err(|e| {
            RollupNodeError::Startup(format!("Failed to register p2p module: {e:?}"))
        })?;
        modules.merge(AdminRpc::new(engine.clone()).into_rpc()).map_err(|e| {
            RollupNodeError::Startup(format!("Failed to register admin module: {e:?}"))
        })?;
        // A node that cannot sequence does not serve the sequencer's methods, so calls to them
        // fail with "method not found".
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
                    l2_query_client,
                    head,
                    safe,
                    finalized,
                    derivation.status(),
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
                ServerConfig::builder().max_response_body_size(max_response_body_size).build(),
            )
            .set_http_middleware(middleware)
            .set_rpc_middleware(rpc_middleware)
            .build(self.rpc_addr)
            .await
            .map_err(|e: std::io::Error| {
                RollupNodeError::Startup(format!("Failed to launch rpc server: {e:?}"))
            })?;
        if let Ok(addr) = server.local_addr() {
            info!(target: "rpc", addr = ?addr, "RPC server bound to address");
        } else {
            error!(target: "rpc", "Failed to get local address for RPC server");
        }
        let rpc = RpcActor::new(server.start(modules));

        let mut supervisor = Supervisor::new();
        supervisor.spawn("rpc", run_node_actor(rpc));
        if let Some(sequencer) = sequencer_actor {
            supervisor.spawn("sequencer", sequencer);
        }
        if let Some(signer) = signer_actor {
            supervisor.spawn("signer", signer);
        }
        supervisor.spawn("network", network_actor);
        supervisor.spawn("discovery", discovery_actor);
        supervisor.spawn("l1_head", l1_head);
        supervisor.spawn("l1_safe", l1_safe);
        supervisor.spawn("l1_finalized", l1_finalized);
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
