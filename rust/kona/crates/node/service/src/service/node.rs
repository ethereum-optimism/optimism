//! Contains the [`RollupNode`] implementation.
use crate::{
    ConductorClient, DelayedL1OriginSelectorProvider, DelegateDerivationActor, DerivationActor,
    DerivationActorRequest, DerivationDelegateClient, DerivationError, EngineActor, EngineHandle,
    JsonrpseeServerLauncher, L1OriginSelector, L1WatcherActor, L1WatcherChain, NetworkActor,
    NetworkBuilder, NetworkConfig, NetworkHandler, NodeActor, NodeMode,
    QueuedL1WatcherDerivationClient, RpcActor, RpcServerLauncher, SequencerActor, SequencerConfig,
    SequencerHandle, SignedPayload, SignerActor,
    actors::{BlockStream, QueuedUnsafePayloadGossipClient},
    service::BufferImportedBlocks,
};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::Address;
use alloy_provider::{Provider, RootProvider};
use alloy_transport::{TransportError, TransportErrorKind};
use backon::{ExponentialBuilder, Retryable};
use jsonrpsee::RpcModule;
use kona_derive::{BlobProviderError, StatefulAttributesBuilder};
use kona_engine::{EngineClient, OpEngineClient};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_gossip::P2pRpcRequest;
use kona_interop::DependencySet;
use kona_protocol::BlockInfo;
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, BufferedAlloyL2ChainProvider, OnlineBeaconClient,
    OnlineBlobProvider, OnlinePipeline,
};
use kona_providers_local::BufferedL2Provider;
use kona_rpc::{
    AdminApiServer, AdminRpc, DevEngineApiServer, DevEngineRpc, HealthzApiServer, HealthzRpc,
    L1WatcherQueries, OpP2PApiServer, P2pRpc, RollupNodeApiServer, RollupRpc, RpcBuilder,
    SequencerAdminApiServer, SequencerAdminRpc, WsRPC, WsServer,
};
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
    /// The binary release version shared by CLI, metrics and RPC.
    pub(crate) version: &'static str,
    /// The rollup configuration.
    pub(crate) config: Arc<RollupConfig>,
    /// The L1 configuration.
    pub(crate) l1_config: L1Config,
    /// The L2 EL provider.
    pub(crate) l2_provider: RootProvider<Optimism>,
    /// Whether to trust the L2 RPC.
    pub(crate) l2_trust_rpc: bool,
    /// The engine wrapper over the node's shared L1/L2 providers and resolved chain rules.
    pub(crate) engine_client: OpEngineClient<RootProvider, RootProvider<Optimism>>,
    /// The mode chosen at composition.
    pub(crate) mode: NodeMode,
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
    Delegate(Box<DelegateDerivationActor<DerivationDelegateClient, AlloyChainProvider>>),
    Normal(Box<DerivationActor<OnlinePipeline>>),
}

#[async_trait::async_trait]
impl NodeActor for ConfiguredDerivationActor {
    type Error = DerivationError;
    async fn step(&mut self) -> Result<(), Self::Error> {
        match self {
            Self::Delegate(actor) => actor.step().await,
            Self::Normal(actor) => actor.step().await,
        }
    }
}

type ConfiguredEngineActor = EngineActor<OpEngineClient<RootProvider, RootProvider<Optimism>>>;

/// Concrete type of the sequencer actor used by `RollupNode`.
type ConfiguredSequencerActor = SequencerActor<
    StatefulAttributesBuilder<AlloyChainProvider, BufferedAlloyL2ChainProvider>,
    ConductorClient,
    L1OriginSelector<DelayedL1OriginSelectorProvider>,
    QueuedUnsafePayloadGossipClient,
>;

/// Concrete type of the rpc actor used by `RollupNode`.
type ConfiguredRpcActor = RpcActor<JsonrpseeServerLauncher>;

impl RollupNode {
    /// The mode of operation for the node.
    const fn mode(&self) -> NodeMode {
        self.mode
    }

