//! This module contains derivation errors thrown within the pipeline.

use crate::BuilderError;
use alloc::string::String;
use alloy_primitives::B256;
use kona_genesis::SystemConfigUpdateError;
use kona_protocol::{DepositError, SpanBatchError};
use thiserror::Error;

/// [crate::ensure] is a short-hand for bubbling up errors in the case of a condition not being met.
#[macro_export]
macro_rules! ensure {
    ($cond:expr, $err:expr) => {
        if !($cond) {
            return Err($err);
        }
    };
}

/// A top-level severity filter for [`PipelineError`] that categorizes errors by handling strategy.
///
/// The [`PipelineErrorKind`] wrapper provides a severity classification system that enables
/// sophisticated error handling in the derivation pipeline. Different error types require
/// different response strategies:
///
/// - **Temporary**: Retry-able errors that may resolve with more data
/// - **Critical**: Fatal errors that require external intervention
/// - **Reset**: Errors that require pipeline state reset but allow continued operation
///
/// # Error Handling Strategy
/// ```text
/// Temporary -> Retry operation, may succeed with more data
/// Critical  -> Stop derivation, external intervention required  
/// Reset     -> Reset pipeline state, continue with clean slate
/// ```
///
/// # Usage in Pipeline
/// Error kinds are used by pipeline stages to determine appropriate error handling:
/// - Temporary errors trigger retries in the main derivation loop
/// - Critical errors halt derivation and bubble up to the caller
/// - Reset errors trigger pipeline resets with appropriate recovery logic
#[derive(Error, Debug, PartialEq, Eq)]
pub enum PipelineErrorKind {
    /// A temporary error that may resolve with additional data or time.
    ///
    /// Temporary errors indicate transient conditions such as insufficient data,
    /// network timeouts, or resource unavailability. These errors suggest that
    /// retrying the operation may succeed once the underlying condition resolves.
    ///
    /// # Examples
    /// - Not enough L1 data available yet
    /// - Network communication timeouts
    /// - Insufficient channel data for frame assembly
    ///
    /// # Handling
    /// The pipeline typically retries temporary errors in a loop, waiting for
    /// conditions to improve or for additional data to become available.
    #[error("Temporary error: {0}")]
    Temporary(#[source] PipelineError),
    /// A critical error that requires external intervention to resolve.
    ///
    /// Critical errors indicate fundamental issues that cannot be resolved through
    /// retries or pipeline resets. These errors require external intervention such
    /// as updated L1 data, configuration changes, or system fixes.
    ///
    /// # Examples
    /// - Data source completely exhausted
    /// - Fundamental configuration errors
    /// - Irrecoverable data corruption
    ///
    /// # Handling
    /// Critical errors halt the derivation process and are returned to the caller
    /// for external resolution. The pipeline cannot continue without intervention.
    #[error("Critical error: {0}")]
    Critical(#[source] PipelineError),
    /// A reset error that requires pipeline state reset but allows continued operation.
    ///
    /// Reset errors indicate conditions that invalidate the current pipeline state
    /// but can be resolved by resetting to a known good state and continuing
    /// derivation. These typically occur due to chain reorganizations or state
    /// inconsistencies.
    ///
    /// # Examples
    /// - L1 chain reorganization detected
    /// - Block hash mismatches indicating reorg
    /// - Hard fork activation requiring state reset
    ///
    /// # Handling
    /// Reset errors trigger pipeline state cleanup and reset to a safe state,
    /// after which derivation can continue with fresh state.
    #[error("Pipeline reset: {0}")]
    Reset(#[from] ResetError),
}

/// An error encountered during derivation pipeline processing.
///
/// [`PipelineError`] represents specific error conditions that can occur during the
/// various stages of L2 block derivation from L1 data. Each error variant provides
/// detailed context about the failure mode and suggests appropriate recovery strategies.
///
/// # Error Categories
///
/// ## Data Availability Errors
/// - [`Self::Eof`]: No more data available from source
/// - [`Self::NotEnoughData`]: Insufficient data for current operation
/// - [`Self::MissingL1Data`]: Required L1 data not available
/// - [`Self::EndOfSource`]: Data source completely exhausted
///
/// ## Stage-Specific Errors  
/// - [`Self::ChannelProviderEmpty`]: No channels available for processing
/// - [`Self::ChannelReaderEmpty`]: Channel reader has no data
/// - [`Self::BatchQueueEmpty`]: No batches available for processing
///
/// ## Validation Errors
/// - [`Self::InvalidBatchType`]: Unsupported or malformed batch type
/// - [`Self::InvalidBatchValidity`]: Batch failed validation checks
/// - [`Self::BadEncoding`]: Data decoding/encoding failures
///
/// ## System Errors
/// - [`Self::SystemConfigUpdate`]: System configuration update failures
/// - [`Self::AttributesBuilder`]: Block attribute construction failures
/// - [`Self::Provider`]: External provider communication failures
#[derive(Error, Debug, PartialEq, Eq)]
pub enum PipelineError {
    /// End of file: no more data available from the channel bank.
    ///
    /// This error indicates that the channel bank has been completely drained
    /// and no additional frame data is available for processing. It typically
    /// occurs at the end of a derivation sequence when all available L1 data
    /// has been consumed.
    ///
    /// # Recovery
    /// Usually indicates completion of derivation for available data. May
    /// require waiting for new L1 blocks to provide additional frame data.
    #[error("EOF")]
    Eof,
    /// Insufficient data available to complete the current processing stage.
    ///
    /// This error indicates that the current operation requires more data than
    /// is currently available, but additional data may become available in the
    /// future. It suggests that retrying the operation later may succeed.
    ///
    /// # Common Scenarios
    /// - Partial frame received, waiting for completion
    /// - Channel assembly requires more frames
    /// - Batch construction needs additional channel data
    ///
    /// # Recovery
    /// Retry the operation after more L1 data becomes available or after
    /// waiting for network propagation delays.
    #[error("Not enough data")]
    NotEnoughData,
    /// No channels are available in the [`ChannelProvider`].
    ///
    /// This error occurs when the channel provider stage has no assembled
    /// channels ready for reading. It typically indicates that frame assembly
    /// is still in progress or that no valid channels have been constructed
    /// from available L1 data.
    ///
    /// [`ChannelProvider`]: crate::stages::ChannelProvider
    #[error("The channel provider is empty")]
    ChannelProviderEmpty,
    /// The channel has already been fully processed by the [`ChannelAssembler`] stage.
    ///
    /// This error indicates an attempt to reprocess a channel that has already
    /// been assembled and consumed. It suggests a logic error in channel tracking
    /// or an attempt to double-process the same channel data.
    ///
    /// [`ChannelAssembler`]: crate::stages::ChannelAssembler
    #[error("Channel already built")]
    ChannelAlreadyBuilt,
    /// Failed to locate the requested channel in the [`ChannelProvider`].
    ///
    /// This error occurs when attempting to access a specific channel that
    /// is not available in the channel provider's cache or storage. It may
    /// indicate a channel ID mismatch or premature channel eviction.
    ///
    /// [`ChannelProvider`]: crate::stages::ChannelProvider
    #[error("Channel not found in channel provider")]
    ChannelNotFound,
    /// No channel data returned by the [`ChannelReader`] stage.
    ///
    /// This error indicates that the channel reader stage has no channels
    /// available for reading. It typically occurs when all channels have
    /// been consumed or when no valid channels have been assembled yet.
    ///
    /// [`ChannelReader`]: crate::stages::ChannelReader
    #[error("The channel reader has no channel available")]
    ChannelReaderEmpty,
    /// The [`BatchQueue`] contains no batches ready for processing.
    ///
    /// This error occurs when the batch queue stage has no assembled batches
    /// available for attribute generation. It indicates that batch assembly
    /// is still in progress or that no valid batches have been constructed.
    ///
    /// [`BatchQueue`]: crate::stages::BatchQueue
    #[error("The batch queue has no batches available")]
    BatchQueueEmpty,
    /// Required L1 origin information is missing from the previous pipeline stage.
    ///
    /// This error indicates a pipeline stage dependency violation where a stage
    /// expects L1 origin information that wasn't provided by the preceding stage.
    /// It suggests a configuration or sequencing issue in the pipeline setup.
    #[error("Missing L1 origin from previous stage")]
    MissingOrigin,
    /// Required L1 data is missing from the [`L1Retrieval`] stage.
    ///
    /// This error occurs when the L1 retrieval stage cannot provide the
    /// requested L1 block data, transactions, or receipts. It may indicate
    /// network issues, data availability problems, or L1 node synchronization lag.
    ///
    /// [`L1Retrieval`]: crate::stages::L1Retrieval
    #[error("L1 Retrieval missing data")]
    MissingL1Data,
    /// Invalid or unsupported batch type encountered during processing.
    ///
    /// This error occurs when a pipeline stage receives a batch type that
    /// it cannot process or that violates the expected batch format. It
    /// indicates either malformed L1 data or unsupported batch versions.
    #[error("Invalid batch type passed to stage")]
    InvalidBatchType,
    /// Batch failed validation checks during processing.
    ///
    /// This error indicates that a batch contains invalid data that fails
    /// validation rules such as timestamp constraints, parent hash checks,
    /// or format requirements. It suggests potentially malicious or corrupted L1 data.
    #[error("Invalid batch validity")]
    InvalidBatchValidity,
    /// [`SystemConfig`] update operation failed.
    ///
    /// This error occurs when attempting to update the system configuration
    /// fails due to invalid parameters, version mismatches, or other
    /// configuration-related issues.
    ///
    /// [`SystemConfig`]: kona_genesis::SystemConfig
    #[error("Error updating system config: {0}")]
    SystemConfigUpdate(SystemConfigUpdateError),
    /// Block attributes construction failed with detailed error information.
    ///
    /// This error wraps [`BuilderError`] variants that occur during the
    /// construction of block attributes from batch data. It indicates issues
    /// with attribute validation, formatting, or consistency checks.
    #[error("Attributes builder error: {0}")]
    AttributesBuilder(#[from] BuilderError),
    /// Data encoding or decoding operation failed.
    ///
    /// This error wraps [`PipelineEncodingError`] variants that occur during
    /// serialization or deserialization of pipeline data structures. It
    /// indicates malformed input data or encoding format violations.
    #[error("Decode error: {0}")]
    BadEncoding(#[from] PipelineEncodingError),
    /// The data source has been completely exhausted and cannot provide more data.
    ///
    /// This error indicates that the underlying L1 data source has reached
    /// its end and no additional data will become available. It typically
    /// occurs when derivation has caught up to the L1 chain head.
    #[error("Data source exhausted")]
    EndOfSource,
    /// External provider communication or operation failed.
    ///
    /// This error wraps failures from external data providers such as L1
    /// nodes, blob providers, or other data sources. It includes network
    /// failures, API errors, and provider-specific issues.
    #[error("Provider error: {0}")]
    Provider(String),
    /// The pipeline received an unsupported signal type.
    ///
    /// This error occurs when a pipeline stage receives a signal that it
    /// cannot process or that is not supported in the current configuration.
    /// It indicates a protocol version mismatch or configuration issue.
    #[error("Unsupported signal")]
    UnsupportedSignal,
}

impl PipelineError {
    /// Wraps this [`PipelineError`] as a [PipelineErrorKind::Critical].
    ///
    /// Critical errors indicate fundamental issues that cannot be resolved through
    /// retries or pipeline resets. They require external intervention to resolve.
    ///
    /// # Usage
    /// Use this method when an error condition is unrecoverable and requires
    /// halting the derivation process for external intervention.
    ///
    /// # Example
    /// ```rust,ignore
    /// if data_source_corrupted {
    ///     return Err(PipelineError::Provider("corrupted data".to_string()).crit());
    /// }
    /// ```
    pub const fn crit(self) -> PipelineErrorKind {
        PipelineErrorKind::Critical(self)
    }

    /// Wraps this [`PipelineError`] as a [PipelineErrorKind::Temporary].
    ///
    /// Temporary errors indicate transient conditions that may resolve with
    /// additional data, time, or retries. The pipeline can attempt to recover
    /// by retrying the operation.
    ///
    /// # Usage
    /// Use this method when an error condition might resolve if the operation
    /// is retried, particularly for data availability or network issues.
    ///
    /// # Example
    /// ```rust,ignore
    /// if insufficient_data {
    ///     return Err(PipelineError::NotEnoughData.temp());
    /// }
    /// ```
    pub const fn temp(self) -> PipelineErrorKind {
        PipelineErrorKind::Temporary(self)
    }
}

/// A reset error
#[derive(Error, Clone, Debug, Eq, PartialEq)]
pub enum ResetError {
    /// The batch has a bad parent hash.
    /// The first argument is the expected parent hash, and the second argument is the actual
    /// parent hash.
    #[error("Bad parent hash: expected {0}, got {1}")]
    BadParentHash(B256, B256),
    /// The batch has a bad timestamp.
    /// The first argument is the expected timestamp, and the second argument is the actual
    /// timestamp.
    #[error("Bad timestamp: expected {0}, got {1}")]
    BadTimestamp(u64, u64),
    /// L1 origin mismatch.
    #[error("L1 origin mismatch. Expected {0:?}, got {1:?}")]
    L1OriginMismatch(u64, u64),
    /// The stage detected a block reorg.
    /// The first argument is the expected block hash.
    /// The second argument is the parent_hash of the next l1 origin block.
    #[error("L1 reorg detected: expected {0}, got {1}")]
    ReorgDetected(B256, B256),
    /// Attributes builder error variant, with [`BuilderError`].
    #[error("Attributes builder error: {0}")]
    AttributesBuilder(#[from] BuilderError),
    /// A Holocene activation temporary error.
    #[error("Holocene activation reset")]
    HoloceneActivation,
    /// The next l1 block provided to the managed traversal stage is not the expected one.
    #[error("Next L1 block hash mismatch: expected {0}, got {1}")]
    NextL1BlockHashMismatch(B256, B256),
}

impl ResetError {
    /// Wrap [`ResetError`] as a [PipelineErrorKind::Reset].
    pub const fn reset(self) -> PipelineErrorKind {
        PipelineErrorKind::Reset(self)
    }
}

/// A decoding error.
#[derive(Error, Debug, PartialEq, Eq)]
pub enum PipelineEncodingError {
    /// The buffer is empty.
    #[error("Empty buffer")]
    EmptyBuffer,
    /// Deposit decoding error.
    #[error("Error decoding deposit: {0}")]
    DepositError(#[from] DepositError),
    /// Alloy RLP Encoding Error.
    #[error("RLP error: {0}")]
    AlloyRlpError(alloy_rlp::Error),
    /// Span Batch Error.
    #[error("{0}")]
    SpanBatchError(#[from] SpanBatchError),
}

#[cfg(test)]
mod tests {
    use super::*;
    use core::error::Error;

    #[test]
    fn test_pipeline_error_kind_source() {
        let err = PipelineErrorKind::Temporary(PipelineError::Eof);
        assert!(err.source().is_some());

        let err = PipelineErrorKind::Critical(PipelineError::Eof);
        assert!(err.source().is_some());

        let err = PipelineErrorKind::Reset(ResetError::BadParentHash(
            Default::default(),
            Default::default(),
        ));
        assert!(err.source().is_some());
    }

    #[test]
    fn test_pipeline_error_source() {
        let err = PipelineError::AttributesBuilder(BuilderError::BlockMismatch(
            Default::default(),
            Default::default(),
        ));
        assert!(err.source().is_some());

        let encoding_err = PipelineEncodingError::EmptyBuffer;
        let err: PipelineError = encoding_err.into();
        assert!(err.source().is_some());

        let err = PipelineError::Eof;
        assert!(err.source().is_none());
    }

    #[test]
    fn test_pipeline_encoding_error_source() {
        let err = PipelineEncodingError::DepositError(DepositError::UnexpectedTopicsLen(0));
        assert!(err.source().is_some());

        let err = SpanBatchError::TooBigSpanBatchSize;
        let err: PipelineEncodingError = err.into();
        assert!(err.source().is_some());

        let err = PipelineEncodingError::EmptyBuffer;
        assert!(err.source().is_none());
    }

    #[test]
    fn test_reset_error_kinds() {
        let reset_errors = [
            ResetError::BadParentHash(Default::default(), Default::default()),
            ResetError::BadTimestamp(0, 0),
            ResetError::L1OriginMismatch(0, 0),
            ResetError::ReorgDetected(Default::default(), Default::default()),
            ResetError::AttributesBuilder(BuilderError::BlockMismatch(
                Default::default(),
                Default::default(),
            )),
            ResetError::HoloceneActivation,
        ];
        for error in reset_errors.into_iter() {
            let expected = PipelineErrorKind::Reset(error.clone());
            assert_eq!(error.reset(), expected);
        }
    }
}
