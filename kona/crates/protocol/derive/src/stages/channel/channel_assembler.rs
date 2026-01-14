//! This module contains the [ChannelAssembler] stage.

use super::{ChannelReaderProvider, NextFrameProvider};
use crate::{
    errors::PipelineError,
    traits::{OriginAdvancer, OriginProvider, SignalReceiver},
    types::{PipelineResult, Signal},
};
use alloc::{boxed::Box, sync::Arc};
use alloy_primitives::{Bytes, hex};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::{
    MAX_RLP_BYTES_PER_CHANNEL_BEDROCK, MAX_RLP_BYTES_PER_CHANNEL_FJORD, RollupConfig,
};
use kona_protocol::{BlockInfo, Channel};

/// The [`ChannelAssembler`] stage is responsible for assembling the [`Frame`]s from the
/// [`FrameQueue`] stage into a raw compressed [`Channel`].
///
/// [`Frame`]: kona_protocol::Frame
/// [`FrameQueue`]: crate::stages::FrameQueue
/// [`Channel`]: kona_protocol::Channel
#[derive(Debug)]
pub struct ChannelAssembler<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
    /// The previous stage of the derivation pipeline.
    pub prev: P,
    /// The current [`Channel`] being assembled.
    pub channel: Option<Channel>,
}

impl<P> ChannelAssembler<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// Creates a new [`ChannelAssembler`] stage with the given configuration and previous stage.
    pub const fn new(cfg: Arc<RollupConfig>, prev: P) -> Self {
        Self { cfg, prev, channel: None }
    }

    /// Returns whether or not the channel currently being assembled has timed out.
    pub fn is_timed_out(&self) -> PipelineResult<bool> {
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        let is_timed_out = self
            .channel
            .as_ref()
            .map(|c| {
                c.open_block_number() + self.cfg.channel_timeout(origin.timestamp) < origin.number
            })
            .unwrap_or_default();

        Ok(is_timed_out)
    }
}

#[async_trait]
impl<P> ChannelReaderProvider for ChannelAssembler<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn next_data(&mut self) -> PipelineResult<Option<Bytes>> {
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;

        // Time out the channel if it has timed out.
        if let Some(channel) = self.channel.as_ref() {
            if self.is_timed_out()? {
                warn!(
                    target: "channel_assembler",
                    "Channel (ID: {}) timed out at L1 origin #{}, open block #{}. Discarding channel.",
                    hex::encode(channel.id()),
                    origin.number,
                    channel.open_block_number()
                );
                self.channel = None;
            }
        }

        // Grab the next frame from the previous stage.
        let next_frame = self.prev.next_frame().await?;

        // Start a new channel if the frame number is 0.
        if next_frame.number == 0 {
            info!(
                target: "channel_assembler",
                "Starting new channel (ID: {}) at L1 origin #{}",
                hex::encode(next_frame.id),
                origin.number
            );
            self.channel = Some(Channel::new(next_frame.id, origin));
        }

        let count = if self.channel.is_some() { 1 } else { 0 };
        kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_CHANNEL_BUFFER, count);

        if let Some(channel) = self.channel.as_mut() {
            // Track the number of blocks until the channel times out.
            let timeout = channel.open_block_number() + self.cfg.channel_timeout(origin.timestamp);
            let margin = timeout.saturating_sub(origin.number) as f64;
            kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_CHANNEL_TIMEOUT, margin);

            // Add the frame to the channel. If this fails, return NotEnoughData and discard the
            // frame.
            debug!(
                target: "channel_assembler",
                "Adding frame #{} to channel (ID: {}) at L1 origin #{}",
                next_frame.number,
                hex::encode(channel.id()),
                origin.number
            );
            if channel.add_frame(next_frame, origin).is_err() {
                error!(
                    target: "channel_assembler",
                    "Failed to add frame to channel (ID: {}) at L1 origin #{}",
                    hex::encode(channel.id()),
                    origin.number
                );
                return Err(PipelineError::NotEnoughData.temp());
            }

            let size = channel.size() as f64;
            kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_CHANNEL_MEM, size);

            let max_rlp_bytes_per_channel = if self.cfg.is_fjord_active(origin.timestamp) {
                MAX_RLP_BYTES_PER_CHANNEL_FJORD
            } else {
                MAX_RLP_BYTES_PER_CHANNEL_BEDROCK
            };
            kona_macros::set!(
                gauge,
                crate::metrics::Metrics::PIPELINE_MAX_RLP_BYTES,
                max_rlp_bytes_per_channel as f64
            );
            if channel.size() > max_rlp_bytes_per_channel as usize {
                warn!(
                    target: "channel_assembler",
                    "Compressed channel size exceeded max RLP bytes per channel, dropping channel (ID: {}) with {} bytes",
                    hex::encode(channel.id()),
                    channel.size()
                );
                self.channel = None;
                return Err(PipelineError::NotEnoughData.temp());
            }

            // If the channel is ready, forward the channel to the next stage.
            if channel.is_ready() {
                let channel_bytes =
                    channel.frame_data().ok_or(PipelineError::ChannelNotFound.crit())?;

                info!(
                    target: "channel_assembler",
                    "Channel (ID: {}) ready for decompression.",
                    hex::encode(channel.id()),
                );

                // Reset the channel and return the compressed bytes.
                self.channel = None;
                return Ok(Some(channel_bytes));
            }
        }

        kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_CHANNEL_MEM, 0);

        Err(PipelineError::NotEnoughData.temp())
    }
}

#[async_trait]
impl<P> OriginAdvancer for ChannelAssembler<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

