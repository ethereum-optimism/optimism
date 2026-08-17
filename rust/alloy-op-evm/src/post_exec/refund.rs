//! The producer-side refund inspector contract.
//!
//! The seam between consensus and producer refund strategy. The block executor consumes
//! [`PostExecExecutedTx::refund_total`]; refund events are optional diagnostics. Verification,
//! settlement, and the `0x7D` consensus type never run a refund inspector.

use alloy_primitives::{Address, U256};
use revm::{
    context_interface::ContextTr,
    inspector::JournalExt,
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
};

use super::{PostExecExecutedTx, PostExecTxContext};

/// Per-transaction refund source, installed as the EVM's post-exec inspector during block
/// production.
///
/// An observing implementor normally also provides an empty or equivalent [`revm::Inspector`]
/// implementation for direct use. The generic factory drives the context-generic observer hooks on
/// this trait because Rust cannot express a bound requiring one type to implement `Inspector` for
/// every possible database context.
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
/// - The observer hooks must never synthesize call/create outcomes and must not mutate EVM state.
///   Every implementation must define every hook explicitly so a downstream observing policy cannot
///   compile after an API migration while silently inheriting no-op behavior.
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

    /// Observe one opcode step while post-exec tracking is active.
    fn inspect_step<CTX>(&mut self, interp: &mut Interpreter, context: &mut CTX)
    where
        CTX: ContextTr<Journal: JournalExt>;

    /// Observe a call frame while post-exec tracking is active.
    fn inspect_call<CTX>(&mut self, context: &mut CTX, inputs: &mut CallInputs)
    where
        CTX: ContextTr<Journal: JournalExt>;

    /// Observe the outcome of a call frame while post-exec tracking is active.
    ///
    /// `outcome` is observe-only: mutating it would change execution, which the seam forbids.
    fn inspect_call_end<CTX>(
        &mut self,
        context: &mut CTX,
        inputs: &CallInputs,
        outcome: &CallOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>;

    /// Observe a create frame while post-exec tracking is active.
    fn inspect_create<CTX>(&mut self, context: &mut CTX, inputs: &mut CreateInputs)
    where
        CTX: ContextTr<Journal: JournalExt>;

    /// Observe the outcome of a create frame while post-exec tracking is active.
    ///
    /// [`inspect_create`](Self::inspect_create) runs before revm's own frame checks, so an early
    /// failure there (depth limit, insufficient balance, nonce overflow) never warms the created
    /// address even though the policy has already observed it. A policy that must distinguish those
    /// cases gates on this hook — noting that `CreateCollision` warms the address but reports no
    /// created address. `outcome` is observe-only.
    fn inspect_create_end<CTX>(
        &mut self,
        context: &mut CTX,
        inputs: &CreateInputs,
        outcome: &CreateOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>;

    /// Observe a self-destruct while post-exec tracking is active.
    fn inspect_selfdestruct(&mut self, contract: Address, target: Address, value: U256);

    /// Snapshot the block-scoped carry-forward state.
    fn snapshot(&self) -> Self::Snapshot;

    /// Restore block-scoped carry-forward state previously captured by
    /// [`snapshot`](Self::snapshot).
    fn restore(&mut self, snapshot: Self::Snapshot);
}
