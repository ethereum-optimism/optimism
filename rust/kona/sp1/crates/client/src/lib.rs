//! This crate contains the core logic for the sp1 proof.

pub mod boot;
pub use boot::AGGREGATION_OUTPUTS_SIZE;

mod oracle;
pub use oracle::BlobStore;

pub mod precompiles;

pub mod types;

extern crate alloc;

pub mod client;

pub mod witness;
