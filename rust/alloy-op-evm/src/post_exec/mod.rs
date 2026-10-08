//! Post-exec execution extensions.

mod inspector;
mod null;
mod refund;

pub use null::NullRefundPolicy;
pub use refund::{PostExecRefundInspector, PostExecRefundPolicyFactory};

use alloc::vec::Vec;
use alloy_evm::{Database, Evm, EvmEnv, EvmFactory};
use alloy_primitives::Bytes;
use core::{
    marker::PhantomData,
    ops::{Deref, DerefMut},
};
use op_alloy::consensus::post_exec::SDMGasEntry;
use revm::{
    Inspector,
    context::{
        DBErrorMarker,
        result::{ExecutionResult, Output, ResultAndState, ResultGas, SuccessReason},
    },
    inspector::NoOpInspector,
    state::EvmState,
};

pub use inspector::{
    PostExecCompositeInspector, PostExecExecutedTx, PostExecRefundEvent, PostExecRefundKind,
    PostExecTxContext, PostExecTxKind,
};

use crate::block::{OpBlockExecutor, receipt_builder::OpReceiptBuilder};

/// The execution result consensus assigns to a post-exec transaction.
///
/// Post-exec transactions are structural no-ops: they consume no gas, emit no logs, and touch no
/// state. They never run through the EVM — the block executor short-circuits them — so every path
/// that would otherwise transact one must synthesize this result instead.
pub fn noop_post_exec_result<HaltReason>() -> ResultAndState<HaltReason> {
    ResultAndState::new(
        ExecutionResult::Success {
            reason: SuccessReason::Stop,
            gas: ResultGas::default(),
            logs: Vec::new(),
            output: Output::Call(Bytes::default()),
        },
        EvmState::default(),
    )
}

/// Extension trait for EVMs that can track post-exec per-transaction warming results.
pub trait PostExecEvm: alloy_evm::Evm {
    /// Opaque marker of the refund policy's block-scoped state.
    type Checkpoint;

    /// Begin post-exec tracking for the next transaction.
    fn begin_post_exec_tx(&mut self, ctx: PostExecTxContext);

    /// Take the extracted post-exec result for the most recently executed transaction.
    fn take_last_post_exec_tx_result(&mut self) -> PostExecExecutedTx;

    /// Mark the refund policy's block-scoped state before a candidate transaction.
    fn refund_checkpoint(&self) -> Self::Checkpoint;

    /// Roll the refund policy's block-scoped state back to `checkpoint`.
    fn revert_refund_checkpoint(&mut self, checkpoint: Self::Checkpoint);
}

/// Extension trait for EVM factories whose produced EVMs support post-exec tracking.
///
/// This exposes factory hooks through [`PostExecEvm`] without constraining every generic EVM
/// associated type directly.
pub trait PostExecEvmFactoryHooks: EvmFactory {
    /// Opaque marker of the block-scoped refund state of EVMs produced by this factory.
    type Checkpoint;

    /// Begin post-exec tracking for the next transaction.
    fn begin_post_exec_tx<DB, I>(evm: &mut Self::Evm<DB, I>, ctx: PostExecTxContext)
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>;

    /// Take the extracted post-exec result for the most recently executed transaction.
    fn take_last_post_exec_tx_result<DB, I>(evm: &mut Self::Evm<DB, I>) -> PostExecExecutedTx
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>;

    /// Mark the refund policy's block-scoped state before a candidate transaction.
    fn refund_checkpoint<DB, I>(evm: &Self::Evm<DB, I>) -> Self::Checkpoint
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>;

    /// Roll the refund policy's block-scoped state back to `checkpoint`.
    fn revert_refund_checkpoint<DB, I>(evm: &mut Self::Evm<DB, I>, checkpoint: Self::Checkpoint)
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>;
}

/// EVM wrapper that makes factory-provided post-exec hooks visible as a [`PostExecEvm`].
#[derive(Debug, Clone)]
pub struct PostExecEvmAdapter<E, F, DB, I> {
    inner: E,
    _factory: PhantomData<fn(F, DB, I)>,
}

impl<E, F, DB, I> PostExecEvmAdapter<E, F, DB, I> {
    /// Creates a new post-exec EVM adapter.
    pub const fn new(inner: E) -> Self {
        Self { inner, _factory: PhantomData }
    }

    /// Consumes the adapter and returns the wrapped EVM.
    pub fn into_inner(self) -> E {
        self.inner
    }
}

impl<E, F, DB, I> Deref for PostExecEvmAdapter<E, F, DB, I> {
    type Target = E;

    fn deref(&self) -> &Self::Target {
        &self.inner
    }
}

impl<E, F, DB, I> DerefMut for PostExecEvmAdapter<E, F, DB, I> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.inner
    }
}

