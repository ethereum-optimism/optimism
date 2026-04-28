use alloy_primitives::{Address, B256, map::HashSet};
use revm::{
    Inspector,
    bytecode::opcode,
    context::Block,
    context_interface::{
        ContextTr, CreateScheme, JournalTr, Transaction,
        transaction::{AccessListItemTr, AuthorizationTr},
    },
    inspector::JournalExt,
    interpreter::{
        CallInputs, CreateInputs, Interpreter,
        interpreter_types::{InputsTr, Jumps},
    },
    primitives::TxKind,
};

// EIP-2929 repeat-access savings. SDM refunds the cold-access premium when a tx
// re-touches something a prior tx in the same block already warmed, making canonical gas
// reflect block-level warming.
//
// Values are derived from EIP-2929 cold-access premiums:
//   account touch: COLD_ACCOUNT_ACCESS_COST (2600) - WARM_STORAGE_READ_COST (100) = 2500
//   SLOAD:         COLD_SLOAD_COST          (2100) - WARM_STORAGE_READ_COST (100) = 2000
//   SSTORE:        cold storage-key surcharge is the full COLD_SLOAD_COST         = 2100
//                  (EIP-2929 uses this SLOAD-named constant for cold SSTORE too)

/// Refund for re-touching an account warmed earlier in the block (BALANCE, EXTCODE*, CALL, …).
const ACCOUNT_REWARM_REFUND: u64 = 2500;
/// Refund for re-touching a storage slot warmed earlier in the block via SLOAD.
const SLOAD_REWARM_REFUND: u64 = 2000;
/// Refund for re-touching a storage slot warmed earlier in the block via SSTORE.
///
/// Higher than the SLOAD refund because EIP-2929 charges cold SSTORE the full
/// `COLD_SLOAD_COST` surcharge too, despite the SLOAD-specific constant name.
const SSTORE_REWARM_REFUND: u64 = 2100;

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
    const fn claims_refunds(self) -> bool {
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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct PostExecExecutedTx {
    /// Total refund for the tx.
    pub refund_total: u64,
}

#[derive(Debug, Clone, Default)]
struct CurrentTxState {
    kind: Option<PostExecTxKind>,
    initialized_top_level: bool,
    refund_total: u64,
    touched_accounts: HashSet<Address>,
    touched_slots: HashSet<(Address, B256)>,
    intrinsic_warm_accounts: HashSet<Address>,
    intrinsic_warm_slots: HashSet<(Address, B256)>,
}

impl CurrentTxState {
    fn begin(&mut self, ctx: PostExecTxContext) {
        self.kind = Some(ctx.kind);
        self.initialized_top_level = false;
        self.refund_total = 0;
        self.touched_accounts.clear();
        self.touched_slots.clear();
        self.intrinsic_warm_accounts.clear();
        self.intrinsic_warm_slots.clear();
    }

    const fn kind(&self) -> Option<PostExecTxKind> {
        self.kind
    }

    fn finish(&mut self) -> PostExecExecutedTx {
        self.kind = None;
        self.initialized_top_level = false;
        PostExecExecutedTx { refund_total: core::mem::take(&mut self.refund_total) }
    }

    fn add_refund(&mut self, amount: u64) {
        if self.kind.is_some_and(PostExecTxKind::claims_refunds) {
            self.refund_total = self.refund_total.saturating_add(amount);
        }
    }
}

/// Lightweight inspector that computes post-exec block-warming refunds.
#[derive(Debug, Clone, Default)]
pub struct SDMWarmingInspector {
    warmed_accounts: HashSet<Address>,
    warmed_slots: HashSet<(Address, B256)>,
    current_tx: CurrentTxState,
    last_tx: PostExecExecutedTx,
}

impl SDMWarmingInspector {
    /// Begins tracking for the next transaction.
    pub fn begin_tx(&mut self, ctx: PostExecTxContext) {
        self.current_tx.begin(ctx);
    }

    /// Notes an account touch that happened outside opcode stepping.
    pub fn note_account_touch(&mut self, address: Address) {
        self.observe_account_touch(address, true);
    }

    /// Finishes the current transaction and stores the extracted result.
    pub fn finish_tx(&mut self) -> PostExecExecutedTx {
        let last = self.current_tx.finish();
        self.last_tx = last;
        last
    }

    /// Takes the extracted result for the most recently finished transaction.
    pub fn take_last_tx_result(&mut self) -> PostExecExecutedTx {
        core::mem::take(&mut self.last_tx)
    }

    fn ensure_top_level_initialized<CTX>(&mut self, context: &CTX)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
        if self.current_tx.kind().is_none() || self.current_tx.initialized_top_level {
            return;
        }

        self.current_tx.initialized_top_level = true;
        self.collect_intrinsic_warmth(context);

        let caller = context.tx().caller();
        self.observe_account_touch(caller, true);

        if let TxKind::Call(target) = context.tx().kind() {
            self.observe_account_touch(target, true);
        }
    }

    fn collect_intrinsic_warmth<CTX>(&mut self, context: &CTX)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
        self.current_tx.intrinsic_warm_accounts.insert(context.block().beneficiary());
        self.current_tx
            .intrinsic_warm_accounts
            .extend(context.journal_ref().precompile_addresses().iter().copied());

        if let Some(access_list) = context.tx().access_list() {
            for item in access_list {
                let address = *item.address();
                self.current_tx.intrinsic_warm_accounts.insert(address);
                for slot in item.storage_slots() {
                    self.current_tx.intrinsic_warm_slots.insert((address, *slot));
                }
            }
        }

        for authority in context.tx().authorization_list() {
            if let Some(authority) = authority.authority() {
                self.current_tx.intrinsic_warm_accounts.insert(authority);
            }
        }
    }

    fn observe_account_touch(&mut self, address: Address, allow_refund: bool) {
        if self.current_tx.kind().is_none() {
            return;
        }

        if self.current_tx.touched_accounts.insert(address) &&
            allow_refund &&
            !self.current_tx.intrinsic_warm_accounts.contains(&address) &&
            self.warmed_accounts.contains(&address)
        {
            self.current_tx.add_refund(ACCOUNT_REWARM_REFUND);
        }

        self.warmed_accounts.insert(address);
    }

    fn observe_slot_touch(&mut self, address: Address, slot: B256, is_sstore: bool) {
        if self.current_tx.kind().is_none() {
            return;
        }

        // Storage accesses should never also claim the account refund.
        self.observe_account_touch(address, false);

        if self.current_tx.touched_slots.insert((address, slot)) &&
            !self.current_tx.intrinsic_warm_slots.contains(&(address, slot)) &&
            self.warmed_slots.contains(&(address, slot))
        {
            self.current_tx.add_refund(if is_sstore {
                SSTORE_REWARM_REFUND
            } else {
                SLOAD_REWARM_REFUND
            });
        }

        self.warmed_slots.insert((address, slot));
    }
}

