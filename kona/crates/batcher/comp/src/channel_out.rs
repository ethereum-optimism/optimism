//! Contains the `ChannelOut` primitive for Optimism.

use crate::{ChannelCompressor, CompressorError};
use alloc::{vec, vec::Vec};
use kona_genesis::RollupConfig;
use kona_protocol::{Batch, ChannelId, Frame};
use rand::{RngCore, SeedableRng, rngs::SmallRng};

/// The frame overhead.
const FRAME_V0_OVERHEAD: usize = 23;

/// An error returned by the [ChannelOut] when adding single batches.
#[derive(Debug, Clone, PartialEq, thiserror::Error)]
pub enum ChannelOutError {
    /// The channel is closed.
    #[error("The channel is already closed")]
    ChannelClosed,
    /// The max frame size is too small.
    #[error("The max frame size is too small")]
    MaxFrameSizeTooSmall,
    /// Missing compressed batch data.
    #[error("Missing compressed batch data")]
    MissingData,
    /// An error from compression.
    #[error("Error from compression")]
    Compression(#[from] CompressorError),
    /// An error encoding the `Batch`.
    #[error("Error encoding the batch")]
    BatchEncoding,
    /// The encoded batch exceeds the max RLP bytes per channel.
    #[error("The encoded batch exceeds the max RLP bytes per channel")]
    ExceedsMaxRlpBytesPerChannel,
}

/// [ChannelOut] constructs a channel from compressed, encoded batch data.
#[allow(missing_debug_implementations)]
pub struct ChannelOut<'a, C>
where
    C: ChannelCompressor,
{
    /// The unique identifier for the channel.
    pub id: ChannelId,
    /// A reference to the [RollupConfig] used to
    /// check the max RLP bytes per channel when
    /// encoding and accepting the batch.
    pub config: &'a RollupConfig,
    /// The rlp length of the channel.
    pub rlp_length: u64,
    /// Whether the channel is closed.
    pub closed: bool,
    /// The frame number.
    pub frame_number: u16,
    /// The compressor.
    pub compressor: C,
}

impl<'a, C> ChannelOut<'a, C>
where
    C: ChannelCompressor,
{
    /// Creates a new [ChannelOut] with the given [ChannelId].
    pub const fn new(id: ChannelId, config: &'a RollupConfig, compressor: C) -> Self {
        Self { id, config, rlp_length: 0, frame_number: 0, closed: false, compressor }
    }

    /// Resets the [ChannelOut] to its initial state.
    pub fn reset(&mut self) {
        self.rlp_length = 0;
        self.frame_number = 0;
        self.closed = false;
        self.compressor.reset();
        // `getrandom` isn't available for wasm and risc targets
        // Thread-based RNGs are not available for no_std
        // So we must use a seeded RNG.
        let mut small_rng = SmallRng::seed_from_u64(43);
        SmallRng::fill_bytes(&mut small_rng, &mut self.id);
    }

    /// Accepts the given [Batch] data into the [ChannelOut], compressing it
    /// into frames.
    pub fn add_batch(&mut self, batch: Batch) -> Result<(), ChannelOutError> {
        if self.closed {
            return Err(ChannelOutError::ChannelClosed);
        }

        // Encode the batch.
        let mut buf = vec![];
        batch.encode(&mut buf).map_err(|_| ChannelOutError::BatchEncoding)?;

        // Validate that the RLP length is within the channel's limits.
        let max_rlp_bytes_per_channel = self.config.max_rlp_bytes_per_channel(batch.timestamp());
        if self.rlp_length + buf.len() as u64 > max_rlp_bytes_per_channel {
            return Err(ChannelOutError::ExceedsMaxRlpBytesPerChannel);
        }

        self.compressor.write(&buf)?;
        self.rlp_length += buf.len() as u64;

        Ok(())
    }

    /// Returns the total amount of rlp-encoded input bytes.
    pub const fn input_bytes(&self) -> u64 {
        self.rlp_length
    }

    /// Returns the number of bytes ready to be output to a frame.
    pub fn ready_bytes(&self) -> usize {
        self.compressor.len()
    }

    /// Flush the internal compressor.
    pub fn flush(&mut self) -> Result<(), ChannelOutError> {
        self.compressor.flush()?;
        Ok(())
    }

    /// Closes the channel if not already closed.
    pub const fn close(&mut self) {
        self.closed = true;
    }

    /// Outputs a [Frame] from the [ChannelOut].
    pub fn output_frame(&mut self, max_size: usize) -> Result<Frame, ChannelOutError> {
        if max_size < FRAME_V0_OVERHEAD {
            return Err(ChannelOutError::MaxFrameSizeTooSmall);
        }

        // Construct an empty frame.
        let mut frame =
            Frame { id: self.id, number: self.frame_number, is_last: self.closed, data: vec![] };

        let mut max_size = max_size - FRAME_V0_OVERHEAD;
        if max_size > self.ready_bytes() {
            max_size = self.ready_bytes();
        }

        // Read `max_size` bytes from the compressed data.
        let mut data = Vec::with_capacity(max_size);
        self.compressor.read(&mut data).map_err(ChannelOutError::Compression)?;
        frame.data.extend_from_slice(data.as_slice());

        // Update the compressed data.
        self.frame_number += 1;
        Ok(frame)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{CompressorWriter, test_utils::MockCompressor};
    use alloy_primitives::Bytes;
    use kona_protocol::{SingleBatch, SpanBatch};

    #[test]
    fn test_output_frame_max_size_too_small() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());
        assert_eq!(channel.output_frame(0), Err(ChannelOutError::MaxFrameSizeTooSmall));
    }

