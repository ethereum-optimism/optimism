use crate::evm::{CustomTxEnv, PaymentTxEnv};
use alloy_evm::{precompiles::PrecompilesMap, Database, Evm, EvmEnv, EvmFactory};
use alloy_op_evm::{OpEvm, OpEvmFactory};
use alloy_primitives::{Address, Bytes};
use op_revm::{
    precompiles::OpPrecompiles, L1BlockInfo, OpContext, OpHaltReason, OpSpecId, OpTransaction,
    OpTransactionError,
};
use reth_ethereum::evm::revm::{
    context::{result::ResultAndState, BlockEnv, CfgEnv},
    handler::PrecompileProvider,
    interpreter::InterpreterResult,
    Context, Inspector, Journal,
};
use revm::{context_interface::result::EVMError, inspector::NoOpInspector};
use std::error::Error;

/// EVM context contains data that EVM needs for execution of [`CustomTxEnv`].
pub type CustomContext<DB> =
    Context<BlockEnv, OpTransaction<PaymentTxEnv>, CfgEnv<OpSpecId>, DB, Journal<DB>, L1BlockInfo>;

pub struct CustomEvm<DB: Database, I, P = OpPrecompiles> {
    inner: OpEvm<DB, I, P>,
}

impl<DB: Database, I, P> CustomEvm<DB, I, P> {
    pub fn new(op: OpEvm<DB, I, P>) -> Self {
        Self { inner: op }
    }
}

impl<DB, I, P> Evm for CustomEvm<DB, I, P>
where
    DB: Database,
    I: Inspector<OpContext<DB>>,
    P: PrecompileProvider<OpContext<DB>, Output = InterpreterResult>,
{
    type DB = DB;
    type Tx = CustomTxEnv;
    type Error = EVMError<DB::Error, OpTransactionError>;
    type HaltReason = OpHaltReason;
    type Spec = OpSpecId;
    type BlockEnv = BlockEnv;
    type Precompiles = P;
    type Inspector = I;

    fn block(&self) -> &BlockEnv {
        self.inner.block()
    }

    fn chain_id(&self) -> u64 {
        self.inner.chain_id()
    }

    fn transact_raw(
        &mut self,
        tx: Self::Tx,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        match tx {
            CustomTxEnv::Op(tx) => self.inner.transact_raw(tx),
            CustomTxEnv::Payment(..) => todo!(),
        }
    }

    fn transact_system_call(
        &mut self,
        caller: Address,
        contract: Address,
        data: Bytes,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        self.inner.transact_system_call(caller, contract, data)
    }

    fn finish(self) -> (Self::DB, EvmEnv<Self::Spec>) {
        self.inner.finish()
    }

    fn set_inspector_enabled(&mut self, enabled: bool) {
        self.inner.set_inspector_enabled(enabled)
    }

    fn components(&self) -> (&Self::DB, &Self::Inspector, &Self::Precompiles) {
        self.inner.components()
    }

    fn components_mut(&mut self) -> (&mut Self::DB, &mut Self::Inspector, &mut Self::Precompiles) {
        self.inner.components_mut()
    }
}

#[derive(Default, Debug, Clone, Copy)]
pub struct CustomEvmFactory(pub OpEvmFactory);

impl CustomEvmFactory {
    pub fn new() -> Self {
        Self::default()
    }
}

impl EvmFactory for CustomEvmFactory {
    type Evm<DB: Database, I: Inspector<OpContext<DB>>> = CustomEvm<DB, I, Self::Precompiles>;
    type Context<DB: Database> = OpContext<DB>;
    type Tx = CustomTxEnv;
    type Error<DBError: Error + Send + Sync + 'static> = EVMError<DBError, OpTransactionError>;
    type HaltReason = OpHaltReason;
    type Spec = OpSpecId;
    type BlockEnv = BlockEnv;
    type Precompiles = PrecompilesMap;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec>,
    ) -> Self::Evm<DB, NoOpInspector> {
        CustomEvm::new(self.0.create_evm(db, input))
    }

    fn create_evm_with_inspector<DB: Database, I: Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        CustomEvm::new(self.0.create_evm_with_inspector(db, input, inspector))
    }
}
