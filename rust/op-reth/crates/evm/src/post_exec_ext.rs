use alloc::{sync::Arc, vec::Vec};
use alloy_consensus::Header;
use alloy_evm::{FromRecoveredTx, FromTxWithEncoded, block::BlockExecutorFor};
use alloy_op_evm::{
    OpBlockExecutor, block::receipt_builder::OpReceiptBuilder, post_exec::PostExecExecutorExt,
};
use reth_chainspec::EthChainSpec;
use reth_evm::{
    ConfigureEvm, Database,
    execute::{BasicBlockBuilder, BlockBuilder},
};
use reth_optimism_forks::OpHardforks;
use reth_optimism_primitives::DepositReceipt;
use reth_primitives_traits::{NodePrimitives, SealedBlock, SealedHeader, SignedTransaction};
use revm::database::State;

use crate::{OpBlockExecutorFactory, OpEvmConfig, OpEvmFactory, OpTx, PostExecMode};

/// Optimism-specific EVM helpers that expose post-exec-aware executors and builders.
pub trait ConfigurePostExecEvm: ConfigureEvm {
    /// Returns a block executor for the given block with explicit post-exec entry access.
    fn post_exec_executor_for_block<'a, DB: Database>(
        &'a self,
        db: &'a mut State<DB>,
        block: &'a SealedBlock<<Self::Primitives as NodePrimitives>::Block>,
        post_exec_mode: PostExecMode,
    ) -> Result<
        impl BlockExecutorFor<'a, Self::BlockExecutorFactory, &'a mut State<DB>> + PostExecExecutorExt,
        Self::Error,
    >;

    /// Returns a block builder for the next block with explicit post-exec entry access.
    fn post_exec_builder_for_next_block<'a, DB: Database + 'a>(
        &'a self,
        db: &'a mut State<DB>,
        parent: &'a SealedHeader<<Self::Primitives as NodePrimitives>::BlockHeader>,
        attributes: Self::NextBlockEnvCtx,
        post_exec_mode: PostExecMode,
    ) -> Result<
        impl BlockBuilder<
            Primitives = Self::Primitives,
            Executor: BlockExecutorFor<'a, Self::BlockExecutorFactory, &'a mut State<DB>>
                          + PostExecExecutorExt,
        > + 'a,
        Self::Error,
    >;
}

impl<ChainSpec, N, R> ConfigurePostExecEvm for OpEvmConfig<ChainSpec, N, R>
where
    ChainSpec: EthChainSpec<Header = Header> + OpHardforks + Send + Sync + Unpin + 'static,
    N: NodePrimitives<
            Receipt = R::Receipt,
            SignedTx = R::Transaction,
            BlockHeader = Header,
            BlockBody = alloy_consensus::BlockBody<R::Transaction>,
            Block = alloy_consensus::Block<R::Transaction>,
        >,
    OpTx: FromRecoveredTx<N::SignedTx> + FromTxWithEncoded<N::SignedTx>,
    R: OpReceiptBuilder<Receipt: DepositReceipt, Transaction: SignedTransaction>
        + Clone
        + Send
        + Sync
        + Unpin
        + 'static,
    Self: Send + Sync + Unpin + Clone + 'static,
{
    fn post_exec_executor_for_block<'a, DB: Database>(
        &'a self,
        db: &'a mut State<DB>,
        block: &'a SealedBlock<<Self::Primitives as NodePrimitives>::Block>,
        post_exec_mode: PostExecMode,
    ) -> Result<
        impl BlockExecutorFor<'a, Self::BlockExecutorFactory, &'a mut State<DB>> + PostExecExecutorExt,
        Self::Error,
    > {
        let evm = self.evm_for_block(db, block.header())?;
        let ctx = self.context_for_block_with_post_exec_mode(block, Some(post_exec_mode));

        Ok(OpBlockExecutor::new(
            evm,
            ctx,
            self.executor_factory.spec(),
            self.executor_factory.receipt_builder(),
        )
        .with_post_exec_begin(alloy_op_evm::post_exec::PostExecEvmExt::begin_post_exec_tx)
        .with_post_exec_result(
            alloy_op_evm::post_exec::PostExecEvmExt::take_last_post_exec_tx_result,
        ))
    }

    fn post_exec_builder_for_next_block<'a, DB: Database + 'a>(
        &'a self,
        db: &'a mut State<DB>,
        parent: &'a SealedHeader<<Self::Primitives as NodePrimitives>::BlockHeader>,
        attributes: Self::NextBlockEnvCtx,
        post_exec_mode: PostExecMode,
    ) -> Result<
        impl BlockBuilder<
            Primitives = Self::Primitives,
            Executor: BlockExecutorFor<'a, Self::BlockExecutorFactory, &'a mut State<DB>>
                          + PostExecExecutorExt,
        > + 'a,
        Self::Error,
    > {
        let evm_env = self.next_evm_env(parent, &attributes)?;
        let evm = self.evm_with_env(db, evm_env);
        let ctx =
            self.context_for_next_block_with_post_exec_mode(parent, attributes, post_exec_mode);
        let executor = OpBlockExecutor::new(
            evm,
            ctx.clone(),
            self.executor_factory.spec(),
            self.executor_factory.receipt_builder(),
        )
        .with_post_exec_begin(alloy_op_evm::post_exec::PostExecEvmExt::begin_post_exec_tx)
        .with_post_exec_result(
            alloy_op_evm::post_exec::PostExecEvmExt::take_last_post_exec_tx_result,
        );

        Ok(BasicBlockBuilder::<
            'a,
            OpBlockExecutorFactory<R, Arc<ChainSpec>, OpEvmFactory<OpTx>>,
            _,
            _,
            N,
        > {
            executor,
            transactions: Vec::new(),
            ctx,
            parent,
            assembler: self.block_assembler(),
        })
    }
}
