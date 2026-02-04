use crate::{
    evm::{
        alloy::{CustomEvm, CustomEvmFactory},
        CustomEvmConfig, CustomTxEnv,
    },
    primitives::CustomTransaction,
};
use alloy_consensus::transaction::Recovered;
use alloy_evm::{
    block::{
        BlockExecutionError, BlockExecutionResult, BlockExecutor, BlockExecutorFactory,
        BlockExecutorFor, ExecutableTx, OnStateHook,
    },
    precompiles::PrecompilesMap,
    Database, Evm, RecoveredTx,
};
use alloy_op_evm::{block::OpTxResult, OpBlockExecutionCtx, OpBlockExecutor};
use reth_ethereum::evm::primitives::InspectorFor;
use reth_op::{chainspec::OpChainSpec, node::OpRethReceiptBuilder, OpReceipt, OpTxType};
use revm::database::State;
use std::sync::Arc;

pub struct CustomBlockExecutor<Evm> {
    inner: OpBlockExecutor<Evm, OpRethReceiptBuilder, Arc<OpChainSpec>>,
}

impl<'db, DB, E> BlockExecutor for CustomBlockExecutor<E>
where
    DB: Database + 'db,
    E: Evm<DB = &'db mut State<DB>, Tx = CustomTxEnv>,
{
    type Transaction = CustomTransaction;
    type Receipt = OpReceipt;
    type Evm = E;
    type Result = OpTxResult<E::HaltReason, OpTxType>;

    fn apply_pre_execution_changes(&mut self) -> Result<(), BlockExecutionError> {
        self.inner.apply_pre_execution_changes()
    }

    fn receipts(&self) -> &[Self::Receipt] {
        self.inner.receipts()
    }

    fn execute_transaction_without_commit(
        &mut self,
        tx: impl ExecutableTx<Self>,
    ) -> Result<Self::Result, BlockExecutionError> {
        let tx = tx.into_parts().1;
        match tx.tx() {
            CustomTransaction::Op(op_tx) => self
                .inner
                .execute_transaction_without_commit(Recovered::new_unchecked(op_tx, *tx.signer())),
            CustomTransaction::Payment(..) => todo!(),
        }
    }

    fn commit_transaction(&mut self, output: Self::Result) -> Result<u64, BlockExecutionError> {
        self.inner.commit_transaction(output)
    }

    fn finish(self) -> Result<(Self::Evm, BlockExecutionResult<OpReceipt>), BlockExecutionError> {
        self.inner.finish()
    }

    fn set_state_hook(&mut self, _hook: Option<Box<dyn OnStateHook>>) {
        self.inner.set_state_hook(_hook)
    }

    fn evm_mut(&mut self) -> &mut Self::Evm {
        self.inner.evm_mut()
    }

    fn evm(&self) -> &Self::Evm {
        self.inner.evm()
    }
}

impl BlockExecutorFactory for CustomEvmConfig {
    type EvmFactory = CustomEvmFactory;
    type ExecutionCtx<'a> = CustomBlockExecutionCtx;
    type Transaction = CustomTransaction;
    type Receipt = OpReceipt;

    fn evm_factory(&self) -> &Self::EvmFactory {
        &self.custom_evm_factory
    }

    fn create_executor<'a, DB, I>(
        &'a self,
        evm: CustomEvm<&'a mut State<DB>, I, PrecompilesMap>,
        ctx: CustomBlockExecutionCtx,
    ) -> impl BlockExecutorFor<'a, Self, DB, I>
    where
        DB: Database + 'a,
        I: InspectorFor<Self, &'a mut State<DB>> + 'a,
    {
        CustomBlockExecutor {
            inner: OpBlockExecutor::new(
                evm,
                ctx.inner,
                self.inner.chain_spec().clone(),
                *self.inner.executor_factory.receipt_builder(),
            ),
        }
    }
}

/// Additional parameters for executing custom transactions.
#[derive(Debug, Clone)]
pub struct CustomBlockExecutionCtx {
    pub inner: OpBlockExecutionCtx,
    pub extension: u64,
}

impl From<CustomBlockExecutionCtx> for OpBlockExecutionCtx {
    fn from(value: CustomBlockExecutionCtx) -> Self {
        value.inner
    }
}