    /// Creates a network builder for the node whose gossip validation follows
    /// `unsafe_block_signer`.
    fn network_builder(&self, unsafe_block_signer: watch::Receiver<Address>) -> NetworkBuilder {
        NetworkBuilder::from_config(self.p2p_config.clone(), &self.config, unsafe_block_signer)
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

    /// Constructs the single owner of execution state and its command handle.
    fn build_engine_actor(&self) -> (ConfiguredEngineActor, EngineHandle) {
        EngineActor::new(
            Arc::new(self.engine_client.clone()),
            self.config.clone(),
            Arc::new(BufferImportedBlocks::new(self.l2_block_buffer.clone())),
        )
    }

    /// Selects between the standard and delegate derivation actor implementations and constructs
    /// the chosen one.
    async fn build_derivation_actor(
        &self,
        engine: EngineHandle,
        derivation_actor_request_rx: mpsc::Receiver<DerivationActorRequest>,
    ) -> Result<ConfiguredDerivationActor, String> {
        if let Some(provider) = self.derivation_delegate_provider.clone() {
            // L1 Provider for sanity checking Derivation Delegation
            let l1_provider = AlloyChainProvider::new(
                self.l1_config.engine_provider.clone(),
                DERIVATION_PROVIDER_CACHE_SIZE,
            );
            Ok(ConfiguredDerivationActor::Delegate(Box::new(DelegateDerivationActor::new(
                engine,
                derivation_actor_request_rx,
                provider,
                l1_provider,
            ))))
        } else {
            let pipeline = self
                .create_pipeline()
                .await
                .map_err(|error| format!("Failed to initialize L1 blob provider: {error}"))?;
            Ok(ConfiguredDerivationActor::Normal(Box::new(DerivationActor::<OnlinePipeline>::new(
                engine,
                derivation_actor_request_rx,
                pipeline,
            ))))
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
        signer_tx: watch::Sender<Address>,
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

        let chain = L1WatcherChain::new(
            self.config.clone(),
            QueuedL1WatcherDerivationClient { derivation_actor_request_tx },
            signer_tx,
            l1_query_rx,
        );

        Ok(L1WatcherActor::new(
            self.l1_config.engine_provider.clone(),
            l1_head_updates_tx,
            head_stream,
            finalized_stream,
            vec![chain],
        ))
    }

    /// Builds the signer actor when the node is in sequencer mode; otherwise returns `None`.
    ///
    /// A sequencer must have a block signer: its blocks are gossiped only once signed.
    async fn build_signer_actor(
        &self,
        payloads_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        signed_payload_tx: mpsc::Sender<SignedPayload>,
        unsafe_block_signer: watch::Receiver<Address>,
    ) -> Result<Option<SignerActor>, String> {
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
        Ok(Some(SignerActor::new(
            signer,
            self.config.l2_chain_id.id(),
            unsafe_block_signer,
            payloads_rx,
            signed_payload_tx,
        )))
    }

    /// Builds the sequencer actor when the node is in sequencer mode; otherwise returns `None`.
    fn build_sequencer(
        &self,
        engine: EngineHandle,
        gossip_payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
        l1_head_updates_rx: watch::Receiver<Option<BlockInfo>>,
    ) -> Option<(ConfiguredSequencerActor, SequencerHandle)> {
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

        let queued_gossip_client = QueuedUnsafePayloadGossipClient::new(gossip_payload_tx);

        Some(SequencerActor::new(
            self.create_attributes_builder(),
            conductor,
            engine,
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
        engine: &EngineHandle,
        sequencer_admin_client: Option<SequencerHandle>,
        p2p_rpc_tx: mpsc::Sender<P2pRpcRequest>,
        l1_watcher_queries_tx: mpsc::Sender<L1WatcherQueries>,
    ) -> Result<Option<ConfiguredRpcActor>, String> {
        let Some(config) = self.rpc_builder() else {
            return Ok(None);
        };

        let mut modules = RpcModule::new(());
        modules
            .merge(HealthzApiServer::into_rpc(HealthzRpc::new(self.version)))
            .map_err(|e| format!("Failed to register healthz module: {e:?}"))?;
        modules
            .merge(P2pRpc::new(p2p_rpc_tx).into_rpc())
            .map_err(|e| format!("Failed to register p2p module: {e:?}"))?;
        merge_admin_module(&mut modules, config.enable_admin(), sequencer_admin_client, engine)?;
        modules
            .merge(
                RollupRpc::new(
                    self.l2_provider.clone(),
                    self.config.clone(),
                    engine.subscribe(),
                    self.version,
                    l1_watcher_queries_tx,
                )
                .into_rpc(),
            )
            .map_err(|e| format!("Failed to register rollup module: {e:?}"))?;
        if config.dev_enabled() {
            modules
                .merge(DevEngineRpc::new(engine.queue_length_subscribe()).into_rpc())
                .map_err(|e| format!("Failed to register dev engine module: {e:?}"))?;
        }
        if config.ws_enabled() {
            modules
                .merge(WsRPC::new(engine.subscribe()).into_rpc())
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
        validate_engine_jwt(&self.engine_client, ExponentialBuilder::default()).await?;
        let initial_signer = unsafe_block_signer_at_l1_head(
            &self.l1_config.engine_provider,
            self.config.l1_system_config_address,
        )
        .await
        .map_err(|error| format!("Failed to read initial unsafe block signer: {error}"))?;

        // Single umbrella cancellation token owned by the spawn_and_wait! macro.
        let cancellation = CancellationToken::new();

        // ─── cross-actor channels ───────────────────────────────────────────────────────────
        // actor request channels
        let (derivation_actor_request_tx, derivation_actor_request_rx) =
            mpsc::channel::<DerivationActorRequest>(1024);
        let (l1_query_tx, l1_query_rx) = mpsc::channel::<L1WatcherQueries>(1024);
        // Network actor inbound channels
        let (p2p_rpc_tx, p2p_rpc_rx) = mpsc::channel::<P2pRpcRequest>(1024);
        // Unsafe payloads to gossip flow from the sequencer to the signer actor and on to the
        // network actor. While signing stalls, a full sequencer queue pauses block production.
        let (gossip_payload_tx, gossip_payload_rx) =
            mpsc::channel::<OpExecutionPayloadEnvelope>(32);
        let (signed_payload_tx, signed_payload_rx) = mpsc::channel::<SignedPayload>(16);
        // watch channels
        // The unsafe block signer: the L1 watcher keeps it current from `SystemConfig`, starting
        // from the value read at startup.
        let (signer_tx, signer_rx) = watch::channel(initial_signer);
        let (l1_head_updates_tx, l1_head_updates_rx) = watch::channel::<Option<BlockInfo>>(None);

        // ─── actor construction ─────────────────────────────────────────────────────────────
        let (engine_actor, engine) = self.build_engine_actor();

        let derivation =
            self.build_derivation_actor(engine.clone(), derivation_actor_request_rx).await?;

        // Start the block signer before the network, so a misconfigured or unreachable remote
        // signer fails before the node binds its p2p ports.
        let signer_actor = self
            .build_signer_actor(gossip_payload_rx, signed_payload_tx, signer_rx.clone())
            .await?;

        // Build and start the libp2p swarm upstream of `NetworkActor::new` so the constructor
        // stays sync.
        let handler: NetworkHandler = self
            .network_builder(signer_rx)
            .build()
            .map_err(|e| format!("Failed to build network: {e:?}"))?
            .start()
            .await
            .map_err(|e| format!("Failed to start network: {e:?}"))?;

        let network = NetworkActor::new(engine.clone(), handler, p2p_rpc_rx, signed_payload_rx);

        let l1_watcher = self.build_l1_watcher(
            derivation_actor_request_tx,
            signer_tx,
            l1_query_rx,
            l1_head_updates_tx,
        )?;

        let (sequencer_actor, sequencer_admin_client) =
            self.build_sequencer(engine.clone(), gossip_payload_tx, l1_head_updates_rx).unzip();

        let rpc =
            self.build_rpc_actor(&engine, sequencer_admin_client, p2p_rpc_tx, l1_query_tx).await?;

        crate::service::spawn_and_wait!(
            cancellation,
            actors = [
                rpc,
                sequencer_actor,
                signer_actor,
                Some(network),
                Some(l1_watcher),
                Some(derivation),
                Some(engine_actor),
            ]
        );
        Ok(())
    }
}

/// Reads the initial signer at the returned L1 head's hash, before gossip starts.
pub async fn unsafe_block_signer_at_l1_head(
    provider: &RootProvider,
    system_config: Address,
) -> Result<Address, TransportError> {
    let head = provider
        .get_block_by_number(BlockNumberOrTag::Latest)
        .await?
        .ok_or_else(|| TransportErrorKind::custom_str("the L1 RPC returned no head block"))?;
    kona_providers_alloy::unsafe_block_signer(provider, system_config, head.header.hash).await
}

#[derive(Debug, thiserror::Error)]
enum EngineStartupError {
    #[error(
        "Engine API rejected the JWT secret; check --l2-engine-jwt-secret/--l2-engine-jwt-encoded"
    )]
    JwtRejected,
    #[error("Failed to exchange capabilities with engine: {0}")]
    Capabilities(TransportError),
}

/// Checks the injected Engine API connection. Authentication failures are permanent; other
/// failures retain the existing bounded startup retry policy.
async fn validate_engine_jwt(
    engine: &impl EngineClient,
    backoff: ExponentialBuilder,
) -> Result<(), String> {
    let exchange = || async {
        let error = match engine.exchange_capabilities(vec![]).await {
            Ok(_) => return Ok(()),
            Err(error) => error,
        };
        let mut source: Option<&dyn std::error::Error> = Some(&error);
        while let Some(cause) = source {
            let message = cause.to_string().to_lowercase();
            if message.contains("signature invalid") ||
                (message.contains("jwt") && message.contains("invalid")) ||
                message.contains("unauthorized") ||
                message.contains("authentication failed")
            {
                return Err(EngineStartupError::JwtRejected);
            }
            source = cause.source();
        }
        Err(EngineStartupError::Capabilities(error))
    };
    exchange
        .retry(backoff)
        .when(|error| !matches!(error, EngineStartupError::JwtRejected))
        .notify(|_, duration| debug!("Retrying engine capability handshake after {duration:?}"))
        .await
        .map_err(|error| error.to_string())
}

/// Registers the admin API namespace on `modules`, but only when `enable_admin` is set.
///
/// The admin API is opt-in via `--rpc.enable-admin`, matching op-node's admin namespace.
fn merge_admin_module(
    modules: &mut RpcModule<()>,
    enable_admin: bool,
    sequencer: Option<SequencerHandle>,
    engine: &EngineHandle,
) -> Result<(), String> {
    if !enable_admin {
        return Ok(());
    }
    modules
        .merge(AdminRpc::new(engine.clone()).into_rpc())
        .map_err(|error| format!("Failed to register admin module: {error}"))?;
    if let Some(sequencer) = sequencer {
        modules
            .merge(SequencerAdminRpc::new(sequencer).into_rpc())
            .map_err(|error| format!("Failed to register sequencer admin module: {error}"))?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn admin_registration_matches_the_enabled_node_capabilities() {
        let base = ["admin_postUnsafePayload", "admin_resetDerivationPipeline"];
        let controls = [
            "admin_sequencerActive",
            "admin_startSequencer",
            "admin_stopSequencer",
            "admin_conductorEnabled",
            "admin_adminRecoverMode",
            "admin_setRecoverMode",
            "admin_overrideLeader",
        ];
        for enabled in [false, true] {
            for is_sequencer in [false, true] {
                let (engine, _engine_peer) =
                    EngineHandle::test_channel(kona_engine::EngineState::default());
                let (sequencer, _sequencer_peer) = SequencerHandle::test_channel(
                    crate::SequencerState { active: false, recovery_mode: false },
                    false,
                );
                let mut modules = RpcModule::new(());
                merge_admin_module(
                    &mut modules,
                    enabled,
                    is_sequencer.then_some(sequencer),
                    &engine,
                )
                .unwrap();
                let names: Vec<_> = modules.method_names().collect();
                for method in base {
                    assert_eq!(names.contains(&method), enabled, "{method}");
                }
                for method in controls {
                    assert_eq!(names.contains(&method), enabled && is_sequencer, "{method}");
                }
                if !enabled || !is_sequencer {
                    let error = modules
                        .call::<_, bool>("admin_sequencerActive", jsonrpsee::rpc_params![])
                        .await
                        .unwrap_err();
                    assert!(
                        matches!(error, jsonrpsee::core::server::MethodsError::JsonRpc(error) if error.code() == -32601)
                    );
                }
            }
        }
    }
    #[tokio::test]
    async fn registered_reset_and_reads_bypass_a_full_sequencer_mailbox() {
        use kona_rpc::SequencerAdminAPIClient;
        let (engine, mut engine_peer) =
            EngineHandle::test_channel(kona_engine::EngineState::default());
        let unpolled_pipeline = engine.subscribe();
        let (sequencer, sequencer_peer) = SequencerHandle::test_channel(
            crate::SequencerState { active: false, recovery_mode: true },
            true,
        );
        let mut waiting = Vec::new();
        for _ in 0..32 {
            let handle = sequencer.clone();
            waiting.push(tokio::spawn(async move { handle.start_sequencer().await }));
        }
        tokio::time::timeout(Duration::from_secs(2), async {
            while sequencer_peer.commands.capacity() != 0 {
                tokio::task::yield_now().await;
            }
        })
        .await
        .unwrap();
        let mut modules = RpcModule::new(());
        merge_admin_module(&mut modules, true, Some(sequencer), &engine).unwrap();
        tokio::time::timeout(Duration::from_secs(2), async {
            assert!(
                !modules
                    .call::<_, bool>("admin_sequencerActive", jsonrpsee::rpc_params![])
                    .await
                    .unwrap()
            );
            assert!(
                modules
                    .call::<_, bool>("admin_adminRecoverMode", jsonrpsee::rpc_params![])
                    .await
                    .unwrap()
            );
            assert!(
                modules
                    .call::<_, bool>("admin_conductorEnabled", jsonrpsee::rpc_params![])
                    .await
                    .unwrap()
            );
        })
        .await
        .unwrap();
        let rpc = modules.clone();
        let reset = tokio::spawn(async move {
            rpc.call::<_, ()>("admin_resetDerivationPipeline", jsonrpsee::rpc_params![]).await
        });
        let reply = tokio::time::timeout(Duration::from_secs(2), engine_peer.resets.recv())
            .await
            .unwrap()
            .unwrap();
        assert_eq!(sequencer_peer.commands.len(), 32);
        engine_peer.state.send_replace(kona_engine::EngineState {
            epoch: 1,
            phase: kona_engine::EnginePhase::Ready,
            ..Default::default()
        });
        reply.send(Ok(())).unwrap();
        tokio::time::timeout(Duration::from_secs(2), reset).await.unwrap().unwrap().unwrap();
        assert_eq!(unpolled_pipeline.borrow().epoch, 1);
        drop(engine_peer);
        let error = modules
            .call::<_, ()>("admin_resetDerivationPipeline", jsonrpsee::rpc_params![])
            .await
            .unwrap_err();
        assert!(
            matches!(error, jsonrpsee::core::server::MethodsError::JsonRpc(error) if error.code() == -32603)
        );
        for task in waiting {
            task.abort();
        }
    }

    fn node_with_providers(
        l1: &alloy_transport::mock::Asserter,
        l2: &alloy_transport::mock::Asserter,
    ) -> (RollupNode, Arc<RollupConfig>) {
        use crate::{EngineConfig, L1ConfigBuilder, RollupNodeBuilder};
        use alloy_rpc_client::RpcClient;
        use alloy_rpc_types_engine::JwtSecret;
        let mut config =
            RollupConfig { l2_chain_id: alloy_chains::Chain::from_id(8453), ..Default::default() };
        config.hardforks.karst_time = Some(123);
        let config = Arc::new(config);
        let enr::CombinedKey::Secp256k1(key) = enr::CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let mut network = NetworkConfig::new(
            kona_disc::LocalNode::new(key, std::net::Ipv4Addr::LOCALHOST.into(), 0, 0),
            "/ip4/127.0.0.1/tcp/0".parse().unwrap(),
        );
        network.gossip_config = kona_gossip::default_config();
        // These URLs are deliberately unusable: supplying providers must replace every consumer.
        let node = RollupNodeBuilder::new(
            "1.2.3",
            config.clone(),
            L1ConfigBuilder {
                chain_config: L1ChainConfig::default(),
                trust_rpc: false,
                beacon: "http://127.0.0.1:1".parse().unwrap(),
                rpc_url: "http://127.0.0.1:1".parse().unwrap(),
                slot_duration_override: Some(12),
            },
            false,
            EngineConfig {
                l2_url: "http://127.0.0.1:1".parse().unwrap(),
                l2_jwt_secret: JwtSecret::random(),
                timeout: Duration::from_millis(321),
                mode: NodeMode::Validator,
            },
            network,
            None,
        )
        .build_with_providers(
            RootProvider::new(RpcClient::mocked(l1.clone())),
            RootProvider::<Optimism>::new(RpcClient::mocked(l2.clone())),
        );
        (node, config)
    }

    #[tokio::test]
    async fn production_wiring_shares_rules_and_injected_clients() {
        use alloy_primitives::{B256, U256};
        use alloy_transport::mock::Asserter;
        let l1 = Asserter::new();
        let l2 = Asserter::new();
        let (node, config) = node_with_providers(&l1, &l2);
        assert_eq!(node.version, "1.2.3");
        assert!(Arc::ptr_eq(&node.config, &config));
        assert!(std::ptr::eq(node.engine_client.cfg(), config.as_ref()));
        assert_eq!(node.engine_client.cfg().hardforks.karst_time, Some(123));
        assert!(!node.l1_config.trust_rpc);
        assert!(!node.l2_trust_rpc);

        l2.push_success(&Vec::<String>::new());
        validate_engine_jwt(&node.engine_client, ExponentialBuilder::default()).await.unwrap();
        l2.push_success(&serde_json::Value::Null);
        assert!(
            node.engine_client.get_l2_block(alloy_eips::BlockId::latest()).await.unwrap().is_none()
        );
        l2.push_success(&"0x11");
        assert_eq!(node.l2_provider.get_block_number().await.unwrap(), 17);
        l1.push_success(&serde_json::Value::Null);
        assert!(
            node.engine_client.get_l1_block(alloy_eips::BlockId::latest()).await.unwrap().is_none()
        );
        l1.push_success(&"0x13");
        assert_eq!(node.l1_config.engine_provider.get_block_number().await.unwrap(), 19);

        let mut block = alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default();
        block.header.hash = B256::repeat_byte(42);
        l1.push_success(&block);
        let signer = Address::repeat_byte(23);
        l1.push_success(&U256::from_be_bytes(signer.into_word().0));
        assert_eq!(
            unsafe_block_signer_at_l1_head(
                &node.l1_config.engine_provider,
                Address::repeat_byte(8)
            )
            .await
            .unwrap(),
            signer
        );
        assert!(l1.read_q().is_empty());
        assert!(l2.read_q().is_empty());
        let network = node.network_builder(watch::channel(signer).1).build().unwrap();
        assert_eq!(network.discovery.chain_id, 8453);
        assert_eq!(network.gossip.handler.rollup_config.hardforks.karst_time, Some(123));
    }

    #[tokio::test(start_paused = true)]
    async fn capability_rejection_is_permanent_but_dependency_failure_retries() {
        use alloy_transport::mock::Asserter;
        let l1 = Asserter::new();
        let l2 = Asserter::new();
        let (node, _) = node_with_providers(&l1, &l2);
        l2.push_failure_msg("signature invalid");
        l2.push_success(&Vec::<String>::new());
        let backoff = ExponentialBuilder::default()
            .with_min_delay(Duration::from_millis(1))
            .with_max_times(2);
        let error = validate_engine_jwt(&node.engine_client, backoff).await.unwrap_err();
        assert!(error.contains("JWT"));
        assert_eq!(l2.read_q().len(), 1, "authentication failure must not retry");
        let _ = l2.pop_response();
        l2.push_failure_msg("temporarily unavailable");
        l2.push_success(&Vec::<String>::new());
        validate_engine_jwt(&node.engine_client, backoff).await.unwrap();
        assert!(l2.read_q().is_empty());
    }
}
