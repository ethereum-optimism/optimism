//! The producer-side refund inspector contract.
//!
//! The seam between consensus (monorepo) and production strategy (sequencer / optimism-premium).
//! The block executor consumes a finished `u64` refund per transaction — never a trace — so the
//! entire production strategy lives behind [`PostExecRefundInspector`]. Verification, settlement
//! and the `0x7D` consensus type stay in the monorepo and never run a refund inspector.

use alloy_primitives::Address;
use revm::{Inspector, interpreter::InterpreterTypes};

use super::PostExecTxContext;

/// Per-transaction refund source, installed as the EVM's post-exec inspector during block
/// production.
///
/// An implementor is also an [`Inspector`] (it observes execution to compute the refund), but that
/// bound is applied at the EVM construction site rather than as a supertrait here: the refund
/// methods below are context-free, so requiring `Inspector<CTX>` on the trait would make them
/// uncallable from the EVM's context-free post-exec hooks.
///
/// **Not consensus.** The executor reads a finished aggregate refund per tx via
/// [`finish_tx`](Self::finish_tx) and bounds it by the structural `refund <= evm_gas_used` rule —
/// it never observes how the refund was computed. A buggy or proprietary producer can therefore
/// only ever yield an *economically* different refund, never an *invalid* block. The seam supports
/// installing [`NoopRefundInspector`] on non-producing paths; flipping those public defaults is a
/// follow-up teardown step after premium production is proven in CI.
pub trait PostExecRefundInspector {
    /// Opaque block-scoped carry-forward state, threaded across flashblock executors and across a
    /// declined candidate's rollback. The monorepo round-trips it without inspecting its shape.
    type Snapshot: Clone;

    /// Begin observing the next transaction.
    fn begin_tx(&mut self, ctx: PostExecTxContext);

    /// Record a protocol-level account touch (e.g. the per-tx fee-vault settlement write) that
    /// happens outside opcode stepping. The current tx never claims a refund for it, but recording
    /// it lets a *later* tx that genuinely accesses the account via an opcode earn its rebate.
    fn note_account_touch(&mut self, address: Address);

    /// Finish the current transaction and return its total refund in gas. This `u64` is the entire
    /// cross-boundary contract.
    fn finish_tx(&mut self) -> u64;

    /// Snapshot the block-scoped carry-forward state.
    fn snapshot(&self) -> Self::Snapshot;

    /// Restore block-scoped carry-forward state previously captured by
    /// [`snapshot`](Self::snapshot).
    fn restore(&mut self, snapshot: Self::Snapshot);
}

/// No-op refund source for non-producing paths: observes nothing, refunds nothing.
///
/// This is the intended public default after the monorepo SDM-production teardown. Until that
/// teardown lands, some public factories still use
/// [`SDMWarmingInspector`](super::SDMWarmingInspector) even when not producing.
#[derive(Debug, Clone, Copy, Default)]
pub struct NoopRefundInspector;

impl<CTX, INTR: InterpreterTypes> Inspector<CTX, INTR> for NoopRefundInspector {}

impl PostExecRefundInspector for NoopRefundInspector {
    type Snapshot = ();

    fn begin_tx(&mut self, _ctx: PostExecTxContext) {}

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> u64 {
        0
    }

    fn snapshot(&self) -> Self::Snapshot {}

    fn restore(&mut self, _snapshot: Self::Snapshot) {}
}
