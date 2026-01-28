//! Frame types for OP Stack L2 data transmission.
//!
//! Frames are the fundamental unit of data transmission in the OP Stack derivation
//! pipeline. They provide a way to split large channel data into smaller, manageable
//! chunks that can be transmitted via L1 transactions and later reassembled.
//!
//! # Frame Structure
//!
//! Each frame contains:
//! - **Channel ID**: Unique identifier linking frames to their parent channel
//! - **Frame Number**: Sequence number for ordering and reassembly
//! - **Data Payload**: The actual frame data content
//! - **Last Flag**: Indicates if this is the final frame in the sequence
//!
//! # Transmission Process
//!
//! ```text
//! Channel Data → Split into Frames → Transmit via L1 → Reassemble Channel
//! ```
//!
//! # Error Handling
//!
//! Frame processing can fail due to:
//! - Size constraints (too large or too small)
//! - Invalid frame identifiers or numbers
//! - Data length mismatches
//! - Unsupported versions

use crate::ChannelId;
use alloc::vec::Vec;

/// Version identifier for the current derivation pipeline format.
///
/// This constant defines the version of the derivation pipeline protocol
/// that this implementation supports. It's included in frame encoding to
/// ensure compatibility and enable future protocol upgrades.
pub const DERIVATION_VERSION_0: u8 = 0;

/// Overhead estimation for frame metadata and tagging information.
///
/// This constant provides an estimate of the additional bytes required
/// for frame metadata (channel ID, frame number, data length, flags) and
/// L1 transaction overhead. Used for buffer size calculations and gas
/// estimation when planning frame transmission.
pub const FRAME_OVERHEAD: usize = 200;

/// Maximum allowed size for a single frame in bytes.
///
/// While typical L1 transactions carrying frames are around 128 KB due to
/// network conditions and gas limits, this larger limit provides headroom
/// for future growth as L1 gas limits and network conditions improve.
///
/// The 1MB limit balances:
/// - **Transmission efficiency**: Larger frames reduce overhead
/// - **Network compatibility**: Must fit within reasonable L1 transaction sizes
/// - **Memory constraints**: Avoid excessive memory usage during processing
pub const MAX_FRAME_LEN: usize = 1_000_000;

/// A frame decoding error.
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq, Hash)]
pub enum FrameDecodingError {
    /// The frame data is too large.
    #[error("Frame data too large: {0} bytes")]
    DataTooLarge(usize),
    /// The frame data is too short.
    #[error("Frame data too short: {0} bytes")]
    DataTooShort(usize),
    /// Error decoding the frame id.
    #[error("Invalid frame id")]
    InvalidId,
    /// Error decoding the frame number.
    #[error("Invalid frame number")]
    InvalidNumber,
    /// Error decoding the frame data length.
    #[error("Invalid frame data length")]
    InvalidDataLength,
}

/// Frame parsing error.
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq, Hash)]
pub enum FrameParseError {
    /// Error parsing the frame data.
    #[error("Frame decoding error: {0}")]
    FrameDecodingError(FrameDecodingError),
    /// No frames to parse.
    #[error("No frames to parse")]
    NoFrames,
    /// Unsupported derivation version.
    #[error("Unsupported derivation version")]
    UnsupportedVersion,
    /// Frame data length mismatch.
    #[error("Frame data length mismatch")]
    DataLengthMismatch,
    /// No frames decoded.
    #[error("No frames decoded")]
    NoFramesDecoded,
}

