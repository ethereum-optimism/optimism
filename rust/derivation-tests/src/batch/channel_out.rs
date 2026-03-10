//! Channel encoder: compress batches and split into frames.

use alloy_primitives::Bytes;
use alloy_rlp::Encodable;
use kona_protocol::{ChannelId, Frame, SingleBatch, SpanBatch, SpanBatchError};

use super::compression::{self, CompressionAlgo};

/// Channel encoder that accumulates batches, compresses, and splits into frames.
#[derive(Debug)]
pub struct ChannelOut {
    channel_id: ChannelId,
    compression: CompressionAlgo,
    uncompressed: Vec<u8>,
    closed: bool,
    compressed: Option<Vec<u8>>,
}

impl ChannelOut {
    /// Create a new channel encoder.
    pub const fn new(channel_id: ChannelId, compression: CompressionAlgo) -> Self {
        Self { channel_id, compression, uncompressed: Vec::new(), closed: false, compressed: None }
    }

    /// Add a singular batch to the channel.
    pub fn add_singular_batch(&mut self, batch: &SingleBatch) -> Result<(), String> {
        if self.closed {
            return Err("channel already closed".into());
        }
        // Batch type prefix: 0x00 = SingularBatch (per derivation spec)
        self.uncompressed.push(0x00);
        // RLP-encode the singular batch
        batch.encode(&mut self.uncompressed);
        Ok(())
    }

    /// Add a span batch to the channel.
    ///
    /// Converts the span batch to its raw encoding and writes the batch type prefix (0x01)
    /// followed by the encoded raw span batch.
    pub fn add_span_batch(&mut self, batch: &SpanBatch) -> Result<(), String> {
        if self.closed {
            return Err("channel already closed".into());
        }
        let raw = batch.to_raw_span_batch().map_err(|e: SpanBatchError| e.to_string())?;
        // Batch type prefix: 0x01 = SpanBatch (per derivation spec)
        self.uncompressed.push(0x01);
        raw.encode(&mut self.uncompressed).map_err(|e: SpanBatchError| e.to_string())?;
        Ok(())
    }

    /// Close the channel, compressing the accumulated data.
    pub fn close(&mut self) -> Result<(), String> {
        if self.closed {
            return Err("channel already closed".into());
        }
        self.closed = true;
        self.compressed = Some(compression::compress(&self.uncompressed, self.compression));
        Ok(())
    }

    /// Split the compressed channel data into frames.
    ///
    /// Each frame is at most `max_frame_size` bytes of data payload.
    pub fn output_frames(&self, max_frame_size: usize) -> Vec<Frame> {
        let compressed = self.compressed.as_ref().expect("channel must be closed first");
        let mut frames = Vec::new();

        if compressed.is_empty() {
            frames.push(Frame { id: self.channel_id, number: 0, data: Vec::new(), is_last: true });
            return frames;
        }

        let chunks: Vec<&[u8]> = compressed.chunks(max_frame_size).collect();
        for (i, chunk) in chunks.iter().enumerate() {
            frames.push(Frame {
                id: self.channel_id,
                number: i as u16,
                data: chunk.to_vec(),
                is_last: i == chunks.len() - 1,
            });
        }

        frames
    }

    /// Output frame data as calldata bytes, prefixed with `DerivationVersion0` (0x00).
    ///
    /// Each entry is one frame encoded as calldata.
    pub fn to_calldata(&self, max_frame_size: usize) -> Vec<Bytes> {
        let frames = self.output_frames(max_frame_size);
        frames
            .into_iter()
            .map(|frame| {
                let mut buf = Vec::new();
                // DerivationVersion0 prefix
                buf.push(0x00);
                buf.extend_from_slice(&frame.encode());
                Bytes::from(buf)
            })
            .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn singular_batch_roundtrip() {
        let batch = SingleBatch {
            parent_hash: Default::default(),
            epoch_num: 1,
            epoch_hash: Default::default(),
            timestamp: 1000,
            transactions: vec![],
        };

        let channel_id: ChannelId = [1u8; 16];
        let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
        channel.add_singular_batch(&batch).unwrap();
        channel.close().unwrap();

        let frames = channel.output_frames(1_000_000);
        assert_eq!(frames.len(), 1);
        assert!(frames[0].is_last);
        assert_eq!(frames[0].id, channel_id);
    }

    #[test]
    fn calldata_format() {
        let batch = SingleBatch {
            parent_hash: Default::default(),
            epoch_num: 1,
            epoch_hash: Default::default(),
            timestamp: 1000,
            transactions: vec![],
        };

        let channel_id: ChannelId = [2u8; 16];
        let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
        channel.add_singular_batch(&batch).unwrap();
        channel.close().unwrap();

        let calldata = channel.to_calldata(1_000_000);
        assert_eq!(calldata.len(), 1);
        // First byte is DerivationVersion0
        assert_eq!(calldata[0][0], 0x00);
    }

    #[test]
    fn test_frame_splitting() {
        // Create many batches with varied data that doesn't compress well,
        // forcing the compressed output to be large enough for multiple frames.
        let channel_id: ChannelId = [3u8; 16];
        let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);

        for i in 0u8..20 {
            // Use pseudo-random bytes that resist compression
            let tx_data: Vec<u8> = (0..500).map(|j| i.wrapping_mul(37).wrapping_add(j as u8)).collect();
            let batch = SingleBatch {
                parent_hash: Default::default(),
                epoch_num: (i as u64) + 1,
                epoch_hash: Default::default(),
                timestamp: 1000 + (i as u64) * 2,
                transactions: vec![Bytes::from(tx_data)],
            };
            channel.add_singular_batch(&batch).unwrap();
        }
        channel.close().unwrap();

        // Use a small max frame size to force multiple frames
        let frames = channel.output_frames(50);
        assert!(frames.len() > 1, "expected multiple frames, got {}", frames.len());

        // Frame numbers should increment
        for (i, frame) in frames.iter().enumerate() {
            assert_eq!(frame.number, i as u16, "frame {i} has wrong number");
            assert_eq!(frame.id, channel_id);
        }

        // Only the last frame should have is_last = true
        for frame in &frames[..frames.len() - 1] {
            assert!(!frame.is_last, "non-last frame should have is_last = false");
        }
        assert!(frames.last().unwrap().is_last, "last frame should have is_last = true");
    }
}
