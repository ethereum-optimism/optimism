//! Node builder setup tests.

use alloy_op_evm::{
    OpEvmContext, OpTxError,
    post_exec::{
        PostExecEvmFactoryAdapter, PostExecEvmFactoryHooks, PostExecExecutedTx, PostExecTxContext,
        WarmingState,
    },
};
use alloy_primitives::{Bytes, address};
use core::marker::PhantomData;
use op_revm::{OpHaltReason, OpSpecId, precompiles::OpPrecompiles};
use reth_db::test_utils::create_test_rw_db;
use reth_evm::{Database, Evm, EvmEnv, EvmFactory, precompiles::PrecompilesMap};
use reth_node_api::{FullNodeComponents, NodeTypesWithDBAdapter};
use reth_node_builder::{
    BuilderContext, FullNodeTypes, Node, NodeBuilder, NodeConfig, NodeTypes,
    components::ExecutorBuilder,
};
use reth_optimism_chainspec::{BASE_MAINNET, OP_SEPOLIA, OpChainSpec};
use reth_optimism_evm::{OpBlockExecutorFactory, OpEvm, OpEvmFactory, OpRethReceiptBuilder, OpTx};
use reth_optimism_node::{OpEvmConfig, OpExecutorBuilder, OpNode, args::RollupArgs};
use reth_optimism_primitives::OpPrimitives;
use reth_provider::providers::BlockchainProvider;
use revm::{
    Inspector,
    context::{BlockEnv, ContextTr},
    context_interface::result::EVMError,
    inspector::NoOpInspector,
    interpreter::interpreter::EthInterpreter,
    precompile::{Precompile, PrecompileId, PrecompileOutput, PrecompileResult, Precompiles},
};
use std::sync::OnceLock;

#[test]
fn test_basic_setup() {
    // parse CLI -> config
    let config = NodeConfig::new(BASE_MAINNET.clone());
    let db = create_test_rw_db();
    let args = RollupArgs::default();
    let op_node = OpNode::new(args);
    let _builder = NodeBuilder::new(config)
        .with_database(db)
        .with_types_and_provider::<OpNode, BlockchainProvider<NodeTypesWithDBAdapter<OpNode, _>>>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .on_component_initialized(move |ctx| {
            let _provider = ctx.provider();
            Ok(())
        })
        .on_node_started(|_full_node| Ok(()))
        .on_rpc_started(|_ctx, handles| {
            let _client = handles.rpc.http_client();
            Ok(())
        })
        .extend_rpc_modules(|ctx| {
            let _ = ctx.config();
            let _ = ctx.node().provider();

            Ok(())
        })
        .check_launch();
}

