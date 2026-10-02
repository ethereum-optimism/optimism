//! Post-exec execution extensions.

mod inspector;
mod null;
mod parallel;
mod refund;
pub use parallel::{
    ObservedRefundPolicy, ParallelObservation, ParallelRefundPolicy, TransactionObserver,
};

pub use null::NullRefundPolicy;
pub use refund::PostExecRefundInspector;

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
    /// Opaque block-scoped refund state.
    type Snapshot: Clone;

    /// Whether this EVM's producer policy can observe transactions independently.
    fn supports_parallel_observation(&self) -> bool {
        false
    }

    /// Captures the policy state staged for the next commit without publishing it.
    fn prepared_refund_snapshot(&self) -> Self::Snapshot {
        self.refund_snapshot()
    }

    /// Compares the currently prepared policy state with a shadow attempt's prepared state.
    fn matches_prepared_refund_snapshot(&self, _snapshot: &Self::Snapshot) -> bool {
        false
    }

    /// Enables isolated speculation with deferred protocol fees. Returns false when unsupported.
    fn set_parallel_execution(&mut self, _enabled: bool) -> bool {
        false
    }

    /// Takes the last transaction's ordered protocol fee operations.
    fn take_deferred_fees(&mut self) -> Vec<op_revm::handler::DeferredFeeCredit> {
        Vec::new()
    }

    /// Takes the last speculative transaction's policy observations.
    fn take_parallel_observation(&mut self) -> Option<ParallelObservation> {
        None
    }

    /// Evaluates validated observations against canonical policy state.
    fn evaluate_parallel_observation(
        &mut self,
        _context: PostExecTxContext,
        _observation: &ParallelObservation,
    ) -> Option<PostExecExecutedTx> {
        None
    }

    /// Snapshot the EVM's block-scoped L1 fee context for isolated execution.
    fn execution_l1_block_info(&self) -> Option<op_revm::L1BlockInfo> {
        None
    }

    /// Carry forward the context produced by a dependency-validated execution attempt.
    fn seed_execution_l1_block_info(&mut self, _info: op_revm::L1BlockInfo) {}

    /// Current execution environment. Workers must use it rather than a cached factory default.
    fn parallel_environment(&self) -> Option<alloy_evm::EvmEnv<op_revm::OpSpecId>> {
        None
    }

    /// Checks that worker warmth and called precompile implementations match the canonical EVM.
    /// Stock maps are known at construction; customized maps require canonical execution for
    /// precompile calls because implementations may be thread-affine or have different behavior.
    fn parallel_precompiles_compatible(
        &self,
        _warm: &[alloy_primitives::Address],
        _calls: &[alloy_primitives::Address],
    ) -> bool {
        false
    }

    /// Publishes a prepared policy update after canonical transaction acceptance.
    fn commit_post_exec_tx(&mut self) {}

    /// Begin post-exec tracking for the next transaction.
    fn begin_post_exec_tx(&mut self, ctx: PostExecTxContext);

    /// Take the extracted post-exec result for the most recently executed transaction.
    fn take_last_post_exec_tx_result(&mut self) -> PostExecExecutedTx;

    /// Snapshot refund state to carry across subblock executors.
    fn refund_snapshot(&self) -> Self::Snapshot;

    /// Seed refund state captured from a prior subblock.
    fn seed_refund_snapshot(&mut self, state: Self::Snapshot);
}

/// Extension trait for EVM factories whose produced EVMs support post-exec tracking.
///
/// This exposes factory hooks through [`PostExecEvm`] without constraining every generic EVM
/// associated type directly.
pub trait PostExecEvmFactoryHooks: EvmFactory {
    /// Opaque block-scoped refund state produced by this factory.
    type Snapshot: Clone;

