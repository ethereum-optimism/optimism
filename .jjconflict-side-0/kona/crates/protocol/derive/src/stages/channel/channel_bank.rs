//! This module contains the `ChannelBank` struct.

use crate::{
    ChannelReaderProvider, NextFrameProvider, OriginAdvancer, OriginProvider, PipelineError,
    PipelineErrorKind, PipelineResult, Signal, SignalReceiver,
};
use alloc::{boxed::Box, collections::VecDeque, sync::Arc};
use alloy_primitives::{Bytes, hex, map::HashMap};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, Channel, ChannelId, Frame};

/// The maximum size of a channel bank.
pub(crate) const MAX_CHANNEL_BANK_SIZE: usize = 100_000_000;

/// The maximum size of a channel bank after the Fjord Hardfork.
pub(crate) const FJORD_MAX_CHANNEL_BANK_SIZE: usize = 1_000_000_000;

/// [`ChannelBank`] is a stateful stage that does the following:
/// 1. Unmarshalls frames from L1 transaction data
/// 2. Applies those frames to a channel
/// 3. Attempts to read from the channel when it is ready
/// 4. Prunes channels (not frames) when the channel bank is too large.
///
/// Note: we prune before we ingest data.
/// As we switch between ingesting data & reading, the prune step occurs at an odd point
/// Specifically, the channel bank is not allowed to become too large between successive calls
/// to `IngestData`. This means that we can do an ingest and then do a read while becoming too
/// large. [`ChannelBank`] buffers channel frames, and emits full channel data
#[derive(Debug)]
pub struct ChannelBank<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
    /// Map of channels by ID.
    pub channels: HashMap<ChannelId, Channel>,
    /// Channels in FIFO order.
    pub channel_queue: VecDeque<ChannelId>,
    /// The previous stage of the derivation pipeline.
    pub prev: P,
}

