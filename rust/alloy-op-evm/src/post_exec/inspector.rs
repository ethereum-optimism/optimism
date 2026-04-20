use alloc::vec::Vec;
use alloy_primitives::{
    Address, B256,
    map::{HashMap, HashSet},
};
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

/// Exact refund categories for post-exec block-level warming.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WarmingRefundKind {
    /// Warm account rebate (+2500).
    WarmAccount,
    /// Warm storage read rebate (+2000).
    WarmSload,
    /// Warm storage write rebate (+2100).
    WarmSstore,
}

/// Exact refund attribution event emitted when a warming rebate is granted.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct WarmingRefundEvent {
    /// Replay-local transaction index that claimed the rebate.
    pub claiming_tx_index: u64,
    /// Refund kind.
    pub kind: WarmingRefundKind,
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
    /// Synthetic post-exec tx: never claims refunds.
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct PostExecExecutedTx {
    /// Total refund for the tx.
    pub refund_total: u64,
    /// Exact attribution events for the tx.
    pub refund_events: Vec<WarmingRefundEvent>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct WarmProvenance {
    first_warmed_by_tx_index: u64,
}

#[derive(Debug, Clone, Default)]
struct CurrentTxState {
    tx_index: u64,
    kind: Option<PostExecTxKind>,
    initialized_top_level: bool,
    refund_total: u64,
    refund_events: Vec<WarmingRefundEvent>,
    touched_accounts: HashSet<Address>,
    touched_slots: HashSet<(Address, B256)>,
    intrinsic_warm_accounts: HashSet<Address>,
    intrinsic_warm_slots: HashSet<(Address, B256)>,
}

impl CurrentTxState {
    fn begin(&mut self, ctx: PostExecTxContext) {
        self.tx_index = ctx.tx_index;
        self.kind = Some(ctx.kind);
        self.initialized_top_level = false;
        self.refund_total = 0;
        self.refund_events.clear();
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
        PostExecExecutedTx {
            refund_total: core::mem::take(&mut self.refund_total),
            refund_events: core::mem::take(&mut self.refund_events),
        }
    }

    fn emit_refund(
        &mut self,
        provenance: WarmProvenance,
        kind: WarmingRefundKind,
        amount: u64,
        address: Address,
        slot: Option<B256>,
    ) {
        if self.kind.is_some_and(PostExecTxKind::claims_refunds) {
            self.refund_total = self.refund_total.saturating_add(amount);
            self.refund_events.push(WarmingRefundEvent {
                claiming_tx_index: self.tx_index,
                kind,
                amount,
                address,
                slot,
                first_warmed_by_tx_index: provenance.first_warmed_by_tx_index,
            });
        }
    }
}

/// Lightweight inspector that computes post-exec block-warming refunds and provenance.
#[derive(Debug, Clone, Default)]
pub struct SDMWarmingInspector {
    warmed_accounts: HashMap<Address, WarmProvenance>,
    warmed_slots: HashMap<(Address, B256), WarmProvenance>,
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
        self.last_tx = last.clone();
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
            !self.current_tx.intrinsic_warm_accounts.contains(&address)
        {
            if let Some(provenance) = self.warmed_accounts.get(&address).copied() {
                self.current_tx.emit_refund(
                    provenance,
                    WarmingRefundKind::WarmAccount,
                    2500,
                    address,
                    None,
                );
            }
        }

        self.warmed_accounts
            .entry(address)
            .or_insert(WarmProvenance { first_warmed_by_tx_index: self.current_tx.tx_index });

        // Non-claiming tx kinds (Deposit, PostExec) never populate these fields because
        // `emit_refund` above gates on `claims_refunds()`. Previously this function also
        // cleared them on every call for those kinds; that was defense-in-depth that
        // obscured the invariant — assert it instead so regressions surface in debug builds.
        debug_assert!(
            self.current_tx.kind().is_some_and(PostExecTxKind::claims_refunds) ||
                (self.current_tx.refund_total == 0 && self.current_tx.refund_events.is_empty()),
            "non-claiming tx kinds must not have emitted refunds — check emit_refund gating",
        );
    }

    fn observe_slot_touch(&mut self, address: Address, slot: B256, is_sstore: bool) {
        if self.current_tx.kind().is_none() {
            return;
        }

        // Storage accesses should never also claim the account rebate.
        self.observe_account_touch(address, false);

        if self.current_tx.touched_slots.insert((address, slot)) &&
            !self.current_tx.intrinsic_warm_slots.contains(&(address, slot))
        {
            if let Some(provenance) = self.warmed_slots.get(&(address, slot)).copied() {
                let (kind, amount) = if is_sstore {
                    (WarmingRefundKind::WarmSstore, 2100)
                } else {
                    (WarmingRefundKind::WarmSload, 2000)
                };
                self.current_tx.emit_refund(provenance, kind, amount, address, Some(slot));
            }
        }

        self.warmed_slots
            .entry((address, slot))
            .or_insert(WarmProvenance { first_warmed_by_tx_index: self.current_tx.tx_index });
    }

    #[cfg(test)]
    fn note_intrinsic_account(&mut self, address: Address) {
        self.current_tx.intrinsic_warm_accounts.insert(address);
    }

    #[cfg(test)]
    fn note_intrinsic_slot(&mut self, address: Address, slot: B256) {
        self.current_tx.intrinsic_warm_slots.insert((address, slot));
    }

    #[cfg(test)]
    fn test_observe_account_touch(&mut self, address: Address) {
        self.observe_account_touch(address, true);
    }

    #[cfg(test)]
    fn test_observe_slot_touch(&mut self, address: Address, slot: B256, is_sstore: bool) {
        self.observe_slot_touch(address, slot, is_sstore);
    }
}

