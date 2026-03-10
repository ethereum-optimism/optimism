//! Cross-implementation derivation pipeline test framework for the OP Stack.
//!
//! This crate provides a high-level builder API for constructing deterministic
//! L1 and L2 chains, serving them over JSON-RPC, and verifying derivation
//! results across multiple implementations (op-program, kona-proofs, op-node,
//! kona-node).
//!
//! # Example
//!
//! ```rust,no_run
//! use derivation_tests::config::DeterministicConfig;
//!
//! let config = DeterministicConfig::default();
//! assert_eq!(config.l2_chain_id, 901);
//! ```

#![warn(missing_docs)]

pub mod batch;
pub mod config;
pub mod harness;
pub mod l1;
pub mod l2;
pub mod roots;
pub mod server;
pub mod state;
