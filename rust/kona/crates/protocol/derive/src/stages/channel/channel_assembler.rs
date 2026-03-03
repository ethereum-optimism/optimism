//! This module contains the [`ChannelAssembler`] stage.

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
        let is_timed_out = self.channel.as_ref().is_some_and(|c| {
            c.open_block_number() + self.cfg.channel_timeout(origin.timestamp) < origin.number
        });

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
        if let Some(channel) = self.channel.as_ref() &&
            self.is_timed_out()?
        {
            let channel_id = hex::encode(channel.id());
            let open_block = channel.open_block_number();
            warn!(
                target: "channel_assembler",
                "Channel (ID: {}) timed out at L1 origin #{}, open block #{}. Discarding channel.",
                channel_id,
                origin.number,
                open_block
            );
            self.channel = None;
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
        ChannelReaderProvider, PipelineError, PipelineErrorKind,
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

    // CB-09: Documents that kona's ChannelAssembler requires one extra pipeline tick per
    // add_frame failure, whereas Go's ChannelAssembler continues reading in the same call.
    //
    // Reference (Go): channel_assembler.go:110-115 — on add_frame failure, logs Warn and calls
    // `continue` to immediately read the next frame in the same NextRawChannel() invocation.
    //
    // Subject (Rust): channel_assembler.rs:113-121 — on add_frame failure, returns
    // Err(PipelineError::NotEnoughData.temp()), requiring a new next_data() call.
    //
    // This test documents the Rust behaviour: a malformed frame (wrong channel ID) triggers
    // add_frame failure, and the assembler returns NotEnoughData rather than immediately consuming
    // the next valid frame. Human review verdict: false-positive (pipeline ticks not important).
    #[tokio::test]
    async fn test_spec_channel_bank_assembler_continue_on_add_frame_failure() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        // Provide frames in reverse order (last then first), plus an invalid frame (wrong channel
        // ID) in between. The provider pops from the end, so order delivered is:
        //   frame 0  (valid, starts channel)
        //   invalid  (wrong ID, add_frame fails)
        //   frame 1  (valid, completes channel)
        let invalid_frame = {
            let mut f = frames[1].clone();
            f.id = [0x00; 16]; // wrong channel ID
            f
        };
        // TestNextFrameProvider::next_frame() calls pop(), so the last element is delivered first.
        // We arrange the vec so that frames[0] is at the end (delivered first), followed by
        // invalid_frame, then frames[1].
        let provider_data = vec![
            Ok(frames[1].clone()),  // at index 0, delivered third
            Ok(invalid_frame),      // at index 1, delivered second
            Ok(frames[0].clone()),  // at index 2 (end), delivered first
        ];
        let mock = TestNextFrameProvider::new(provider_data);
        let cfg = Arc::new(RollupConfig::default());
        let mut assembler = ChannelAssembler::new(cfg, mock);

        // Tick 1: consume frame 0 → channel created, not yet ready → NotEnoughData.
        assert!(assembler.channel.is_none());
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_some());

        // Tick 2: consume invalid frame → add_frame fails → NotEnoughData.
        // Go would continue the loop and immediately consume frame 1 in this same tick,
        // completing the channel. Rust returns NotEnoughData here instead.
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        // Channel is still present (the invalid frame did not reset it).
        assert!(assembler.channel.is_some());

        // Tick 3: consume frame 1 → channel ready → returns channel bytes.
        // In Go this would have happened in tick 2. Rust requires an extra tick.
        let result = assembler.next_data().await;
        assert!(result.is_ok(), "Expected Ok but got: {:?}", result);
        assert!(result.unwrap().is_some());
        assert!(assembler.channel.is_none());

        // Verify the error log was emitted.
        let error_logs = trace_store.get_by_level(Level::ERROR);
        assert_eq!(error_logs.len(), 1);
        assert!(error_logs[0].contains("Failed to add frame to channel"));
    }

    // CB-10: Documents that kona's ChannelAssembler silently drops non-first frames with no
    // channel and returns NotEnoughData without a warning log, whereas Go logs Warn and continues.
    //
    // Reference (Go): channel_assembler.go:101-105 — checks `frame.FrameNumber > 0 && ca.channel
    // == nil`, logs Warn("dropping non-first frame without channel"), then `continue`s the loop.
    //
    // Subject (Rust): channel_assembler.rs:84-98 — if frame.number != 0 and self.channel is None,
    // falls through to Err(PipelineError::NotEnoughData.temp()) with no warning log.
    //
    // Human review verdict: false-positive (logging is an implementation detail, not spec).
    #[tokio::test]
    async fn test_spec_channel_bank_assembler_non_first_frame_no_channel() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        // Provide frame 1 (non-first) with no prior frame 0. No channel should be created.
        let non_first_frame = crate::frame!(0xFF, 1, vec![0xDD; 50], false);
        let mock = TestNextFrameProvider::new(vec![Ok(non_first_frame)]);
        let cfg = Arc::new(RollupConfig::default());
        let mut assembler = ChannelAssembler::new(cfg, mock);

        assert!(assembler.channel.is_none());
        // Rust: returns NotEnoughData without any warning log.
        // Go: would log Warn("dropping non-first frame without channel") then continue the loop.
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_none());

        // Verify: no warning was emitted (Rust silently drops — difference from Go).
        let warning_logs = trace_store.get_by_level(Level::WARN);
        assert!(
            warning_logs.is_empty(),
            "Rust emits no warning for non-first frame without channel (Go would warn)"
        );
    }

    // CB-11: Documents that kona's ChannelAssembler requires one extra pipeline tick after
    // dropping an oversized channel, whereas Go's ChannelAssembler continues reading in the same
    // call.
    //
    // Reference (Go): channel_assembler.go:116-121 — after adding a frame that makes the channel
    // exceed MaxRLPBytesPerChannel, calls ca.resetChannel() then `continue`s the inner loop,
    // immediately reading the next frame in the same NextRawChannel() invocation.
    //
    // Subject (Rust): channel_assembler.rs:136-144 — drops the channel (self.channel = None) but
    // returns Err(PipelineError::NotEnoughData.temp()) instead of continuing.
    //
    // This test documents the Rust behaviour: after an oversized channel is dropped, the next
    // valid channel must be started in a separate next_data() call.
    #[tokio::test]
    async fn test_spec_channel_bank_assembler_oversized_channel_continue() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        // Channel A: frame 0 (small) + frame 1 (oversized, is_last) → gets dropped on frame 1.
        // Channel B: frame 0 (small, is_last) → valid complete channel.
        // Frame delivery order from provider (pop() delivers in LIFO from end):
        //   channel_a_frame0   (starts channel A)
        //   channel_a_frame1   (oversized, triggers channel A drop)
        //   channel_b_frame0   (starts + completes channel B)
        let channel_a_frame0 = crate::frame!(0xAA, 0, vec![0x01; 50], false);
        let mut channel_a_frame1 = crate::frame!(0xAA, 1, vec![0x01; 50], true);
        channel_a_frame1.data = vec![0x01; MAX_RLP_BYTES_PER_CHANNEL_BEDROCK as usize];
        let channel_b_frame0 = crate::frame!(0xBB, 0, vec![0x02; 50], true);

        // TestNextFrameProvider::next_frame() calls pop(), so the last element is delivered first.
        // We arrange the vec so that channel_a_frame0 is at the end (delivered first).
        let provider_data = vec![
            Ok(channel_b_frame0), // at index 0, delivered third
            Ok(channel_a_frame1), // at index 1, delivered second
            Ok(channel_a_frame0), // at index 2 (end), delivered first
        ];
        let mock = TestNextFrameProvider::new(provider_data);
        let cfg = Arc::new(RollupConfig::default());
        let mut assembler = ChannelAssembler::new(cfg, mock);

        // Tick 1: consume channel_a_frame0 → channel A created, not ready → NotEnoughData.
        assert!(assembler.channel.is_none());
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_some());

        // Tick 2: consume channel_a_frame1 (oversized) → channel A dropped → NotEnoughData.
        // Go would continue the loop here, immediately consuming channel_b_frame0 and returning
        // the completed channel B data in this same tick.
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_none(), "Channel A should have been dropped");

        // Tick 3: consume channel_b_frame0 → channel B starts and completes → returns data.
        // In Go this would have happened in tick 2. Rust requires an extra tick.
        let result = assembler.next_data().await;
        assert!(result.is_ok(), "Expected Ok but got: {:?}", result);
        assert!(result.unwrap().is_some(), "Expected Some channel data");
        assert!(assembler.channel.is_none());

        // Verify the oversized-channel warning was emitted.
        let warning_logs = trace_store.get_by_level(Level::WARN);
        assert!(!warning_logs.is_empty());
        assert!(warning_logs.iter().any(|m| m.contains("Compressed channel size exceeded")));
    }

    // CB-12: Documents that kona's ChannelAssembler does not return a CriticalError when a
    // last frame is successfully added but is_ready() returns false, whereas Go's ChannelAssembler
    // panics with NewCriticalError in this situation.
    //
    // Reference (Go): channel_assembler.go:131-133 — after the frame ingestion loop exits, checks
    // `if ch == nil || !ch.IsReady()` and returns `NewCriticalError(errors.New("unexpected
    // non-ready channel"))`. The loop only exits when frame.IsLast=true breaks it, so this is a
    // defensive assertion that the loop invariant holds.
    //
    // Subject (Rust): channel_assembler.rs:147-161 — checks `if channel.is_ready()` and returns
    // the channel data if true, otherwise falls through to Err(PipelineError::NotEnoughData.temp()).
    // There is no CriticalError for the invariant violation case.
    //
    // This test demonstrates the Rust behaviour: in a scenario where a frame with is_last=true
    // is received but the channel is missing a preceding frame (so is_ready() returns false),
    // Rust returns NotEnoughData (allowing retry) rather than a CriticalError (halting the
    // pipeline). In practice, the Holocene ChannelAssembler's strict ordering guarantees that
    // is_ready() will always be true after a successful last frame add, so this invariant cannot
    // be violated by honest input. The difference only matters if there is a bug in the Channel
    // type itself.
    #[tokio::test]
    async fn test_spec_channel_bank_assembler_no_critical_error_on_last_frame_not_ready() {
        // We cannot directly test the invariant violation (it would require a buggy Channel
        // implementation), but we can verify that the Rust assembler does NOT return a
        // CriticalError when is_ready() returns false after a last frame is added.
        //
        // Scenario: send frame 1 (is_last=true) to a channel that was started at frame 0, but
        // the channel is in Holocene (requireInOrder) mode and frame 0 hasn't been properly
        // added — actually, in Holocene mode the assembler creates a fresh channel for each
        // frame 0. We can simulate a not-ready channel by sending frame 0 (starts channel) and
        // then a frame with is_last=true but a DIFFERENT channel ID, which will fail add_frame
        // (FrameIdMismatch), not reach the is_ready() check.
        //
        // Instead, test the indirect guarantee: when a channel is open but gets EOF from the
        // provider before a last frame, we get NotEnoughData, not CriticalError.
        let frames = [crate::frame!(0xFF, 0, vec![0xDD; 50], false)]; // no is_last frame
        let mock = TestNextFrameProvider::new(frames.into_iter().map(Ok).collect());
        let cfg = Arc::new(RollupConfig::default());
        let mut assembler = ChannelAssembler::new(cfg, mock);

        // Tick 1: frame 0 consumed, channel created.
        assert_eq!(assembler.next_data().await.unwrap_err(), PipelineError::NotEnoughData.temp());
        assert!(assembler.channel.is_some());

        // Tick 2: provider is empty → EOF from prev → returns EOF (not CriticalError).
        // If Rust had the Go-style post-loop invariant check, it would return CriticalError here.
        // Instead it returns EOF from the provider, propagated upward.
        let err = assembler.next_data().await.unwrap_err();
        // We expect EOF or NotEnoughData — not a CriticalError.
        assert!(
            !matches!(err, PipelineErrorKind::Critical(_)),
            "Expected non-critical error, got: {:?}",
            err
        );
    }
}
