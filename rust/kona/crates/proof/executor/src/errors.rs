//! Error types for the `kona-executor` crate.
//!
//! This module provides comprehensive error handling for the stateless L2 block
//! execution engine, covering validation errors, execution failures, and
//! database operation errors.

use alloc::string::String;
use alloy_evm::block::BlockExecutionError;
use kona_mpt::TrieNodeError;
use op_alloy_consensus::EIP1559ParamError;
use revm::context::DBErrorMarker;
use thiserror::Error;

/// Errors that can occur when validating EIP-1559 parameters from block header extra data.
///
/// This error type is used for validation errors during decoding of EIP-1559 parameters,
/// providing more specific error variants than the upstream [`EIP1559ParamError`] which
/// is primarily designed for encoding errors.
#[derive(Error, Debug, PartialEq, Eq)]
pub enum Eip1559ValidationError {
    /// The EIP-1559 denominator cannot be zero as it would cause division by zero.
    ///
    /// This error occurs when the decoded denominator value from the block header's
    /// extra data field is zero, which violates protocol specifications.
    #[error("EIP-1559 denominator cannot be zero")]
    ZeroDenominator,
    /// Error from EIP-1559 parameter decoding.
    ///
    /// This variant wraps errors from the underlying EIP-1559 parameter decoding,
    /// such as invalid version bytes, incorrect data lengths, or missing parameters.
    #[error(transparent)]
    Decode(#[from] EIP1559ParamError),
}

/// Error type for the [`StatelessL2Builder`] block execution engine.
///
/// [`ExecutorError`] represents various failure modes that can occur during
/// stateless L2 block building and execution. These errors provide detailed
/// context for debugging execution failures and enable appropriate error
/// handling strategies.
///
/// # Error Categories
///
/// ## Input Validation Errors
/// - Missing required fields in payload attributes
/// - Invalid block header parameters
/// - Unsupported transaction types
///
/// ## Execution Errors  
/// - Block gas limit violations
/// - Transaction execution failures
/// - EVM-level execution errors
///
/// ## Data Integrity Errors
/// - Trie database operation failures
/// - RLP encoding/decoding errors
/// - Cryptographic signature recovery failures
///
/// [`StatelessL2Builder`]: crate::StatelessL2Builder
#[derive(Error, Debug)]
pub enum ExecutorError {
    /// Gas limit not provided in the payload attributes.
    ///
    /// This error occurs when the payload attributes are missing the required
    /// gas limit field, which is necessary for block execution validation.
    /// The gas limit defines the maximum amount of gas that can be consumed
    /// by all transactions in the block.
    ///
    /// # Common Causes
    /// - Malformed payload attributes from the driver
    /// - Missing fields in payload construction
    /// - Protocol version mismatches
    #[error("Gas limit not provided in payload attributes")]
    MissingGasLimit,
    /// Transactions list not provided in the payload attributes.
    ///
    /// This error occurs when the payload attributes don't include the
    /// required transactions list. Even empty blocks require an empty
    /// transactions array to be explicitly provided.
    ///
    /// # Common Causes
    /// - Incomplete payload attribute construction
    /// - Serialization/deserialization errors
    /// - Protocol specification violations
    #[error("Transactions not provided in payload attributes")]
    MissingTransactions,
    /// EIP-1559 fee parameters missing in execution payload after Holocene activation.
    ///
    /// Post-Holocene blocks require EIP-1559 base fee and blob base fee parameters
    /// to be present in the execution payload for proper fee market operation.
    /// This error indicates these required parameters are missing.
    ///
    /// # Common Causes
    /// - Holocene upgrade not properly implemented in payload construction
    /// - Missing fee parameter calculation
    /// - Incorrect hard fork activation detection
    #[error("Missing EIP-1559 parameters in execution payload post-Holocene")]
    MissingEIP1559Params,
    /// Parent beacon block root not provided in the payload attributes.
    ///
    /// This error occurs when the payload attributes are missing the parent
    /// beacon block root, which is required for post-Dencun blocks to enable
    /// proper beacon chain integration and blob transaction validation.
    ///
    /// # Common Causes
    /// - Missing beacon chain data in payload construction
    /// - Incorrect Dencun upgrade implementation
    /// - Beacon client communication failures
    #[error("Parent beacon block root not provided in payload attributes")]
    MissingParentBeaconBlockRoot,
    /// Invalid `extraData` field in the block header.
    ///
    /// This error occurs when the block header's `extraData` field contains
    /// invalid data that violates protocol specifications. The `extraData`
    /// field has specific format requirements depending on the network.
    ///
    /// # Common Causes
    /// - Malformed extraData during header construction
    /// - Incorrect protocol version handling
    /// - Data corruption during header assembly
    /// - Zero denominator in EIP-1559 parameters
    #[error("Invalid `extraData` field in the block header: {0}")]
    InvalidExtraData(#[from] Eip1559ValidationError),
    /// Block gas limit exceeded during execution.
    ///
    /// This error occurs when the cumulative gas consumption of all transactions
    /// in the block exceeds the block's gas limit. This violates consensus rules
    /// and indicates either invalid transaction inclusion or incorrect gas accounting.
    ///
    /// # Common Causes
    /// - Transaction gas estimation errors
    /// - Incorrect gas limit calculation
    /// - Invalid transaction ordering
    /// - Gas accounting bugs in execution
    #[error("Block gas limit exceeded")]
    BlockGasLimitExceeded,
    /// Unsupported transaction type encountered during execution.
    ///
    /// This error occurs when the executor encounters a transaction type that
    /// it doesn't know how to process. This may indicate a protocol upgrade
    /// that hasn't been implemented or invalid transaction data.
    ///
    /// # Arguments
    /// * `0` - The unsupported transaction type identifier
    ///
    /// # Common Causes
    /// - New transaction types not yet supported
    /// - Corrupted transaction data
    /// - Protocol version mismatches
    #[error("Unsupported transaction type: {0}")]
    UnsupportedTransactionType(u8),
    /// Trie database operation failed.
    ///
    /// This error wraps [`TrieDBError`] variants that occur during state
    /// tree operations, including node lookups, proof verification, and
    /// state root computation.
    #[error("Trie error: {0}")]
    TrieDBError(#[from] TrieDBError),
    /// Block execution failed at the EVM level.
    ///
    /// This error wraps [`BlockExecutionError`] variants that occur during
    /// the actual execution of transactions within the block, including
    /// transaction failures, state conflicts, and EVM-level errors.
    #[error("Execution error: {0}")]
    ExecutionError(#[from] BlockExecutionError),
    /// Cryptographic signature recovery failed.
    ///
    /// This error occurs when attempting to recover the sender address from
    /// a transaction signature fails due to invalid signatures, malformed
    /// signature data, or cryptographic errors.
    ///
    /// # Common Causes
    /// - Invalid transaction signatures
    /// - Corrupted signature data
    /// - Unsupported signature algorithms
    /// - Chain ID mismatches
    #[error("sender recovery error: {0}")]
    Recovery(#[from] alloy_consensus::crypto::RecoveryError),
    /// RLP encoding or decoding error.
    ///
    /// This error occurs when RLP (Recursive Length Prefix) serialization
    /// or deserialization fails due to malformed data, invalid length
    /// prefixes, or unsupported data structures.
    ///
    /// # Common Causes
    /// - Corrupted transaction or block data
    /// - Invalid RLP formatting
    /// - Unsupported data types in RLP stream
    #[error("RLP error: {0}")]
    RLPError(alloy_eips::eip2718::Eip2718Error),
    /// Executor instance not available when required.
    ///
    /// This error occurs when attempting to perform an operation that requires
    /// an active executor instance, but none is available. This typically
    /// indicates a lifecycle or initialization issue.
    ///
    /// # Common Causes
    /// - Executor not properly initialized
    /// - Executor already consumed or dropped
    /// - Incorrect executor lifecycle management
    #[error("Missing the executor")]
    MissingExecutor,
}

/// Result type alias for operations that may fail with [`ExecutorError`].
///
/// This type alias provides a convenient way to handle executor operations
/// that can fail, encapsulating the success value type `T` and the standard
/// [`ExecutorError`] for failures.
///
/// # Usage
/// ```rust,ignore
/// fn build_block(attrs: PayloadAttributes) -> ExecutorResult<BlockBuildingOutcome> {
///     // ... block building logic
/// }
/// ```
pub type ExecutorResult<T> = Result<T, ExecutorError>;

/// Result type alias for trie database operations that may fail with [`TrieDBError`].
///
/// This type alias provides a convenient way to handle trie database operations
/// that can fail, specifically for operations involving state tree access,
/// proof verification, and account data retrieval.
///
/// # Usage
/// ```rust,ignore
/// fn get_account_info(address: Address) -> TrieDBResult<AccountInfo> {
///     // ... trie database lookup logic
/// }
/// ```
pub type TrieDBResult<T> = Result<T, TrieDBError>;

/// Error type for [`TrieDB`] database operations.
///
/// [`TrieDBError`] represents failures that can occur during trie database
/// operations, including state tree traversal, proof verification, and
/// account data retrieval. These errors are critical for stateless execution
/// as they indicate issues with the underlying state data.
///
/// # Error Categories
///
/// ## State Tree Errors
/// - Blinded trie node issues
/// - Missing account information
/// - Trie node corruption or invalid proofs
///
/// ## Provider Errors
/// - Communication failures with trie data providers
/// - Missing or corrupted state data
/// - Network or I/O related issues
///
/// [`TrieDB`]: crate::TrieDB
#[derive(Error, Debug, PartialEq, Eq)]
pub enum TrieDBError {
    /// Trie root node has not been properly blinded for stateless execution.
    ///
    /// This error occurs when attempting to perform stateless execution on a trie
    /// where the root node hasn't been blinded (converted to a form suitable for
    /// stateless proofs). Blinding is required to enable stateless verification.
    ///
    /// # Common Causes
    /// - Incorrect trie preparation for stateless execution
    /// - Missing proof data or invalid proof format
    /// - State tree not properly configured for witness-based execution
    #[error("Trie root node has not been blinded")]
    RootNotBlinded,
    /// Account information missing for a bundle account during state access.
    ///
    /// This error occurs when the trie database cannot find required account
    /// information during execution. In stateless execution, all required
    /// account data must be provided via witnesses or proofs.
    ///
    /// # Common Causes
    /// - Incomplete witness data for accessed accounts
    /// - Missing account proofs in the witness
    /// - Account state changes not properly tracked
    /// - Proof verification failures
    #[error("Missing account info for bundle account.")]
    MissingAccountInfo,
    /// Trie node operation failed due to invalid node data or proof.
    ///
    /// This error wraps [`TrieNodeError`] variants that occur during individual
    /// trie node operations, including hash verification, node parsing, and
    /// proof validation.
    ///
    /// # Common Causes
    /// - Corrupted trie node data
    /// - Invalid Merkle proofs
    /// - Hash mismatches in trie verification
    /// - Malformed node structure
    #[error("Trie node error: {0}")]
    TrieNode(#[from] TrieNodeError),
    /// Trie data provider communication or operation failed.
    ///
    /// This error occurs when the underlying trie data provider fails to
    /// retrieve or validate trie data. It includes network failures, data
    /// corruption, and provider-specific errors.
    ///
    /// # Common Causes
    /// - Network communication failures
    /// - Provider service unavailable
    /// - Data corruption in provider storage
    /// - Timeout during data retrieval
    #[error("Trie provider error: {0}")]
    Provider(String),
}

impl DBErrorMarker for TrieDBError {}

impl From<EIP1559ParamError> for ExecutorError {
    fn from(err: EIP1559ParamError) -> Self {
        Self::InvalidExtraData(Eip1559ValidationError::Decode(err))
    }
}
