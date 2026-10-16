//! Module for witness generation components.

pub mod online_blob_store;
pub use online_blob_store::OnlineBlobStore;

pub mod preimage_witness_collector;
pub use preimage_witness_collector::PreimageWitnessCollector;

pub mod witness_size_limit;
pub use witness_size_limit::{BLOB_WITNESS_BYTES, PREIMAGE_ENTRY_OVERHEAD_BYTES, WitnessSizeLimit};