impl<CTX> Inspector<CTX> for SDMWarmingInspector
where
    CTX: ContextTr<Journal: JournalExt>,
{
    fn step(&mut self, interp: &mut Interpreter, context: &mut CTX) {
        self.ensure_top_level_initialized(context);

        match interp.bytecode.opcode() {
            opcode::SLOAD | opcode::SSTORE => {
                if let Ok(slot) = interp.stack.peek(0) {
                    let slot = B256::from(slot.to_be_bytes());
                    self.observe_slot_touch(
                        interp.input.target_address(),
                        slot,
                        interp.bytecode.opcode() == opcode::SSTORE,
                    );
                }
            }
            opcode::EXTCODECOPY |
            opcode::EXTCODEHASH |
            opcode::EXTCODESIZE |
            opcode::BALANCE |
            opcode::SELFDESTRUCT => {
                if let Ok(word) = interp.stack.peek(0) {
                    self.observe_account_touch(
                        Address::from_word(B256::from(word.to_be_bytes())),
                        true,
                    );
                }
            }
            _ => {}
        }
    }

    fn call(
        &mut self,
        context: &mut CTX,
        inputs: &mut CallInputs,
    ) -> Option<revm::interpreter::CallOutcome> {
        if context.journal().depth() == 0 {
            self.ensure_top_level_initialized(context);
        }
        self.observe_account_touch(inputs.bytecode_address, true);
        None
    }

    fn create(
        &mut self,
        context: &mut CTX,
        inputs: &mut CreateInputs,
    ) -> Option<revm::interpreter::CreateOutcome> {
        if context.journal().depth() == 0 {
            self.ensure_top_level_initialized(context);
        }

        let caller = inputs.caller();
        self.observe_account_touch(caller, true);

        let created_address = match inputs.scheme() {
            CreateScheme::Create => {
                let nonce = context
                    .journal_ref()
                    .evm_state()
                    .get(&caller)
                    .map(|account| account.info.nonce)
                    .unwrap_or_default();
                inputs.created_address(nonce)
            }
            _ => inputs.created_address(0),
        };
        self.observe_account_touch(created_address, true);
        None
    }

    fn selfdestruct(
        &mut self,
        _contract: Address,
        target: Address,
        _value: alloy_primitives::U256,
    ) {
        self.observe_account_touch(target, true);
    }
}

