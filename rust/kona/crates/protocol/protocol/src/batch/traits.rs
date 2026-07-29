//! Traits for working with protocol types.

use alloc::boxed::Box;
use async_trait::async_trait;
use core::fmt::Display;
use op_alloy_consensus::OpBlock;

use crate::L2BlockInfo;

/// Describes the functionality of a data source that fetches safe blocks.
#[async_trait]
pub trait BatchValidationProvider {
    /// The error type for the [`BatchValidationProvider`].
    type Error: Display;

    /// Returns the [`L2BlockInfo`] given a block number.
    ///
    /// Errors if the block does not exist.
    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error>;

    /// Returns the [`OpBlock`] for a given number.
    ///
    /// Errors if no block is available for the given block number.
    async fn block_by_number(&mut self, number: u64) -> Result<OpBlock, Self::Error>;
}
