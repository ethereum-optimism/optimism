//! The producer-side refund inspector contract.
//!
//! The seam between consensus (monorepo) and production strategy (sequencer / optimism-premium).
//! The block executor consumes a finished [`PostExecExecutedTx`] per transaction. Its
//! `refund_total` is consensus-facing, while `refund_events` are optional diagnostics.
//! Verification, settlement, and the `0x7D` consensus type stay in the monorepo and never run a
//! refund inspector.

use alloy_primitives::Address;

use super::{PostExecExecutedTx, PostExecTxContext};

/// Per-transaction refund source, installed as the EVM's post-exec inspector during block
/// production.
///
/// An implementor is also a [`revm::Inspector`] (it observes execution to compute the refund), but
/// that bound is applied at the EVM construction site rather than as a supertrait here: the refund
/// methods below are context-free, so requiring `Inspector<CTX>` on the trait would make them
/// uncallable from the EVM's context-free post-exec hooks.
///
/// **Not consensus.** The executor reads [`PostExecExecutedTx::refund_total`] via
/// [`finish_tx`](Self::finish_tx) and bounds it by the structural `refund <= evm_gas_used` rule —
/// it never observes how the refund was computed. A buggy or proprietary producer can therefore
/// only ever yield an *economically* different refund, never an *invalid* block.
/// [`PostExecExecutedTx::refund_events`] are optional diagnostics and may be empty.
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

    /// Finish the current transaction. The result's aggregate refund is consensus-facing; its
    /// attribution events are optional diagnostics.
    fn finish_tx(&mut self) -> PostExecExecutedTx;

    /// Snapshot the block-scoped carry-forward state.
    fn snapshot(&self) -> Self::Snapshot;

    /// Restore block-scoped carry-forward state previously captured by
    /// [`snapshot`](Self::snapshot).
    fn restore(&mut self, snapshot: Self::Snapshot);
}
