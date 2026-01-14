//! Channel Types

use alloc::vec::Vec;
use alloy_primitives::{Bytes, map::HashMap};

use crate::{BlockInfo, Frame};

/// [`CHANNEL_ID_LENGTH`] is the length of the channel ID.
pub const CHANNEL_ID_LENGTH: usize = 16;

/// [`ChannelId`] is an opaque identifier for a channel.
pub type ChannelId = [u8; CHANNEL_ID_LENGTH];

/// [`MAX_RLP_BYTES_PER_CHANNEL`] is the maximum amount of bytes that will be read from
/// a channel. This limit is set when decoding the RLP.
pub const MAX_RLP_BYTES_PER_CHANNEL: u64 = 10_000_000;

/// [`FJORD_MAX_RLP_BYTES_PER_CHANNEL`] is the maximum amount of bytes that will be read from
/// a channel when the Fjord Hardfork is activated. This limit is set when decoding the RLP.
pub const FJORD_MAX_RLP_BYTES_PER_CHANNEL: u64 = 100_000_000;

/// An error returned when adding a frame to a channel.
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq, Hash)]
pub enum ChannelError {
    /// The frame id does not match the channel id.
    #[error("Frame id does not match channel id")]
    FrameIdMismatch,
    /// The channel is closed.
    #[error("Channel is closed")]
    ChannelClosed,
    /// The frame number is already in the channel.
    #[error("Frame number {0} already exists")]
    FrameNumberExists(usize),
    /// The frame number is beyond the end frame.
    #[error("Frame number {0} is beyond end frame")]
    FrameBeyondEndFrame(usize),
}

/// A Channel is a set of batches that are split into at least one, but possibly multiple frames.
///
/// Frames are allowed to be ingested out of order.
/// Each frame is ingested one by one. Once a frame with `closed` is added to the channel, the
/// channel may mark itself as ready for reading once all intervening frames have been added
#[derive(Debug, Clone, Default)]
pub struct Channel {
    /// The unique identifier for this channel
    pub id: ChannelId,
    /// The block that the channel is currently open at
    pub open_block: BlockInfo,
    /// Estimated memory size, used to drop the channel if we have too much data
    pub estimated_size: usize,
    /// True if the last frame has been buffered
    pub closed: bool,
    /// The highest frame number that has been ingested
    pub highest_frame_number: u16,
    /// The frame number of the frame where `is_last` is true
    /// No other frame number may be higher than this
    pub last_frame_number: u16,
    /// Store a map of frame number to frame for constant time ordering
    pub inputs: HashMap<u16, Frame>,
    /// The highest L1 inclusion block that a frame was included in
    pub highest_l1_inclusion_block: BlockInfo,
}

impl Channel {
    /// Create a new [`Channel`] with the given [`ChannelId`] and [`BlockInfo`].
    pub fn new(id: ChannelId, open_block: BlockInfo) -> Self {
        Self { id, open_block, inputs: HashMap::default(), ..Default::default() }
    }

    /// Returns the current [`ChannelId`] for the channel.
    pub const fn id(&self) -> ChannelId {
        self.id
    }

    /// Returns the number of frames ingested.
    pub fn len(&self) -> usize {
        self.inputs.len()
    }

    /// Returns if the channel is empty.
    pub fn is_empty(&self) -> bool {
        self.inputs.is_empty()
    }

    /// Add a frame to the channel.
    ///
    /// ## Takes
    /// - `frame`: The frame to add to the channel
    /// - `l1_inclusion_block`: The block that the frame was included in
    ///
    /// ## Returns
    /// - `Ok(()):` If the frame was successfully buffered
    /// - `Err(_):` If the frame was invalid
    pub fn add_frame(
        &mut self,
        frame: Frame,
        l1_inclusion_block: BlockInfo,
    ) -> Result<(), ChannelError> {
        // Ensure that the frame ID is equal to the channel ID.
        if frame.id != self.id {
            return Err(ChannelError::FrameIdMismatch);
        }
        if frame.is_last && self.closed {
            return Err(ChannelError::ChannelClosed);
        }
        if self.inputs.contains_key(&frame.number) {
            return Err(ChannelError::FrameNumberExists(frame.number as usize));
        }
        if self.closed && frame.number >= self.last_frame_number {
            return Err(ChannelError::FrameBeyondEndFrame(frame.number as usize));
        }

        // Guaranteed to succeed at this point. Update the channel state.
        if frame.is_last {
            self.last_frame_number = frame.number;
            self.closed = true;

            // Prune frames with a higher number than the last frame number when we receive a
            // closing frame.
            if self.last_frame_number < self.highest_frame_number {
                self.inputs.retain(|id, frame| {
                    self.estimated_size -= frame.size();
                    *id < self.last_frame_number
                });
                self.highest_frame_number = self.last_frame_number;
            }
        }

        // Update the highest frame number.
        if frame.number > self.highest_frame_number {
            self.highest_frame_number = frame.number;
        }

        if self.highest_l1_inclusion_block.number < l1_inclusion_block.number {
            self.highest_l1_inclusion_block = l1_inclusion_block;
        }

        self.estimated_size += frame.size();
        self.inputs.insert(frame.number, frame);
        Ok(())
    }

    /// Returns the block number of the L1 block that contained the first [`Frame`] in this channel.
    pub const fn open_block_number(&self) -> u64 {
        self.open_block.number
    }

    /// Returns the estimated size of the channel including [`Frame`] overhead.
    pub const fn size(&self) -> usize {
        self.estimated_size
    }

