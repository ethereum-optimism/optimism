//! Optimism Node types config.

use crate::{
    OpEngineApiBuilder, OpEngineTypes,
    args::RollupArgs,
    engine::OpEngineValidator,
    txpool::{OpTransactionPool, OpTransactionValidator},
};
use op_alloy_consensus::{OpPooledTransaction, interop::SafetyLevel};
use op_alloy_rpc_types_engine::OpExecutionData;
use reth_chainspec::{ChainSpecProvider, EthChainSpec, Hardforks};
use reth_engine_local::LocalPayloadAttributesBuilder;
use reth_evm::ConfigureEvm;
use reth_network::{
    NetworkConfig, NetworkHandle, NetworkManager, NetworkPrimitives, PeersInfo,
    types::BasicNetworkPrimitives,
};
use reth_node_api::{
    AddOnsContext, BuildNextEnv, EngineTypes, FullNodeComponents, HeaderTy, NodeAddOns,
    NodePrimitives, PayloadAttributesBuilder, PayloadTypes, PrimitivesTy, TxTy,
};
use reth_node_builder::{
    BuilderContext, DebugNode, Node, NodeAdapter, NodeComponentsBuilder,
    components::{
        BasicPayloadServiceBuilder, ComponentsBuilder, ConsensusBuilder, ExecutorBuilder,
        NetworkBuilder, PayloadBuilderBuilder, PoolBuilder, PoolBuilderConfigOverrides,
        TxPoolBuilder,
    },
    node::{FullNodeTypes, NodeTypes},
    rpc::{
        BasicEngineValidatorBuilder, EngineApiBuilder, EngineValidatorAddOn,
        EngineValidatorBuilder, EthApiBuilder, Identity, PayloadValidatorBuilder, RethRpcAddOns,
        RethRpcMiddleware, RethRpcServerHandles, RpcAddOns, RpcContext, RpcHandle,
    },
};
use reth_optimism_chainspec::{OpChainSpec, OpHardfork};
use reth_optimism_consensus::OpBeaconConsensus;
use reth_optimism_evm::{OpEvmConfig, OpRethReceiptBuilder};
use reth_optimism_forks::OpHardforks;
use reth_optimism_payload_builder::{
    OpAttributes, OpBuiltPayload, OpPayloadPrimitives,
    builder::OpPayloadTransactions,
    config::{OpBuilderConfig, OpDAConfig, OpGasLimitConfig},
};
use reth_optimism_primitives::{DepositReceipt, OpPrimitives};
use reth_optimism_rpc::{
    SequencerClient,
    eth::{OpEthApiBuilder, ext::OpEthExtApi},
    historical::{HistoricalRpc, HistoricalRpcClient},
    miner::{MinerApiExtServer, OpMinerExtApi},
    witness::{DebugExecutionWitnessApiServer, OpDebugWitnessApi},
};
use reth_optimism_storage::OpStorage;
use reth_optimism_txpool::{
    OpPooledTx,
    supervisor::{DEFAULT_SUPERVISOR_URL, SupervisorClient},
};
use reth_provider::{CanonStateSubscriptions, providers::ProviderFactoryBuilder};
use reth_rpc_api::{DebugApiServer, L2EthApiExtServer, eth::RpcTypes};
use reth_rpc_server_types::RethRpcModule;
use reth_tracing::tracing::{debug, info};
use reth_transaction_pool::{
    EthPoolTransaction, PoolPooledTx, PoolTransaction, TransactionPool,
    TransactionValidationTaskExecutor, blobstore::DiskFileBlobStore,
};
use reth_trie_common::KeccakKeyHasher;
use serde::de::DeserializeOwned;
use std::{marker::PhantomData, sync::Arc};
use url::Url;

/// Marker trait for Optimism node types with standard engine, chain spec, and primitives.
pub trait OpNodeTypes:
    NodeTypes<Payload = OpEngineTypes, ChainSpec: OpHardforks + Hardforks, Primitives = OpPrimitives>
{
}
/// Blanket impl for all node types that conform to the Optimism spec.
impl<N> OpNodeTypes for N where
    N: NodeTypes<
            Payload = OpEngineTypes,
            ChainSpec: OpHardforks + Hardforks,
            Primitives = OpPrimitives,
        >
{
}

/// Helper trait for Optimism node types with full configuration including storage and execution
/// data.
pub trait OpFullNodeTypes:
    NodeTypes<
        ChainSpec: OpHardforks,
        Primitives: OpPayloadPrimitives,
        Storage = OpStorage,
        Payload: EngineTypes<ExecutionData = OpExecutionData>,
    >
{
}

impl<N> OpFullNodeTypes for N where
    N: NodeTypes<
            ChainSpec: OpHardforks,
            Primitives: OpPayloadPrimitives,
            Storage = OpStorage,
            Payload: EngineTypes<ExecutionData = OpExecutionData>,
        >
{
}

/// Type configuration for a regular Optimism node.
#[derive(Debug, Default, Clone)]
#[non_exhaustive]
pub struct OpNode {
    /// Additional Optimism args
    pub args: RollupArgs,
    /// Data availability configuration for the OP builder.
    ///
    /// Used to throttle the size of the data availability payloads (configured by the batcher via
    /// the `miner_` api).
    ///
    /// By default no throttling is applied.
    pub da_config: OpDAConfig,
    /// Gas limit configuration for the OP builder.
    /// Used to control the gas limit of the blocks produced by the OP builder.(configured by the
    /// batcher via the `miner_` api)
    pub gas_limit_config: OpGasLimitConfig,
}

/// A [`ComponentsBuilder`] with its generic arguments set to a stack of Optimism specific builders.
pub type OpNodeComponentBuilder<Node, Payload = OpPayloadBuilder> = ComponentsBuilder<
    Node,
    OpPoolBuilder,
    BasicPayloadServiceBuilder<Payload>,
    OpNetworkBuilder,
    OpExecutorBuilder,
    OpConsensusBuilder,
>;

impl OpNode {
    /// Creates a new instance of the Optimism node type.
    pub fn new(args: RollupArgs) -> Self {
        Self {
            args,
            da_config: OpDAConfig::default(),
            gas_limit_config: OpGasLimitConfig::default(),
        }
    }

