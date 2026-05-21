//! Traits for the `kona-interop` crate.

use alloc::{boxed::Box, vec::Vec};
use alloy_consensus::Header;
use alloy_primitives::B256;
use async_trait::async_trait;
use core::error::Error;
use op_alloy_consensus::OpReceiptEnvelope;

/// Describes the interface of the interop data provider. This provider is multiplexed over several
/// chains, with each method consuming a chain ID to determine the target chain.
#[async_trait]
pub trait InteropProvider {
    /// The error type for the provider.
    type Error: Error;

    /// Fetch a [Header] by its number.
    async fn header_by_number(&self, chain_id: u64, number: u64) -> Result<Header, Self::Error>;

    /// Fetch all receipts for a given block by number.
    async fn receipts_by_number(
        &self,
        chain_id: u64,
        number: u64,
    ) -> Result<Vec<OpReceiptEnvelope>, Self::Error>;

    /// Fetch all receipts for a given block by hash.
    async fn receipts_by_hash(
        &self,
        chain_id: u64,
        block_hash: B256,
    ) -> Result<Vec<OpReceiptEnvelope>, Self::Error>;
}
