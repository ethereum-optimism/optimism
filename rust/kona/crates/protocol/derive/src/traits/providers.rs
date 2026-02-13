//! Chain providers for the derivation pipeline.

use crate::PipelineErrorKind;
use alloc::{boxed::Box, sync::Arc, vec::Vec};
use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_primitives::B256;
use async_trait::async_trait;
use core::fmt::Display;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, BlockInfo};

/// Describes the functionality of a data source that can provide information from the blockchain.
#[async_trait]
pub trait ChainProvider {
    /// The error type for the [`ChainProvider`].
    type Error: Display + Into<PipelineErrorKind>;

    /// Fetch the L1 [`Header`] for the given [`B256`] hash.
    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error>;

    /// Returns the block at the given number, or an error if the block does not exist in the data
    /// source.
    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error>;

    /// Returns all receipts in the block with the given hash, or an error if the block does not
    /// exist in the data source.
    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error>;

    /// Returns the [`BlockInfo`] and list of [`TxEnvelope`]s from the given block hash.
    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error>;
}

/// Describes the functionality of a data source that fetches safe blocks.
#[async_trait]
pub trait L2ChainProvider: BatchValidationProviderDerive {
    /// The error type for the [`L2ChainProvider`].
    type Error: Display + Into<PipelineErrorKind>;

    /// Returns the [`SystemConfig`] by L2 number.
    async fn system_config_by_number(
        &mut self,
        number: u64,
        rollup_config: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as L2ChainProvider>::Error>;
}

/// A super-trait for [`BatchValidationProvider`] that binds `Self::Error` to have a conversion into
/// [`PipelineErrorKind`].
pub trait BatchValidationProviderDerive: BatchValidationProvider {}

// Auto-implement the [BatchValidationProviderDerive] trait for all types that implement
// [BatchValidationProvider] where the error can be converted into [PipelineErrorKind].
impl<T> BatchValidationProviderDerive for T
where
    T: BatchValidationProvider,
    <T as BatchValidationProvider>::Error: Into<PipelineErrorKind>,
{
}