impl<P> ChannelBank<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// Create a new [`ChannelBank`] stage.
    pub fn new(cfg: Arc<RollupConfig>, prev: P) -> Self {
        Self { cfg, channels: HashMap::default(), channel_queue: VecDeque::new(), prev }
    }

    /// Returns the size of the channel bank by accumulating over all channels.
    pub fn size(&self) -> usize {
        self.channels.iter().fold(0, |acc, (_, c)| acc + c.size())
    }

    /// Prunes the Channel bank, until it is below the max channel bank size.
    /// Prunes from the high-priority channel since it failed to be read.
    pub fn prune(&mut self) -> PipelineResult<()> {
        let mut total_size = self.size();
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        let max_channel_bank_size = if self.cfg.is_fjord_active(origin.timestamp) {
            FJORD_MAX_CHANNEL_BANK_SIZE
        } else {
            MAX_CHANNEL_BANK_SIZE
        };
        while total_size > max_channel_bank_size {
            let id =
                self.channel_queue.pop_front().ok_or(PipelineError::ChannelProviderEmpty.crit())?;
            let channel = self.channels.remove(&id).ok_or(PipelineError::ChannelNotFound.crit())?;
            total_size -= channel.size();
        }
        Ok(())
    }

    /// Adds new L1 data to the channel bank. Should only be called after all data has been read.
    pub fn ingest_frame(&mut self, frame: Frame) -> PipelineResult<()> {
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;

        // Get the channel for the frame, or create a new one if it doesn't exist.
        let current_channel = match self.channels.get_mut(&frame.id) {
            Some(c) => c,
            None => {
                let channel = Channel::new(frame.id, origin);
                self.channel_queue.push_back(frame.id);
                self.channels.insert(frame.id, channel);
                self.channels.get_mut(&frame.id).expect("Channel must be in queue")
            }
        };

        // Check if the channel is not timed out. If it has, ignore the frame.
        if current_channel.open_block_number() + self.cfg.channel_timeout(origin.timestamp) <
            origin.number
        {
            warn!(
                target: "channel_bank",
                "Channel (ID: {}) timed out", hex::encode(frame.id)
            );
            return Ok(());
        }

        // Ingest the frame. If it fails, ignore the frame.
        let frame_id = frame.id;
        if current_channel.add_frame(frame, origin).is_err() {
            warn!(target: "channel_bank", "Failed to add frame to channel: {:?}", frame_id);
            return Ok(());
        }

        self.prune()
    }

    /// Read the raw data of the first channel, if it's timed-out or closed.
    ///
    /// Returns an error if there is nothing new to read.
    pub fn read(&mut self) -> PipelineResult<Option<Bytes>> {
        // Bail if there are no channels to read from.
        if self.channel_queue.is_empty() {
            trace!(target: "channel_bank", "No channels to read from");
            return Err(PipelineError::Eof.temp());
        }

        // Return an `Ok(None)` if the first channel is timed out. There may be more timed
        // out channels at the head of the queue and we want to remove them all.
        let first = self.channel_queue[0];
        let channel =
            self.channels.get(&first).ok_or(PipelineError::ChannelProviderEmpty.crit())?;
        let origin = self.origin().ok_or(PipelineError::ChannelProviderEmpty.crit())?;
        if channel.open_block_number() + self.cfg.channel_timeout(origin.timestamp) < origin.number
        {
            warn!(
                target: "channel_bank",
                "Channel (ID: {}) timed out", hex::encode(first)
            );
            self.channels.remove(&first);
            self.channel_queue.pop_front();
            return Ok(None);
        }

        // At this point we have removed all timed out channels from the front of the
        // `channel_queue`. Pre-Canyon we simply check the first index.
        // Post-Canyon we read the entire channelQueue for the first ready channel.
        // If no channel is available, we return `PipelineError::Eof`.
        // Canyon is activated when the first L1 block whose time >= CanyonTime, not on the L2
        // timestamp.
        if !self.cfg.is_canyon_active(origin.timestamp) {
            return self.try_read_channel_at_index(0).map(Some);
        }

        let channel_data =
            (0..self.channel_queue.len()).find_map(|i| self.try_read_channel_at_index(i).ok());
        channel_data.map_or_else(|| Err(PipelineError::Eof.temp()), |data| Ok(Some(data)))
    }

    /// Attempts to read the channel at the specified index. If the channel is not ready or timed
    /// out, it will return an error.
    /// If the channel read was successful, it will remove the channel from the channel queue.
    fn try_read_channel_at_index(&mut self, index: usize) -> PipelineResult<Bytes> {
        let channel_id = self.channel_queue[index];
        let channel =
            self.channels.get(&channel_id).ok_or(PipelineError::ChannelProviderEmpty.crit())?;
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;

        let timed_out = channel.open_block_number() + self.cfg.channel_timeout(origin.timestamp) <
            origin.number;
        if timed_out || !channel.is_ready() {
            return Err(PipelineError::Eof.temp());
        }

        let frame_data = channel.frame_data();
        self.channels.remove(&channel_id);
        self.channel_queue.remove(index);

        frame_data.ok_or(PipelineError::ChannelProviderEmpty.crit())
    }
}

#[async_trait]
impl<P> OriginAdvancer for ChannelBank<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

#[async_trait]
impl<P> ChannelReaderProvider for ChannelBank<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn next_data(&mut self) -> PipelineResult<Option<Bytes>> {
        match self.read() {
            Err(e) => {
                if !matches!(e, PipelineErrorKind::Temporary(PipelineError::Eof)) {
                    return Err(PipelineError::ChannelProviderEmpty.crit());
                }
            }
            data => return data,
        };

        // Load the data into the channel bank
        let frame = match self.prev.next_frame().await {
            Ok(f) => f,
            Err(e) => {
                return Err(e);
            }
        };
        let res = self.ingest_frame(frame);
        res?;
        Err(PipelineError::NotEnoughData.temp())
    }
}

