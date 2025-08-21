use crate::{ExecutionPayloadBaseV1, ExecutionPayloadFlashblockDeltaV1, FlashBlock, Metadata};
use alloy_primitives::bytes::Bytes;
use alloy_rpc_types_engine::PayloadId;
use serde::{Deserialize, Serialize};
use std::{fmt::Debug, io};

#[derive(Clone, Debug, PartialEq, Default, Deserialize, Serialize)]
struct FlashblocksPayloadV1 {
    /// The payload id of the flashblock
    pub payload_id: PayloadId,
    /// The index of the flashblock in the block
    pub index: u64,
    /// The base execution payload configuration
    #[serde(skip_serializing_if = "Option::is_none")]
    pub base: Option<ExecutionPayloadBaseV1>,
    /// The delta/diff containing modified portions of the execution payload
    pub diff: ExecutionPayloadFlashblockDeltaV1,
    /// Additional metadata associated with the flashblock
    pub metadata: serde_json::Value,
}

impl FlashBlock {
    /// Decodes `bytes` into [`FlashBlock`].
    ///
    /// This function is specific to the Base Optimism websocket encoding.
    ///
    /// It is assumed that the `bytes` are encoded in JSON and optionally compressed using brotli.
    /// Whether the `bytes` is compressed or not is determined by looking at the first
    /// non ascii-whitespace character.
    pub(crate) fn decode(bytes: Bytes) -> eyre::Result<Self> {
        let bytes = try_parse_message(bytes)?;

        let payload: FlashblocksPayloadV1 = serde_json::from_slice(&bytes)
            .map_err(|e| eyre::eyre!("failed to parse message: {e}"))?;

        let metadata: Metadata = serde_json::from_value(payload.metadata.clone())
            .map_err(|e| eyre::eyre!("failed to parse message metadata: {e}"))?;

        Ok(Self {
            payload_id: payload.payload_id,
            index: payload.index,
            base: payload.base,
            diff: payload.diff,
            metadata,
        })
    }
}

/// Maps `bytes` into a potentially different [`Bytes`].
///
/// If the bytes start with a "{" character, prepended by any number of ASCII-whitespaces,
/// then it assumes that it is JSON-encoded and returns it as-is.
///
/// Otherwise, the `bytes` are passed through a brotli decompressor and returned.
fn try_parse_message(bytes: Bytes) -> eyre::Result<Bytes> {
    if bytes.trim_ascii_start().starts_with(b"{") {
        return Ok(bytes);
    }

    let mut decompressor = brotli::Decompressor::new(bytes.as_ref(), 4096);
    let mut decompressed = Vec::new();
    io::copy(&mut decompressor, &mut decompressed)?;

    Ok(decompressed.into())
}