impl<CTX> Inspector<CTX> for SDMWarmingInspector
where
    CTX: ContextTr<Journal: JournalExt>,
{
    fn step(&mut self, interp: &mut Interpreter, context: &mut CTX) {
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

        self.ensure_top_level_initialized(context);
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

/// Composite inspector that always includes the post-exec warming inspector.
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
mod tests {
    use super::*;
    use alloy_primitives::{address, b256};

    #[test]
    fn repeated_account_touch_refunds_once() {
        let account = address!("00000000000000000000000000000000000000aa");
        let mut inspector = SDMWarmingInspector::default();

        inspector.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
        inspector.test_observe_account_touch(account);
        let first = inspector.finish_tx();
        assert_eq!(first.refund_total, 0);

        inspector.begin_tx(PostExecTxContext { tx_index: 1, kind: PostExecTxKind::Normal });
        inspector.test_observe_account_touch(account);
        let second = inspector.finish_tx();
        assert_eq!(second.refund_total, 2500);
        assert_eq!(second.refund_events.len(), 1);
        assert_eq!(second.refund_events[0].kind, WarmingRefundKind::WarmAccount);
        assert_eq!(second.refund_events[0].first_warmed_by_tx_index, 0);
    }

    #[test]
    fn repeated_sload_refunds_without_account_double_count() {
        let account = address!("00000000000000000000000000000000000000aa");
        let slot = b256!("0000000000000000000000000000000000000000000000000000000000000001");
        let mut inspector = SDMWarmingInspector::default();

        inspector.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
        inspector.test_observe_slot_touch(account, slot, false);
        let first = inspector.finish_tx();
        assert_eq!(first.refund_total, 0);

        inspector.begin_tx(PostExecTxContext { tx_index: 1, kind: PostExecTxKind::Normal });
        inspector.test_observe_slot_touch(account, slot, false);
        let second = inspector.finish_tx();
        assert_eq!(second.refund_total, 2000);
        assert_eq!(second.refund_events.len(), 1);
        assert_eq!(second.refund_events[0].kind, WarmingRefundKind::WarmSload);
    }

    #[test]
    fn repeated_sstore_refunds_without_account_double_count() {
        let account = address!("00000000000000000000000000000000000000aa");
        let slot = b256!("0000000000000000000000000000000000000000000000000000000000000002");
        let mut inspector = SDMWarmingInspector::default();

        inspector.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
        inspector.test_observe_slot_touch(account, slot, true);
        let first = inspector.finish_tx();
        assert_eq!(first.refund_total, 0);

        inspector.begin_tx(PostExecTxContext { tx_index: 1, kind: PostExecTxKind::Normal });
        inspector.test_observe_slot_touch(account, slot, true);
        let second = inspector.finish_tx();
        assert_eq!(second.refund_total, 2100);
        assert_eq!(second.refund_events.len(), 1);
        assert_eq!(second.refund_events[0].kind, WarmingRefundKind::WarmSstore);
    }

    #[test]
    fn deposit_warms_but_does_not_claim() {
        let account = address!("00000000000000000000000000000000000000bb");
        let mut inspector = SDMWarmingInspector::default();

        inspector.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Deposit });
        inspector.test_observe_account_touch(account);
        let deposit = inspector.finish_tx();
        assert_eq!(deposit.refund_total, 0);
        assert!(deposit.refund_events.is_empty());

        inspector.begin_tx(PostExecTxContext { tx_index: 1, kind: PostExecTxKind::Normal });
        inspector.test_observe_account_touch(account);
        let later = inspector.finish_tx();
        assert_eq!(later.refund_total, 2500);
        assert_eq!(later.refund_events[0].first_warmed_by_tx_index, 0);
    }

    #[test]
    fn post_exec_tx_never_claims_refunds() {
        let account = address!("00000000000000000000000000000000000000cc");
        let mut inspector = SDMWarmingInspector::default();

        inspector.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
        inspector.test_observe_account_touch(account);
        let _ = inspector.finish_tx();

        inspector.begin_tx(PostExecTxContext { tx_index: 1, kind: PostExecTxKind::PostExec });
        inspector.test_observe_account_touch(account);
        let post_exec = inspector.finish_tx();
        assert_eq!(post_exec.refund_total, 0);
        assert!(post_exec.refund_events.is_empty());
    }

    #[test]
    fn intrinsic_access_list_warmth_does_not_claim_or_steal_provenance() {
        let account = address!("00000000000000000000000000000000000000dd");
        let slot = b256!("0000000000000000000000000000000000000000000000000000000000000003");
        let mut inspector = SDMWarmingInspector::default();

        inspector.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
        inspector.note_intrinsic_account(account);
        inspector.note_intrinsic_slot(account, slot);
        inspector.test_observe_slot_touch(account, slot, false);
        let first = inspector.finish_tx();
        assert_eq!(first.refund_total, 0);

        inspector.begin_tx(PostExecTxContext { tx_index: 1, kind: PostExecTxKind::Normal });
        inspector.test_observe_slot_touch(account, slot, false);
        let second = inspector.finish_tx();
        assert_eq!(second.refund_total, 2000);
        assert_eq!(second.refund_events[0].first_warmed_by_tx_index, 0);
    }

    #[test]
    fn warming_provenance_chains_across_three_txs() {
        let account_a = address!("00000000000000000000000000000000000000aa");
        let account_b = address!("00000000000000000000000000000000000000bb");
        let slot = b256!("0000000000000000000000000000000000000000000000000000000000000042");
        let mut inspector = SDMWarmingInspector::default();

        // Tx 0 warms account A (no refund — it's the first toucher).
        inspector.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
        inspector.test_observe_account_touch(account_a);
        let tx0 = inspector.finish_tx();
        assert_eq!(tx0.refund_total, 0);
        assert!(tx0.refund_events.is_empty());

        // Tx 1 re-warms A (refund, provenance = 0) AND is the first toucher of slot (B, slot).
        inspector.begin_tx(PostExecTxContext { tx_index: 1, kind: PostExecTxKind::Normal });
        inspector.test_observe_account_touch(account_a);
        inspector.test_observe_slot_touch(account_b, slot, true);
        let tx1 = inspector.finish_tx();
        assert_eq!(tx1.refund_total, 2500);
        assert_eq!(tx1.refund_events.len(), 1);
        assert_eq!(tx1.refund_events[0].kind, WarmingRefundKind::WarmAccount);
        assert_eq!(tx1.refund_events[0].address, account_a);
        assert_eq!(tx1.refund_events[0].first_warmed_by_tx_index, 0);

        // Tx 2 re-hits (B, slot) via SSTORE — should refund 2100, attributed to tx 1.
        inspector.begin_tx(PostExecTxContext { tx_index: 2, kind: PostExecTxKind::Normal });
        inspector.test_observe_slot_touch(account_b, slot, true);
        let tx2 = inspector.finish_tx();
        assert_eq!(tx2.refund_total, 2100);
        assert_eq!(tx2.refund_events.len(), 1);
        assert_eq!(tx2.refund_events[0].kind, WarmingRefundKind::WarmSstore);
        assert_eq!(tx2.refund_events[0].address, account_b);
        assert_eq!(tx2.refund_events[0].slot, Some(slot));
        assert_eq!(tx2.refund_events[0].first_warmed_by_tx_index, 1);
    }

    #[test]
    fn take_last_tx_result_round_trips() {
        let account = address!("00000000000000000000000000000000000000ee");
        let mut inspector = SDMWarmingInspector::default();

        inspector.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
        inspector.test_observe_account_touch(account);
        let _ = inspector.finish_tx();
        let last = inspector.take_last_tx_result();
        assert_eq!(last.refund_total, 0);
        assert!(inspector.take_last_tx_result().refund_events.is_empty());
    }
}
