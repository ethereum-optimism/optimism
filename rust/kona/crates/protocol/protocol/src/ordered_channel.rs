//! Ordered Channel Type
//!
//! An [`OrderedChannel`] enforces strict sequential frame ordering, rejecting any frame whose
//! number does not match the expected next frame. This is the post-Holocene channel type, matching
//! `op-node`'s `requireInOrder` behavior.

use alloc::vec::Vec;
use alloy_primitives::Bytes;

use crate::{BlockInfo, ChannelError, ChannelId, Frame};

/// An error returned when reading data from an [`OrderedChannel`].
#[derive(Debug, thiserror::Error, Clone, Copy, PartialEq, Eq, Hash)]
pub enum ReadError {
    /// The channel is not ready (not all frames have been received).
    #[error("Channel is not ready")]
    NotReady,
    /// The channel has no frames.
    #[error("Channel is empty")]
    Empty,
}

/// An ordered channel that enforces strict sequential frame ingestion.
///
/// Unlike [`Channel`], which accepts frames out of order and checks contiguity at read time,
/// `OrderedChannel` rejects any frame whose number does not equal the current frame count.
/// This matches `op-node`'s Holocene behavior where `requireInOrder` is true.
///
/// [`Channel`]: crate::Channel
#[derive(Debug, Clone)]
pub struct OrderedChannel {
    /// The unique identifier for this channel.
    pub id: ChannelId,
    /// The block that the channel was opened at.
    pub open_block: BlockInfo,
    /// Estimated memory size, used to drop the channel if we have too much data.
    pub estimated_size: usize,
    /// True if the last frame has been buffered.
    pub closed: bool,
    /// Frames stored in sequential order.
    pub inputs: Vec<Frame>,
    /// The highest L1 inclusion block that a frame was included in.
    pub highest_l1_inclusion_block: BlockInfo,
}

impl OrderedChannel {
    /// Create a new [`OrderedChannel`] with the given [`ChannelId`] and [`BlockInfo`].
    pub fn new(id: ChannelId, open_block: BlockInfo) -> Self {
        Self {
            id,
            open_block,
            estimated_size: 0,
            closed: false,
            inputs: Vec::new(),
            highest_l1_inclusion_block: BlockInfo::default(),
        }
    }

    /// Returns the [`ChannelId`].
    pub const fn id(&self) -> ChannelId {
        self.id
    }

    /// Returns the number of frames ingested.
    pub const fn len(&self) -> usize {
        self.inputs.len()
    }

    /// Returns if the channel is empty.
    pub const fn is_empty(&self) -> bool {
        self.inputs.is_empty()
    }

    /// Returns the block number of the L1 block that contained the first [`Frame`].
    pub const fn open_block_number(&self) -> u64 {
        self.open_block.number
    }

    /// Returns the estimated size of the channel including [`Frame`] overhead.
    pub const fn size(&self) -> usize {
        self.estimated_size
    }

    /// Add a frame to the channel. The frame number must equal the current frame count
    /// (strict sequential ordering).
    pub fn add_frame(
        &mut self,
        frame: Frame,
        l1_inclusion_block: BlockInfo,
    ) -> Result<(), ChannelError> {
        if frame.id != self.id {
            return Err(ChannelError::FrameIdMismatch);
        }
        if self.closed {
            return Err(ChannelError::ChannelClosed);
        }

        let expected = self.inputs.len() as u16;
        if frame.number != expected {
            return Err(ChannelError::FrameOutOfOrder { expected, got: frame.number });
        }

        if frame.is_last {
            self.closed = true;
        }

        if self.highest_l1_inclusion_block.number < l1_inclusion_block.number {
            self.highest_l1_inclusion_block = l1_inclusion_block;
        }

        self.estimated_size += frame.size();
        self.inputs.push(frame);
        Ok(())
    }

    /// Returns `true` if the channel is ready to be read.
    /// Since frames are ingested in order, the channel is ready as soon as it is closed.
    pub const fn is_ready(&self) -> bool {
        self.closed
    }

