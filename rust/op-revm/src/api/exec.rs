//! Implementation of the [`ExecuteEvm`] trait for the [`OpEvm`].
//!
//! Every upstream API impl below must also be checked for newly defaulted trait methods.
use crate::{
    L1BlockInfo, OpHaltReason, OpSpecId, OpTransactionError, evm::OpEvm, handler::OpHandler,
    transaction::OpTxTr,
};
use revm::{
    DatabaseCommit, ExecuteCommitEvm, ExecuteEvm,
    context::{ContextSetters, result::ExecResultAndState},
    context_interface::{
        Cfg, ContextTr, Database, JournalTr,
        result::{EVMError, ExecutionResult},
    },
    handler::{
        EthFrame, Handler, PrecompileProvider, SystemCallTx, instructions::EthInstructions,
        system_call::SystemCallEvm,
    },
    inspector::{
        InspectCommitEvm, InspectEvm, InspectSystemCallEvm, Inspector, InspectorHandler, JournalExt,
    },
    interpreter::{InterpreterResult, interpreter::EthInterpreter},
    primitives::{Address, Bytes},
    state::EvmState,
};

/// Type alias for Optimism context
pub trait OpContextTr:
    ContextTr<
        Journal: JournalTr<State = EvmState>,
        Tx: OpTxTr,
        Cfg: Cfg<Spec = OpSpecId>,
        Chain = L1BlockInfo,
    >
{
}

impl<T> OpContextTr for T where
    T: ContextTr<
            Journal: JournalTr<State = EvmState>,
            Tx: OpTxTr,
            Cfg: Cfg<Spec = OpSpecId>,
            Chain = L1BlockInfo,
        >
{
}

/// Type alias for the error type of the `OpEvm`.
pub type OpError<CTX> = EVMError<<<CTX as ContextTr>::Db as Database>::Error, OpTransactionError>;

/// UPSTREAM-MIRROR(copy): revm-handler@41.0.0 `revm_handler::api::ExecuteEvm`
///
/// Copies upstream's `ExecuteEvm` implementation with `OpHandler` substituted for
/// `MainnetHandler`. Re-diff the bodies when the upstream implementation changes.
impl<CTX, INSP, PRECOMPILE> ExecuteEvm
    for OpEvm<CTX, INSP, EthInstructions<EthInterpreter, CTX>, PRECOMPILE>
where
    CTX: OpContextTr + ContextSetters,
    PRECOMPILE: PrecompileProvider<CTX, Output = InterpreterResult>,
{
    type Tx = <CTX as ContextTr>::Tx;
    type Block = <CTX as ContextTr>::Block;
    type State = EvmState;
    type Error = OpError<CTX>;
    type ExecutionResult = ExecutionResult<OpHaltReason>;

    fn set_block(&mut self, block: Self::Block) {
        self.0.ctx.set_block(block);
    }

    fn transact_one(&mut self, tx: Self::Tx) -> Result<Self::ExecutionResult, Self::Error> {
        self.0.ctx.set_tx(tx);
        let mut h = OpHandler::<_, _, EthFrame<EthInterpreter>>::new();
        h.run(self)
    }

    fn finalize(&mut self) -> Self::State {
        self.0.ctx.journal_mut().finalize()
    }

    fn replay(
        &mut self,
    ) -> Result<ExecResultAndState<Self::ExecutionResult, Self::State>, Self::Error> {
        let mut h = OpHandler::<_, _, EthFrame<EthInterpreter>>::new();
        h.run(self).map(|result| {
            let state = self.finalize();
            ExecResultAndState::new(result, state)
        })
    }
}

/// UPSTREAM-MIRROR(copy): revm-handler@41.0.0 `revm_handler::api::ExecuteCommitEvm`
///
/// Copies the upstream commit implementation for the OP EVM.
impl<CTX, INSP, PRECOMPILE> ExecuteCommitEvm
    for OpEvm<CTX, INSP, EthInstructions<EthInterpreter, CTX>, PRECOMPILE>
where
    CTX: OpContextTr<Db: DatabaseCommit> + ContextSetters,
    PRECOMPILE: PrecompileProvider<CTX, Output = InterpreterResult>,
{
    fn commit(&mut self, state: Self::State) {
        self.0.ctx.db_mut().commit(state);
    }
}