/// A channel frame representing a segment of channel data for transmission.
///
/// Frames are the atomic units of data transmission in the OP Stack derivation pipeline.
/// Large channel data is split into multiple frames to fit within L1 transaction size
/// constraints while maintaining the ability to reassemble the original data.
///
/// # Binary Encoding Format
///
/// The frame is encoded as a concatenated byte sequence:
/// ```text
/// frame = channel_id ++ frame_number ++ frame_data_length ++ frame_data ++ is_last
/// ```
///
/// ## Field Specifications
/// - **channel_id** (16 bytes): Unique identifier linking this frame to its parent channel
/// - **frame_number** (2 bytes, uint16): Sequence number for proper reassembly ordering
/// - **frame_data_length** (4 bytes, uint32): Length of the frame_data field in bytes
/// - **frame_data** (variable): The actual payload data for this frame segment
/// - **is_last** (1 byte, bool): Flag indicating if this is the final frame in the sequence
///
/// ## Total Overhead
/// Each frame has a fixed overhead of 23 bytes (16 + 2 + 4 + 1) plus the variable data payload.
///
/// # Frame Sequencing
///
/// Frames within a channel must be:
/// 1. **Sequentially numbered**: Starting from 0 and incrementing by 1
/// 2. **Properly terminated**: Exactly one frame marked with `is_last = true`
/// 3. **Complete**: All frame numbers from 0 to the last frame must be present
///
/// # Reassembly Process
///
/// Channel reassembly involves:
/// 1. **Collection**: Gather all frames with the same channel ID
/// 2. **Sorting**: Order frames by their frame number
/// 3. **Validation**: Verify sequential numbering and last frame flag
/// 4. **Concatenation**: Combine frame data in order to reconstruct channel
///
/// # Error Conditions
///
/// Frame processing can fail due to:
/// - Missing frames in the sequence
/// - Duplicate frame numbers
/// - Multiple frames marked as last
/// - Frame data exceeding size limits
/// - Invalid encoding or corruption
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Frame {
    /// Unique identifier linking this frame to its parent channel.
    ///
    /// All frames belonging to the same channel share this identifier,
    /// enabling proper grouping during the reassembly process. The channel
    /// ID is typically derived from the first frame's metadata.
    pub id: ChannelId,
    /// Sequence number for frame ordering within the channel.
    ///
    /// Frame numbers start at 0 and increment sequentially. This field
    /// is critical for proper reassembly ordering and detecting missing
    /// or duplicate frames during channel reconstruction.
    pub number: u16,
    /// Payload data carried by this frame.
    ///
    /// Contains a segment of the original channel data. When all frames
    /// in a channel are reassembled, concatenating this data in frame
    /// number order reconstructs the complete channel payload.
    pub data: Vec<u8>,
    /// Flag indicating whether this is the final frame in the channel sequence.
    ///
    /// Exactly one frame per channel should have this flag set to `true`.
    /// This enables detection of complete channel reception and validation
    /// that no frames are missing from the end of the sequence.
    pub is_last: bool,
}

impl Frame {
    /// Creates a new [`Frame`].
    pub const fn new(id: ChannelId, number: u16, data: Vec<u8>, is_last: bool) -> Self {
        Self { id, number, data, is_last }
    }

    /// Encode the frame into a byte vector.
    pub fn encode(&self) -> Vec<u8> {
        let mut encoded = Vec::with_capacity(16 + 2 + 4 + self.data.len() + 1);
        encoded.extend_from_slice(&self.id);
        encoded.extend_from_slice(&self.number.to_be_bytes());
        encoded.extend_from_slice(&(self.data.len() as u32).to_be_bytes());
        encoded.extend_from_slice(&self.data);
        encoded.push(self.is_last as u8);
        encoded
    }

    /// Decode a frame from a byte vector.
    pub fn decode(encoded: &[u8]) -> Result<(usize, Self), FrameDecodingError> {
        const BASE_FRAME_LEN: usize = 16 + 2 + 4 + 1;

        if encoded.len() < BASE_FRAME_LEN {
            return Err(FrameDecodingError::DataTooShort(encoded.len()));
        }

        let id = encoded[..16].try_into().map_err(|_| FrameDecodingError::InvalidId)?;
        let number = u16::from_be_bytes(
            encoded[16..18].try_into().map_err(|_| FrameDecodingError::InvalidNumber)?,
        );
        let data_len = u32::from_be_bytes(
            encoded[18..22].try_into().map_err(|_| FrameDecodingError::InvalidDataLength)?,
        ) as usize;

        if data_len > MAX_FRAME_LEN || data_len >= encoded.len() - (BASE_FRAME_LEN - 1) {
            return Err(FrameDecodingError::DataTooLarge(data_len));
        }

        let data = encoded[22..22 + data_len].to_vec();
        let is_last = encoded[22 + data_len] == 1;
        Ok((BASE_FRAME_LEN + data_len, Self { id, number, data, is_last }))
    }

