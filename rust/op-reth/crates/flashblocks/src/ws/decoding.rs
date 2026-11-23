use crate::FlashBlock;
use alloy_primitives::bytes::Bytes;
use std::io::Read;

/// Largest flashblock message accepted, as received or after decompression.
pub(crate) const MAX_MESSAGE_SIZE: usize = 64 * 1024 * 1024;

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
/// Otherwise, the `bytes` are passed through a brotli decompressor and returned. Fails if they
/// decompress to more than [`MAX_MESSAGE_SIZE`] bytes.
fn try_parse_message(bytes: Bytes) -> eyre::Result<Bytes> {
    if bytes.trim_ascii_start().starts_with(b"{") {
        return Ok(bytes);
    }

    let mut decompressed = Vec::new();
    brotli::Decompressor::new(bytes.as_ref(), 4096)
        .take(MAX_MESSAGE_SIZE as u64 + 1)
        .read_to_end(&mut decompressed)?;
    if decompressed.len() > MAX_MESSAGE_SIZE {
        eyre::bail!("message decompresses to more than {MAX_MESSAGE_SIZE} bytes");
    }

    Ok(decompressed.into())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;

    /// Brotli-compresses `len` spaces, which the decoder takes for a compressed message.
    fn compressed_spaces(len: usize) -> Bytes {
        let mut compressed = Vec::new();
        let mut writer = brotli::CompressorWriter::new(&mut compressed, 4096, 1, 22);
        writer.write_all(&vec![b' '; len]).unwrap();
        drop(writer);
        compressed.into()
    }

    #[test]
    fn test_decompresses_up_to_max_message_size() {
        let decompressed = try_parse_message(compressed_spaces(MAX_MESSAGE_SIZE)).unwrap();
        assert_eq!(decompressed.len(), MAX_MESSAGE_SIZE);
    }

    #[test]
    fn test_rejects_message_decompressing_beyond_max_message_size() {
        let compressed = compressed_spaces(MAX_MESSAGE_SIZE + 1);
        assert!(compressed.len() < 64 * 1024, "compressed to {} bytes", compressed.len());

        assert!(try_parse_message(compressed).is_err());
    }
}
