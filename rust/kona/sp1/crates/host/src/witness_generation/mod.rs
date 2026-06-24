//! Module for witness generation components.

pub mod traits;
pub use traits::{DefaultOracleBase, WitnessGenerator};

pub mod online_blob_store;
pub use online_blob_store::OnlineBlobStore;

pub mod preimage_witness_collector;
pub use preimage_witness_collector::PreimageWitnessCollector;
