//! Compact codec implementations for Optimism types.
//!
//! This module provides newtype wrappers around `op-alloy-consensus` types
//! and implements the `reth-codecs` `Compact` trait for them.
//!
//! These wrappers exist to satisfy the Rust orphan rule: neither `Compact` (from `reth-codecs`)
//! nor the OP types (from `op-alloy-consensus`) are defined in this crate, so we need local
//! newtypes to implement the trait.

pub mod receipt;
pub mod transaction;
