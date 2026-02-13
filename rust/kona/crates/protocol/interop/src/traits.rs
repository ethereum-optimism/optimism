//! Traits for the `kona-interop` crate.

use crate::InteropValidationError;
use alloc::{boxed::Box, vec::Vec};
use alloy_consensus::Header;
use alloy_primitives::{B256, ChainId};
use async_trait::async_trait;
use core::error::Error;
use kona_protocol::BlockInfo;
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

/// Trait for validating interop-related timestamps and blocks.
pub trait InteropValidator: Send + Sync {
    /// Validates that the provided timestamps and chain IDs are eligible for interop execution.
    ///
    /// # Arguments
    /// * `initiating_chain_id` - The chain ID where the message was initiated
    /// * `initiating_timestamp` - The timestamp when the message was initiated
    /// * `executing_chain_id` - The chain ID where the message is being executed
    /// * `executing_timestamp` - The timestamp when the message is being executed
    /// * `timeout` - Optional timeout value to add to the execution deadline
    ///
    /// # Returns
    /// * `Ok(())` if the timestamps are valid for interop execution
    /// * `Err(InteropValidationError)` if validation fails
    fn validate_interop_timestamps(
        &self,
        initiating_chain_id: ChainId,
        initiating_timestamp: u64,
        executing_chain_id: ChainId,
        executing_timestamp: u64,
        timeout: Option<u64>,
    ) -> Result<(), InteropValidationError>;

    /// Returns `true` if the timestamp is strictly after the interop activation block.
    ///
    /// This function checks whether the provided timestamp is *after* that activation,
    /// skipping the activation block itself.
    ///
    /// Returns `false` if `interop_time` is not configured.
    fn is_post_interop(&self, chain_id: ChainId, timestamp: u64) -> bool;

    /// Returns `true` if the block is the interop activation block for the specified chain.
    ///
    /// An interop activation block is defined as the block that is right after the
    /// interop activation time.
    ///
    /// Returns `false` if `interop_time` is not configured.
    fn is_interop_activation_block(&self, chain_id: ChainId, block: BlockInfo) -> bool;
}
