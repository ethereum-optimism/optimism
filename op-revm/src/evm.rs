//! Contains the `[OpEvm]` type and its implementation of the execution EVM traits.
use crate::{precompiles::OpPrecompiles, OpSpecId};
use revm::{
    context::{Cfg, ContextError, ContextSetters, Evm, FrameStack},
    context_interface::ContextTr,
    handler::{
        evm::FrameTr,
        instructions::{EthInstructions, InstructionProvider},
        EthFrame, EvmTr, FrameInitOrResult, ItemOrResult, PrecompileProvider,
    },
    inspector::{InspectorEvmTr, JournalExt},
    interpreter::{interpreter::EthInterpreter, InterpreterResult},
    Database, Inspector,
};

/// Optimism EVM extends the [`Evm`] type with Optimism specific types and logic.
#[derive(Debug, Clone)]
pub struct OpEvm<
    CTX,
    INSP,
    I = EthInstructions<EthInterpreter, CTX>,
    P = OpPrecompiles,
    F = EthFrame<EthInterpreter>,
>(
    /// Inner EVM type.
    pub Evm<CTX, INSP, I, P, F>,
);

impl<CTX: ContextTr<Cfg: Cfg<Spec: Into<OpSpecId> + Clone>>, INSP>
    OpEvm<CTX, INSP, EthInstructions<EthInterpreter, CTX>, OpPrecompiles>
{
    /// Create a new Optimism EVM.
    pub fn new(ctx: CTX, inspector: INSP) -> Self {
        let spec: OpSpecId = ctx.cfg().spec().into();
        Self(Evm {
            ctx,
            inspector,
            instruction: EthInstructions::new_mainnet_with_spec(spec.into()),
            precompiles: OpPrecompiles::new_with_spec(spec),
            frame_stack: FrameStack::new_prealloc(8),
        })
    }

    /// Consumes self and returns the inner context.
    pub fn into_context(self) -> CTX {
        self.0.ctx
    }
}

impl<CTX, INSP, I, P> OpEvm<CTX, INSP, I, P> {
    /// Consumed self and returns a new Evm type with given Inspector.
    pub fn with_inspector<OINSP>(self, inspector: OINSP) -> OpEvm<CTX, OINSP, I, P> {
        OpEvm(self.0.with_inspector(inspector))
    }

    /// Consumes self and returns a new Evm type with given Precompiles.
    pub fn with_precompiles<OP>(self, precompiles: OP) -> OpEvm<CTX, INSP, I, OP> {
        OpEvm(self.0.with_precompiles(precompiles))
    }

    /// Consumes self and returns the inner Inspector.
    pub fn into_inspector(self) -> INSP {
        self.0.into_inspector()
    }
}

impl<CTX, INSP, I, P> InspectorEvmTr for OpEvm<CTX, INSP, I, P>
where
    CTX: ContextTr<Journal: JournalExt> + ContextSetters,
    I: InstructionProvider<Context = CTX, InterpreterTypes = EthInterpreter>,
    P: PrecompileProvider<CTX, Output = InterpreterResult>,
    INSP: Inspector<CTX, I::InterpreterTypes>,
{
    type Inspector = INSP;

    #[inline]
    fn all_inspector(
        &self,
    ) -> (
        &Self::Context,
        &Self::Instructions,
        &Self::Precompiles,
        &FrameStack<Self::Frame>,
        &Self::Inspector,
    ) {
        self.0.all_inspector()
    }

    #[inline]
    fn all_mut_inspector(
        &mut self,
    ) -> (
        &mut Self::Context,
        &mut Self::Instructions,
        &mut Self::Precompiles,
        &mut FrameStack<Self::Frame>,
        &mut Self::Inspector,
    ) {
        self.0.all_mut_inspector()
    }
}

impl<CTX, INSP, I, P> EvmTr for OpEvm<CTX, INSP, I, P, EthFrame<EthInterpreter>>
where
    CTX: ContextTr,
    I: InstructionProvider<Context = CTX, InterpreterTypes = EthInterpreter>,
    P: PrecompileProvider<CTX, Output = InterpreterResult>,
{
    type Context = CTX;
    type Instructions = I;
    type Precompiles = P;
    type Frame = EthFrame<EthInterpreter>;

    #[inline]
    fn all(
        &self,
    ) -> (
        &Self::Context,
        &Self::Instructions,
        &Self::Precompiles,
        &FrameStack<Self::Frame>,
    ) {
        self.0.all()
    }

    #[inline]
    fn all_mut(
        &mut self,
    ) -> (
        &mut Self::Context,
        &mut Self::Instructions,
        &mut Self::Precompiles,
        &mut FrameStack<Self::Frame>,
    ) {
        self.0.all_mut()
    }

    fn frame_init(
        &mut self,
        frame_input: <Self::Frame as FrameTr>::FrameInit,
    ) -> Result<
        ItemOrResult<&mut Self::Frame, <Self::Frame as FrameTr>::FrameResult>,
        ContextError<<<Self::Context as ContextTr>::Db as Database>::Error>,
    > {
        self.0.frame_init(frame_input)
    }

    fn frame_run(
        &mut self,
    ) -> Result<
        FrameInitOrResult<Self::Frame>,
        ContextError<<<Self::Context as ContextTr>::Db as Database>::Error>,
    > {
        self.0.frame_run()
    }

    #[doc = " Returns the result of the frame to the caller. Frame is popped from the frame stack."]
    #[doc = " Consumes the frame result or returns it if there is more frames to run."]
    fn frame_return_result(
        &mut self,
        result: <Self::Frame as FrameTr>::FrameResult,
    ) -> Result<
        Option<<Self::Frame as FrameTr>::FrameResult>,
        ContextError<<<Self::Context as ContextTr>::Db as Database>::Error>,
    > {
        self.0.frame_return_result(result)
    }
}