    /// Parses a single frame from the given data at the given starting position,
    /// returning the frame and the number of bytes consumed.
    pub fn parse_frame(data: &[u8], start: usize) -> Result<(usize, Self), FrameDecodingError> {
        let (frame_len, frame) = Self::decode(&data[start..])?;
        Ok((frame_len, frame))
    }

    /// Parse the on chain serialization of frame(s) in an L1 transaction. Currently
    /// only version 0 of the serialization format is supported. All frames must be parsed
    /// without error and there must not be any left over data and there must be at least one
    /// frame.
    ///
    /// Frames are stored in L1 transactions with the following format:
    /// * `data = DerivationVersion0 ++ Frame(s)` Where there is one or more frames concatenated
    ///   together.
    pub fn parse_frames(encoded: &[u8]) -> Result<Vec<Self>, FrameParseError> {
        if encoded.is_empty() {
            return Err(FrameParseError::NoFrames);
        }
        if encoded[0] != DERIVATION_VERSION_0 {
            return Err(FrameParseError::UnsupportedVersion);
        }

        let data = &encoded[1..];
        let mut frames = Vec::new();
        let mut offset = 0;
        while offset < data.len() {
            let (frame_length, frame) =
                Self::decode(&data[offset..]).map_err(FrameParseError::FrameDecodingError)?;
            frames.push(frame);
            offset += frame_length;
        }

        if offset != data.len() {
            return Err(FrameParseError::DataLengthMismatch);
        }
        if frames.is_empty() {
            return Err(FrameParseError::NoFramesDecoded);
        }

        Ok(frames)
    }

    /// Calculates the size of the frame + overhead for storing the frame. The sum of the frame size
    /// of each frame in a channel determines the channel's size. The sum of the channel sizes
    /// is used for pruning & compared against the max channel bank size.
    pub const fn size(&self) -> usize {
        self.data.len() + FRAME_OVERHEAD
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloc::vec;

    #[test]
    fn test_encode_frame_roundtrip() {
        let frame = Frame { id: [0xFF; 16], number: 0xEE, data: vec![0xDD; 50], is_last: true };

        let (_, frame_decoded) = Frame::decode(&frame.encode()).unwrap();
        assert_eq!(frame, frame_decoded);
    }

    #[test]
    fn test_data_too_short() {
        let frame = Frame { id: [0xFF; 16], number: 0xEE, data: vec![0xDD; 22], is_last: true };
        let err = Frame::decode(&frame.encode()[..22]).unwrap_err();
        assert_eq!(err, FrameDecodingError::DataTooShort(22));
    }

    #[test]
    fn test_decode_exceeds_max_data_len() {
        let frame = Frame {
            id: [0xFF; 16],
            number: 0xEE,
            data: vec![0xDD; MAX_FRAME_LEN + 1],
            is_last: true,
        };
        let err = Frame::decode(&frame.encode()).unwrap_err();
        assert_eq!(err, FrameDecodingError::DataTooLarge(MAX_FRAME_LEN + 1));
    }

    #[test]
    fn test_decode_malicious_data_len() {
        let frame = Frame { id: [0xFF; 16], number: 0xEE, data: vec![0xDD; 50], is_last: true };
        let mut encoded = frame.encode();
        let data_len = (encoded.len() - 22) as u32;
        encoded[18..22].copy_from_slice(&data_len.to_be_bytes());

        let err = Frame::decode(&encoded).unwrap_err();
        assert_eq!(err, FrameDecodingError::DataTooLarge(encoded.len() - 22_usize));

        let valid_data_len = (encoded.len() - 23) as u32;
        encoded[18..22].copy_from_slice(&valid_data_len.to_be_bytes());
        let (_, frame_decoded) = Frame::decode(&encoded).unwrap();
        assert_eq!(frame, frame_decoded);
    }

    #[test]
    fn test_decode_many() {
        let frame = Frame { id: [0xFF; 16], number: 0xEE, data: vec![0xDD; 50], is_last: true };
        let mut bytes = Vec::new();
        bytes.extend_from_slice(&[DERIVATION_VERSION_0]);
        (0..5).for_each(|_| {
            bytes.extend_from_slice(&frame.encode());
        });

        let frames = Frame::parse_frames(bytes.as_slice()).unwrap();
        assert_eq!(frames.len(), 5);
        (0..5).for_each(|i| {
            assert_eq!(frames[i], frame);
        });
    }
}
