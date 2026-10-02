//! Transaction-local observation and ordered evaluation for parallel refund policies.

use super::{PostExecExecutedTx, PostExecRefundInspector, PostExecTxContext};
use alloc::sync::Arc;
use alloy_primitives::{Address, U256};
use core::{any::Any, fmt, marker::PhantomData};
use revm::{
    context_interface::ContextTr,
    inspector::JournalExt,
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
};

/// Type-erased, immutable observations carried from an EVM worker to its coordinator.
///
/// Erasure keeps the existing inspector and EVM factory interfaces compatible with opaque,
/// sequential policies. [`ObservedRefundPolicy`] performs the checked downcast at evaluation.
#[derive(Clone)]
pub struct ParallelObservation {
    value: Arc<dyn Any + Send + Sync>,
    bytes: usize,
}

impl ParallelObservation {
    /// Wraps observations and their retained heap/inline size for resource accounting.
    pub fn new<T: Any + Send + Sync>(value: T, bytes: usize) -> Self {
        Self { value: Arc::new(value), bytes }
    }

    /// Returns the retained size reported by the observing policy.
    pub const fn size_bytes(&self) -> usize {
        self.bytes
    }

    /// Borrows observations of the expected policy type.
    pub fn downcast_ref<T: Any>(&self) -> Option<&T> {
        self.value.downcast_ref()
    }
}

impl fmt::Debug for ParallelObservation {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("ParallelObservation").field("bytes", &self.bytes).finish_non_exhaustive()
    }
}

/// An observer whose output depends only on the current transaction's execution.
///
/// All cross-transaction state belongs to [`ParallelRefundPolicy::State`]. Implementations must
/// bound their own collection, include reads made through the EVM database, and reset on every
/// `begin_tx`. The provisional transaction index is not available during parallel observation;
/// index-dependent decisions belong in the ordered evaluator.
pub trait TransactionObserver: PostExecRefundInspector<Snapshot = ()> + Default {
    /// Facts sufficient to reproduce the sequential policy's decision.
    type Observation: Send + Sync + 'static;

    /// Takes the completed observation after `finish_tx`, along with its retained size in bytes.
    fn take_observation(&mut self) -> (Self::Observation, usize);
}

/// A producer policy that can evaluate transaction-local observations in canonical block order.
pub trait ParallelRefundPolicy: 'static {
    /// EVM observer. It must not consult mutable state outside the current EVM.
    type Observer: TransactionObserver;
    /// Committed policy state, also used for rollback and subblock snapshots.
    type State: Default + Clone + PartialEq;

    /// Prepares a decision without mutating committed policy state or the EVM.
    fn evaluate(
        state: &Self::State,
        context: PostExecTxContext,
        observation: &<Self::Observer as TransactionObserver>::Observation,
    ) -> (PostExecExecutedTx, Self::State);
}

/// Adapts an observation/evaluation policy to the existing SDM inspector interface.
///
/// Sequential execution and parallel commit use the same evaluator. Prepared updates stay private
/// until `commit_tx`; rejected candidates and restored snapshots discard the pending update.
pub struct ObservedRefundPolicy<P: ParallelRefundPolicy> {
    observer: P::Observer,
    state: P::State,
    pending: Option<P::State>,
    context: Option<PostExecTxContext>,
    observation: Option<ParallelObservation>,
    observation_only: bool,
    _policy: PhantomData<P>,
}

impl<P: ParallelRefundPolicy> Default for ObservedRefundPolicy<P> {
    fn default() -> Self {
        Self {
            observer: Default::default(),
            state: Default::default(),
            pending: None,
            context: None,
            observation: None,
            observation_only: false,
            _policy: PhantomData,
        }
    }
}

impl<P: ParallelRefundPolicy> fmt::Debug for ObservedRefundPolicy<P> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("ObservedRefundPolicy")
            .field("observation_only", &self.observation_only)
            .finish_non_exhaustive()
    }
}

