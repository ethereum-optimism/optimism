use alloc::{sync::Arc, vec::Vec};
use alloy_evm::{FromRecoveredTx, FromTxWithEncoded, block::BlockExecutor};
use alloy_op_evm::{
    OpBlockExecutor, PreRefundGasUsed,
    block::OpTxEnv,
    post_exec::{PostExecEvmFactoryAdapter, PostExecEvmFactoryHooks, PostExecExecutorExt},
};
use core::fmt::Debug;
use op_revm::OpSpecId;
use reth_evm::{
    ConfigureEvm, Database,
    execute::{BasicBlockBuilder, BlockBuilder},
    precompiles::PrecompilesMap,
};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
use reth_primitives_traits::{NodePrimitives, SealedBlock, SealedHeader};
use revm::{context::BlockEnv, database::State};

use crate::{
    OpBlockExecutorFactory, OpEvmConfig, OpEvmFactory, OpRethReceiptBuilder, OpTx, PostExecMode,
};

/// Optimism-specific EVM helpers that expose post-exec-aware executors and builders.
#[allow(clippy::type_complexity)]
pub trait ConfigurePostExecEvm: ConfigureEvm {
    /// Returns a block executor for the given block with explicit post-exec entry access.
    ///
    /// # Errors
    ///
    /// Returns an error if creating the block EVM fails.
    fn post_exec_executor_for_block<'a, DB: Database>(
        &'a self,
        db: &'a mut State<DB>,
        block: &'a SealedBlock<<Self::Primitives as NodePrimitives>::Block>,
        post_exec_mode: PostExecMode,
    ) -> Result<
        impl BlockExecutor<
            Transaction = <Self::Primitives as NodePrimitives>::SignedTx,
            Receipt = <Self::Primitives as NodePrimitives>::Receipt,
        > + PostExecExecutorExt
        + 'a,
        Self::Error,
    >;

    /// Returns a block builder for the next block with explicit post-exec entry access.
    ///
    /// # Errors
    ///
    /// Returns an error if deriving the next-block EVM environment fails.
    fn post_exec_builder_for_next_block<'a, DB: Database + 'a>(
        &'a self,
        db: &'a mut State<DB>,
        parent: &'a SealedHeader<<Self::Primitives as NodePrimitives>::BlockHeader>,
        attributes: Self::NextBlockEnvCtx,
        post_exec_mode: PostExecMode,
    ) -> Result<
        impl BlockBuilder<
            Primitives = Self::Primitives,
            Executor: PostExecExecutorExt
                          + BlockExecutor<
                Evm: alloy_evm::Evm<DB: core::ops::DerefMut<Target = State<DB>>>,
                Result: PreRefundGasUsed,
            >,
        > + 'a,
        Self::Error,
    >;
}

impl ConfigurePostExecEvm for OpEvmConfig {
    fn post_exec_executor_for_block<'a, DB: Database>(
        &'a self,
        db: &'a mut State<DB>,
        block: &'a SealedBlock<<Self::Primitives as NodePrimitives>::Block>,
        post_exec_mode: PostExecMode,
    ) -> Result<
        impl BlockExecutor<
            Transaction = <Self::Primitives as NodePrimitives>::SignedTx,
            Receipt = <Self::Primitives as NodePrimitives>::Receipt,
        > + PostExecExecutorExt
        + 'a,
        Self::Error,
    > {
        let evm = self.evm_for_block(db, block.header())?;
        let ctx = self.context_for_block_with_post_exec_mode(block, Some(post_exec_mode));

        Ok(OpBlockExecutor::new(
            evm,
            ctx,
            self.executor_factory.spec(),
            self.executor_factory.receipt_builder(),
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
            Executor: PostExecExecutorExt
                          + BlockExecutor<
                Evm: alloy_evm::Evm<DB: core::ops::DerefMut<Target = State<DB>>>,
                Result: PreRefundGasUsed,
            >,
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
        );

        Ok(BasicBlockBuilder::<
            'a,
            OpBlockExecutorFactory<OpRethReceiptBuilder, Arc<OpChainSpec>, OpEvmFactory<OpTx>>,
            _,
            _,
            OpPrimitives,
        > {
            executor,
            transactions: Vec::new(),
            ctx,
            parent,
            assembler: self.block_assembler(),
        })
    }
}

impl<F> ConfigurePostExecEvm for OpEvmConfig<PostExecEvmFactoryAdapter<F>>
where
    F: PostExecEvmFactoryHooks<
            Tx: FromRecoveredTx<OpTransactionSigned>
                    + FromTxWithEncoded<OpTransactionSigned>
                    + alloy_evm::TransactionEnvMut
                    + OpTxEnv,
            Precompiles = PrecompilesMap,
            Spec = OpSpecId,
            BlockEnv = BlockEnv,
        > + Debug
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
        impl BlockExecutor<
            Transaction = <Self::Primitives as NodePrimitives>::SignedTx,
            Receipt = <Self::Primitives as NodePrimitives>::Receipt,
        > + PostExecExecutorExt
        + 'a,
        Self::Error,
    > {
        let evm = self.evm_for_block(db, block.header())?;
        let ctx = self.context_for_block_with_post_exec_mode(block, Some(post_exec_mode));

        Ok(OpBlockExecutor::new(
            evm,
            ctx,
            self.executor_factory.spec(),
            self.executor_factory.receipt_builder(),
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
            Executor: PostExecExecutorExt
                          + BlockExecutor<
                Evm: alloy_evm::Evm<DB: core::ops::DerefMut<Target = State<DB>>>,
                Result: PreRefundGasUsed,
            >,
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
        );

        Ok(BasicBlockBuilder::<
            'a,
            OpBlockExecutorFactory<
                OpRethReceiptBuilder,
                Arc<OpChainSpec>,
                PostExecEvmFactoryAdapter<F>,
            >,
            _,
            _,
            OpPrimitives,
        > {
            executor,
            transactions: Vec::new(),
            ctx,
            parent,
            assembler: self.block_assembler(),
        })
    }
}