impl<E, F, DB, I> Evm for PostExecEvmAdapter<E, F, DB, I>
where
    E: Evm,
{
    type DB = E::DB;
    type Tx = E::Tx;
    type Error = E::Error;
    type HaltReason = E::HaltReason;
    type Spec = E::Spec;
    type BlockEnv = E::BlockEnv;
    type Precompiles = E::Precompiles;
    type Inspector = E::Inspector;

    fn block(&self) -> &Self::BlockEnv {
        self.inner.block()
    }

    fn cfg_env(&self) -> &revm::context::CfgEnv<Self::Spec> {
        self.inner.cfg_env()
    }

    fn chain_id(&self) -> u64 {
        self.inner.chain_id()
    }

    fn transact_raw(
        &mut self,
        tx: Self::Tx,
    ) -> Result<revm::context_interface::result::ResultAndState<Self::HaltReason>, Self::Error>
    {
        self.inner.transact_raw(tx)
    }

    fn transact_system_call(
        &mut self,
        caller: alloy_primitives::Address,
        contract: alloy_primitives::Address,
        data: alloy_primitives::Bytes,
    ) -> Result<revm::context_interface::result::ResultAndState<Self::HaltReason>, Self::Error>
    {
        self.inner.transact_system_call(caller, contract, data)
    }

    fn finish(self) -> (Self::DB, EvmEnv<Self::Spec, Self::BlockEnv>) {
        self.inner.finish()
    }

    fn set_inspector_enabled(&mut self, enabled: bool) {
        self.inner.set_inspector_enabled(enabled);
    }

    fn components(&self) -> (&Self::DB, &Self::Inspector, &Self::Precompiles) {
        self.inner.components()
    }

    fn components_mut(&mut self) -> (&mut Self::DB, &mut Self::Inspector, &mut Self::Precompiles) {
        self.inner.components_mut()
    }
}

impl<F, DB, I> PostExecEvm for PostExecEvmAdapter<F::Evm<DB, I>, F, DB, I>
where
    F: PostExecEvmFactoryHooks,
    DB: Database,
    I: Inspector<F::Context<DB>>,
{
    type Checkpoint = F::Checkpoint;

    fn begin_post_exec_tx(&mut self, ctx: PostExecTxContext) {
        F::begin_post_exec_tx(&mut self.inner, ctx);
    }

    fn take_last_post_exec_tx_result(&mut self) -> PostExecExecutedTx {
        F::take_last_post_exec_tx_result(&mut self.inner)
    }

    fn refund_checkpoint(&self) -> Self::Checkpoint {
        F::refund_checkpoint(&self.inner)
    }

    fn revert_refund_checkpoint(&mut self, checkpoint: Self::Checkpoint) {
        F::revert_refund_checkpoint(&mut self.inner, checkpoint);
    }
}

/// EVM factory adapter that wraps produced EVMs in [`PostExecEvmAdapter`].
///
/// This is needed for generic/custom [`EvmFactory`] implementations because
/// [`BlockExecutorFactory::create_executor`](alloy_evm::block::BlockExecutorFactory::create_executor)
/// is handed an already-produced EVM value, not the factory that produced it. Without this wrapper,
/// generic executor code cannot assume that every `F::Evm<DB, I>` associated type implements
/// [`PostExecEvm`], even when `F` knows how to drive post-exec hooks for those EVMs.
#[derive(Debug, Clone, Copy)]
pub struct PostExecEvmFactoryAdapter<F> {
    inner: F,
}

impl<F> PostExecEvmFactoryAdapter<F> {
    /// Creates a new post-exec EVM factory adapter.
    pub const fn new(inner: F) -> Self {
        Self { inner }
    }

    /// Returns the wrapped EVM factory.
    pub const fn inner(&self) -> &F {
        &self.inner
    }

    /// Consumes the adapter and returns the wrapped EVM factory.
    pub fn into_inner(self) -> F {
        self.inner
    }
}

impl<F> EvmFactory for PostExecEvmFactoryAdapter<F>
where
    F: PostExecEvmFactoryHooks,
{
    type Evm<DB: Database, I: Inspector<Self::Context<DB>>> =
        PostExecEvmAdapter<F::Evm<DB, I>, F, DB, I>;
    type Context<DB: Database> = F::Context<DB>;
    type Tx = F::Tx;
    type Error<DBError: DBErrorMarker> = F::Error<DBError>;
    type HaltReason = F::HaltReason;
    type Spec = F::Spec;
    type BlockEnv = F::BlockEnv;
    type Precompiles = F::Precompiles;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec, Self::BlockEnv>,
    ) -> Self::Evm<DB, NoOpInspector> {
        PostExecEvmAdapter::new(self.inner.create_evm(db, input))
    }

    fn create_evm_with_inspector<DB: Database, I: Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec, Self::BlockEnv>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        PostExecEvmAdapter::new(self.inner.create_evm_with_inspector(db, input, inspector))
    }
}

/// Extension trait for block executors that collect post-exec payload entries.
pub trait PostExecExecutorExt {
    /// Returns the accumulated post-exec entries for the current block without clearing them.
    fn post_exec_entries(&self) -> &[SDMGasEntry];

    /// Take the accumulated post-exec entries for the current block.
    fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry>;

    /// Take the exact per-transaction policy-provided refund events aligned with receipts.
    fn take_refund_events_by_tx(&mut self) -> Vec<Vec<PostExecRefundEvent>>;
}

impl<E, R, Spec> PostExecExecutorExt for OpBlockExecutor<E, R, Spec>
where
    E: PostExecEvm,
    R: OpReceiptBuilder,
    Spec: alloy_op_hardforks::OpHardforks + Clone,
{
    fn post_exec_entries(&self) -> &[SDMGasEntry] {
        Self::post_exec_entries(self)
    }

    fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry> {
        Self::take_post_exec_entries(self)
    }

    fn take_refund_events_by_tx(&mut self) -> Vec<Vec<PostExecRefundEvent>> {
        Self::take_refund_events_by_tx(self)
    }
}
