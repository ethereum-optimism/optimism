//! The pure (sync, IO-free) derivation surface.
//!
//! Phase 3 of the pure-derivation migration. See the design plan and
//! brainstorm under `plans/` for the full rationale. Quick summary:
//!
//! - **No async, no IO.** [`Deriver`] runs entirely on pre-fetched L1 inputs supplied by the
//!   caller. The caller is responsible for L1 RPC, blob KZG verification, and L2 lookups; the
//!   deriver runs the protocol-level state machine.
//! - **No `tracing::*` calls.** Every observable event lives in [`DeriveTrace`]; the caller
//!   translates it to whatever observability substrate is appropriate (`tracing::*` in kona-node,
//!   oracle hints in kona-client).
//! - **`derive` never errors.** Malformed inputs are dropped silently with a trace entry. Only
//!   caller-contract violations on [`Deriver::add_l1_input`] / [`Deriver::add_span_batch_overlap`]
//!   surface as [`CriticalError`].
//! - **Closes the span-batch overlap content spec gap** post-Holocene. `overlap.rs` runs the full
//!   byte-wise tx compare that op-node and kona's async pipeline skip today.
//!
//! `pure/` is `no_std` (alloc-only). Every match on [`Derivation`] and
//! [`TraceEntry`] inside this module is exhaustive — no `_ =>` wildcards.

mod deriver;
mod extract;
mod overlap;
mod types;

pub use deriver::Deriver;
pub use extract::{L1TxView, extract_l1_input};
pub use types::{
    BatchKind, BatchVerdict, CriticalError, Derivation, DeriveTrace, EmptyBatchReason,
    FrameDropReason, L1Input, SpanBatchOverlap, SpanBatchOverlapBlock, TraceEntry,
};
