//! Sync (IO-free) derivation building blocks.
//!
//! This module hosts the post-Holocene derivation primitives that have no IO
//! dependency. The existing async stages in `stages/` delegate their pure
//! state-machine work here; phase 3 will build `pure::Deriver` on top of the
//! same primitives.
//!
//! Visibility is `pub(crate)` through phase 5. Phase 6b promotes the module
//! to `pub` as a stable seam for future extraction into a dedicated
//! `kona-core` crate.
//!
//! Rules for code that lives here:
//! - **No async.** `core::*` is sync only.
//! - **No `tracing::*`.** Callers (async stages today, `pure::Deriver` tomorrow) translate outcomes
//!   to traces or logs at their own level.
//! - **`no_std`-clean.** Verified by the workspace's `no_std` build target.

// Some `core::*` items are consumed only by the async stages today (with the
// `async` Cargo feature on); the `--no-default-features` build legitimately
// drags in unused symbols. Phase 3 wires them all up via `pure::Deriver`.
#![cfg_attr(not(feature = "async"), allow(dead_code))]

pub(crate) mod attributes;
pub(crate) mod batch;
pub(crate) mod channel;
