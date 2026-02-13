//! RPC component builder
//!
//! # Example
//!
//! Builds offline `TraceApi` with only EVM and database. This can be useful
//! for example when downloading a state snapshot (pre-synced node) from some mirror.
//!
//! ```rust
//! use alloy_rpc_types_eth::BlockId;
//! use op_alloy_network::Optimism;
//! use reth_db::test_utils::create_test_rw_db_with_path;
//! use reth_node_builder::{
//!     ConsensusEngineHandle, LaunchContext, NodeConfig, RethFullAdapter,
//!     components::ComponentsBuilder,
//!     hooks::OnComponentInitializedHook,
//!     rpc::{EthApiBuilder, EthApiCtx},
//! };
//! use reth_optimism_chainspec::OP_SEPOLIA;
//! use reth_optimism_evm::OpEvmConfig;
//! use reth_optimism_node::{OpExecutorBuilder, OpNetworkPrimitives, OpNode};
//! use reth_optimism_rpc::OpEthApiBuilder;
//! use reth_optimism_txpool::OpPooledTransaction;
//! use reth_provider::providers::BlockchainProvider;
//! use reth_rpc::TraceApi;
//! use reth_rpc_eth_types::{EthConfig, EthStateCache};
//! use reth_tasks::{Runtime, pool::BlockingTaskGuard};
//! use reth_trie_db::ChangesetCache;
//! use std::sync::Arc;
//!
//! #[tokio::main]
//! async fn main() {
//!     // build core node with all components disabled except EVM and state
//!     let sepolia = NodeConfig::new(OP_SEPOLIA.clone());
//!     let db = create_test_rw_db_with_path(sepolia.datadir());
//!     let runtime = Runtime::with_existing_handle(tokio::runtime::Handle::current()).unwrap();
//!     let launch_ctx = LaunchContext::new(runtime, sepolia.datadir());
//!     let node = launch_ctx
//!         .with_loaded_toml_config(sepolia)
//!         .unwrap()
//!         .attach(Arc::new(db))
//!         .with_provider_factory::<_, OpEvmConfig>(ChangesetCache::new())
//!         .await
//!         .unwrap()
//!         .with_genesis()
//!         .unwrap()
//!         .with_metrics_task() // todo: shouldn't be req to set up blockchain db
//!         .with_blockchain_db::<RethFullAdapter<_, OpNode>, _>(move |provider_factory| {
//!             Ok(BlockchainProvider::new(provider_factory).unwrap())
//!         })
//!         .unwrap()
//!         .with_components(
//!             ComponentsBuilder::default()
//!                 .node_types::<RethFullAdapter<_, OpNode>>()
//!                 .noop_pool::<OpPooledTransaction>()
//!                 .executor(OpExecutorBuilder::default())
//!                 .noop_consensus()
//!                 .noop_network::<OpNetworkPrimitives>()
//!                 .noop_payload(),
//!             Box::new(()) as Box<dyn OnComponentInitializedHook<_>>,
//!         )
//!         .await
//!         .unwrap();
//!
//!     // build `eth` namespace API
//!     let config = EthConfig::default();
//!     let cache = EthStateCache::spawn_with(
//!         node.provider_factory().clone(),
//!         config.cache,
//!         node.task_executor().clone(),
//!     );
//!     // Create a dummy beacon engine handle for offline mode
//!     let (tx, _) = tokio::sync::mpsc::unbounded_channel();
//!     let ctx = EthApiCtx {
//!         components: node.node_adapter(),
//!         config,
//!         cache,
//!         engine_handle: ConsensusEngineHandle::new(tx),
//!     };
//!     let eth_api = OpEthApiBuilder::<Optimism>::default().build_eth_api(ctx).await.unwrap();
//!
//!     // build `trace` namespace API
//!     let trace_api = TraceApi::new(eth_api, BlockingTaskGuard::new(10), EthConfig::default());
//!
//!     // fetch traces for latest block
//!     let traces = trace_api.trace_block(BlockId::latest()).await.unwrap();
//! }
//! ```

pub use reth_optimism_rpc::{OpEngineApi, OpEthApi, OpEthApiBuilder};

use crate::OP_NAME_CLIENT;
use alloy_rpc_types_engine::ClientVersionV1;
use op_alloy_rpc_types_engine::OpExecutionData;
use reth_chainspec::EthereumHardforks;
use reth_node_api::{
    AddOnsContext, EngineApiValidator, EngineTypes, FullNodeComponents, NodeTypes,
};
use reth_node_builder::rpc::{EngineApiBuilder, PayloadValidatorBuilder};
use reth_node_core::version::{CLIENT_CODE, version_metadata};
use reth_optimism_rpc::engine::OP_ENGINE_CAPABILITIES;
use reth_payload_builder::PayloadStore;
use reth_rpc_engine_api::{EngineApi, EngineCapabilities};

/// Builder for basic [`OpEngineApi`] implementation.
#[derive(Debug, Default, Clone)]
pub struct OpEngineApiBuilder<EV> {
    engine_validator_builder: EV,
}

impl<N, EV> EngineApiBuilder<N> for OpEngineApiBuilder<EV>
where
    N: FullNodeComponents<
        Types: NodeTypes<
            ChainSpec: EthereumHardforks,
            Payload: EngineTypes<ExecutionData = OpExecutionData>,
        >,
    >,
    EV: PayloadValidatorBuilder<N>,
    EV::Validator: EngineApiValidator<<N::Types as NodeTypes>::Payload>,
{
    type EngineApi = OpEngineApi<
        N::Provider,
        <N::Types as NodeTypes>::Payload,
        N::Pool,
        EV::Validator,
        <N::Types as NodeTypes>::ChainSpec,
    >;

    async fn build_engine_api(self, ctx: &AddOnsContext<'_, N>) -> eyre::Result<Self::EngineApi> {
        let Self { engine_validator_builder } = self;

        let engine_validator = engine_validator_builder.build(ctx).await?;
        let client = ClientVersionV1 {
            code: CLIENT_CODE,
            name: OP_NAME_CLIENT.to_string(),
            version: version_metadata().cargo_pkg_version.to_string(),
            commit: version_metadata().vergen_git_sha.to_string(),
        };
        let inner = EngineApi::new(
            ctx.node.provider().clone(),
            ctx.config.chain.clone(),
            ctx.beacon_engine_handle.clone(),
            PayloadStore::new(ctx.node.payload_builder_handle().clone()),
            ctx.node.pool().clone(),
            Box::new(ctx.node.task_executor().clone()),
            client,
            EngineCapabilities::new(OP_ENGINE_CAPABILITIES.iter().copied()),
            engine_validator,
            ctx.config.engine.accept_execution_requests_hash,
            ctx.node.network().clone(),
        );

        Ok(OpEngineApi::new(inner))
    }
}
