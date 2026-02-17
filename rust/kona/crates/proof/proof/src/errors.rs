//! Error types for the proof program.
//!
//! This module defines error types used throughout the proof system, including
//! oracle provider errors and hint parsing errors. These errors provide detailed
//! context about failures during proof generation and data retrieval.

use alloc::string::{String, ToString};
use kona_derive::{PipelineError, PipelineErrorKind};
use kona_mpt::{OrderedListWalkerError, TrieNodeError};
use kona_preimage::errors::PreimageOracleError;
use kona_protocol::{FromBlockError, OpBlockConversionError};
use thiserror::Error;

/// Error from an oracle-backed provider.
///
/// [`OracleProviderError`] represents various failure modes when interacting with
/// oracle-backed data providers. These errors can occur during preimage retrieval,
/// data parsing, or when validating oracle responses.
///
/// # Error Categories
/// - **Data Availability**: Block numbers beyond chain head, missing preimages
/// - **Communication**: Oracle communication failures, timeouts
/// - **Data Integrity**: Malformed trie data, invalid RLP encoding
/// - **Conversion**: Type conversion failures, slice length mismatches
/// - **Network**: Unknown chain IDs, unsupported configurations
#[derive(Error, Debug)]
pub enum OracleProviderError {
    /// Requested block number is past the current chain head.
    ///
    /// This error occurs when attempting to access block data for a block number
    /// that exceeds the highest block available in the chain. It typically indicates
    /// that the requested block has not been produced yet or the chain data is stale.
    ///
    /// # Arguments
    /// * `0` - The requested block number
    /// * `1` - The current chain head block number
    #[error("Block number ({0}) past chain head ({_1})")]
    BlockNumberPastHead(u64, u64),
    /// Preimage oracle communication or data retrieval error.
    ///
    /// This error wraps underlying oracle communication failures, including
    /// network timeouts, missing preimage keys, or invalid preimage data.
    /// It's the most common error type when oracle operations fail.
    #[error("Preimage oracle error: {0}")]
    Preimage(#[from] PreimageOracleError),
    /// Ordered list walker error during trie traversal.
    ///
    /// This error occurs when walking through ordered lists in Merkle Patricia
    /// tries fails due to malformed data, incorrect proofs, or missing nodes.
    /// It typically indicates corrupted trie data or invalid proof structures.
    #[error("Trie walker error: {0}")]
    TrieWalker(#[from] OrderedListWalkerError),
    /// Trie node parsing or validation error.
    ///
    /// This error occurs when processing individual trie nodes fails due to
    /// invalid node structure, incorrect hashing, or malformed node data.
    /// It indicates fundamental issues with the trie data integrity.
    #[error("Trie node error: {0}")]
    TrieNode(#[from] TrieNodeError),
    /// Block information extraction or conversion error.
    ///
    /// This error occurs when converting raw block data into structured
    /// [`kona_protocol::BlockInfo`] objects fails due to missing fields, invalid data
    /// formats, or unsupported block versions.
    #[error("From block error: {0}")]
    BlockInfo(FromBlockError),
    /// OP Stack specific block conversion error.
    ///
    /// This error occurs when converting between different OP Stack block
    /// formats fails due to incompatible data structures, missing OP-specific
    /// fields, or version mismatches between block formats.
    #[error("Op block conversion error: {0}")]
    OpBlockConversion(OpBlockConversionError),
    /// RLP (Recursive Length Prefix) encoding or decoding error.
    ///
    /// This error occurs when parsing or encoding RLP data fails due to
    /// malformed input, invalid length prefixes, or unsupported data types.
    /// RLP is used extensively for Ethereum data serialization.
    #[error("RLP error: {0}")]
    Rlp(alloy_rlp::Error),
    /// Slice to array conversion error.
    ///
    /// This error occurs when attempting to convert a byte slice to a fixed-size
    /// array fails due to length mismatches. It typically happens when parsing
    /// hash values or other fixed-length data from variable-length sources.
    #[error("Slice conversion error: {0}")]
    SliceConversion(core::array::TryFromSliceError),
    /// JSON serialization or deserialization error.
    ///
    /// This error occurs when parsing JSON data (e.g., rollup configurations)
    /// fails due to invalid JSON syntax, missing required fields, or type
    /// mismatches between expected and actual data structures.
    #[error("Serde error: {0}")]
    Serde(serde_json::Error),
    /// Unknown or unsupported chain ID.
    ///
    /// This error occurs when encountering a chain ID that is not recognized
    /// by the system. It typically happens when trying to load rollup
    /// configurations for networks that are not supported or configured.
    ///
    /// # Argument
    /// * `0` - The unknown chain ID that was encountered
    #[error("Unknown chain ID: {0}")]
    UnknownChainId(u64),
}

impl From<OracleProviderError> for PipelineErrorKind {
    fn from(val: OracleProviderError) -> Self {
        match val {
            OracleProviderError::BlockNumberPastHead(_, _) => PipelineError::EndOfSource.crit(),
            _ => PipelineError::Provider(val.to_string()).crit(),
        }
    }
}

/// Error parsing a hint from string format.
///
/// [`HintParsingError`] occurs when attempting to parse a hint string fails due to
/// invalid format, unknown hint types, or malformed hint data. Hints are expected
/// to follow the format `<hint_type> <hint_data>` where data is hex-encoded.
///
/// # Common Causes
/// - Invalid hint format (wrong number of parts, missing data)
/// - Unknown hint type strings that don't map to [`crate::HintType`] variants
/// - Malformed hex encoding in hint data
/// - Empty or malformed hint strings
///
/// # Example Error Scenarios
/// ```text
/// "invalid-hint-type 0x1234"      // Unknown hint type
/// "l1-block-header"               // Missing hint data
/// "l1-block-header invalid-hex"   // Invalid hex encoding
/// "too many parts here"           // Wrong format
/// ```
#[derive(Error, Debug)]
#[error("Hint parsing error: {_0}")]
pub struct HintParsingError(pub String);
