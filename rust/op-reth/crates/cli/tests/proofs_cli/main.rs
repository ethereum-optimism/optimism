//! Behavioral tests of the `op-reth proofs` subcommands.
//!
//! Every test seeds a real reth datadir with a short chain, runs the subcommands against it the
//! way the binary does, and asserts on the proof window and snapshot the proofs storage holds
//! afterward.

mod backfill;
mod fixture;
mod init;
mod prune;
mod snapshot;
mod unwind;