    #[test]
    fn test_channel_out_output_frame_no_data() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(
            ChannelId::default(),
            &config,
            MockCompressor { read_error: true, compressed: Some(Default::default()) },
        );
        let err = channel.output_frame(FRAME_V0_OVERHEAD).unwrap_err();
        assert_eq!(err, ChannelOutError::Compression(CompressorError::Full));
    }

    #[test]
    fn test_channel_out_output() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(
            ChannelId::default(),
            &config,
            MockCompressor { compressed: Some(Default::default()), ..Default::default() },
        );
        let frame = channel.output_frame(FRAME_V0_OVERHEAD).unwrap();
        assert_eq!(frame.id, ChannelId::default());
        assert_eq!(frame.number, 0);
        assert!(!frame.is_last);
    }

    #[test]
    fn test_channel_out_reset() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut {
            id: ChannelId::default(),
            config: &config,
            rlp_length: 10,
            closed: true,
            frame_number: 11,
            compressor: MockCompressor::default(),
        };
        channel.reset();
        assert_eq!(channel.rlp_length, 0);
        assert_eq!(channel.frame_number, 0);
        // The odds of a randomized channel id being equal to the
        // default are so astronomically low, this test will always pass.
        // The randomized [u8; 16] is about 1/255^16.
        assert!(channel.id != ChannelId::default());
        assert!(!channel.closed);
    }

    #[test]
    fn test_channel_out_ready_bytes_empty() {
        let config = RollupConfig::default();
        let channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());
        assert_eq!(channel.ready_bytes(), 0);
    }

    #[test]
    fn test_channel_out_ready_bytes_some() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());
        channel.compressor.write(&[1, 2, 3]).unwrap();
        assert_eq!(channel.ready_bytes(), 3);
    }

    #[test]
    fn test_channel_out_close() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());
        assert!(!channel.closed);

        channel.close();
        assert!(channel.closed);
    }

    #[test]
    fn test_channel_out_add_batch_closed() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());
        channel.close();

        let batch = Batch::Single(SingleBatch::default());
        assert_eq!(channel.add_batch(batch), Err(ChannelOutError::ChannelClosed));
    }

    #[test]
    fn test_channel_out_empty_span_batch_decode_error() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());

        let batch = Batch::Span(SpanBatch::default());
        assert_eq!(channel.add_batch(batch), Err(ChannelOutError::BatchEncoding));
    }

    #[test]
    fn test_channel_out_max_rlp_bytes_per_channel() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());

        let batch = Batch::Single(SingleBatch::default());
        channel.rlp_length = config.max_rlp_bytes_per_channel(batch.timestamp());

        assert_eq!(channel.add_batch(batch), Err(ChannelOutError::ExceedsMaxRlpBytesPerChannel));
    }

    #[test]
    fn test_channel_out_add_batch() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());

        let batch = Batch::Single(SingleBatch::default());
        assert_eq!(channel.add_batch(batch), Ok(()));
    }

    #[test]
    fn test_channel_out_add_batch_enforces_cumulative_rlp_limit() {
        let config = RollupConfig::default();
        let mut channel = ChannelOut::new(ChannelId::default(), &config, MockCompressor::default());

        let timestamp = 0;
        let max_rlp = config.max_rlp_bytes_per_channel(timestamp);
        let payload_size = (max_rlp / 2 + 1) as usize;

        let large_batch = Batch::Single(SingleBatch {
            timestamp,
            transactions: vec![Bytes::from(vec![0u8; payload_size])],
            ..Default::default()
        });

        let mut encoded = Vec::new();
        large_batch.encode(&mut encoded).expect("test batch should encode");
        assert!(encoded.len() as u64 <= max_rlp, "test batch should fit within per-channel limit");

        channel.add_batch(large_batch.clone()).expect("first batch should fit");
        assert_eq!(channel.rlp_length, encoded.len() as u64);

        let err = channel.add_batch(large_batch).unwrap_err();
        assert_eq!(err, ChannelOutError::ExceedsMaxRlpBytesPerChannel);
    }
}
