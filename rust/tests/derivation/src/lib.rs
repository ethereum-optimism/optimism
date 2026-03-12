//! Derivation pipeline equivalence test harness.
//!
//! Tests that the kona-node-service derivation path and the kona-proof fault proof
//! path produce identical output roots for the same L2 chain state.

pub mod genesis;
pub mod harness;
pub mod mock_batcher;
pub mod mock_l1;
pub mod mock_sequencer;
pub mod node_actor;

// Phase 2: fault proof actor path
// pub mod fault_proof_actor;