impl<P: ParallelRefundPolicy> ObservedRefundPolicy<P> {
    fn evaluate(
        state: &P::State,
        context: PostExecTxContext,
        observation: &<P::Observer as TransactionObserver>::Observation,
    ) -> (PostExecExecutedTx, P::State) {
        #[cfg(feature = "metrics")]
        let started = std::time::Instant::now();
        let result = P::evaluate(state, context, observation);
        #[cfg(feature = "metrics")]
        metrics::histogram!("optimism_parallel.policy_evaluation_seconds")
            .record(started.elapsed().as_secs_f64());
        result
    }
}

impl<P: ParallelRefundPolicy> PostExecRefundInspector for ObservedRefundPolicy<P> {
    type Snapshot = P::State;

    fn begin_tx(&mut self, ctx: PostExecTxContext) {
        self.pending = None;
        self.context = Some(ctx);
        self.observation = None;
        // Observation must not depend on an index that can change when candidates are skipped.
        self.observer.begin_tx(PostExecTxContext { tx_index: 0, kind: ctx.kind });
    }

    fn note_account_touch(&mut self, address: Address) {
        self.observer.note_account_touch(address);
    }

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        let _ = self.observer.finish_tx();
        let (observation, bytes) = self.observer.take_observation();
        let context = self.context.take().expect("finish_tx follows begin_tx");
        if self.observation_only {
            self.observation = Some(ParallelObservation::new(observation, bytes));
            PostExecExecutedTx::default()
        } else {
            let (result, state) = Self::evaluate(&self.state, context, &observation);
            self.pending = Some(state);
            result
        }
    }

    fn inspect_step<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        interp: &mut Interpreter,
        context: &mut CTX,
    ) {
        self.observer.inspect_step(interp, context);
    }
    fn inspect_call<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        context: &mut CTX,
        inputs: &mut CallInputs,
    ) {
        self.observer.inspect_call(context, inputs);
    }
    fn inspect_call_end<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        context: &mut CTX,
        inputs: &CallInputs,
        outcome: &CallOutcome,
    ) {
        self.observer.inspect_call_end(context, inputs, outcome);
    }
    fn inspect_create<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        context: &mut CTX,
        inputs: &mut CreateInputs,
    ) {
        self.observer.inspect_create(context, inputs);
    }
    fn inspect_create_end<CTX: ContextTr<Journal: JournalExt>>(
        &mut self,
        context: &mut CTX,
        inputs: &CreateInputs,
        outcome: &CreateOutcome,
    ) {
        self.observer.inspect_create_end(context, inputs, outcome);
    }
    fn inspect_selfdestruct(&mut self, contract: Address, target: Address, value: U256) {
        self.observer.inspect_selfdestruct(contract, target, value);
    }
    fn snapshot(&self) -> Self::Snapshot {
        self.state.clone()
    }
    fn prepared_snapshot(&self) -> Self::Snapshot {
        self.pending.as_ref().unwrap_or(&self.state).clone()
    }
    fn matches_prepared_snapshot(&self, snapshot: &Self::Snapshot) -> bool {
        self.pending.as_ref().unwrap_or(&self.state) == snapshot
    }
    fn restore(&mut self, snapshot: Self::Snapshot) {
        self.state = snapshot;
        self.pending = None;
    }
    fn commit_tx(&mut self) {
        if let Some(state) = self.pending.take() {
            self.state = state;
        }
    }

    fn supports_parallel_observation(&self) -> bool {
        true
    }
    fn set_parallel_observation(&mut self, enabled: bool) {
        self.observation_only = enabled;
    }
    fn take_parallel_observation(&mut self) -> Option<ParallelObservation> {
        self.observation.take()
    }
    fn evaluate_parallel_observation(
        &mut self,
        context: PostExecTxContext,
        observation: &ParallelObservation,
    ) -> Option<PostExecExecutedTx> {
        let observation =
            observation.downcast_ref::<<P::Observer as TransactionObserver>::Observation>()?;
        let (result, state) = Self::evaluate(&self.state, context, observation);
        self.pending = Some(state);
        Some(result)
    }
}
