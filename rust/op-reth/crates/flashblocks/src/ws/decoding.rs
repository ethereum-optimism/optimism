use crate::FlashBlock;
use alloy_primitives::bytes::Bytes;
use std::io;

/// A trait for decoding flashblocks from bytes.
pub trait FlashBlockDecoder: Send + 'static {
    /// Decodes `bytes` into a [`FlashBlock`].
    fn decode(&self, bytes: Bytes) -> eyre::Result<FlashBlock>;
}

/// Default implementation of the decoder.
impl FlashBlockDecoder for () {
    fn decode(&self, bytes: Bytes) -> eyre::Result<FlashBlock> {
        decode_flashblock(bytes)
    }
}

pub(crate) fn decode_flashblock(bytes: Bytes) -> eyre::Result<FlashBlock> {
    let bytes = crate::ws::decoding::try_parse_message(bytes)?;

    let payload: FlashBlock =
        serde_json::from_slice(&bytes).map_err(|e| eyre::eyre!("failed to parse message: {e}"))?;

    Ok(payload)
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
