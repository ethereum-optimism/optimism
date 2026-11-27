use crate::FlashBlock;
use alloy_primitives::bytes::Bytes;
use std::io::Read;

/// The largest flashblock message, in bytes.
///
/// `WsConnector` limits each websocket message to this size, and the default decoder limits the
/// output of brotli decompression to this size. Tungstenite also limits each websocket frame to
/// 16 MiB by default, so a message sent as a single frame must not exceed 16 MiB.
pub(crate) const MAX_FLASHBLOCK_SIZE: usize = 64 << 20;

/// The low seven bits of the first byte of a brotli large-window stream.
///
/// RFC 7932 §9.1 marks this window bits pattern as invalid. Only the non-standard large-window
/// extension uses it.
const LARGE_WINDOW_MARKER: u8 = 0b001_0001;

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
/// Otherwise, [`decompress_brotli`] decompresses the `bytes` with a limit of
/// [`MAX_FLASHBLOCK_SIZE`].
fn try_parse_message(bytes: Bytes) -> eyre::Result<Bytes> {
    if bytes.trim_ascii_start().starts_with(b"{") {
        return Ok(bytes);
    }

    Ok(decompress_brotli(&bytes, MAX_FLASHBLOCK_SIZE)?.into())
}

/// Decompresses a standard (RFC 7932) brotli stream whose output is at most `limit` bytes.
///
/// Returns an error if the stream uses the large-window extension, if decoding fails, or if the
/// output exceeds `limit` bytes. Reads at most `limit + 1` bytes of output. The output buffer can
/// reserve about twice that, because `read_to_end` grows it by doubling.
fn decompress_brotli(compressed: &[u8], limit: usize) -> eyre::Result<Vec<u8>> {
    // `brotli::Decompressor` always enables the large-window extension and offers no strict
    // constructor.
    // Before it writes any output, it can allocate a ring buffer as large as the declared window:
    // at most 16 MiB for a standard stream, but up to 1 GiB for a large-window stream. `limit`
    // bounds neither allocation, so the extension is rejected before decoding.
    let is_large_window =
        compressed.first().is_some_and(|byte| (byte & 0x7f) == LARGE_WINDOW_MARKER);
    eyre::ensure!(!is_large_window, "brotli large-window streams are not supported");

    let mut decompressed = Vec::new();
    brotli::Decompressor::new(compressed, 4096)
        .take(limit as u64 + 1)
        .read_to_end(&mut decompressed)?;
    eyre::ensure!(
        decompressed.len() <= limit,
        "decompressed flashblock exceeds the limit of {limit} bytes"
    );

    Ok(decompressed)
}

#[cfg(test)]
mod tests {
    use super::*;
    use brotli::enc::BrotliEncoderParams;
    use std::io;

    const LIMIT: usize = 1024;

    /// Compresses `len` zero bytes with `params`.
    fn compress_zeros(len: usize, params: &BrotliEncoderParams) -> Vec<u8> {
        let mut compressed = Vec::new();
        brotli::BrotliCompress(&mut io::repeat(0).take(len as u64), &mut compressed, params)
            .unwrap();
        compressed
    }

    #[test]
    fn test_decompress_brotli_accepts_output_up_to_limit() {
        let compressed = compress_zeros(LIMIT, &BrotliEncoderParams::default());

        assert_eq!(decompress_brotli(&compressed, LIMIT).unwrap(), vec![0; LIMIT]);
    }

    #[test]
    fn test_decompress_brotli_rejects_output_above_limit() {
        let compressed = compress_zeros(LIMIT + 1, &BrotliEncoderParams::default());

        let err = decompress_brotli(&compressed, LIMIT).unwrap_err();
        assert!(err.to_string().contains("exceeds the limit"), "{err}");
    }

    #[test]
    fn test_decompress_brotli_accepts_every_standard_window_size() {
        // RFC 7932 window sizes are 10 to 24 bits, and each has its own header encoding.
        for lgwin in 10..=24 {
            let params = BrotliEncoderParams { lgwin, ..Default::default() };
            let compressed = compress_zeros(LIMIT, &params);

            decompress_brotli(&compressed, LIMIT)
                .unwrap_or_else(|err| panic!("lgwin {lgwin}: {err}"));
        }
    }

    #[test]
    fn test_decompress_brotli_rejects_large_window_streams() {
        let params = BrotliEncoderParams { large_window: true, ..Default::default() };
        let compressed = compress_zeros(LIMIT, &params);

        // `brotli::Decompressor` alone accepts this stream, so only the check rejects it.
        let mut decompressed = Vec::new();
        brotli::Decompressor::new(compressed.as_slice(), 4096)
            .read_to_end(&mut decompressed)
            .unwrap();
        assert_eq!(decompressed, vec![0; LIMIT]);

        let err = decompress_brotli(&compressed, LIMIT).unwrap_err();
        assert!(err.to_string().contains("large-window"), "{err}");
    }

    #[test]
    fn test_decompress_brotli_rejects_empty_input() {
        assert!(decompress_brotli(&[], LIMIT).is_err());
    }

    #[test]
    fn test_try_parse_message_applies_max_flashblock_size() {
        let params = BrotliEncoderParams { quality: 5, ..Default::default() };
        let compressed = compress_zeros(MAX_FLASHBLOCK_SIZE + 1, &params);

        let err = try_parse_message(compressed.into()).unwrap_err();
        assert!(err.to_string().contains("exceeds the limit"), "{err}");
    }
}
