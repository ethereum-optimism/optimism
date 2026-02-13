//! An abstraction for the driver's block executor.
//!
//! This module provides the [`Executor`] trait which abstracts block execution for the driver.
//! The executor is responsible for building and executing blocks from payload attributes,
//! maintaining safe head state, and computing output roots for the execution results.

use alloc::boxed::Box;
use alloy_consensus::{Header, Sealed};
use alloy_primitives::B256;
use async_trait::async_trait;
use core::error::Error;
use kona_executor::BlockBuildingOutcome;
use op_alloy_rpc_types_engine::OpPayloadAttributes;

/// Executor trait for block execution in the driver pipeline.
///
/// This trait abstracts the block execution functionality needed by the driver.
/// Implementations are responsible for:
/// - Building blocks from payload attributes
/// - Maintaining execution state and safe head tracking
/// - Computing output roots after block execution
/// - Handling execution errors and recovery scenarios
///
/// The executor operates in a sequential manner where blocks must be executed
/// in order to maintain proper state transitions.
#[async_trait]
pub trait Executor {
    /// The error type for the Executor.
    ///
    /// Should implement [`Error`] and provide detailed information about
    /// execution failures, including transaction-level errors and state issues.
    type Error: Error;

    /// Waits for the executor to be ready for block execution.
    ///
    /// This method blocks until the executor has completed any necessary
    /// initialization or synchronization required before it can begin
    /// processing blocks.
    ///
    /// # Usage
    /// This should be called before attempting to execute any payloads
    /// to ensure the executor is in a valid state.
    async fn wait_until_ready(&mut self);

    /// Updates the safe head to the specified header.
    ///
    /// Sets the executor's internal safe head state to the provided sealed header.
    /// This is used to establish the starting point for subsequent block execution.
    ///
    /// # Arguments
    /// * `header` - The sealed header to set as the new safe head
    ///
    /// # Usage
    /// This must be called before executing a payload to ensure the executor
    /// builds on the correct parent block.
    fn update_safe_head(&mut self, header: Sealed<Header>);

    /// Execute the given payload attributes to build and execute a block.
    ///
    /// Takes the provided payload attributes and builds a complete block,
    /// executing all transactions and computing the resulting state changes.
    ///
    /// # Arguments
    /// * `attributes` - The payload attributes containing transactions and metadata
    ///
    /// # Returns
    /// * `Ok(BlockBuildingOutcome)` - Successful execution result with the built block
    /// * `Err(Self::Error)` - Execution failure with detailed error information
    ///
    /// # Errors
    /// This method can fail due to:
    /// - Invalid transactions in the payload
    /// - Execution errors (e.g., out of gas, revert)
    /// - State inconsistencies
    /// - Block validation failures
    ///
    /// # Usage
    /// Must be called after setting the safe head with [`Self::update_safe_head`].
    /// The execution builds on the current safe head state.
    async fn execute_payload(
        &mut self,
        attributes: OpPayloadAttributes,
    ) -> Result<BlockBuildingOutcome, Self::Error>;

    /// Computes the output root for the most recently executed block.
    ///
    /// Calculates the Merkle root of the execution outputs which is used
    /// for verification and state commitment purposes.
    ///
    /// # Returns
    /// * `Ok(B256)` - The computed output root hash
    /// * `Err(Self::Error)` - Computation failure
    ///
    /// # Errors
    /// This method can fail if:
    /// - No block has been executed yet
    /// - State is inconsistent or corrupted
    /// - Output computation encounters an internal error
    ///
    /// # Usage
    /// Expected to be called immediately after successful payload execution
    /// via [`Self::execute_payload`]. The computed root corresponds to the state
    /// after the most recent block execution.
    fn compute_output_root(&mut self) -> Result<B256, Self::Error>;
}