impl<P> OriginProvider for ChannelBank<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P> SignalReceiver for ChannelBank<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        self.prev.signal(signal).await?;
        self.channels.clear();
        self.channel_queue = VecDeque::with_capacity(10);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        test_utils::{CollectingLayer, TestNextFrameProvider, TraceStorage},
        types::ResetSignal,
    };
    use alloc::{vec, vec::Vec};
    use kona_genesis::HardForkConfig;
    use tracing::Level;
    use tracing_subscriber::layer::SubscriberExt;

    #[test]
    fn test_try_read_channel_at_index_missing_channel() {
        let mock = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        channel_bank.channel_queue.push_back([0xFF; 16]);
        let err = channel_bank.try_read_channel_at_index(0).unwrap_err();
        assert_eq!(err, PipelineError::ChannelProviderEmpty.crit());
    }

    #[test]
    fn test_try_read_channel_at_index_missing_origin() {
        let mut mock = TestNextFrameProvider::new(vec![]);
        mock.block_info = None;
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        channel_bank.channel_queue.push_back([0xFF; 16]);
        channel_bank.channels.insert([0xFF; 16], Channel::new([0xFF; 16], BlockInfo::default()));
        let err = channel_bank.try_read_channel_at_index(0).unwrap_err();
        assert_eq!(err, PipelineError::MissingOrigin.crit());
    }

    #[test]
    fn test_try_read_channel_at_index_timed_out() {
        let mut mock = TestNextFrameProvider::new(vec![]);
        mock.block_info = Some(BlockInfo { number: 10, ..Default::default() });
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        channel_bank.channel_queue.push_back([0xFF; 16]);
        channel_bank.channels.insert([0xFF; 16], Channel::new([0xFF; 16], BlockInfo::default()));
        let err = channel_bank.try_read_channel_at_index(0).unwrap_err();
        assert_eq!(err, PipelineError::Eof.temp());
    }

    #[test]
    fn test_try_read_channel_at_index() {
        let mock = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        let id: ChannelId = [0xFF; 16];
        channel_bank.channel_queue.push_back(id);
        let mut channel = Channel::new(id, BlockInfo::default());
        channel
            .add_frame(
                Frame { id, number: 0, data: b"seven__".to_vec(), is_last: false },
                BlockInfo::default(),
            )
            .unwrap();
        channel
            .add_frame(
                Frame { id, number: 1, data: b"seven__".to_vec(), is_last: false },
                BlockInfo::default(),
            )
            .unwrap();
        channel
            .add_frame(
                Frame { id, number: 2, data: b"seven__".to_vec(), is_last: true },
                BlockInfo::default(),
            )
            .unwrap();
        assert!(channel.is_ready());
        channel_bank.channels.insert([0xFF; 16], channel);
        let frame_data = channel_bank.try_read_channel_at_index(0).unwrap();
        assert_eq!(
            frame_data,
            alloy_primitives::bytes!("736576656e5f5f736576656e5f5f736576656e5f5f")
        );
    }

    #[test]
    fn test_read_channel_canyon_not_active() {
        let mock = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        let id: ChannelId = [0xFF; 16];
        channel_bank.channel_queue.push_back(id);
        let mut channel = Channel::new(id, BlockInfo::default());
        channel
            .add_frame(
                Frame { id, number: 0, data: b"seven__".to_vec(), is_last: false },
                BlockInfo::default(),
            )
            .unwrap();
        channel
            .add_frame(
                Frame { id, number: 1, data: b"seven__".to_vec(), is_last: false },
                BlockInfo::default(),
            )
            .unwrap();
        channel
            .add_frame(
                Frame { id, number: 2, data: b"seven__".to_vec(), is_last: true },
                BlockInfo::default(),
            )
            .unwrap();
        assert!(channel.is_ready());
        channel_bank.channels.insert([0xFF; 16], channel);
        let frame_data = channel_bank.read().unwrap();
        assert_eq!(
            frame_data,
            Some(alloy_primitives::bytes!("736576656e5f5f736576656e5f5f736576656e5f5f"))
        );
    }

    #[test]
    fn test_read_channel_active() {
        let mock = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { canyon_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let mut channel_bank = ChannelBank::new(cfg, mock);
        let id: ChannelId = [0xFF; 16];
        channel_bank.channel_queue.push_back(id);
        let mut channel = Channel::new(id, BlockInfo::default());
        channel
            .add_frame(
                Frame { id, number: 0, data: b"seven__".to_vec(), is_last: false },
                BlockInfo::default(),
            )
            .unwrap();
        channel
            .add_frame(
                Frame { id, number: 1, data: b"seven__".to_vec(), is_last: false },
                BlockInfo::default(),
            )
            .unwrap();
        channel
            .add_frame(
                Frame { id, number: 2, data: b"seven__".to_vec(), is_last: true },
                BlockInfo::default(),
            )
            .unwrap();
        assert!(channel.is_ready());
        channel_bank.channels.insert([0xFF; 16], channel);
        let frame_data = channel_bank.read().unwrap();
        assert_eq!(
            frame_data,
            Some(alloy_primitives::bytes!("736576656e5f5f736576656e5f5f736576656e5f5f"))
        );
    }

    #[test]
    fn test_ingest_empty_origin() {
        let mut mock = TestNextFrameProvider::new(vec![]);
        mock.block_info = None;
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        let frame = Frame::default();
        let err = channel_bank.ingest_frame(frame).unwrap_err();
        assert_eq!(err, PipelineError::MissingOrigin.crit());
    }

    #[tokio::test]
    async fn test_reset() {
        let mock = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        channel_bank.channels.insert([0xFF; 16], Channel::default());
        channel_bank.channel_queue.push_back([0xFF; 16]);
        assert!(!channel_bank.prev.reset);
        channel_bank.signal(ResetSignal::default().signal()).await.unwrap();
        assert_eq!(channel_bank.channels.len(), 0);
        assert_eq!(channel_bank.channel_queue.len(), 0);
        assert!(channel_bank.prev.reset);
    }

    #[test]
    fn test_ingest_invalid_frame() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let mock = TestNextFrameProvider::new(vec![]);
        let mut channel_bank = ChannelBank::new(Arc::new(RollupConfig::default()), mock);
        let frame = Frame { id: [0xFF; 16], ..Default::default() };
        assert_eq!(channel_bank.size(), 0);
        assert!(channel_bank.channels.is_empty());
        assert_eq!(trace_store.lock().iter().filter(|(l, _)| matches!(l, &Level::WARN)).count(), 0);
        assert_eq!(channel_bank.ingest_frame(frame.clone()), Ok(()));
        assert_eq!(channel_bank.size(), kona_protocol::FRAME_OVERHEAD);
        assert_eq!(channel_bank.channels.len(), 1);
        // This should fail since the frame is already ingested.
        assert_eq!(channel_bank.ingest_frame(frame), Ok(()));
        assert_eq!(channel_bank.size(), kona_protocol::FRAME_OVERHEAD);
        assert_eq!(channel_bank.channels.len(), 1);
        assert_eq!(trace_store.lock().iter().filter(|(l, _)| matches!(l, &Level::WARN)).count(), 1);
    }

    #[test]
    fn test_ingest_and_prune_channel_bank() {
        let mut frames = crate::frames!(0xFF, 0, vec![0xDD; 50], 100000);
        let mock = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        // Ingest frames until the channel bank is full and it stops increasing in size
        let mut current_size = 0;
        let next_frame = frames.pop().unwrap();
        channel_bank.ingest_frame(next_frame).unwrap();
        while channel_bank.size() > current_size {
            current_size = channel_bank.size();
            let next_frame = frames.pop().unwrap();
            channel_bank.ingest_frame(next_frame).unwrap();
            assert!(channel_bank.size() <= MAX_CHANNEL_BANK_SIZE);
        }
        // There should be a bunch of frames leftover
        assert!(!frames.is_empty());
        // If we ingest one more frame, the channel bank should prune
        // and the size should be the same
        let next_frame = frames.pop().unwrap();
        channel_bank.ingest_frame(next_frame).unwrap();
        assert_eq!(channel_bank.size(), current_size);
    }

    #[test]
    fn test_ingest_and_prune_channel_bank_fjord() {
        let mut frames = crate::frames!(0xFF, 0, vec![0xDD; 50], 100000);
        let mock = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { fjord_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let mut channel_bank = ChannelBank::new(cfg, mock);
        // Ingest frames until the channel bank is full and it stops increasing in size
        let mut current_size = 0;
        let next_frame = frames.pop().unwrap();
        channel_bank.ingest_frame(next_frame).unwrap();
        while channel_bank.size() > current_size {
            current_size = channel_bank.size();
            let next_frame = frames.pop().unwrap();
            channel_bank.ingest_frame(next_frame).unwrap();
            assert!(channel_bank.size() <= FJORD_MAX_CHANNEL_BANK_SIZE);
        }
        // There should be a bunch of frames leftover
        assert!(!frames.is_empty());
        // If we ingest one more frame, the channel bank should prune
        // and the size should be the same
        let next_frame = frames.pop().unwrap();
        channel_bank.ingest_frame(next_frame).unwrap();
        assert_eq!(channel_bank.size(), current_size);
    }

    #[tokio::test]
    async fn test_read_empty_channel_bank() {
        let frames = [crate::frame!(0xFF, 0, vec![0xDD; 50], true)];
        let mock = TestNextFrameProvider::new(vec![Ok(frames[0].clone())]);
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_bank = ChannelBank::new(cfg, mock);
        let err = channel_bank.read().unwrap_err();
        assert_eq!(err, PipelineError::Eof.temp());
        let err = channel_bank.next_data().await.unwrap_err();
        assert_eq!(err, PipelineError::NotEnoughData.temp());
    }

    #[tokio::test]
    async fn test_channel_timeout() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let configs: [RollupConfig; 2] = [
            kona_registry::ROLLUP_CONFIGS.get(&10).cloned().unwrap(),
            kona_registry::ROLLUP_CONFIGS.get(&8453).cloned().unwrap(),
        ];

        for cfg in configs {
            let frames = [
                crate::frame!(0xFF, 0, vec![0xDD; 50], false),
                crate::frame!(0xFF, 1, vec![0xDD; 50], true),
            ];
            let mock = TestNextFrameProvider::new(frames.into_iter().map(Ok).collect::<Vec<_>>());
            let cfg = Arc::new(cfg);
            let mut channel_bank = ChannelBank::new(cfg.clone(), mock);

            // Ingest first frame
            let err = channel_bank.next_data().await.unwrap_err();
            assert_eq!(err, PipelineError::NotEnoughData.temp());

            for _ in 0..cfg.channel_timeout + 1 {
                channel_bank.advance_origin().await.unwrap();
            }

            // There should be an in-progress channel.
            assert_eq!(channel_bank.channels.len(), 1);
            assert_eq!(channel_bank.channel_queue.len(), 1);

            // Should be `Ok(())`, channel timed out.
            channel_bank.next_data().await.unwrap();

            // The channel should have been pruned.
            assert_eq!(channel_bank.channels.len(), 0);
            assert_eq!(channel_bank.channel_queue.len(), 0);

            // Ensure the channel was successfully timed out.
            let (_, warning_trace) = trace_store
                .lock()
                .iter()
                .find(|(l, _)| matches!(l, &Level::WARN))
                .cloned()
                .unwrap();
            assert!(warning_trace.contains("timed out"));
        }
    }
}