/// Composite inspector that always includes the [`SDMWarmingInspector`] alongside a
/// caller-provided inner inspector, fanning every hook to both.
#[derive(Debug, Clone)]
pub struct PostExecCompositeInspector<I> {
    inner: I,
    post_exec: SDMWarmingInspector,
}

impl<I> PostExecCompositeInspector<I> {
    /// Creates a new composite inspector.
    pub fn new(inner: I) -> Self {
        Self { inner, post_exec: SDMWarmingInspector::default() }
    }

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

    /// Begin tracking the next transaction.
    pub fn begin_post_exec_tx(&mut self, ctx: PostExecTxContext) {
        self.post_exec.begin_tx(ctx);
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

impl<CTX, INTR, I> Inspector<CTX, INTR> for PostExecCompositeInspector<I>
where
    INTR: revm::interpreter::InterpreterTypes,
    I: Inspector<CTX, INTR>,
    SDMWarmingInspector: Inspector<CTX, INTR>,
{
    fn initialize_interp(&mut self, interp: &mut Interpreter<INTR>, context: &mut CTX) {
        self.inner.initialize_interp(interp, context);
        self.post_exec.initialize_interp(interp, context);
    }

    fn step(&mut self, interp: &mut Interpreter<INTR>, context: &mut CTX) {
        self.inner.step(interp, context);
        self.post_exec.step(interp, context);
    }

    fn step_end(&mut self, interp: &mut Interpreter<INTR>, context: &mut CTX) {
        self.inner.step_end(interp, context);
        self.post_exec.step_end(interp, context);
    }

    fn log(&mut self, context: &mut CTX, log: alloy_primitives::Log) {
        self.inner.log(context, log.clone());
        self.post_exec.log(context, log);
    }

    fn log_full(
        &mut self,
        interp: &mut Interpreter<INTR>,
        context: &mut CTX,
        log: alloy_primitives::Log,
    ) {
        self.inner.log_full(interp, context, log.clone());
        self.post_exec.log_full(interp, context, log);
    }

    fn call(
        &mut self,
        context: &mut CTX,
        inputs: &mut CallInputs,
    ) -> Option<revm::interpreter::CallOutcome> {
        // Always run both inspectors: the warming inspector's first-touch observations drive
        // block-scoped refund attribution and must not be gated on whether the user inspector
        // short-circuits the frame. The warming inspector is expected to never synthesize an
        // outcome, so inner's return value is authoritative.
        let inner = self.inner.call(context, inputs);
        let post_exec = self.post_exec.call(context, inputs);
        debug_assert!(
            post_exec.is_none(),
            "SDMWarmingInspector must not synthesize a call outcome",
        );
        inner
    }

    fn call_end(
        &mut self,
        context: &mut CTX,
        inputs: &CallInputs,
        outcome: &mut revm::interpreter::CallOutcome,
    ) {
        self.inner.call_end(context, inputs, outcome);
        self.post_exec.call_end(context, inputs, outcome);
    }

    fn create(
        &mut self,
        context: &mut CTX,
        inputs: &mut CreateInputs,
    ) -> Option<revm::interpreter::CreateOutcome> {
        // See `call` above: always observe; inner's outcome wins.
        let inner = self.inner.create(context, inputs);
        let post_exec = self.post_exec.create(context, inputs);
        debug_assert!(
            post_exec.is_none(),
            "SDMWarmingInspector must not synthesize a create outcome",
        );
        inner
    }

    fn create_end(
        &mut self,
        context: &mut CTX,
        inputs: &CreateInputs,
        outcome: &mut revm::interpreter::CreateOutcome,
    ) {
        self.inner.create_end(context, inputs, outcome);
        self.post_exec.create_end(context, inputs, outcome);
    }

    fn selfdestruct(&mut self, contract: Address, target: Address, value: alloy_primitives::U256) {
        self.inner.selfdestruct(contract, target, value);
        self.post_exec.selfdestruct(contract, target, value);
    }
}

#[cfg(test)]
mod tests;
