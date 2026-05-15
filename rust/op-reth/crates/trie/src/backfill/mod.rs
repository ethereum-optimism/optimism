//! Backfill: extend the proofs window from `[earliest, latest]` to
//! `[target_earliest, latest]` where `target_earliest < earliest`. See
//! [`BackfillJob`] for the per-step implementation.
//!
//! `earliest` is a **base-state boundary**, not "oldest block with its own
//! changeset rows". To move it from `E` to `E-1` the job materializes block
//! `E`'s historical records (changesets + history-bitmap entries) and then
//! flips the marker — mirroring prune in reverse.
//!
//! ## Invariants per step
//!
//! - The proofs current-state tables are untouched; only history is written.
//! - `earliest` decreases by exactly one per successful step.
//! - Each step commits atomically, so a crash mid-backfill resumes cleanly from the current
//!   `earliest`.

mod changesets;
mod error;
mod job;

#[cfg(test)]
mod tests;

pub use error::BackfillError;
pub use job::BackfillJob;