    /// Publishes a prepared policy update, when the factory supports staged decisions.
    fn commit_post_exec_tx<DB, I>(_evm: &mut Self::Evm<DB, I>)
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
    }

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

    /// Snapshot refund state to carry across subblock executors.
    fn refund_snapshot<DB, I>(evm: &Self::Evm<DB, I>) -> Self::Snapshot
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>;

    /// Seed refund state captured from a prior subblock.
    fn seed_refund_snapshot<DB, I>(evm: &mut Self::Evm<DB, I>, state: Self::Snapshot)
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
    type Snapshot = F::Snapshot;

    fn commit_post_exec_tx(&mut self) {
        F::commit_post_exec_tx(&mut self.inner);
    }

    fn begin_post_exec_tx(&mut self, ctx: PostExecTxContext) {
        F::begin_post_exec_tx(&mut self.inner, ctx);
    }

    fn take_last_post_exec_tx_result(&mut self) -> PostExecExecutedTx {
        F::take_last_post_exec_tx_result(&mut self.inner)
    }

    fn refund_snapshot(&self) -> Self::Snapshot {
        F::refund_snapshot(&self.inner)
    }

    fn seed_refund_snapshot(&mut self, state: Self::Snapshot) {
        F::seed_refund_snapshot(&mut self.inner, state);
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
    /// Opaque block-scoped refund state.
    type Snapshot: Clone;

    /// Supplies non-authoritative candidate previews without advancing a transaction selector.
    fn set_parallel_candidates(&mut self, _candidates: Vec<crate::block::ParallelCandidate>) {}

    /// Retires a skipped candidate and, when requested, all invalid sender descendants.
    fn reject_parallel_candidate(
        &mut self,
        _hash: alloy_primitives::B256,
        _descendants: Option<(alloy_primitives::Address, u64)>,
    ) {
    }

    /// Drains a subblock or selection phase while retaining the execution configuration.
    fn drain_parallel_work(&mut self) {}

    /// Invalidates all speculative output when a build is cancelled or its parent changes.
    fn invalidate_parallel_work(&mut self) {}

    /// Returns the accumulated post-exec entries for the current block without clearing them.
    fn post_exec_entries(&self) -> &[SDMGasEntry];

    /// Take the accumulated post-exec entries for the current block.
    fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry>;

    /// Take the exact per-transaction policy-provided refund events aligned with receipts.
    fn take_refund_events_by_tx(&mut self) -> Vec<Vec<PostExecRefundEvent>>;

    /// Snapshot refund state to carry across subblock executors.
    fn refund_snapshot(&self) -> Self::Snapshot;

    /// Seed refund state captured from a prior subblock.
    fn seed_refund_snapshot(&mut self, state: Self::Snapshot);
}

impl<E, R, Spec> PostExecExecutorExt for OpBlockExecutor<E, R, Spec>
where
    E: PostExecEvm,
    R: OpReceiptBuilder,
    Spec: alloy_op_hardforks::OpHardforks + Clone,
{
    type Snapshot = E::Snapshot;

    fn drain_parallel_work(&mut self) {
        #[cfg(feature = "parallel")]
        if let Some(parallel) = &mut self.parallel {
            parallel.clear();
        }
    }

    fn invalidate_parallel_work(&mut self) {
        #[cfg(feature = "parallel")]
        {
            self.parallel = None;
        }
    }

    fn reject_parallel_candidate(
        &mut self,
        _hash: alloy_primitives::B256,
        _descendants: Option<(alloy_primitives::Address, u64)>,
    ) {
        #[cfg(feature = "parallel")]
        if let Some(parallel) = &mut self.parallel {
            parallel.rejected(_hash, _descendants);
        }
    }

    fn set_parallel_candidates(&mut self, candidates: Vec<crate::block::ParallelCandidate>) {
        #[cfg(feature = "parallel")]
        if let Some(parallel) = self.parallel.as_mut() {
            parallel.candidates_changed(&candidates);
        }
        self.ctx.parallel_candidates = candidates;
    }

    fn post_exec_entries(&self) -> &[SDMGasEntry] {
        Self::post_exec_entries(self)
    }

    fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry> {
        Self::take_post_exec_entries(self)
    }

    fn take_refund_events_by_tx(&mut self) -> Vec<Vec<PostExecRefundEvent>> {
        Self::take_refund_events_by_tx(self)
    }

    fn refund_snapshot(&self) -> Self::Snapshot {
        Self::refund_snapshot(self)
    }

    fn seed_refund_snapshot(&mut self, state: Self::Snapshot) {
        Self::seed_refund_snapshot(self, state);
    }
}