/// UPSTREAM-MIRROR(copy): revm-inspector@41.0.0 `revm_inspector::mainnet_inspect::InspectEvm`
///
/// Copies the upstream inspector implementation with `OpHandler`.
impl<CTX, INSP, PRECOMPILE> InspectEvm
    for OpEvm<CTX, INSP, EthInstructions<EthInterpreter, CTX>, PRECOMPILE>
where
    CTX: OpContextTr<Journal: JournalExt> + ContextSetters,
    INSP: Inspector<CTX, EthInterpreter>,
    PRECOMPILE: PrecompileProvider<CTX, Output = InterpreterResult>,
{
    type Inspector = INSP;

    fn set_inspector(&mut self, inspector: Self::Inspector) {
        self.0.inspector = inspector;
    }

    fn inspect_one_tx(&mut self, tx: Self::Tx) -> Result<Self::ExecutionResult, Self::Error> {
        self.0.ctx.set_tx(tx);
        let mut h = OpHandler::<_, _, EthFrame<EthInterpreter>>::new();
        h.inspect_run(self)
    }
}

/// UPSTREAM-MIRROR(copy): revm-inspector@41.0.0 `revm_inspector::mainnet_inspect::InspectCommitEvm`
///
/// Mirrors the upstream marker implementation for inspector commits.
impl<CTX, INSP, PRECOMPILE> InspectCommitEvm
    for OpEvm<CTX, INSP, EthInstructions<EthInterpreter, CTX>, PRECOMPILE>
where
    CTX: OpContextTr<Journal: JournalExt, Db: DatabaseCommit> + ContextSetters,
    INSP: Inspector<CTX, EthInterpreter>,
    PRECOMPILE: PrecompileProvider<CTX, Output = InterpreterResult>,
{
}

/// UPSTREAM-MIRROR(copy): revm-handler@41.0.0 `revm_handler::system_call::SystemCallEvm`
///
/// Copies upstream system-call setup with `OpHandler`.
impl<CTX, INSP, PRECOMPILE> SystemCallEvm
    for OpEvm<CTX, INSP, EthInstructions<EthInterpreter, CTX>, PRECOMPILE>
where
    CTX: OpContextTr<Tx: SystemCallTx> + ContextSetters,
    PRECOMPILE: PrecompileProvider<CTX, Output = InterpreterResult>,
{
    fn system_call_one_with_caller(
        &mut self,
        caller: Address,
        system_contract_address: Address,
        data: Bytes,
    ) -> Result<Self::ExecutionResult, Self::Error> {
        self.0.ctx.set_tx(CTX::Tx::new_system_tx_with_caller(
            caller,
            system_contract_address,
            data,
        ));
        let mut h = OpHandler::<_, _, EthFrame<EthInterpreter>>::new();
        h.run_system_call(self)
    }
}

/// UPSTREAM-MIRROR(copy): revm-inspector@41.0.0
/// `revm_inspector::mainnet_inspect::InspectSystemCallEvm`
///
/// Copies upstream inspected system-call setup with `OpHandler`.
impl<CTX, INSP, PRECOMPILE> InspectSystemCallEvm
    for OpEvm<CTX, INSP, EthInstructions<EthInterpreter, CTX>, PRECOMPILE>
where
    CTX: OpContextTr<Journal: JournalExt, Tx: SystemCallTx> + ContextSetters,
    INSP: Inspector<CTX, EthInterpreter>,
    PRECOMPILE: PrecompileProvider<CTX, Output = InterpreterResult>,
{
    fn inspect_one_system_call_with_caller(
        &mut self,
        caller: Address,
        system_contract_address: Address,
        data: Bytes,
    ) -> Result<Self::ExecutionResult, Self::Error> {
        self.0.ctx.set_tx(CTX::Tx::new_system_tx_with_caller(
            caller,
            system_contract_address,
            data,
        ));
        let mut h = OpHandler::<_, _, EthFrame<EthInterpreter>>::new();
        h.inspect_run_system_call(self)
    }
}
