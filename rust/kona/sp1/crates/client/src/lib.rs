//! This crate contains the core logic for the sp1 proof.

pub mod boot;

mod oracle;
pub use oracle::BlobStore;

pub mod precompiles;

pub mod range;

pub mod super_root;

#[cfg(any(test, feature = "aggregation-test-utils"))]
pub mod test_utils;

pub mod types;

extern crate alloc;

pub mod metrics;

pub mod witness;
