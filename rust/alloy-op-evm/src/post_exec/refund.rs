//! The producer-side refund inspector contract.
//!
//! The seam between consensus and producer refund strategy. The block executor consumes
//! [`PostExecExecutedTx::refund_total`]; refund events are optional diagnostics. Verification,
//! settlement, and the `0x7D` consensus type never run a refund inspector.

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
/// it never observes how the refund was computed. Verifiers run the default inspector and discard
/// its refund, so a proprietary producer policy can never make a verifier accept an *invalid*
/// block. It can, however, produce a *self-rejecting* block: the seam requires the implementor to
/// be side-effect-free w.r.t. EVM state (see the [`Inspector`](revm::Inspector) rule below); a
/// violation just fails the producer's own block, never verifier acceptance.
/// [`PostExecExecutedTx::refund_events`] are optional diagnostics and may be empty.
///
/// # Implementor contract
/// - [`finish_tx`](Self::finish_tx) is called exactly once per [`begin_tx`](Self::begin_tx),
///   **including when the EVM call itself errors** (`transact_raw` runs its finish block before
///   propagating). Implementors must tolerate finishing a failed tx.
/// - [`begin_tx`](Self::begin_tx) must fully reset per-transaction state.
///   [`snapshot`](Self::snapshot)/[`restore`](Self::restore) only cover block-scoped carry-forward,
///   so a failed or declined candidate relies on the next `begin_tx` for per-tx cleanup.
/// - The implementor's [`Inspector`](revm::Inspector) impl must never synthesize call/create
///   outcomes and must not mutate EVM state (enforced in debug builds by `debug_assert!`s in
///   `PostExecCompositeInspector`). A non-observing implementor can satisfy the `Inspector` bound
///   with an empty impl.
pub trait PostExecRefundInspector {
    /// Opaque block-scoped state carried across subblocks and candidate rollback.
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
