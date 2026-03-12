//! Channel encoding: Batch -> RLP encode -> zlib compress.

use alloy_rlp::Encodable;
use kona_protocol::{BatchType, ChannelId, Frame, SingleBatch};
use miniz_oxide::deflate::compress_to_vec_zlib;

/// Encodes a set of [`SingleBatch`]es into compressed channel data.
///
/// Pipeline: for each batch, prepend BatchType::Single byte + RLP encode,
/// then zlib-compress the concatenated output.
pub fn encode_batches(batches: &[SingleBatch]) -> Vec<u8> {
    let mut raw = Vec::new();
    for batch in batches {
        // Batch type prefix
        raw.push(BatchType::Single as u8);
        // RLP encode the SingleBatch
        batch.encode(&mut raw);
    }

    // Compress with zlib (level 6, good balance of speed/ratio)
    compress_to_vec_zlib(&raw, 6)
}

/// Splits compressed channel data into frames.
pub fn split_into_frames(
    channel_id: ChannelId,
    compressed_data: &[u8],
    max_frame_data_size: usize,
) -> Vec<Frame> {
    let mut frames = Vec::new();
    let mut offset = 0;
    let mut frame_number: u16 = 0;

    while offset < compressed_data.len() {
        let end = (offset + max_frame_data_size).min(compressed_data.len());
        let is_last = end == compressed_data.len();
        let data = compressed_data[offset..end].to_vec();

        frames.push(Frame::new(channel_id, frame_number, data, is_last));

        offset = end;
        frame_number += 1;
    }

    // If we had no data, push an empty last frame
    if frames.is_empty() {
        frames.push(Frame::new(channel_id, 0, Vec::new(), true));
    }

    frames
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encode_single_batch_roundtrip() {
        let batch = SingleBatch {
            parent_hash: Default::default(),
            epoch_num: 1,
            epoch_hash: Default::default(),
            timestamp: 1000,
            transactions: vec![],
        };
        let compressed = encode_batches(&[batch]);
        assert!(!compressed.is_empty());

        // Decompress and verify it starts with BatchType::Single
        let decompressed =
            miniz_oxide::inflate::decompress_to_vec_zlib(&compressed).expect("decompress");
        assert_eq!(decompressed[0], BatchType::Single as u8);
    }

    #[test]
    fn test_frame_splitting() {
        let data = vec![0xAB; 300];
        let channel_id = [0u8; 16];
        let frames = split_into_frames(channel_id, &data, 100);

        assert_eq!(frames.len(), 3);
        assert!(!frames[0].is_last);
        assert!(!frames[1].is_last);
        assert!(frames[2].is_last);

        // Reassemble
        let mut reassembled = Vec::new();
        for frame in &frames {
            reassembled.extend_from_slice(&frame.data);
        }
        assert_eq!(reassembled, data);
    }
}