#[test]
fn test_setup_custom_precompiles() {
    /// Unichain custom precompiles.
    struct UniPrecompiles;

    impl UniPrecompiles {
        /// Returns map of precompiles for Unichain.
        fn precompiles(spec_id: OpSpecId) -> PrecompilesMap {
            static INSTANCE: OnceLock<Precompiles> = OnceLock::new();

            PrecompilesMap::from_static(INSTANCE.get_or_init(|| {
                let mut precompiles = OpPrecompiles::new_with_spec(spec_id).precompiles().clone();
                // Custom precompile.
                let precompile = Precompile::new(
                    PrecompileId::custom("custom"),
                    address!("0x0000000000000000000000000000000000756e69"),
                    |_, _, reservoir| {
                        PrecompileResult::Ok(PrecompileOutput::new(0, Bytes::new(), reservoir))
                    },
                );
                precompiles.extend([precompile]);
                precompiles
            }))
        }
    }

    /// Builds Unichain EVM configuration.
    #[derive(Clone, Debug)]
    struct UniEvmFactory;

    impl EvmFactory for UniEvmFactory {
        type Evm<DB: Database, I: Inspector<OpEvmContext<DB>>> =
            OpEvm<DB, I, Self::Precompiles, OpTx>;
        type Context<DB: Database> = OpEvmContext<DB>;
        type Tx = OpTx;
        type Error<DBError: core::error::Error + Send + Sync + 'static> =
            EVMError<DBError, OpTxError>;
        type HaltReason = OpHaltReason;
        type Spec = OpSpecId;
        type BlockEnv = BlockEnv;
        type Precompiles = PrecompilesMap;

        fn create_evm<DB: Database>(
            &self,
            db: DB,
            input: EvmEnv<OpSpecId>,
        ) -> Self::Evm<DB, NoOpInspector> {
            let mut op_evm = OpEvmFactory::default().create_evm(db, input);
            *op_evm.components_mut().2 = UniPrecompiles::precompiles(*op_evm.ctx().cfg().spec());

            op_evm
        }

        fn create_evm_with_inspector<
            DB: Database,
            I: Inspector<Self::Context<DB>, EthInterpreter>,
        >(
            &self,
            db: DB,
            input: EvmEnv<OpSpecId>,
            inspector: I,
        ) -> Self::Evm<DB, I> {
            let mut op_evm =
                OpEvmFactory::default().create_evm_with_inspector(db, input, inspector);
            *op_evm.components_mut().2 = UniPrecompiles::precompiles(*op_evm.ctx().cfg().spec());

            op_evm
        }
    }

    impl PostExecEvmFactoryHooks for UniEvmFactory {
        fn begin_post_exec_tx<DB, I>(evm: &mut Self::Evm<DB, I>, ctx: PostExecTxContext)
        where
            DB: Database,
            I: Inspector<Self::Context<DB>>,
        {
            evm.begin_post_exec_tx(ctx);
        }

        fn take_last_post_exec_tx_result<DB, I>(evm: &mut Self::Evm<DB, I>) -> PostExecExecutedTx
        where
            DB: Database,
            I: Inspector<Self::Context<DB>>,
        {
            evm.take_last_post_exec_tx_result()
        }

        fn warming_state<DB, I>(evm: &Self::Evm<DB, I>) -> WarmingState
        where
            DB: Database,
            I: Inspector<Self::Context<DB>>,
        {
            evm.warming_state()
        }

        fn seed_warming_state<DB, I>(evm: &mut Self::Evm<DB, I>, state: WarmingState)
        where
            DB: Database,
            I: Inspector<Self::Context<DB>>,
        {
            evm.seed_warming_state(state);
        }
    }

    /// Unichain executor builder.
    ///
    /// This is a type-level/builder-plumbing test for downstream OP Stack chains that need to
    /// customize the EVM executor, for example to add chain-specific precompiles. `check_launch`
    /// does not execute a block or call the custom precompile; it verifies that a custom executor
    /// builder, custom EVM factory, and OP node components compose into a launchable node config.
    struct UniExecutorBuilder;

    impl<Node> ExecutorBuilder<Node> for UniExecutorBuilder
    where
        Node: FullNodeTypes<Types: NodeTypes<ChainSpec = OpChainSpec, Primitives = OpPrimitives>>,
    {
        type EVM = OpEvmConfig<
            OpChainSpec,
            <Node::Types as NodeTypes>::Primitives,
            OpRethReceiptBuilder,
            PostExecEvmFactoryAdapter<UniEvmFactory>,
        >;

        async fn build_evm(self, ctx: &BuilderContext<Node>) -> eyre::Result<Self::EVM> {
            let OpEvmConfig { executor_factory, block_assembler, _pd: _ } =
                OpExecutorBuilder::default().build_evm(ctx).await?;
            let uni_executor_factory = OpBlockExecutorFactory::new(
                *executor_factory.receipt_builder(),
                ctx.chain_spec(),
                PostExecEvmFactoryAdapter::new(UniEvmFactory),
            );
            let uni_evm_config = OpEvmConfig {
                executor_factory: uni_executor_factory,
                block_assembler,
                _pd: PhantomData,
            };
            Ok(uni_evm_config)
        }
    }

    NodeBuilder::new(NodeConfig::new(OP_SEPOLIA.clone()))
        .with_database(create_test_rw_db())
        .with_types::<OpNode>()
        .with_components(
            OpNode::default()
                .components()
                // Custom EVM configuration
                .executor(UniExecutorBuilder),
        )
        .check_launch();
}
