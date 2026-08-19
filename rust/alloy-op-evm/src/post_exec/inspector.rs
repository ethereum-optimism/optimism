use alloc::vec::Vec;
use alloy_primitives::{Address, B256};
use revm::{
    Inspector,
    context_interface::ContextTr,
    inspector::JournalExt,
    interpreter::{CallInputs, CreateInputs, Interpreter},
};

/// Refund categories a policy can attribute a rebate to.
///
/// The categories are shared vocabulary for diagnostics; the rebate amount each one carries is a
/// property of the policy that emitted the event, not of this enum.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PostExecRefundKind {
    /// Rebate for re-touching an account already touched in the block.
    WarmAccount,
    /// Rebate for re-reading a storage slot already read in the block.
    WarmSload,
    /// Rebate for re-writing a storage slot already touched in the block.
    WarmSstore,
}

/// Refund attribution event emitted when a policy grants a rebate.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PostExecRefundEvent {
    /// Replay-local transaction index that claimed the rebate.
    pub claiming_tx_index: u64,
    /// Refund kind.
    pub kind: PostExecRefundKind,
    /// Rebate amount in gas.
    pub amount: u64,
    /// Account touched by the rebate.
    pub address: Address,
    /// Storage slot touched by the rebate, when applicable.
    pub slot: Option<B256>,
    /// Replay-local transaction index that first warmed this account or slot.
    pub first_warmed_by_tx_index: u64,
}

/// Classification for the currently executing transaction.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PostExecTxKind {
    /// Regular user transaction that can claim post-exec refunds.
    Normal,
    /// Deposit transaction: warms for later txs, but never claims refunds.
    Deposit,
    /// Post-exec tx: never claims refunds.
    PostExec,
}

impl PostExecTxKind {
    /// Whether a transaction of this kind may claim post-exec refunds.
    ///
    /// Only a [`Normal`](Self::Normal) transaction claims. Deposits and the trailing post-exec
    /// transaction only warm state. External producer policies can reuse this classification.
    pub const fn claims_refunds(self) -> bool {
        matches!(self, Self::Normal)
    }
}

/// Metadata supplied before executing a transaction.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PostExecTxContext {
    /// Replay-local transaction index.
    pub tx_index: u64,
    /// Transaction classification.
    pub kind: PostExecTxKind,
}

/// Extracted result for the most recently executed transaction.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct PostExecExecutedTx {
    /// Consensus-facing total refund for the tx.
    pub refund_total: u64,
    /// Optional diagnostic attribution events for the tx.
    pub refund_events: Vec<PostExecRefundEvent>,
}

/// Composite inspector that always includes a refund inspector `R` alongside a caller-provided
/// inner inspector `I`, fanning every hook to both.
///
/// `R` is fixed by the EVM factory (it defaults to
/// [`NullRefundPolicy`](super::NullRefundPolicy)); the always-present refund inspector is what lets
/// `OpEvm<DB, I, R>` expose post-exec hooks for *any* user inspector `I` — as `alloy_evm`'s
/// `BlockExecutorFactory` requires.
#[derive(Debug, Clone)]
pub struct PostExecCompositeInspector<I, R = super::NullRefundPolicy> {
    inner: I,
    post_exec: R,
}

impl<I, R: Default> PostExecCompositeInspector<I, R> {
    /// Creates a new composite inspector with a default-constructed refund inspector.
    pub fn new(inner: I) -> Self {
        Self { inner, post_exec: R::default() }
    }
}

impl<I, R> PostExecCompositeInspector<I, R> {
    /// Returns the wrapped user inspector.
    pub const fn inner(&self) -> &I {
        &self.inner
    }

    /// Returns the wrapped user inspector mutably.
    pub const fn inner_mut(&mut self) -> &mut I {
        &mut self.inner
    }

    /// Consumes the composite inspector and returns the wrapped user inspector.
    pub fn into_inner(self) -> I {
        self.inner
    }
}

impl<I, R: super::PostExecRefundInspector> PostExecCompositeInspector<I, R> {
    /// Begin tracking the next transaction.
    pub fn begin_post_exec_tx(&mut self, ctx: PostExecTxContext) {
        self.post_exec.begin_tx(ctx);
    }

    /// Snapshot refund state to carry across subblock executors.
    pub fn refund_snapshot(&self) -> R::Snapshot {
        self.post_exec.snapshot()
    }

    /// Seed refund state captured from a prior subblock.
    pub fn seed_refund_snapshot(&mut self, state: R::Snapshot) {
        self.post_exec.restore(state);
    }

    /// Notes an account touch that happened outside opcode stepping.
    pub fn note_account_touch(&mut self, address: Address) {
        self.post_exec.note_account_touch(address);
    }

    /// Finish tracking the current transaction.
    pub fn finish_post_exec_tx(&mut self) -> PostExecExecutedTx {
        self.post_exec.finish_tx()
    }
}

impl<CTX, I, R> Inspector<CTX> for PostExecCompositeInspector<I, R>
where
    CTX: ContextTr<Journal: JournalExt>,
    I: Inspector<CTX>,
    R: super::PostExecRefundInspector,
{
    fn initialize_interp(&mut self, interp: &mut Interpreter, context: &mut CTX) {
        self.inner.initialize_interp(interp, context);
    }

    fn step(&mut self, interp: &mut Interpreter, context: &mut CTX) {
        self.inner.step(interp, context);
        self.post_exec.inspect_step(interp, context);
    }

    fn step_end(&mut self, interp: &mut Interpreter, context: &mut CTX) {
        self.inner.step_end(interp, context);
    }

    fn log(&mut self, context: &mut CTX, log: alloy_primitives::Log) {
        self.inner.log(context, log);
    }

    fn log_full(
        &mut self,
        interp: &mut Interpreter,
        context: &mut CTX,
        log: alloy_primitives::Log,
    ) {
        self.inner.log_full(interp, context, log);
    }

    fn call(
        &mut self,
        context: &mut CTX,
        inputs: &mut CallInputs,
    ) -> Option<revm::interpreter::CallOutcome> {
        // Always run the refund observer: its first-touch observations must not be gated on
        // whether the user inspector short-circuits the frame. The user inspector's return value
        // remains authoritative because the refund seam is observer-only.
        let inner = self.inner.call(context, inputs);
        self.post_exec.inspect_call(context, inputs);
        inner
    }

    fn call_end(
        &mut self,
        context: &mut CTX,
        inputs: &CallInputs,
        outcome: &mut revm::interpreter::CallOutcome,
    ) {
        self.inner.call_end(context, inputs, outcome);
        // The refund observer sees the outcome the user inspector settled on, and only by
        // shared reference: the seam is observer-only, so it cannot rewrite the frame result.
        self.post_exec.inspect_call_end(context, inputs, outcome);
    }

    fn create(
        &mut self,
        context: &mut CTX,
        inputs: &mut CreateInputs,
    ) -> Option<revm::interpreter::CreateOutcome> {
        let inner = self.inner.create(context, inputs);
        self.post_exec.inspect_create(context, inputs);
        inner
    }

    fn create_end(
        &mut self,
        context: &mut CTX,
        inputs: &CreateInputs,
        outcome: &mut revm::interpreter::CreateOutcome,
    ) {
        self.inner.create_end(context, inputs, outcome);
        self.post_exec.inspect_create_end(context, inputs, outcome);
    }

    fn selfdestruct(&mut self, contract: Address, target: Address, value: alloy_primitives::U256) {
        self.inner.selfdestruct(contract, target, value);
        self.post_exec.inspect_selfdestruct(contract, target, value);
    }
}