impl<P> OriginProvider for ChannelAssembler<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P> SignalReceiver for ChannelAssembler<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        self.prev.signal(signal).await?;
        self.channel = None;
        Ok(())
    }
}

#[cfg(test)]
mod test {
    use super::ChannelAssembler;
    use crate::{
        ChannelReaderProvider, PipelineError,
        test_utils::{CollectingLayer, TestNextFrameProvider, TraceStorage},
    };
    use alloc::{sync::Arc, vec};
    use kona_genesis::{
        HardForkConfig, MAX_RLP_BYTES_PER_CHANNEL_BEDROCK, MAX_RLP_BYTES_PER_CHANNEL_FJORD,
        RollupConfig,
    };
    use kona_protocol::BlockInfo;
    use tracing::Level;
    use tracing_subscriber::layer::SubscriberExt;

    #[tokio::test]
    async fn test_assembler_channel_timeout() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let mock = TestNextFrameProvider::new(frames.into_iter().rev().map(Ok).collect());
        let cfg = Arc::new(RollupConfig::default());
        let mut assembler = ChannelAssembler::new(cfg, mock);

        // Set the origin to default block info @ block # 0.
        assembler.prev.block_info = Some(BlockInfo::default());

        // Read in the first frame. Since the frame isn't the last, the assembler
        // should return None.
        assert!(assembler.channel.is_none());
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_some());

        // Push the origin forward past channel timeout.
        assembler.prev.block_info =
            Some(BlockInfo { number: assembler.cfg.channel_timeout(0) + 1, ..Default::default() });

        // Assert that the assembler has timed out the channel.
        assert!(assembler.is_timed_out().unwrap());
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_none());

        // Assert that the info log was emitted.
        let info_logs = trace_store.get_by_level(Level::INFO);
        assert_eq!(info_logs.len(), 1);
        let info_str = "Starting new channel";
        assert!(info_logs[0].contains(info_str));

        // Assert that the warning log was emitted.
        let warning_logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(warning_logs.len(), 1);
        let warn_str = "timed out at L1 origin";
        assert!(warning_logs[0].contains(warn_str));
    }

    #[tokio::test]
    async fn test_assembler_non_starting_frame() {
        let frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let mock = TestNextFrameProvider::new(frames.into_iter().map(Ok).collect());
        let cfg = Arc::new(RollupConfig::default());
        let mut assembler = ChannelAssembler::new(cfg, mock);

        // Send in the second frame first. This should result in no channel being created,
        // and the frame being discarded.
        assert!(assembler.channel.is_none());
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_none());
    }

    #[tokio::test]
    async fn test_assembler_already_built() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let mock = TestNextFrameProvider::new(frames.clone().into_iter().rev().map(Ok).collect());
        let cfg = Arc::new(RollupConfig::default());
        let mut assembler = ChannelAssembler::new(cfg, mock);

        // Send in the first frame. This should result in a channel being created.
        assert!(assembler.channel.is_none());
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_some());

        // Send in a malformed second frame. This should result in an error in `add_frame`.
        assembler.prev.data.push(Ok(frames[1].clone()).map(|mut f| {
            f.id = Default::default();
            f
        }));
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_some());

        // Send in the second frame again. This should return the channel bytes.
        assert!(assembler.next_data().await.unwrap().is_some());
        assert!(assembler.channel.is_none());

        // Assert that the error log was emitted.
        let error_logs = trace_store.get_by_level(Level::ERROR);
        assert_eq!(error_logs.len(), 1);
        let error_str = "Failed to add frame to channel";
        assert!(error_logs[0].contains(error_str));
    }

    #[tokio::test]
    async fn test_assembler_size_limit_exceeded_bedrock() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let mut frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        frames[1].data = vec![0; MAX_RLP_BYTES_PER_CHANNEL_BEDROCK as usize];
        let mock = TestNextFrameProvider::new(frames.into_iter().rev().map(Ok).collect());
        let cfg = Arc::new(RollupConfig::default());

        let mut assembler = ChannelAssembler::new(cfg, mock);

        // Send in the first frame. This should result in a channel being created.
        assert!(assembler.channel.is_none());
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_some());

        // Send in the second frame. This should result in the channel being dropped due to the size
        // limit being reached.
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_none());

        let trace_store_lock = trace_store.lock();
        assert_eq!(trace_store_lock.iter().filter(|(l, _)| matches!(l, &Level::WARN)).count(), 1);

        let (_, message) =
            trace_store_lock.iter().find(|(l, _)| matches!(l, &Level::WARN)).unwrap();
        assert!(message.contains("Compressed channel size exceeded max RLP bytes per channel"));
    }

    #[tokio::test]
    async fn test_assembler_size_limit_exceeded_fjord() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let mut frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        frames[1].data = vec![0; MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize];
        let mock = TestNextFrameProvider::new(frames.into_iter().rev().map(Ok).collect());
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { fjord_time: Some(0), ..Default::default() },
            ..Default::default()
        });

        let mut assembler = ChannelAssembler::new(cfg, mock);

        // Send in the first frame. This should result in a channel being created.
        assert!(assembler.channel.is_none());
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_some());

        // Send in the second frame. This should result in the channel being dropped due to the size
        // limit being reached.
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_none());

        let trace_store_lock = trace_store.lock();
        assert_eq!(trace_store_lock.iter().filter(|(l, _)| matches!(l, &Level::WARN)).count(), 1);

        let (_, message) =
            trace_store_lock.iter().find(|(l, _)| matches!(l, &Level::WARN)).unwrap();
        assert!(message.contains("Compressed channel size exceeded max RLP bytes per channel"));
    }
}
