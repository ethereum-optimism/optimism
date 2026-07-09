//! This crate contains the core logic for the sp1 proof.

pub mod boot;

mod oracle;
pub use oracle::BlobStore;

pub mod precompiles;

pub mod range;

pub mod super_root;

pub mod types;

extern crate alloc;

pub mod client;

pub mod metrics;

pub mod witness;
