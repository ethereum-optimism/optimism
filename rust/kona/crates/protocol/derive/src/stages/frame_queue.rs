//! This module contains the [`FrameQueue`] stage of the derivation pipeline.

use crate::{
    NextFrameProvider, OriginAdvancer, OriginProvider, PipelineError, PipelineResult, Stage,
};
use alloc::{boxed::Box, collections::VecDeque, sync::Arc};
use alloy_eips::BlockNumHash;
use alloy_primitives::Bytes;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::{RollupConfig, SystemConfig};
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
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + Stage + Debug,
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
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + Stage + Debug,
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

            // If the frames are in different channels, and the current channel is not last,
            // drop ONLY the current (unclosed channel's) frame and re-examine the preceding
            // pair. This peels the unclosed channel one frame at a time, without discarding
            // interleaved frames of other (valid, closed) channels that happen to sit in the
            // index range. Matches op-node `pruneFrameQueue` (op-node/rollup/derive/
            // frame_queue.go): `discard(0)` + `i--`.
            if !extends_channel && !prev_frame.is_last && next_frame.number == 0 {
                self.queue.remove(i);
                i = i.saturating_sub(1);
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
        let _queue_size = self.queue.iter().map(|f| f.size()).sum::<usize>() as f64;
        kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_FRAME_QUEUE_MEM, _queue_size);

        // Prune frames if Holocene is active.
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        self.prune(origin);

        Ok(())
    }
}

#[async_trait]
impl<P> OriginAdvancer for FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + Stage + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

