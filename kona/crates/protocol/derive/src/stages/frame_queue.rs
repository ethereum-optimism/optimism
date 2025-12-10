//! This module contains the [FrameQueue] stage of the derivation pipeline.

use crate::{
    NextFrameProvider, OriginAdvancer, OriginProvider, PipelineError, PipelineResult, Signal,
    SignalReceiver,
};
use alloc::{boxed::Box, collections::VecDeque, sync::Arc};
use alloy_primitives::Bytes;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, Frame};

/// Provides data frames for the [`FrameQueue`] stage.
#[async_trait]
pub trait FrameQueueProvider {
    /// An item that can be converted into a byte array.
    type Item: Into<Bytes>;

    /// Retrieves the next data item from the L1 retrieval stage.
    /// If there is data, it pushes it into the next stage.
    /// If there is no data, it returns an error.
    async fn next_data(&mut self) -> PipelineResult<Self::Item>;
}

/// The [`FrameQueue`] stage of the derivation pipeline.
/// This stage takes the output of the [`L1Retrieval`] stage and parses it into frames.
///
/// [`L1Retrieval`]: crate::stages::L1Retrieval
#[derive(Debug)]
pub struct FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// The previous stage in the pipeline.
    pub prev: P,
    /// The current frame queue.
    pub queue: VecDeque<Frame>,
    /// The rollup config.
    pub rollup_config: Arc<RollupConfig>,
}

impl<P> FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// Create a new [`FrameQueue`] stage with the given previous [`L1Retrieval`] stage.
    ///
    /// [`L1Retrieval`]: crate::stages::L1Retrieval
    pub const fn new(prev: P, cfg: Arc<RollupConfig>) -> Self {
        Self { prev, queue: VecDeque::new(), rollup_config: cfg }
    }

    /// Returns if holocene is active.
    pub fn is_holocene_active(&self, origin: BlockInfo) -> bool {
        self.rollup_config.is_holocene_active(origin.timestamp)
    }

    /// Prunes frames if Holocene is active.
    pub fn prune(&mut self, origin: BlockInfo) {
        if !self.is_holocene_active(origin) {
            return;
        }

        let mut i = 0;
        while i < self.queue.len() - 1 {
            let prev_frame = &self.queue[i];
            let next_frame = &self.queue[i + 1];
            let extends_channel = prev_frame.id == next_frame.id;

            // If the frames are in the same channel, and the frame numbers are not sequential,
            // drop the next frame.
            if extends_channel && prev_frame.number + 1 != next_frame.number {
                self.queue.remove(i + 1);
                continue;
            }

            // If the frames are in the same channel, and the previous is last, drop the next frame.
            if extends_channel && prev_frame.is_last {
                self.queue.remove(i + 1);
                continue;
            }

            // If the frames are in different channels, the next frame must be first.
            if !extends_channel && next_frame.number != 0 {
                self.queue.remove(i + 1);
                continue;
            }

            // If the frames are in different channels, and the current channel is not last, walk
            // back the channel and drop all prev frames.
            if !extends_channel && !prev_frame.is_last && next_frame.number == 0 {
                // Find the index of the first frame in the queue with the same channel ID
                // as the previous frame.
                let first_frame =
                    self.queue.iter().position(|f| f.id == prev_frame.id).expect("infallible");

                // Drain all frames from the previous channel.
                let drained = self.queue.drain(first_frame..=i);
                i = i.saturating_sub(drained.len());
                continue;
            }

            i += 1;
        }
    }

    /// Loads more frames into the [`FrameQueue`].
    pub async fn load_frames(&mut self) -> PipelineResult<()> {
        // Skip loading frames if the queue is not empty.
        if !self.queue.is_empty() {
            return Ok(());
        }

        let data = match self.prev.next_data().await {
            Ok(data) => data,
            Err(e) => {
                debug!(target: "frame_queue", "Failed to retrieve data: {:?}", e);
                // SAFETY: Bubble up potential EOF error without wrapping.
                return Err(e);
            }
        };

        let Ok(frames) = Frame::parse_frames(&data.into()) else {
            // There may be more frames in the queue for the
            // pipeline to advance, so don't return an error here.
            error!(target: "frame_queue", "Failed to parse frames from data.");
            return Ok(());
        };

        // Optimistically extend the queue with the new frames.
        self.queue.extend(frames);

        // Update metrics with last frame count
        kona_macros::set!(
            gauge,
            crate::metrics::Metrics::PIPELINE_FRAME_QUEUE_BUFFER,
            self.queue.len() as f64
        );
        let queue_size = self.queue.iter().map(|f| f.size()).sum::<usize>() as f64;
        kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_FRAME_QUEUE_MEM, queue_size);

        // Prune frames if Holocene is active.
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        self.prune(origin);

        Ok(())
    }
}

