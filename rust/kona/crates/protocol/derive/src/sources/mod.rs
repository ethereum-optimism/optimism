//! The data source module.
//!
//! Data sources are data providers for the kona derivation pipeline.
//! They implement the [`DataAvailabilityProvider`](crate::traits::DataAvailabilityProvider) trait,
//! providing a way to iterate over data for a given (L2)
//! [`BlockInfo`](kona_protocol::BlockInfo).

mod blob_data;
pub use blob_data::BlobData;

mod ethereum;
pub use ethereum::EthereumDataSource;

mod blobs;
pub use blobs::BlobSource;

mod calldata;
pub use calldata::CalldataSource;