    /// Configure the data availability configuration for the OP builder.
    pub fn with_da_config(mut self, da_config: OpDAConfig) -> Self {
        self.da_config = da_config;
        self
    }

    /// Configure the gas limit configuration for the OP builder.
    pub fn with_gas_limit_config(mut self, gas_limit_config: OpGasLimitConfig) -> Self {
        self.gas_limit_config = gas_limit_config;
        self
    }

    /// Returns the components for the given [`RollupArgs`].
    pub fn components<Node>(&self) -> OpNodeComponentBuilder<Node>
    where
        Node: FullNodeTypes<Types: OpNodeTypes>,
    {
        let RollupArgs { disable_txpool_gossip, compute_pending_block, discovery_v4, .. } =
            self.args;
        ComponentsBuilder::default()
            .node_types::<Node>()
            .executor(OpExecutorBuilder::default())
            .pool(
                OpPoolBuilder::default()
                    .with_enable_tx_conditional(self.args.enable_tx_conditional)
                    .with_supervisor(
                        self.args.supervisor_http.clone(),
                        self.args.supervisor_safety_level,
                    ),
            )
            .payload(BasicPayloadServiceBuilder::new(
                OpPayloadBuilder::new(compute_pending_block)
                    .with_da_config(self.da_config.clone())
                    .with_gas_limit_config(self.gas_limit_config.clone()),
            ))
            .network(OpNetworkBuilder::new(disable_txpool_gossip, !discovery_v4))
            .consensus(OpConsensusBuilder::default())
    }

    /// Returns [`OpAddOnsBuilder`] with configured arguments.
    pub fn add_ons_builder<NetworkT: RpcTypes>(&self) -> OpAddOnsBuilder<NetworkT> {
        OpAddOnsBuilder::default()
            .with_sequencer(self.args.sequencer.clone())
            .with_sequencer_headers(self.args.sequencer_headers.clone())
            .with_da_config(self.da_config.clone())
            .with_gas_limit_config(self.gas_limit_config.clone())
            .with_enable_tx_conditional(self.args.enable_tx_conditional)
            .with_min_suggested_priority_fee(self.args.min_suggested_priority_fee)
            .with_historical_rpc(self.args.historical_rpc.clone())
            .with_flashblocks(self.args.flashblocks_url.clone())
            .with_flashblock_consensus(self.args.flashblock_consensus)
    }

    /// Instantiates the [`ProviderFactoryBuilder`] for an opstack node.
    ///
    /// # Open a Providerfactory in read-only mode from a datadir
    ///
    /// See also: [`ProviderFactoryBuilder`] and
    /// [`ReadOnlyConfig`](reth_provider::providers::ReadOnlyConfig).
    ///
    /// ```no_run
    /// use reth_optimism_chainspec::BASE_MAINNET;
    /// use reth_optimism_node::OpNode;
    ///
    /// fn demo(runtime: reth_tasks::Runtime) {
    ///     let factory = OpNode::provider_factory_builder()
    ///         .open_read_only(BASE_MAINNET.clone(), "datadir", runtime)
    ///         .unwrap();
    /// }
    /// ```
    ///
    /// # Open a Providerfactory with custom config
    ///
    /// ```no_run
    /// use reth_optimism_chainspec::OpChainSpecBuilder;
    /// use reth_optimism_node::OpNode;
    /// use reth_provider::providers::ReadOnlyConfig;
    ///
    /// fn demo(runtime: reth_tasks::Runtime) {
    ///     let factory = OpNode::provider_factory_builder()
    ///         .open_read_only(
    ///             OpChainSpecBuilder::base_mainnet().build().into(),
    ///             ReadOnlyConfig::from_datadir("datadir").no_watch(),
    ///             runtime,
    ///         )
    ///         .unwrap();
    /// }
    /// ```
    pub fn provider_factory_builder() -> ProviderFactoryBuilder<Self> {
        ProviderFactoryBuilder::default()
    }
}

impl<N> Node<N> for OpNode
where
    N: FullNodeTypes<Types: OpFullNodeTypes + OpNodeTypes>,
{
    type ComponentsBuilder = ComponentsBuilder<
        N,
        OpPoolBuilder,
        BasicPayloadServiceBuilder<OpPayloadBuilder>,
        OpNetworkBuilder,
        OpExecutorBuilder,
        OpConsensusBuilder,
    >;

    type AddOns = OpAddOns<
        NodeAdapter<N, <Self::ComponentsBuilder as NodeComponentsBuilder<N>>::Components>,
        OpEthApiBuilder,
        OpEngineValidatorBuilder,
        OpEngineApiBuilder<OpEngineValidatorBuilder>,
        BasicEngineValidatorBuilder<OpEngineValidatorBuilder>,
    >;

    fn components_builder(&self) -> Self::ComponentsBuilder {
        Self::components(self)
    }

    fn add_ons(&self) -> Self::AddOns {
        self.add_ons_builder().build()
    }
}

impl<N> DebugNode<N> for OpNode
where
    N: FullNodeComponents<Types = Self>,
{
    type RpcBlock = alloy_rpc_types_eth::Block<op_alloy_consensus::OpTxEnvelope>;

    fn rpc_to_primitive_block(rpc_block: Self::RpcBlock) -> reth_node_api::BlockTy<Self> {
        rpc_block.into_consensus()
    }

    fn local_payload_attributes_builder(
        chain_spec: &Self::ChainSpec,
    ) -> impl PayloadAttributesBuilder<<Self::Payload as PayloadTypes>::PayloadAttributes> {
        LocalPayloadAttributesBuilder::new(Arc::new(chain_spec.clone()))
    }
}

impl NodeTypes for OpNode {
    type Primitives = OpPrimitives;
    type ChainSpec = OpChainSpec;
    type Storage = OpStorage;
    type Payload = OpEngineTypes;
}

/// Add-ons w.r.t. optimism.
///
/// This type provides optimism-specific addons to the node and exposes the RPC server and engine
/// API.
#[derive(Debug)]
pub struct OpAddOns<
    N: FullNodeComponents,
    EthB: EthApiBuilder<N>,
    PVB,
    EB = OpEngineApiBuilder<PVB>,
    EVB = BasicEngineValidatorBuilder<PVB>,
    RpcMiddleware = Identity,