    /// Returns all of the channel's [`Frame`] data concatenated together.
    ///
    /// Returns an error if the channel is empty or not yet ready.
    pub fn data(&self) -> Result<Bytes, ReadError> {
        if self.inputs.is_empty() {
            return Err(ReadError::Empty);
        }
        if !self.closed {
            return Err(ReadError::NotReady);
        }
        Ok(self.inputs.iter().flat_map(|f| &f.data).copied().collect::<Vec<_>>().into())
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloc::string::ToString;

    fn test_id() -> ChannelId {
        [0xFF; 16]
    }

    fn frame(id: ChannelId, number: u16, data: &[u8], is_last: bool) -> Frame {
        Frame { id, number, data: data.to_vec(), is_last }
    }

    #[test]
    fn test_ordered_channel_accessors() {
        let id = test_id();
        let block = BlockInfo { number: 42, timestamp: 0, ..Default::default() };
        let channel = OrderedChannel::new(id, block);

        assert_eq!(channel.id(), id);
        assert_eq!(channel.open_block_number(), 42);
        assert_eq!(channel.size(), 0);
        assert_eq!(channel.len(), 0);
        assert!(channel.is_empty());
        assert!(!channel.is_ready());
    }

    #[test]
    fn test_ordered_frames_accepted() {
        let id = test_id();
        let block = BlockInfo::default();
        let mut channel = OrderedChannel::new(id, block);

        assert!(channel.add_frame(frame(id, 0, b"hello", false), block).is_ok());
        assert!(channel.add_frame(frame(id, 1, b"world", true), block).is_ok());
        assert!(channel.is_ready());
        assert_eq!(channel.len(), 2);
        assert_eq!(channel.data().unwrap().as_ref(), b"helloworld");
    }

    #[test]
    fn test_wrong_channel_id_rejected() {
        let id = test_id();
        let block = BlockInfo::default();
        let mut channel = OrderedChannel::new(id, block);

        let err = channel.add_frame(frame([0xEE; 16], 0, b"bad", false), block).unwrap_err();
        assert_eq!(err, ChannelError::FrameIdMismatch);
    }

    #[test]
    fn test_out_of_order_frame_rejected() {
        let id = test_id();
        let block = BlockInfo::default();
        let mut channel = OrderedChannel::new(id, block);

        // Frame 0 succeeds
        assert!(channel.add_frame(frame(id, 0, b"first", false), block).is_ok());

        // Frame 2 (skipping 1) is rejected
        let err = channel.add_frame(frame(id, 2, b"skip", false), block).unwrap_err();
        assert_eq!(err, ChannelError::FrameOutOfOrder { expected: 1, got: 2 });
        assert_eq!(channel.len(), 1);
    }

    #[test]
    fn test_frame_after_close_rejected() {
        let id = test_id();
        let block = BlockInfo::default();
        let mut channel = OrderedChannel::new(id, block);

        assert!(channel.add_frame(frame(id, 0, b"only", true), block).is_ok());
        assert!(channel.is_ready());

        let err = channel.add_frame(frame(id, 1, b"extra", false), block).unwrap_err();
        assert_eq!(err, ChannelError::ChannelClosed);
    }

    #[test]
    fn test_attack_scenario_cross_tx_out_of_order() {
        // Attack from the finding: T1=[F0,F1,F2], T2=[F4,F5,F6(is_last)], T3=[F3]
        // OrderedChannel should accept F0-F2 then reject F4 (expected F3).
        let id = test_id();
        let block = BlockInfo::default();
        let mut channel = OrderedChannel::new(id, block);

        // T1 frames arrive in order
        assert!(channel.add_frame(frame(id, 0, b"f0", false), block).is_ok());
        assert!(channel.add_frame(frame(id, 1, b"f1", false), block).is_ok());
        assert!(channel.add_frame(frame(id, 2, b"f2", false), block).is_ok());

        // T2 starts at frame 4 — out of order, rejected
        let err = channel.add_frame(frame(id, 4, b"f4", false), block).unwrap_err();
        assert_eq!(err, ChannelError::FrameOutOfOrder { expected: 3, got: 4 });

        // Channel is not ready and not closed
        assert!(!channel.is_ready());
        assert_eq!(channel.len(), 3);
    }

    #[test]
    fn test_single_frame_channel() {
        let id = test_id();
        let block = BlockInfo::default();
        let mut channel = OrderedChannel::new(id, block);

        assert!(channel.add_frame(frame(id, 0, b"all", true), block).is_ok());
        assert!(channel.is_ready());
        assert_eq!(channel.data().unwrap().as_ref(), b"all");
    }

    #[test]
    fn test_data_empty_channel() {
        let id = test_id();
        let block = BlockInfo::default();
        let channel = OrderedChannel::new(id, block);

        assert_eq!(channel.data(), Err(ReadError::Empty));
    }

    #[test]
    fn test_data_not_ready() {
        let id = test_id();
        let block = BlockInfo::default();
        let mut channel = OrderedChannel::new(id, block);

        assert!(channel.add_frame(frame(id, 0, b"partial", false), block).is_ok());
        assert_eq!(channel.data(), Err(ReadError::NotReady));
    }

    #[test]
    fn test_l1_inclusion_block_tracking() {
        let id = test_id();
        let block1 = BlockInfo { number: 10, ..Default::default() };
        let block2 = BlockInfo { number: 20, ..Default::default() };
        let mut channel = OrderedChannel::new(id, block1);

        assert!(channel.add_frame(frame(id, 0, b"a", false), block1).is_ok());
        assert_eq!(channel.highest_l1_inclusion_block.number, 10);

        assert!(channel.add_frame(frame(id, 1, b"b", true), block2).is_ok());
        assert_eq!(channel.highest_l1_inclusion_block.number, 20);
    }

    #[test]
    fn test_error_display() {
        let err = ChannelError::FrameOutOfOrder { expected: 3, got: 5 };
        assert_eq!(err.to_string(), "Frame out of order: expected 3, got 5");
    }
}