#[async_trait]
impl<P> OriginAdvancer for FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

#[async_trait]
impl<P> NextFrameProvider for FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn next_frame(&mut self) -> PipelineResult<Frame> {
        self.load_frames().await?;

        // If we did not add more frames but still have more data, retry this function.
        if self.queue.is_empty() {
            trace!(target: "frame_queue", "Queue is empty after fetching data. Retrying next_frame.");
            return Err(PipelineError::NotEnoughData.temp());
        }

        Ok(self.queue.pop_front().expect("Frame queue impossibly empty"))
    }
}

impl<P> OriginProvider for FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P> SignalReceiver for FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        self.prev.signal(signal).await?;
        self.queue = VecDeque::default();
        Ok(())
    }
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::{test_utils::TestFrameQueueProvider, types::ResetSignal};
    use alloc::vec;
    use kona_genesis::HardForkConfig;

    #[tokio::test]
    async fn test_frame_queue_reset() {
        let mock = TestFrameQueueProvider::new(vec![]);
        let mut frame_queue = FrameQueue::new(mock, Default::default());
        assert!(!frame_queue.prev.reset);
        frame_queue.signal(ResetSignal::default().signal()).await.unwrap();
        assert_eq!(frame_queue.queue.len(), 0);
        assert!(frame_queue.prev.reset);
    }

    #[tokio::test]
    async fn test_frame_queue_empty_bytes() {
        let data = vec![Ok(Bytes::from(vec![0x00]))];
        let mut mock = TestFrameQueueProvider::new(data);
        mock.set_origin(BlockInfo::default());
        let mut frame_queue = FrameQueue::new(mock, Default::default());
        assert!(!frame_queue.is_holocene_active(BlockInfo::default()));
        let err = frame_queue.next_frame().await.unwrap_err();
        assert_eq!(err, PipelineError::NotEnoughData.temp());
    }

    #[tokio::test]
    async fn test_frame_queue_no_frames_decoded() {
        let data = vec![Err(PipelineError::Eof.temp()), Ok(Bytes::default())];
        let mut mock = TestFrameQueueProvider::new(data);
        mock.set_origin(BlockInfo::default());
        let mut frame_queue = FrameQueue::new(mock, Default::default());
        assert!(!frame_queue.is_holocene_active(BlockInfo::default()));
        let err = frame_queue.next_frame().await.unwrap_err();
        assert_eq!(err, PipelineError::NotEnoughData.temp());
    }

    #[tokio::test]
    async fn test_frame_queue_wrong_derivation_version() {
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_origin(BlockInfo::default())
            .with_raw_frames(Bytes::from(vec![0x01]))
            .with_expected_err(PipelineError::NotEnoughData.temp())
            .build();
        assert.holocene_active(false);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_frame_queue_frame_too_short() {
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_origin(BlockInfo::default())
            .with_raw_frames(Bytes::from(vec![0x00, 0x01]))
            .with_expected_err(PipelineError::NotEnoughData.temp())
            .build();
        assert.holocene_active(false);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_frame_queue_single_frame() {
        let frames = [crate::frame!(0xFF, 0, vec![0xDD; 50], true)];
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_expected_frames(&frames)
            .with_origin(BlockInfo::default())
            .with_frames(&frames)
            .build();
        assert.holocene_active(false);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_frame_queue_multiple_frames() {
        let frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], false),
            crate::frame!(0xFF, 2, vec![0xDD; 50], true),
        ];
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_expected_frames(&frames)
            .with_origin(BlockInfo::default())
            .with_frames(&frames)
            .build();
        assert.holocene_active(false);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_frame_queue_missing_origin() {
        let frames = [crate::frame!(0xFF, 0, vec![0xDD; 50], true)];
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_expected_frames(&frames)
            .with_frames(&frames)
            .build();
        assert.holocene_active(false);
        assert.missing_origin().await;
    }

    #[tokio::test]
    async fn test_holocene_valid_frames() {
        let frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], false),
            crate::frame!(0xFF, 2, vec![0xDD; 50], true),
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&frames)
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_single_frame() {
        let frames = [crate::frame!(0xFF, 1, vec![0xDD; 50], true)];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&frames)
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_unordered_frames() {
        let frames = [
            // -- First Channel --
            crate::frame!(0xEE, 0, vec![0xDD; 50], false),
            crate::frame!(0xEE, 1, vec![0xDD; 50], false),
            crate::frame!(0xEE, 2, vec![0xDD; 50], true),
            crate::frame!(0xEE, 3, vec![0xDD; 50], false), // Dropped
            // -- Next Channel --
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&[&frames[0..3], &frames[4..]].concat())
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_non_sequential_frames() {
        let frames = [
            // -- First Channel --
            crate::frame!(0xEE, 0, vec![0xDD; 50], false),
            crate::frame!(0xEE, 1, vec![0xDD; 50], false),
            crate::frame!(0xEE, 3, vec![0xDD; 50], true), // Dropped
            crate::frame!(0xEE, 4, vec![0xDD; 50], false), // Dropped
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&frames[0..2])
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_unclosed_channel() {
        let frames = [
            // -- First Channel --
            crate::frame!(0xEE, 0, vec![0xDD; 50], false),
            crate::frame!(0xEE, 1, vec![0xDD; 50], false),
            crate::frame!(0xEE, 2, vec![0xDD; 50], false),
            crate::frame!(0xEE, 3, vec![0xDD; 50], false),
            // -- Next Channel --
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&frames[4..])
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_unstarted_channel() {
        let frames = [
            // -- First Channel --
            crate::frame!(0xDD, 0, vec![0xDD; 50], false),
            crate::frame!(0xDD, 1, vec![0xDD; 50], false),
            crate::frame!(0xDD, 2, vec![0xDD; 50], false),
            crate::frame!(0xDD, 3, vec![0xDD; 50], true),
            // -- Second Channel --
            crate::frame!(0xEE, 1, vec![0xDD; 50], false), // Dropped
            crate::frame!(0xEE, 2, vec![0xDD; 50], true),  // Dropped
            // -- Third Channel --
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&[&frames[0..4], &frames[6..]].concat())
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_unclosed_channel_with_invalid_start() {
        let frames = [
            // -- First Channel --
            crate::frame!(0xEE, 0, vec![0xDD; 50], false),
            crate::frame!(0xEE, 1, vec![0xDD; 50], false),
            crate::frame!(0xEE, 2, vec![0xDD; 50], false),
            crate::frame!(0xEE, 3, vec![0xDD; 50], false),
            // -- Next Channel --
            crate::frame!(0xFF, 1, vec![0xDD; 50], false), // Dropped
            crate::frame!(0xFF, 2, vec![0xDD; 50], true),  // Dropped
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&frames[0..4])
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_replace_channel() {
        let frames = [
            // -- First Channel - VALID & CLOSED --
            crate::frame!(0xDD, 0, vec![0xDD; 50], false),
            crate::frame!(0xDD, 1, vec![0xDD; 50], true),
            // -- Second Channel - VALID & NOT CLOSED / DROPPED --
            crate::frame!(0xEE, 0, vec![0xDD; 50], false),
            crate::frame!(0xEE, 1, vec![0xDD; 50], false),
            // -- Third Channel - VALID & CLOSED / REPLACES CHANNEL #2 --
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&[&frames[0..2], &frames[4..]].concat())
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_interleaved_invalid_channel() {
        let frames = [
            // -- First channel is dropped since it is replaced by the second channel --
            // -- Second channel is dropped since it isn't closed --
            crate::frame!(0x01, 0, vec![0xDD; 50], false),
            crate::frame!(0x02, 0, vec![0xDD; 50], false),
            crate::frame!(0x01, 1, vec![0xDD; 50], true),
            crate::frame!(0x02, 1, vec![0xDD; 50], false),
            // -- Third Channel - VALID & CLOSED --
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&frames[4..])
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }

    #[tokio::test]
    async fn test_holocene_interleaved_valid_channel() {
        let frames = [
            // -- First channel is dropped since it is replaced by the second channel --
            // -- Second channel is successfully closed so it's valid --
            crate::frame!(0x01, 0, vec![0xDD; 50], false),
            crate::frame!(0x02, 0, vec![0xDD; 50], false),
            crate::frame!(0x01, 1, vec![0xDD; 50], true),
            crate::frame!(0x02, 1, vec![0xDD; 50], true),
            // -- Third Channel - VALID & CLOSED --
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        };
        let assert = crate::test_utils::FrameQueueBuilder::new()
            .with_rollup_config(&cfg)
            .with_origin(BlockInfo::default())
            .with_expected_frames(&[&frames[1..2], &frames[3..]].concat())
            .with_frames(&frames)
            .build();
        assert.holocene_active(true);
        assert.next_frames().await;
    }
}