> {
    /// Rpc add-ons responsible for launching the RPC servers and instantiating the RPC handlers
    /// and eth-api.
    pub rpc_add_ons: RpcAddOns<N, EthB, PVB, EB, EVB, RpcMiddleware>,
    /// Data availability configuration for the OP builder.
    pub da_config: OpDAConfig,
    /// Gas limit configuration for the OP builder.
    pub gas_limit_config: OpGasLimitConfig,
    /// Sequencer client, configured to forward submitted transactions to sequencer of given OP
    /// network.
    pub sequencer_url: Option<String>,
    /// Headers to use for the sequencer client requests.
    pub sequencer_headers: Vec<String>,
    /// RPC endpoint for historical data.
    ///
    /// This can be used to forward pre-bedrock rpc requests (op-mainnet).
    pub historical_rpc: Option<String>,
    /// Enable transaction conditionals.
    enable_tx_conditional: bool,
    min_suggested_priority_fee: u64,
}

impl<N, EthB, PVB, EB, EVB, RpcMiddleware> OpAddOns<N, EthB, PVB, EB, EVB, RpcMiddleware>
where
    N: FullNodeComponents,
    EthB: EthApiBuilder<N>,
{
    /// Creates a new instance from components.
    #[allow(clippy::too_many_arguments)]
    pub const fn new(
        rpc_add_ons: RpcAddOns<N, EthB, PVB, EB, EVB, RpcMiddleware>,
        da_config: OpDAConfig,
        gas_limit_config: OpGasLimitConfig,
        sequencer_url: Option<String>,
        sequencer_headers: Vec<String>,
        historical_rpc: Option<String>,
        enable_tx_conditional: bool,
        min_suggested_priority_fee: u64,
    ) -> Self {
        Self {
            rpc_add_ons,
            da_config,
            gas_limit_config,
            sequencer_url,
            sequencer_headers,
            historical_rpc,
            enable_tx_conditional,
            min_suggested_priority_fee,
        }
    }
}

impl<N> Default for OpAddOns<N, OpEthApiBuilder, OpEngineValidatorBuilder>
where
    N: FullNodeComponents<Types: OpNodeTypes>,
    OpEthApiBuilder: EthApiBuilder<N>,
{
    fn default() -> Self {
        Self::builder().build()
    }
}

impl<N, NetworkT, RpcMiddleware>
    OpAddOns<
        N,
        OpEthApiBuilder<NetworkT>,
        OpEngineValidatorBuilder,
        OpEngineApiBuilder<OpEngineValidatorBuilder>,
        RpcMiddleware,
    >
where
    N: FullNodeComponents<Types: OpNodeTypes>,
    OpEthApiBuilder<NetworkT>: EthApiBuilder<N>,
{
    /// Build a [`OpAddOns`] using [`OpAddOnsBuilder`].
    pub fn builder() -> OpAddOnsBuilder<NetworkT> {
        OpAddOnsBuilder::default()
    }
}

