//! Errors shared by RPC sources and their wrappers.
use alloy_eips::{BlockId, eip2718::Eip2718Error};
use alloy_primitives::B256;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_derive::{PipelineError, PipelineErrorKind, ResetError};

/// A failure fetching or verifying RPC consensus data.
#[allow(clippy::enum_variant_names)]
#[derive(Debug, thiserror::Error)]
pub enum SourceError {
    /// Transport error
    #[error(transparent)]
    Transport(#[from] RpcError<TransportErrorKind>),
    /// Block not found.
    #[error("Block not found: {0}")]
    BlockNotFound(BlockId),
    /// `debug_getRawReceipts` returned null for an existing or unknown block.
    #[error("debug_getRawReceipts returned null for L1 block {0}")]
    RawReceiptsNull(B256),
    /// A raw receipt failed EIP-2718 decoding.
    #[error("Failed to decode raw receipt {index} for L1 block {hash}: {source}")]
    RawReceiptDecode {
        /// The requested L1 block hash.
        hash: B256,
        /// The index of the invalid receipt in the RPC response.
        index: usize,
        /// The concrete EIP-2718 decoding error.
        #[source]
        source: Eip2718Error,
    },
    /// An empty raw receipt response was returned for a block that contains transactions.
    #[error(
        "debug_getRawReceipts returned no receipts for non-empty L1 block {hash} \
         (transactions root: {transactions_root})"
    )]
    EmptyRawReceiptsForNonEmptyBlock {
        /// The requested L1 block hash.
        hash: B256,
        /// The transaction root in the block header.
        transactions_root: B256,
    },
    /// Both receipt RPC methods failed.
    #[error("debug_getRawReceipts failed: {raw}; eth_getBlockReceipts fallback failed: {fallback}")]
    ReceiptMethodsFailed {
        /// The error returned by `debug_getRawReceipts`.
        raw: Box<Self>,
        /// The error returned by the `eth_getBlockReceipts` fallback.
        fallback: Box<Self>,
    },
}

impl SourceError {
    /// Returns whether this is the JSON-RPC `MethodNotFound` response.
    pub(crate) fn is_method_not_found(&self) -> bool {
        const METHOD_NOT_FOUND: i64 = -32601;

        let Self::Transport(error) = self else {
            return false;
        };
        let error_code =
            error.as_error_resp().map(|payload| payload.code).or_else(|| match error {
                // Some RPC services return JSON-RPC errors with a non-successful HTTP status, which
                // Alloy exposes as an HTTP transport error rather than an `ErrorResp`.
                RpcError::Transport(error) => {
                    error.as_http_error().and_then(|error| Self::error_code_from_body(&error.body))
                }
                // Also recognize an exact error code in a malformed JSON-RPC response. Alloy
                // normally promotes these to `ErrorResp`, but a response with an
                // invalid or missing ID may remain a deserialization error.
                RpcError::DeserError { text, .. } => Self::error_code_from_body(text),
                _ => None,
            });
        error_code == Some(METHOD_NOT_FOUND)
    }

    /// Extracts a JSON-RPC error code from either an error payload or a full response body.
    fn error_code_from_body(body: &str) -> Option<i64> {
        let value: serde_json::Value = serde_json::from_str(body).ok()?;
        value.get("error").unwrap_or(&value).get("code").and_then(serde_json::Value::as_i64)
    }

    /// Returns whether the standard receipt method confirmed that a hash is no longer available.
    pub(crate) const fn is_hash_block_not_found(&self) -> bool {
        matches!(self, Self::BlockNotFound(BlockId::Hash(_)))
    }
}

impl From<SourceError> for PipelineErrorKind {
    fn from(e: SourceError) -> Self {
        match e {
            SourceError::Transport(e) => {
                Self::Temporary(PipelineError::Provider(format!("Transport error: {e}")))
            }
            SourceError::BlockNotFound(id) => {
                // A hash-based lookup returning not-found means the block was reorged
                // out of the chain — retrying will never succeed, so reset.
                // A number-based lookup returning not-found means the next L1 block
                // hasn't been produced yet — this is transient, so Temporary.
                match id {
                    BlockId::Hash(_) => ResetError::BlockNotFound(id).reset(),
                    BlockId::Number(_) => Self::Temporary(PipelineError::Provider(format!(
                        "L1 Block not found: {id}"
                    ))),
                }
            }
            error @ (SourceError::RawReceiptsNull(_) |
            SourceError::RawReceiptDecode { .. } |
            SourceError::EmptyRawReceiptsForNonEmptyBlock { .. }) => {
                Self::Temporary(PipelineError::Provider(error.to_string()))
            }
            SourceError::ReceiptMethodsFailed { raw, fallback } => {
                let message = format!(
                    "debug_getRawReceipts failed: {raw}; eth_getBlockReceipts fallback failed: \
                     {fallback}"
                );

                // The standard method determines the derivation impact. A missing hash confirms
                // the possible reorg signaled by the preferred method and requires a reset. Every
                // other failure remains temporary so that receipt fetching can be retried.
                match *fallback {
                    SourceError::BlockNotFound(id @ BlockId::Hash(_)) => {
                        ResetError::BlockNotFound(id).reset()
                    }
                    _ => Self::Temporary(PipelineError::Provider(message)),
                }
            }
        }
    }
}