    /// Returns `true` if the channel is ready to be read.
    pub fn is_ready(&self) -> bool {
        // Must have buffered the last frame before the channel is ready.
        if !self.closed {
            return false;
        }

        // Must have the possibility of contiguous frames.
        if self.inputs.len() != (self.last_frame_number + 1) as usize {
            return false;
        }

        // Check for contiguous frames.
        for i in 0..=self.last_frame_number {
            if !self.inputs.contains_key(&i) {
                return false;
            }
        }

        true
    }

    /// Returns all of the channel's [`Frame`]s concatenated together.
    ///
    /// ## Returns
    ///
    /// - `Some(Bytes)`: The concatenated frame data
    /// - `None`: If the channel is missing frames
    pub fn frame_data(&self) -> Option<Bytes> {
        if self.is_empty() {
            return None;
        }
        let mut data = Vec::with_capacity(self.size());
        (0..=self.last_frame_number).try_for_each(|i| {
            let frame = self.inputs.get(&i)?;
            data.extend_from_slice(&frame.data);
            Some(())
        })?;
        Some(data.into())
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloc::{
        string::{String, ToString},
        vec,
    };

    struct FrameValidityTestCase {
        #[allow(dead_code)]
        name: String,
        frames: Vec<Frame>,
        should_error: Vec<bool>,
        sizes: Vec<u64>,
        frame_data: Option<Bytes>,
    }

    fn run_frame_validity_test(test_case: FrameValidityTestCase) {
        // #[cfg(feature = "std")]
        // println!("Running test: {}", test_case.name);

        let id = [0xFF; 16];
        let block = BlockInfo::default();
        let mut channel = Channel::new(id, block);

        if test_case.frames.len() != test_case.should_error.len() ||
            test_case.frames.len() != test_case.sizes.len()
        {
            panic!("Test case length mismatch");
        }

        for (i, frame) in test_case.frames.iter().enumerate() {
            let result = channel.add_frame(frame.clone(), block);
            if test_case.should_error[i] {
                assert!(result.is_err());
            } else {
                assert!(result.is_ok());
            }
            assert_eq!(channel.size(), test_case.sizes[i] as usize);
        }

        if test_case.frame_data.is_some() {
            assert_eq!(channel.frame_data().unwrap(), test_case.frame_data.unwrap());
        }
    }

    #[test]
    fn test_channel_accessors() {
        let id = [0xFF; 16];
        let block = BlockInfo { number: 42, timestamp: 0, ..Default::default() };
        let channel = Channel::new(id, block);

        assert_eq!(channel.id(), id);
        assert_eq!(channel.open_block_number(), block.number);
        assert_eq!(channel.size(), 0);
        assert_eq!(channel.len(), 0);
        assert!(channel.is_empty());
        assert!(!channel.is_ready());
    }

    #[test]
    fn test_frame_validity() {
        let id = [0xFF; 16];
        let test_cases = [
            FrameValidityTestCase {
                name: "wrong channel".to_string(),
                frames: vec![Frame { id: [0xEE; 16], ..Default::default() }],
                should_error: vec![true],
                sizes: vec![0],
                frame_data: None,
            },
            FrameValidityTestCase {
                name: "double close".to_string(),
                frames: vec![
                    Frame { id, is_last: true, number: 2, data: b"four".to_vec() },
                    Frame { id, is_last: true, number: 1, ..Default::default() },
                ],
                should_error: vec![false, true],
                sizes: vec![204, 204],
                frame_data: None,
            },
            FrameValidityTestCase {
                name: "duplicate frame".to_string(),
                frames: vec![
                    Frame { id, number: 2, data: b"four".to_vec(), ..Default::default() },
                    Frame { id, number: 2, data: b"seven".to_vec(), ..Default::default() },
                ],
                should_error: vec![false, true],
                sizes: vec![204, 204],
                frame_data: None,
            },
            FrameValidityTestCase {
                name: "duplicate closing frames".to_string(),
                frames: vec![
                    Frame { id, number: 2, is_last: true, data: b"four".to_vec() },
                    Frame { id, number: 2, is_last: true, data: b"seven".to_vec() },
                ],
                should_error: vec![false, true],
                sizes: vec![204, 204],
                frame_data: None,
            },
            FrameValidityTestCase {
                name: "frame past closing".to_string(),
                frames: vec![
                    Frame { id, number: 2, is_last: true, data: b"four".to_vec() },
                    Frame { id, number: 10, data: b"seven".to_vec(), ..Default::default() },
                ],
                should_error: vec![false, true],
                sizes: vec![204, 204],
                frame_data: None,
            },
            FrameValidityTestCase {
                name: "prune after close frame".to_string(),
                frames: vec![
                    Frame { id, number: 0, is_last: false, data: b"seven".to_vec() },
                    Frame { id, number: 1, is_last: true, data: b"four".to_vec() },
                ],
                should_error: vec![false, false],
                sizes: vec![205, 409],
                frame_data: Some(b"sevenfour".to_vec().into()),
            },
            FrameValidityTestCase {
                name: "multiple valid frames, no data".to_string(),
                frames: vec![
                    Frame { id, number: 1, data: b"seven__".to_vec(), ..Default::default() },
                    Frame { id, number: 2, data: b"four".to_vec(), ..Default::default() },
                ],
                should_error: vec![false, false],
                sizes: vec![207, 411],
                // Notice: this is none because there is no frame at index 0,
                //         which causes the frame_data to short-circuit to None.
                frame_data: None,
            },
            FrameValidityTestCase {
                name: "multiple valid frames".to_string(),
                frames: vec![
                    Frame { id, number: 0, data: b"seven__".to_vec(), ..Default::default() },
                    Frame { id, number: 1, data: b"four".to_vec(), ..Default::default() },
                ],
                should_error: vec![false, false],
                sizes: vec![207, 411],
                frame_data: Some(b"seven__".to_vec().into()),
            },
        ];

        test_cases.into_iter().for_each(run_frame_validity_test);
    }
}