#[async_trait]
impl<P> NextFrameProvider for FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + Stage + Send + Debug,
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
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + Stage + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P> Stage for FrameQueue<P>
where
    P: FrameQueueProvider + OriginAdvancer + OriginProvider + Stage + Send + Debug,
{
    async fn reset(
        &mut self,
        l1_origin: BlockNumHash,
        system_config: SystemConfig,
    ) -> PipelineResult<()> {
        self.prev.reset(l1_origin, system_config).await?;
        self.queue = VecDeque::default();
        Ok(())
    }

    async fn activate(&mut self) -> PipelineResult<()> {
        self.prev.activate().await?;
        self.queue = VecDeque::default();
        Ok(())
    }

    async fn flush_channel(&mut self) -> PipelineResult<()> {
        self.prev.flush_channel().await?;
        self.queue = VecDeque::default();
        Ok(())
    }

    async fn provide_block(&mut self, block: BlockInfo) -> PipelineResult<()> {
        self.prev.provide_block(block).await
    }
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::test_utils::TestFrameQueueProvider;
    use alloc::vec;
    use kona_genesis::HardForkConfig;

    #[tokio::test]
    async fn test_frame_queue_reset() {
        let mock = TestFrameQueueProvider::new(vec![]);
        let mut frame_queue = FrameQueue::new(mock, Default::default());
        assert!(!frame_queue.prev.reset);
        frame_queue.reset(BlockNumHash::default(), SystemConfig::default()).await.unwrap();
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


    /// A faithful Rust port of op-node's `pruneFrameQueue`
    /// (op-node/rollup/derive/frame_queue.go). This is the differential ORACLE. It is an
    /// independent implementation (NOT calling kona's `prune`) so a divergence between this
    /// and the real `FrameQueue::prune` is meaningful.
    /// ```
    fn opnode_prune_oracle(frames: &[Frame]) -> alloc::vec::Vec<Frame> {
        let mut frames: alloc::vec::Vec<Frame> = frames.to_vec();
        if frames.is_empty() {
            return frames;
        }
        let mut i: usize = 0;
        while i < frames.len() - 1 {
            let current = frames[i].clone();
            let next = frames[i + 1].clone();
            if current.id == next.id {
                if current.is_last {
                    frames.remove(i + 1); // discard(1): drop next
                    continue;
                }
                if next.number != current.number + 1 {
                    frames.remove(i + 1); // discard(1): drop next
                    continue;
                }
            } else {
                if next.number == 0 && !current.is_last {
                    frames.remove(i); // discard(0): drop current
                    // op-node: `if i > 0 { i-- }`. saturating_sub is identical given i >= 0.
                    i = i.saturating_sub(1);
                    continue;
                }
                if next.number != 0 {
                    frames.remove(i + 1); // discard(1): drop next
                    continue;
                }
            }
            i += 1;
        }
        frames
    }

    /// Runs the REAL (fixed) `FrameQueue::prune` over an input frame slice with Holocene active,
    /// returning the surviving queue. The same-crate test can set the private `queue` field
    /// directly and call `prune`.
    fn real_prune(frames: &[Frame]) -> alloc::vec::Vec<Frame> {
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        // The `prev` stage is irrelevant to `prune`; we only exercise the queue/prune logic.
        let prev = TestFrameQueueProvider::new(vec![]);
        let mut fq = FrameQueue::new(prev, cfg);
        fq.queue = frames.iter().cloned().collect::<VecDeque<Frame>>();
        // Holocene-active origin so prune actually runs.
        fq.prune(BlockInfo::default());
        fq.queue.into_iter().collect()
    }

    /// Compare two frame sequences by the `(id, number, is_last)` tuple (ignoring payload bytes).
    fn frames_eq_by_key(a: &[Frame], b: &[Frame]) -> bool {
        a.len() == b.len()
            && a.iter()
                .zip(b.iter())
                .all(|(x, y)| x.id == y.id && x.number == y.number && x.is_last == y.is_last)
    }

    fn fmt_frames(frames: &[Frame]) -> alloc::string::String {
        use alloc::string::String;
        use core::fmt::Write;
        let mut s = String::from("[");
        for (i, f) in frames.iter().enumerate() {
            if i > 0 {
                s.push_str(", ");
            }
            let _ = write!(s, "({:#04x},{},{})", f.id[0], f.number, f.is_last);
        }
        s.push(']');
        s
    }

    /// Emits a fuzz summary. When the `metrics` feature is enabled (which links `std`), prints to
    /// stderr so `--nocapture` shows the counts. Under default `no_std` test builds it routes the
    /// summary through `tracing::info!` (always linked). Counts are also embedded in assertion
    /// messages, so they are visible on failure regardless of feature set.
    fn log_fuzz(summary: &str, first_div: Option<&str>) {
        #[cfg(feature = "metrics")]
        {
            std::eprintln!("{summary}");
            if let Some(d) = first_div {
                std::eprintln!("first divergence:\n{d}");
            }
        }
        #[cfg(not(feature = "metrics"))]
        {
            info!(target: "frame_queue", "{}", summary);
            if let Some(d) = first_div {
                info!(target: "frame_queue", "first divergence: {}", d);
            }
        }
    }

    /// The adversarial interleaved-channel vector (id2 == 0x02, id0 == 0x00):
    /// `[(id2,0,last),(id2,2,false),(id0,0,last),(id0,2,false),(id2,0,false),(id0,0,false)]`.
    fn adversarial_vector() -> [Frame; 6] {
        [
            crate::frame!(0x02, 0, vec![0xDD; 4], true),
            crate::frame!(0x02, 2, vec![0xDD; 4], false),
            crate::frame!(0x00, 0, vec![0xDD; 4], true),
            crate::frame!(0x00, 2, vec![0xDD; 4], false),
            crate::frame!(0x02, 0, vec![0xDD; 4], false),
            crate::frame!(0x00, 0, vec![0xDD; 4], false),
        ]
    }

    /// Test 1: ORACLE self-check. The op-node oracle must keep BOTH valid closed channels on
    /// the adversarial vector, matching real op-node behavior.
    #[test]
    fn test_oracle_matches_opnode() {
        let input = adversarial_vector();
        let expected = [
            crate::frame!(0x02, 0, vec![0xDD; 4], true),
            crate::frame!(0x00, 0, vec![0xDD; 4], true),
        ];
        let got = opnode_prune_oracle(&input);
        assert!(
            frames_eq_by_key(&got, &expected),
            "oracle must match real op-node: expected {}, got {}",
            fmt_frames(&expected),
            fmt_frames(&got)
        );
    }

    /// Test 2: the REAL FIXED `FrameQueue::prune` must now keep BOTH valid closed channels on
    /// the adversarial vector (== op-node). Before the fix it kept only `[(id0,0,false)]`.
    #[test]
    fn test_real_prune_matches_opnode() {
        let input = adversarial_vector();
        let expected = [
            crate::frame!(0x02, 0, vec![0xDD; 4], true),
            crate::frame!(0x00, 0, vec![0xDD; 4], true),
        ];
        let got = real_prune(&input);
        assert!(
            frames_eq_by_key(&got, &expected),
            "FIXED prune must keep both closed channels: expected {}, got {}",
            fmt_frames(&expected),
            fmt_frames(&got)
        );
        // And it must agree with the oracle on this vector.
        assert!(
            frames_eq_by_key(&got, &opnode_prune_oracle(&input)),
            "FIXED prune must equal op-node oracle on the adversarial vector"
        );
    }

    /// Test 3a: EXHAUSTIVE differential fuzz over the small space that previously had 14,972
    /// divergences. ids ∈ {0,1,2}, numbers ∈ {0,1,2,3}, `is_last` ∈ {true,false}, lengths 1..=6.
    /// Asserts ZERO divergences between the real fixed prune and the op-node oracle.
    #[test]
    fn test_exhaustive_differential_fuzz() {
        // Frame "alphabet": (id, number, is_last). 3 ids * 4 numbers * 2 is_last = 24 cells.
        let mut cells: alloc::vec::Vec<Frame> = alloc::vec::Vec::with_capacity(24);
        for id in 0u8..3 {
            for number in 0u16..4 {
                for is_last in [false, true] {
                    cells.push(Frame { id: [id; 16], number, data: vec![0xAB; 2], is_last });
                }
            }
        }
        let base = cells.len();

        let mut total: u64 = 0;
        let mut divergences: u64 = 0;
        let mut first_div: Option<alloc::string::String> = None;

        // All sequences of length L for L in 1..=6, enumerated in base-`base` counting.
        for len in 1usize..=6 {
            let combos: u64 = (base as u64).pow(len as u32);
            for n in 0..combos {
                let mut seq: alloc::vec::Vec<Frame> = alloc::vec::Vec::with_capacity(len);
                let mut acc = n;
                for _ in 0..len {
                    let idx = (acc % base as u64) as usize;
                    acc /= base as u64;
                    seq.push(cells[idx].clone());
                }

                let got = real_prune(&seq);
                let want = opnode_prune_oracle(&seq);
                total += 1;
                if !frames_eq_by_key(&got, &want) {
                    divergences += 1;
                    if first_div.is_none() {
                        first_div = Some(alloc::format!(
                            "INPUT {}\n  real:   {}\n  opnode: {}",
                            fmt_frames(&seq),
                            fmt_frames(&got),
                            fmt_frames(&want)
                        ));
                    }
                }
            }
        }

        let summary = alloc::format!(
            "[prune-parity exhaustive fuzz] inputs checked = {total}, divergences = {divergences}"
        );
        log_fuzz(&summary, first_div.as_deref());
        // Counts are embedded in the assert message so they are visible even without --nocapture
        // on failure, and the exhaustive coverage is asserted to be large.
        assert!(total > 200_000, "expected a large exhaustive space, only checked {total}");
        assert_eq!(divergences, 0, "exhaustive fuzz: {summary}");
    }

    /// Test 3b: RANDOM large-space differential fuzz. >= 2,000,000 random sequences, lengths up
    /// to 16, ids 0..4, numbers 0..5, random `is_last`. Asserts ZERO divergences.
    #[test]
    fn test_random_differential_fuzz() {
        // SplitMix64 — a tiny, deterministic, no_std-friendly PRNG. Fixed seed for reproducibility.
        struct SplitMix64(u64);
        impl SplitMix64 {
            fn next(&mut self) -> u64 {
                self.0 = self.0.wrapping_add(0x9E3779B97F4A7C15);
                let mut z = self.0;
                z = (z ^ (z >> 30)).wrapping_mul(0xBF58476D1CE4E5B9);
                z = (z ^ (z >> 27)).wrapping_mul(0x94D049BB133111EB);
                z ^ (z >> 31)
            }
            fn below(&mut self, n: u64) -> u64 {
                self.next() % n
            }
        }

        const ITERS: u64 = 2_000_000;
        const MAX_LEN: usize = 16;
        const N_IDS: u8 = 4; // ids 0..4
        const N_NUMS: u16 = 5; // numbers 0..5

        // Fixed seed for reproducibility (fixed constant).
        let mut rng = SplitMix64(0x4E4B44303031C0FFu64 ^ 0x5DEECE66D);
        let mut total: u64 = 0;
        let mut divergences: u64 = 0;
        let mut first_div: Option<alloc::string::String> = None;

        for _ in 0..ITERS {
            // Length 1..=MAX_LEN (avoid empty: prune's `len()-1` underflows on empty, and the
            // real pipeline never prunes an empty queue).
            let len = 1 + (rng.below(MAX_LEN as u64) as usize);
            let mut seq: alloc::vec::Vec<Frame> = alloc::vec::Vec::with_capacity(len);
            for _ in 0..len {
                let id = rng.below(N_IDS as u64) as u8;
                let number = rng.below(N_NUMS as u64) as u16;
                let is_last = rng.below(2) == 1;
                seq.push(Frame { id: [id; 16], number, data: vec![0x7C; 2], is_last });
            }

            let got = real_prune(&seq);
            let want = opnode_prune_oracle(&seq);
            total += 1;
            if !frames_eq_by_key(&got, &want) {
                divergences += 1;
                if first_div.is_none() {
                    first_div = Some(alloc::format!(
                        "INPUT {}\n  real:   {}\n  opnode: {}",
                        fmt_frames(&seq),
                        fmt_frames(&got),
                        fmt_frames(&want)
                    ));
                }
            }
        }

        let summary = alloc::format!(
            "[prune-parity random fuzz] inputs checked = {total}, divergences = {divergences}"
        );
        log_fuzz(&summary, first_div.as_deref());
        assert_eq!(total, ITERS, "must check exactly {ITERS} random inputs");
        assert_eq!(divergences, 0, "random fuzz: {summary}");
    }
}
