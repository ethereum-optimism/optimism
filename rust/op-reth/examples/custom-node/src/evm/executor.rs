use crate::{
    evm::{
        CustomEvmConfig, CustomTxEnv,
        alloy::{CustomEvm, CustomEvmFactory},
    },
    primitives::CustomTransaction,
};
use alloy_consensus::transaction::Recovered;
use alloy_evm::{
    RecoveredTx,
    block::{
        BlockExecutionError, BlockExecutionResult, BlockExecutor, BlockExecutorFactory,
        ExecutableTx, GasOutput, StateDB,
    },
    precompiles::PrecompilesMap,
};
use alloy_op_evm::{
    OpBlockExecutionCtx, OpBlockExecutor, OpEvmContext,
    block::OpTxResult,
    post_exec::{PostExecEvm, PostExecExecutorExt, WarmingRefundEvent, WarmingState},
};
use op_alloy_consensus::SDMGasEntry;
use reth_op::{OpReceipt, OpTxType, chainspec::OpChainSpec, node::OpRethReceiptBuilder};
use revm::Inspector;
use std::sync::Arc;

pub struct CustomBlockExecutor<Evm> {
    inner: OpBlockExecutor<Evm, OpRethReceiptBuilder, Arc<OpChainSpec>>,
}

impl<Evm> CustomBlockExecutor<Evm> {
    pub const fn new(inner: OpBlockExecutor<Evm, OpRethReceiptBuilder, Arc<OpChainSpec>>) -> Self {
        Self { inner }
    }
}

impl<DB, E> BlockExecutor for CustomBlockExecutor<E>
where
    DB: StateDB,
    E: PostExecEvm<DB = DB, Tx = CustomTxEnv>,
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

    fn commit_transaction(&mut self, output: Self::Result) -> GasOutput {
        self.inner.commit_transaction(output)
    }

    fn finish(self) -> Result<(Self::Evm, BlockExecutionResult<OpReceipt>), BlockExecutionError> {
        self.inner.finish()
    }

    fn evm_mut(&mut self) -> &mut Self::Evm {
        self.inner.evm_mut()
    }

    fn evm(&self) -> &Self::Evm {
        self.inner.evm()
    }
}

impl<E> PostExecExecutorExt for CustomBlockExecutor<E>
where
    E: PostExecEvm,
{
    fn post_exec_entries(&self) -> &[SDMGasEntry] {
        self.inner.post_exec_entries()
    }

    fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry> {
        self.inner.take_post_exec_entries()
    }

    fn take_warming_events_by_tx(&mut self) -> Vec<Vec<WarmingRefundEvent>> {
        self.inner.take_warming_events_by_tx()
    }

    fn warming_state(&self) -> WarmingState {
        self.inner.warming_state()
    }

    fn seed_warming_state(&mut self, state: WarmingState) {
        self.inner.seed_warming_state(state);
    }
}

impl BlockExecutorFactory for CustomEvmConfig {
    type EvmFactory = CustomEvmFactory;
    type ExecutionCtx<'a> = CustomBlockExecutionCtx;
    type Transaction = CustomTransaction;
    type Receipt = OpReceipt;
    type TxExecutionResult =
        OpTxResult<<CustomEvmFactory as alloy_evm::EvmFactory>::HaltReason, OpTxType>;
    type Executor<'a, DB: StateDB, I: Inspector<OpEvmContext<DB>>> =
        CustomBlockExecutor<CustomEvm<DB, I, PrecompilesMap>>;

    fn evm_factory(&self) -> &Self::EvmFactory {
        &self.custom_evm_factory
    }

    fn create_executor<'a, DB, I>(
        &'a self,
        evm: CustomEvm<DB, I, PrecompilesMap>,
        ctx: CustomBlockExecutionCtx,
    ) -> Self::Executor<'a, DB, I>
    where
        DB: StateDB,
        I: Inspector<OpEvmContext<DB>>,
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
