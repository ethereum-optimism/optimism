//! Optimism builder trait [`OpBuilder`] used to build [`OpEvm`].
use crate::{evm::OpEvm, precompiles::OpPrecompiles, transaction::OpTxTr, L1BlockInfo, OpSpecId};
use revm::{
    context::Cfg,
    context_interface::{Block, JournalTr},
    handler::instructions::EthInstructions,
    interpreter::interpreter::EthInterpreter,
    state::EvmState,
    Context, Database,
};

/// Type alias for default OpEvm
pub type DefaultOpEvm<CTX, INSP = ()> =
    OpEvm<CTX, INSP, EthInstructions<EthInterpreter, CTX>, OpPrecompiles>;

/// Trait that allows for optimism OpEvm to be built.
pub trait OpBuilder: Sized {
    /// Type of the context.
    type Context;

    /// Build the op.
    fn build_op(self) -> DefaultOpEvm<Self::Context>;

    /// Build the op with an inspector.
    fn build_op_with_inspector<INSP>(self, inspector: INSP) -> DefaultOpEvm<Self::Context, INSP>;
}

impl<BLOCK, TX, CFG, DB, JOURNAL> OpBuilder for Context<BLOCK, TX, CFG, DB, JOURNAL, L1BlockInfo>
where
    BLOCK: Block,
    TX: OpTxTr,
    CFG: Cfg<Spec = OpSpecId>,
    DB: Database,
    JOURNAL: JournalTr<Database = DB, State = EvmState>,
{
    type Context = Self;

    fn build_op(self) -> DefaultOpEvm<Self::Context> {
        OpEvm::new(self, ())
    }

    fn build_op_with_inspector<INSP>(self, inspector: INSP) -> DefaultOpEvm<Self::Context, INSP> {
        OpEvm::new(self, inspector)
    }
}