impl<N, EthB, PVB, EB, EVB, RpcMiddleware> OpAddOns<N, EthB, PVB, EB, EVB, RpcMiddleware>
where
    N: FullNodeComponents,
    EthB: EthApiBuilder<N>,
{
    /// Maps the [`reth_node_builder::rpc::EngineApiBuilder`] builder type.
    pub fn with_engine_api<T>(
        self,
        engine_api_builder: T,
    ) -> OpAddOns<N, EthB, PVB, T, EVB, RpcMiddleware> {
        let Self {
            rpc_add_ons,
            da_config,
            gas_limit_config,
            sequencer_url,
            sequencer_headers,
            historical_rpc,
            enable_tx_conditional,
            min_suggested_priority_fee,
            ..
        } = self;
        OpAddOns::new(
            rpc_add_ons.with_engine_api(engine_api_builder),
            da_config,
            gas_limit_config,
            sequencer_url,
            sequencer_headers,
            historical_rpc,
            enable_tx_conditional,
            min_suggested_priority_fee,
        )
    }

    /// Maps the [`PayloadValidatorBuilder`] builder type.
    pub fn with_payload_validator<T>(
        self,
        payload_validator_builder: T,
    ) -> OpAddOns<N, EthB, T, EB, EVB, RpcMiddleware> {
        let Self {
            rpc_add_ons,
            da_config,
            gas_limit_config,
            sequencer_url,
            sequencer_headers,
            enable_tx_conditional,
            min_suggested_priority_fee,
            historical_rpc,
            ..
        } = self;
        OpAddOns::new(
            rpc_add_ons.with_payload_validator(payload_validator_builder),
            da_config,
            gas_limit_config,
            sequencer_url,
            sequencer_headers,
            historical_rpc,
            enable_tx_conditional,
            min_suggested_priority_fee,
        )
    }

    /// Sets the RPC middleware stack for processing RPC requests.
    ///
    /// This method configures a custom middleware stack that will be applied to all RPC requests
    /// across HTTP, `WebSocket`, and IPC transports. The middleware is applied to the RPC service
    /// layer, allowing you to intercept, modify, or enhance RPC request processing.
    ///
    /// See also [`RpcAddOns::with_rpc_middleware`].
    pub fn with_rpc_middleware<T>(self, rpc_middleware: T) -> OpAddOns<N, EthB, PVB, EB, EVB, T> {
        let Self {
            rpc_add_ons,
            da_config,
            gas_limit_config,
            sequencer_url,
            sequencer_headers,
            enable_tx_conditional,
            min_suggested_priority_fee,
            historical_rpc,
            ..
        } = self;
        OpAddOns::new(
            rpc_add_ons.with_rpc_middleware(rpc_middleware),
            da_config,
            gas_limit_config,
            sequencer_url,
            sequencer_headers,
            historical_rpc,
            enable_tx_conditional,
            min_suggested_priority_fee,
        )
    }

    /// Sets the hook that is run once the rpc server is started.
    pub fn on_rpc_started<F>(mut self, hook: F) -> Self
    where
        F: FnOnce(RpcContext<'_, N, EthB::EthApi>, RethRpcServerHandles) -> eyre::Result<()>
            + Send
            + 'static,
    {
        self.rpc_add_ons = self.rpc_add_ons.on_rpc_started(hook);
        self
    }

    /// Sets the hook that is run to configure the rpc modules.
    pub fn extend_rpc_modules<F>(mut self, hook: F) -> Self
    where
        F: FnOnce(RpcContext<'_, N, EthB::EthApi>) -> eyre::Result<()> + Send + 'static,
    {
        self.rpc_add_ons = self.rpc_add_ons.extend_rpc_modules(hook);
        self
    }
}

impl<N, EthB, PVB, EB, EVB, Attrs, RpcMiddleware> NodeAddOns<N>
    for OpAddOns<N, EthB, PVB, EB, EVB, RpcMiddleware>
where
    N: FullNodeComponents<
            Types: NodeTypes<
                ChainSpec: OpHardforks,
                Primitives: OpPayloadPrimitives,
                Payload: PayloadTypes<PayloadBuilderAttributes = Attrs>,
            >,
            Evm: ConfigureEvm<
                NextBlockEnvCtx: BuildNextEnv<
                    Attrs,
                    HeaderTy<N::Types>,
                    <N::Types as NodeTypes>::ChainSpec,
                >,
            >,
            Pool: TransactionPool<Transaction: OpPooledTx>,
        >,
    EthB: EthApiBuilder<N>,
    PVB: Send,
    EB: EngineApiBuilder<N>,
    EVB: EngineValidatorBuilder<N>,
    RpcMiddleware: RethRpcMiddleware,
    Attrs: OpAttributes<Transaction = TxTy<N::Types>, RpcPayloadAttributes: DeserializeOwned>,
{
    type Handle = RpcHandle<N, EthB::EthApi>;

    async fn launch_add_ons(
        self,
        ctx: reth_node_api::AddOnsContext<'_, N>,
    ) -> eyre::Result<Self::Handle> {
        let Self {
            rpc_add_ons,
            da_config,
            gas_limit_config,
            sequencer_url,
            sequencer_headers,
            enable_tx_conditional,
            historical_rpc,
            ..
        } = self;

        let maybe_pre_bedrock_historical_rpc = historical_rpc
            .and_then(|historical_rpc| {
                ctx.node
                    .provider()
                    .chain_spec()
                    .op_fork_activation(OpHardfork::Bedrock)
                    .block_number()
                    .filter(|activation| *activation > 0)
                    .map(|bedrock_block| (historical_rpc, bedrock_block))
            })
            .map(|(historical_rpc, bedrock_block)| -> eyre::Result<_> {
                info!(target: "reth::cli", %bedrock_block, ?historical_rpc, "Using historical RPC endpoint pre bedrock");
                let provider = ctx.node.provider().clone();
                let client = HistoricalRpcClient::new(&historical_rpc)?;
                let layer = HistoricalRpc::new(provider, client, bedrock_block);
                Ok(layer)
            })
            .transpose()?
            ;

        let rpc_add_ons = rpc_add_ons.option_layer_rpc_middleware(maybe_pre_bedrock_historical_rpc);

        let builder = reth_optimism_payload_builder::OpPayloadBuilder::new(
            ctx.node.pool().clone(),
            ctx.node.provider().clone(),
            ctx.node.evm_config().clone(),
        );
        // install additional OP specific rpc methods
        let debug_ext = OpDebugWitnessApi::<_, _, _, Attrs>::new(
            ctx.node.provider().clone(),
            Box::new(ctx.node.task_executor().clone()),
            builder,
        );
        let miner_ext = OpMinerExtApi::new(da_config, gas_limit_config);

        let sequencer_client = if let Some(url) = sequencer_url {
            Some(SequencerClient::new_with_headers(url, sequencer_headers).await?)
        } else {
            None
        };

        let tx_conditional_ext: OpEthExtApi<N::Pool, N::Provider> = OpEthExtApi::new(
            sequencer_client,
            ctx.node.pool().clone(),
            ctx.node.provider().clone(),
        );

        rpc_add_ons
            .launch_add_ons_with(ctx, move |container| {
                let reth_node_builder::rpc::RpcModuleContainer { modules, auth_module, registry } =
                    container;

                debug!(target: "reth::cli", "Installing debug payload witness rpc endpoint");
                modules.merge_if_module_configured(RethRpcModule::Debug, debug_ext.into_rpc())?;

                // extend the miner namespace if configured in the regular http server
                modules.add_or_replace_if_module_configured(
                    RethRpcModule::Miner,
                    miner_ext.clone().into_rpc(),
                )?;

                // install the miner extension in the authenticated if configured
                if modules.module_config().contains_any(&RethRpcModule::Miner) {
                    debug!(target: "reth::cli", "Installing miner DA rpc endpoint");
                    auth_module.merge_auth_methods(miner_ext.into_rpc())?;
                }

                // install the debug namespace in the authenticated if configured
                if modules.module_config().contains_any(&RethRpcModule::Debug) {
                    debug!(target: "reth::cli", "Installing debug rpc endpoint");
                    auth_module.merge_auth_methods(registry.debug_api().into_rpc())?;
                }

                if enable_tx_conditional {
                    // extend the eth namespace if configured in the regular http server
                    modules.merge_if_module_configured(
                        RethRpcModule::Eth,
                        tx_conditional_ext.into_rpc(),
                    )?;
                }

                Ok(())
            })
            .await
    }
}

impl<N, EthB, PVB, EB, EVB, Attrs, RpcMiddleware> RethRpcAddOns<N>
    for OpAddOns<N, EthB, PVB, EB, EVB, RpcMiddleware>
where
    N: FullNodeComponents<
            Types: NodeTypes<
                ChainSpec: OpHardforks,
                Primitives: OpPayloadPrimitives,
                Payload: PayloadTypes<PayloadBuilderAttributes = Attrs>,
            >,
            Evm: ConfigureEvm<
                NextBlockEnvCtx: BuildNextEnv<
                    Attrs,
                    HeaderTy<N::Types>,
                    <N::Types as NodeTypes>::ChainSpec,
                >,
            >,
        >,
    <<N as FullNodeComponents>::Pool as TransactionPool>::Transaction: OpPooledTx,
    EthB: EthApiBuilder<N>,
    PVB: PayloadValidatorBuilder<N>,
    EB: EngineApiBuilder<N>,
    EVB: EngineValidatorBuilder<N>,
    RpcMiddleware: RethRpcMiddleware,
    Attrs: OpAttributes<Transaction = TxTy<N::Types>, RpcPayloadAttributes: DeserializeOwned>,
{
    type EthApi = EthB::EthApi;

    fn hooks_mut(&mut self) -> &mut reth_node_builder::rpc::RpcHooks<N, Self::EthApi> {
        self.rpc_add_ons.hooks_mut()
    }
}

impl<N, EthB, PVB, EB, EVB, RpcMiddleware> EngineValidatorAddOn<N>
    for OpAddOns<N, EthB, PVB, EB, EVB, RpcMiddleware>
where
    N: FullNodeComponents,
    EthB: EthApiBuilder<N>,
    PVB: Send,
    EB: EngineApiBuilder<N>,
    EVB: EngineValidatorBuilder<N>,
    RpcMiddleware: Send,
{
    type ValidatorBuilder = EVB;

    fn engine_validator_builder(&self) -> Self::ValidatorBuilder {
        EngineValidatorAddOn::engine_validator_builder(&self.rpc_add_ons)
    }
}

/// A regular optimism evm and executor builder.
#[derive(Debug, Clone)]
#[non_exhaustive]
pub struct OpAddOnsBuilder<NetworkT, RpcMiddleware = Identity> {
    /// Sequencer client, configured to forward submitted transactions to sequencer of given OP
    /// network.
    sequencer_url: Option<String>,
    /// Headers to use for the sequencer client requests.
    sequencer_headers: Vec<String>,
    /// RPC endpoint for historical data.
    historical_rpc: Option<String>,
    /// Data availability configuration for the OP builder.
    da_config: Option<OpDAConfig>,
    /// Gas limit configuration for the OP builder.
    gas_limit_config: Option<OpGasLimitConfig>,
    /// Enable transaction conditionals.
    enable_tx_conditional: bool,
    /// Marker for network types.
    _nt: PhantomData<NetworkT>,
    /// Minimum suggested priority fee (tip)
    min_suggested_priority_fee: u64,
    /// RPC middleware to use
    rpc_middleware: RpcMiddleware,
    /// Optional tokio runtime to use for the RPC server.
    tokio_runtime: Option<tokio::runtime::Handle>,
    /// A URL pointing to a secure websocket service that streams out flashblocks.
    flashblocks_url: Option<Url>,
    /// Enable flashblock consensus client to drive chain forward.
    flashblock_consensus: bool,
}

impl<NetworkT> Default for OpAddOnsBuilder<NetworkT> {
    fn default() -> Self {
        Self {
            sequencer_url: None,
            sequencer_headers: Vec::new(),
            historical_rpc: None,
            da_config: None,
            gas_limit_config: None,
            enable_tx_conditional: false,
            min_suggested_priority_fee: 1_000_000,
            _nt: PhantomData,
            rpc_middleware: Identity::new(),
            tokio_runtime: None,
            flashblocks_url: None,
            flashblock_consensus: false,
        }
    }
}

impl<NetworkT, RpcMiddleware> OpAddOnsBuilder<NetworkT, RpcMiddleware> {
    /// With a [`SequencerClient`].
    pub fn with_sequencer(mut self, sequencer_client: Option<String>) -> Self {
        self.sequencer_url = sequencer_client;
        self
    }

    /// With headers to use for the sequencer client requests.
    pub fn with_sequencer_headers(mut self, sequencer_headers: Vec<String>) -> Self {
        self.sequencer_headers = sequencer_headers;
        self
    }

    /// Configure the data availability configuration for the OP builder.
    pub fn with_da_config(mut self, da_config: OpDAConfig) -> Self {
        self.da_config = Some(da_config);
        self
    }

    /// Configure the gas limit configuration for the OP payload builder.
    pub fn with_gas_limit_config(mut self, gas_limit_config: OpGasLimitConfig) -> Self {
        self.gas_limit_config = Some(gas_limit_config);
        self
    }

    /// Configure if transaction conditional should be enabled.
    pub const fn with_enable_tx_conditional(mut self, enable_tx_conditional: bool) -> Self {
        self.enable_tx_conditional = enable_tx_conditional;
        self
    }

    /// Configure the minimum priority fee (tip)
    pub const fn with_min_suggested_priority_fee(mut self, min: u64) -> Self {
        self.min_suggested_priority_fee = min;
        self
    }

    /// Configures the endpoint for historical RPC forwarding.
    pub fn with_historical_rpc(mut self, historical_rpc: Option<String>) -> Self {
        self.historical_rpc = historical_rpc;
        self
    }

    /// Configures a custom tokio runtime for the RPC server.
    ///
    /// Caution: This runtime must not be created from within asynchronous context.
    pub fn with_tokio_runtime(mut self, tokio_runtime: Option<tokio::runtime::Handle>) -> Self {
        self.tokio_runtime = tokio_runtime;
        self
    }

    /// Configure the RPC middleware to use
    pub fn with_rpc_middleware<T>(self, rpc_middleware: T) -> OpAddOnsBuilder<NetworkT, T> {
        let Self {
            sequencer_url,
            sequencer_headers,
            historical_rpc,
            da_config,
            gas_limit_config,
            enable_tx_conditional,
            min_suggested_priority_fee,
            tokio_runtime,
            _nt,
            flashblocks_url,
            flashblock_consensus,
            ..
        } = self;
        OpAddOnsBuilder {
            sequencer_url,
            sequencer_headers,
            historical_rpc,
            da_config,
            gas_limit_config,
            enable_tx_conditional,
            min_suggested_priority_fee,
            _nt,
            rpc_middleware,
            tokio_runtime,
            flashblocks_url,
            flashblock_consensus,
        }
    }

    /// With a URL pointing to a flashblocks secure websocket subscription.
    pub fn with_flashblocks(mut self, flashblocks_url: Option<Url>) -> Self {
        self.flashblocks_url = flashblocks_url;
        self
    }

    /// With a flashblock consensus client to drive chain forward.
    pub const fn with_flashblock_consensus(mut self, flashblock_consensus: bool) -> Self {
        self.flashblock_consensus = flashblock_consensus;
        self
    }
}

impl<NetworkT, RpcMiddleware> OpAddOnsBuilder<NetworkT, RpcMiddleware> {
    /// Builds an instance of [`OpAddOns`].
    pub fn build<N, PVB, EB, EVB>(
        self,
    ) -> OpAddOns<N, OpEthApiBuilder<NetworkT>, PVB, EB, EVB, RpcMiddleware>
    where
        N: FullNodeComponents<Types: NodeTypes>,
        OpEthApiBuilder<NetworkT>: EthApiBuilder<N>,
        PVB: PayloadValidatorBuilder<N> + Default,
        EB: Default,
        EVB: Default,
    {
        let Self {
            sequencer_url,
            sequencer_headers,
            da_config,
            gas_limit_config,
            enable_tx_conditional,
            min_suggested_priority_fee,
            historical_rpc,
            rpc_middleware,
            tokio_runtime,
            flashblocks_url,
            flashblock_consensus,
            ..
        } = self;

        OpAddOns::new(
            RpcAddOns::new(
                OpEthApiBuilder::default()
                    .with_sequencer(sequencer_url.clone())
                    .with_sequencer_headers(sequencer_headers.clone())
                    .with_min_suggested_priority_fee(min_suggested_priority_fee)
                    .with_flashblocks(flashblocks_url)
                    .with_flashblock_consensus(flashblock_consensus),
                PVB::default(),
                EB::default(),
                EVB::default(),
                rpc_middleware,
            )
            .with_tokio_runtime(tokio_runtime),
            da_config.unwrap_or_default(),
            gas_limit_config.unwrap_or_default(),
            sequencer_url,
            sequencer_headers,
            historical_rpc,
            enable_tx_conditional,
            min_suggested_priority_fee,
        )
    }
}

/// A regular optimism evm and executor builder.
#[derive(Debug, Copy, Clone, Default)]
#[non_exhaustive]
pub struct OpExecutorBuilder;

impl<Node> ExecutorBuilder<Node> for OpExecutorBuilder
where
    Node: FullNodeTypes<Types: NodeTypes<ChainSpec: OpHardforks, Primitives = OpPrimitives>>,
{
    type EVM =
        OpEvmConfig<<Node::Types as NodeTypes>::ChainSpec, <Node::Types as NodeTypes>::Primitives>;

    async fn build_evm(self, ctx: &BuilderContext<Node>) -> eyre::Result<Self::EVM> {
        let evm_config = OpEvmConfig::new(ctx.chain_spec(), OpRethReceiptBuilder::default());

        Ok(evm_config)
    }
}

/// A basic optimism transaction pool.
///
/// This contains various settings that can be configured and take precedence over the node's
/// config.
#[derive(Debug)]
pub struct OpPoolBuilder<T = crate::txpool::OpPooledTransaction> {
    /// Enforced overrides that are applied to the pool config.
    pub pool_config_overrides: PoolBuilderConfigOverrides,
    /// Enable transaction conditionals.
    pub enable_tx_conditional: bool,
    /// Supervisor client url
    pub supervisor_http: String,
    /// Supervisor safety level
    pub supervisor_safety_level: SafetyLevel,
    /// Marker for the pooled transaction type.
    _pd: core::marker::PhantomData<T>,
}

impl<T> Default for OpPoolBuilder<T> {
    fn default() -> Self {
        Self {
            pool_config_overrides: Default::default(),
            enable_tx_conditional: false,
            supervisor_http: DEFAULT_SUPERVISOR_URL.to_string(),
            supervisor_safety_level: SafetyLevel::CrossUnsafe,
            _pd: Default::default(),
        }
    }
}

impl<T> Clone for OpPoolBuilder<T> {
    fn clone(&self) -> Self {
        Self {
            pool_config_overrides: self.pool_config_overrides.clone(),
            enable_tx_conditional: self.enable_tx_conditional,
            supervisor_http: self.supervisor_http.clone(),
            supervisor_safety_level: self.supervisor_safety_level,
            _pd: core::marker::PhantomData,
        }
    }
}

impl<T> OpPoolBuilder<T> {
    /// Sets the `enable_tx_conditional` flag on the pool builder.
    pub const fn with_enable_tx_conditional(mut self, enable_tx_conditional: bool) -> Self {
        self.enable_tx_conditional = enable_tx_conditional;
        self
    }

    /// Sets the [`PoolBuilderConfigOverrides`] on the pool builder.
    pub fn with_pool_config_overrides(
        mut self,
        pool_config_overrides: PoolBuilderConfigOverrides,
    ) -> Self {
        self.pool_config_overrides = pool_config_overrides;
        self
    }

    /// Sets the supervisor client
    pub fn with_supervisor(
        mut self,
        supervisor_client: String,
        supervisor_safety_level: SafetyLevel,
    ) -> Self {
        self.supervisor_http = supervisor_client;
        self.supervisor_safety_level = supervisor_safety_level;
        self
    }
}

impl<Node, T, Evm> PoolBuilder<Node, Evm> for OpPoolBuilder<T>
where
    Node: FullNodeTypes<Types: NodeTypes<ChainSpec: OpHardforks>>,
    T: EthPoolTransaction<Consensus = TxTy<Node::Types>> + OpPooledTx,
    Evm: ConfigureEvm<Primitives = PrimitivesTy<Node::Types>> + Clone + 'static,
{
    type Pool = OpTransactionPool<Node::Provider, DiskFileBlobStore, Evm, T>;

    async fn build_pool(
        self,
        ctx: &BuilderContext<Node>,
        evm_config: Evm,
    ) -> eyre::Result<Self::Pool> {
        let Self { pool_config_overrides, .. } = self;

        // supervisor used for interop
        if ctx.chain_spec().is_interop_active_at_timestamp(ctx.head().timestamp) &&
            self.supervisor_http == DEFAULT_SUPERVISOR_URL
        {
            info!(target: "reth::cli",
                url=%DEFAULT_SUPERVISOR_URL,
                "Default supervisor url is used, consider changing --rollup.supervisor-http."
            );
        }
        let supervisor_client = SupervisorClient::builder(self.supervisor_http.clone())
            .minimum_safety(self.supervisor_safety_level)
            .build()
            .await;

        let blob_store = reth_node_builder::components::create_blob_store(ctx)?;
        let validator =
            TransactionValidationTaskExecutor::eth_builder(ctx.provider().clone(), evm_config)
                .no_eip4844()
                .with_max_tx_input_bytes(ctx.config().txpool.max_tx_input_bytes)
                .kzg_settings(ctx.kzg_settings()?)
                .set_tx_fee_cap(ctx.config().rpc.rpc_tx_fee_cap)
                .with_max_tx_gas_limit(ctx.config().txpool.max_tx_gas_limit)
                .with_minimum_priority_fee(ctx.config().txpool.minimum_priority_fee)
                .with_additional_tasks(
                    pool_config_overrides
                        .additional_validation_tasks
                        .unwrap_or_else(|| ctx.config().txpool.additional_validation_tasks),
                )
                .build_with_tasks(ctx.task_executor().clone(), blob_store.clone())
                .map(|validator| {
                    OpTransactionValidator::new(validator)
                        // In --dev mode we can't require gas fees because we're unable to decode
                        // the L1 block info
                        .require_l1_data_gas_fee(!ctx.config().dev.dev)
                        .with_supervisor(supervisor_client.clone())
                });

        let final_pool_config = pool_config_overrides.apply(ctx.pool_config());

        let transaction_pool = TxPoolBuilder::new(ctx)
            .with_validator(validator)
            .build_and_spawn_maintenance_task(blob_store, final_pool_config)?;

        info!(target: "reth::cli", "Transaction pool initialized");
        debug!(target: "reth::cli", "Spawned txpool maintenance task");

        // The Op txpool maintenance task is only spawned when interop is active
        if ctx.chain_spec().is_interop_active_at_timestamp(ctx.head().timestamp) {
            // spawn the Op txpool maintenance task
            let chain_events = ctx.provider().canonical_state_stream();
            ctx.task_executor().spawn_critical_task(
                "Op txpool interop maintenance task",
                reth_optimism_txpool::maintain::maintain_transaction_pool_interop_future(
                    transaction_pool.clone(),
                    chain_events,
                    supervisor_client,
                ),
            );
            debug!(target: "reth::cli", "Spawned Op interop txpool maintenance task");
        }

        if self.enable_tx_conditional {
            // spawn the Op txpool maintenance task
            let chain_events = ctx.provider().canonical_state_stream();
            ctx.task_executor().spawn_critical_task(
                "Op txpool conditional maintenance task",
                reth_optimism_txpool::maintain::maintain_transaction_pool_conditional_future(
                    transaction_pool.clone(),
                    chain_events,
                ),
            );
            debug!(target: "reth::cli", "Spawned Op conditional txpool maintenance task");
        }

        Ok(transaction_pool)
    }
}

/// A basic optimism payload service builder
#[derive(Debug, Default, Clone)]
pub struct OpPayloadBuilder<Txs = ()> {
    /// By default the pending block equals the latest block
    /// to save resources and not leak txs from the tx-pool,
    /// this flag enables computing of the pending block
    /// from the tx-pool instead.
    ///
    /// If `compute_pending_block` is not enabled, the payload builder
    /// will use the payload attributes from the latest block. Note
    /// that this flag is not yet functional.
    pub compute_pending_block: bool,
    /// The type responsible for yielding the best transactions for the payload if mempool
    /// transactions are allowed.
    pub best_transactions: Txs,
    /// This data availability configuration specifies constraints for the payload builder
    /// when assembling payloads
    pub da_config: OpDAConfig,
    /// Gas limit configuration for the OP builder.
    /// This is used to configure gas limit related constraints for the payload builder.
    pub gas_limit_config: OpGasLimitConfig,
}

impl OpPayloadBuilder {
    /// Create a new instance with the given `compute_pending_block` flag and data availability
    /// config.
    pub fn new(compute_pending_block: bool) -> Self {
        Self {
            compute_pending_block,
            best_transactions: (),
            da_config: OpDAConfig::default(),
            gas_limit_config: OpGasLimitConfig::default(),
        }
    }

    /// Configure the data availability configuration for the OP payload builder.
    pub fn with_da_config(mut self, da_config: OpDAConfig) -> Self {
        self.da_config = da_config;
        self
    }

    /// Configure the gas limit configuration for the OP payload builder.
    pub fn with_gas_limit_config(mut self, gas_limit_config: OpGasLimitConfig) -> Self {
        self.gas_limit_config = gas_limit_config;
        self
    }
}

impl<Txs> OpPayloadBuilder<Txs> {
    /// Configures the type responsible for yielding the transactions that should be included in the
    /// payload.
    pub fn with_transactions<T>(self, best_transactions: T) -> OpPayloadBuilder<T> {
        let Self { compute_pending_block, da_config, gas_limit_config, .. } = self;
        OpPayloadBuilder { compute_pending_block, best_transactions, da_config, gas_limit_config }
    }
}

impl<Node, Pool, Txs, Evm, Attrs> PayloadBuilderBuilder<Node, Pool, Evm> for OpPayloadBuilder<Txs>
where
    Node: FullNodeTypes<
            Provider: ChainSpecProvider<ChainSpec: OpHardforks>,
            Types: NodeTypes<
                Primitives: OpPayloadPrimitives,
                Payload: PayloadTypes<
                    BuiltPayload = OpBuiltPayload<PrimitivesTy<Node::Types>>,
                    PayloadBuilderAttributes = Attrs,
                >,
            >,
        >,
    Evm: ConfigureEvm<
            Primitives = PrimitivesTy<Node::Types>,
            NextBlockEnvCtx: BuildNextEnv<
                Attrs,
                HeaderTy<Node::Types>,
                <Node::Types as NodeTypes>::ChainSpec,
            >,
        > + 'static,
    Pool: TransactionPool<Transaction: OpPooledTx<Consensus = TxTy<Node::Types>>> + Unpin + 'static,
    Txs: OpPayloadTransactions<Pool::Transaction>,
    Attrs: OpAttributes<Transaction = TxTy<Node::Types>>,
{
    type PayloadBuilder =
        reth_optimism_payload_builder::OpPayloadBuilder<Pool, Node::Provider, Evm, Txs, Attrs>;

    async fn build_payload_builder(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
        evm_config: Evm,
    ) -> eyre::Result<Self::PayloadBuilder> {
        let payload_builder = reth_optimism_payload_builder::OpPayloadBuilder::with_builder_config(
            pool,
            ctx.provider().clone(),
            evm_config,
            OpBuilderConfig {
                da_config: self.da_config.clone(),
                gas_limit_config: self.gas_limit_config.clone(),
            },
        )
        .with_transactions(self.best_transactions.clone())
        .set_compute_pending_block(self.compute_pending_block);
        Ok(payload_builder)
    }
}

/// A basic optimism network builder.
#[derive(Debug, Default)]
pub struct OpNetworkBuilder {
    /// Disable transaction pool gossip
    pub disable_txpool_gossip: bool,
    /// Disable discovery v4
    pub disable_discovery_v4: bool,
}

impl Clone for OpNetworkBuilder {
    fn clone(&self) -> Self {
        Self::new(self.disable_txpool_gossip, self.disable_discovery_v4)
    }
}

impl OpNetworkBuilder {
    /// Creates a new `OpNetworkBuilder`.
    pub const fn new(disable_txpool_gossip: bool, disable_discovery_v4: bool) -> Self {
        Self { disable_txpool_gossip, disable_discovery_v4 }
    }
}

impl OpNetworkBuilder {
    /// Returns the [`NetworkConfig`] that contains the settings to launch the p2p network.
    ///
    /// This applies the configured [`OpNetworkBuilder`] settings.
    pub fn network_config<Node, NetworkP>(
        &self,
        ctx: &BuilderContext<Node>,
    ) -> eyre::Result<NetworkConfig<Node::Provider, NetworkP>>
    where
        Node: FullNodeTypes<Types: NodeTypes<ChainSpec: Hardforks>>,
        NetworkP: NetworkPrimitives,
    {
        let disable_txpool_gossip = self.disable_txpool_gossip;
        let disable_discovery_v4 = self.disable_discovery_v4;
        let args = &ctx.config().network;
        let network_builder = ctx
            .network_config_builder()?
            // apply discovery settings
            .apply(|mut builder| {
                let rlpx_socket = (args.addr, args.port).into();
                if disable_discovery_v4 || args.discovery.disable_discovery {
                    builder = builder.disable_discv4_discovery();
                }
                if !args.discovery.disable_discovery {
                    builder = builder.discovery_v5(
                        args.discovery.discovery_v5_builder(
                            rlpx_socket,
                            ctx.config()
                                .network
                                .resolved_bootnodes()
                                .or_else(|| ctx.chain_spec().bootnodes())
                                .unwrap_or_default(),
                        ),
                    );
                }

                builder
            });

        let mut network_config = ctx.build_network_config(network_builder);

        // When `sequencer_endpoint` is configured, the node will forward all transactions to a
        // Sequencer node for execution and inclusion on L1, and disable its own txpool
        // gossip to prevent other parties in the network from learning about them.
        network_config.tx_gossip_disabled = disable_txpool_gossip;

        Ok(network_config)
    }
}

impl<Node, Pool> NetworkBuilder<Node, Pool> for OpNetworkBuilder
where
    Node: FullNodeTypes<Types: NodeTypes<ChainSpec: Hardforks>>,
    Pool: TransactionPool<Transaction: PoolTransaction<Consensus = TxTy<Node::Types>>>
        + Unpin
        + 'static,
{
    type Network =
        NetworkHandle<BasicNetworkPrimitives<PrimitivesTy<Node::Types>, PoolPooledTx<Pool>>>;

    async fn build_network(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
    ) -> eyre::Result<Self::Network> {
        let network_config = self.network_config(ctx)?;
        let network = NetworkManager::builder(network_config).await?;
        let handle = ctx.start_network(network, pool);
        info!(target: "reth::cli", enode=%handle.local_node_record(), "P2P networking initialized");

        Ok(handle)
    }
}

/// A basic optimism consensus builder.
#[derive(Debug, Default, Clone)]
#[non_exhaustive]
pub struct OpConsensusBuilder;

impl<Node> ConsensusBuilder<Node> for OpConsensusBuilder
where
    Node: FullNodeTypes<
        Types: NodeTypes<
            ChainSpec: OpHardforks,
            Primitives: NodePrimitives<Receipt: DepositReceipt>,
        >,
    >,
{
    type Consensus = Arc<OpBeaconConsensus<<Node::Types as NodeTypes>::ChainSpec>>;

    async fn build_consensus(self, ctx: &BuilderContext<Node>) -> eyre::Result<Self::Consensus> {
        Ok(Arc::new(OpBeaconConsensus::new(ctx.chain_spec())))
    }
}

/// Builder for [`OpEngineValidator`].
#[derive(Debug, Default, Clone)]
#[non_exhaustive]
pub struct OpEngineValidatorBuilder;

impl<Node> PayloadValidatorBuilder<Node> for OpEngineValidatorBuilder
where
    Node: FullNodeComponents<
        Types: NodeTypes<
            ChainSpec: OpHardforks,
            Payload: PayloadTypes<ExecutionData = OpExecutionData>,
        >,
    >,
{
    type Validator = OpEngineValidator<
        Node::Provider,
        <<Node::Types as NodeTypes>::Primitives as NodePrimitives>::SignedTx,
        <Node::Types as NodeTypes>::ChainSpec,
    >;

    async fn build(self, ctx: &AddOnsContext<'_, Node>) -> eyre::Result<Self::Validator> {
        Ok(OpEngineValidator::new::<KeccakKeyHasher>(
            ctx.config.chain.clone(),
            ctx.node.provider().clone(),
        ))
    }
}

/// Network primitive types used by Optimism networks.
pub type OpNetworkPrimitives = BasicNetworkPrimitives<OpPrimitives, OpPooledTransaction>;
