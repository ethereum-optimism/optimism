//! Sync core of the post-Holocene channel assembler.
//!
//! Lifted from `stages/channel/channel_assembler.rs` so that both the async
//! stage (today) and `pure::Deriver` (phase 3) drive the same state machine.
//!
//! This module is sysconfig-aware only via the `RollupConfig` it carries
//! (Fjord-vs-Bedrock max-RLP gate, channel timeout window). It does no IO,
//! emits no traces, and never panics. All outcomes — including drops — are
//! returned through [`FrameOutcome`] / [`TimeoutOutcome`] so callers can map
//! them to whatever observability they need.

use alloy_primitives::Bytes;
use kona_genesis::{
    MAX_RLP_BYTES_PER_CHANNEL_BEDROCK, MAX_RLP_BYTES_PER_CHANNEL_FJORD, RollupConfig,
};
use kona_protocol::{BlockInfo, ChannelError, Frame, OrderedChannel};

/// Why a frame was dropped during channel assembly. Each variant maps to an
/// observability event in the caller (warning, error, or info log today;
/// trace entry tomorrow).
#[derive(Debug, PartialEq, Eq, Clone, Copy)]
pub(crate) enum FrameDropReason {
    /// `add_frame` rejected the frame (out-of-order, mismatched id, etc.).
    AddFrameFailed(ChannelError),
    /// The cumulative channel size exceeded the max RLP bytes for the active
    /// fork.
    ChannelSizeExceeded,
    /// A non-zero frame arrived with no active channel — Holocene's
    /// single-channel rule.
    NoActiveChannel,
}

/// Outcome of feeding one frame into [`process_frame`].
#[derive(Debug)]
pub(crate) enum FrameOutcome {
    /// A new channel was opened by this frame.
    OpenedChannel,
    /// The frame was added to an existing channel; channel not yet ready.
    Buffered,
    /// The frame closed the channel. Carries the assembled (still compressed)
    /// channel bytes.
    ChannelReady(Bytes),
    /// The frame was discarded; the assembler advances without producing
    /// output. The reason carries enough information for the caller to emit
    /// the appropriate log / trace entry.
    Dropped(FrameDropReason),
}

/// Whether the active channel has timed out at the given origin.
#[derive(Debug, PartialEq, Eq, Clone, Copy)]
pub(crate) enum TimeoutOutcome {
    /// No channel is currently active.
    NoChannel,
    /// The channel is still within its timeout window.
    Active,
    /// The channel has timed out at this origin.
    TimedOut,
}

/// Returns whether the given channel — if any — has timed out at the given
/// L1 origin under the supplied rollup config.
pub(crate) fn check_timeout(
    cfg: &RollupConfig,
    channel: Option<&OrderedChannel>,
    origin: BlockInfo,
) -> TimeoutOutcome {
    let Some(channel) = channel else { return TimeoutOutcome::NoChannel };
    let timeout = channel.open_block_number() + cfg.channel_timeout(origin.timestamp);
    if timeout < origin.number { TimeoutOutcome::TimedOut } else { TimeoutOutcome::Active }
}

/// Returns the max RLP bytes per channel for the active fork at this origin
/// timestamp. Exposed so callers can surface it to metrics.
pub(crate) fn max_rlp_bytes_per_channel(cfg: &RollupConfig, origin_timestamp: u64) -> u64 {
    if cfg.is_fjord_active(origin_timestamp) {
        MAX_RLP_BYTES_PER_CHANNEL_FJORD
    } else {
        MAX_RLP_BYTES_PER_CHANNEL_BEDROCK
    }
}

/// Feeds one frame into the post-Holocene channel state machine.
///
/// The caller is expected to have already cleared `*channel` on a prior
/// [`TimeoutOutcome::TimedOut`] result.
///
/// Returns the structural outcome — opened / buffered / ready / dropped.
/// Tracing / metrics happen in the caller.
pub(crate) fn process_frame(
    cfg: &RollupConfig,
    channel: &mut Option<OrderedChannel>,
    next_frame: Frame,
    origin: BlockInfo,
) -> FrameOutcome {
    // Holocene's single-channel rule: frame number 0 starts a fresh channel.
    let opened = if next_frame.number == 0 {
        *channel = Some(OrderedChannel::new(next_frame.id, origin));
        true
    } else {
        false
    };

    let Some(active) = channel.as_mut() else {
        return FrameOutcome::Dropped(FrameDropReason::NoActiveChannel);
    };

    if let Err(err) = active.add_frame(next_frame, origin) {
        return FrameOutcome::Dropped(FrameDropReason::AddFrameFailed(err));
    }

    let limit = max_rlp_bytes_per_channel(cfg, origin.timestamp) as usize;
    if active.size() > limit {
        *channel = None;
        return FrameOutcome::Dropped(FrameDropReason::ChannelSizeExceeded);
    }

    if active.is_ready() {
        // `data()` only errors when the channel is unready, which we just
        // checked. Surface the error as a drop rather than panicking so the
        // caller can re-emit it as a critical PipelineError.
        return match active.data() {
            Ok(bytes) => {
                *channel = None;
                FrameOutcome::ChannelReady(bytes)
            }
            Err(_) => {
                *channel = None;
                FrameOutcome::Dropped(FrameDropReason::AddFrameFailed(
                    ChannelError::FrameIdMismatch,
                ))
            }
        };
    }

    if opened { FrameOutcome::OpenedChannel } else { FrameOutcome::Buffered }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::frame;
    use alloc::{sync::Arc, vec};
    use kona_genesis::{HardForkConfig, RollupConfig};
    use kona_protocol::BlockInfo;

    fn cfg_bedrock() -> Arc<RollupConfig> {
        Arc::new(RollupConfig::default())
    }

    fn cfg_fjord() -> Arc<RollupConfig> {
        Arc::new(RollupConfig {
            hardforks: HardForkConfig { fjord_time: Some(0), ..Default::default() },
            ..Default::default()
        })
    }

    #[test]
    fn happy_path_opens_and_closes_channel() {
        let cfg = cfg_bedrock();
        let mut channel = None;
        let origin = BlockInfo::default();
        let f0 = frame!(0xAB, 0, vec![0xDD; 50], false);
        let f1 = frame!(0xAB, 1, vec![0xDD; 50], true);

        assert!(matches!(
            process_frame(&cfg, &mut channel, f0, origin),
            FrameOutcome::OpenedChannel
        ));
        assert!(channel.is_some());
        match process_frame(&cfg, &mut channel, f1, origin) {
            FrameOutcome::ChannelReady(bytes) => assert!(!bytes.is_empty()),
            other => panic!("expected ChannelReady, got {other:?}"),
        }
        assert!(channel.is_none());
    }

    #[test]
    fn timeout_detected_after_window() {
        let cfg = cfg_bedrock();
        let mut channel = None;
        let origin = BlockInfo::default();
        let f0 = frame!(0xAB, 0, vec![0xDD; 50], false);
        let _ = process_frame(&cfg, &mut channel, f0, origin);

        assert_eq!(check_timeout(&cfg, channel.as_ref(), origin), TimeoutOutcome::Active);
        let future = BlockInfo { number: cfg.channel_timeout(0) + 1, ..Default::default() };
        assert_eq!(check_timeout(&cfg, channel.as_ref(), future), TimeoutOutcome::TimedOut);
    }

    #[test]
    fn non_starting_frame_with_no_channel_drops() {
        let cfg = cfg_bedrock();
        let mut channel = None;
        let origin = BlockInfo::default();
        let f1 = frame!(0xAB, 1, vec![0xDD; 50], true);
        match process_frame(&cfg, &mut channel, f1, origin) {
            FrameOutcome::Dropped(FrameDropReason::NoActiveChannel) => {}
            other => panic!("expected Dropped(NoActiveChannel), got {other:?}"),
        }
        assert!(channel.is_none());
    }

    #[test]
    fn size_limit_exceeded_drops_channel_bedrock() {
        let cfg = cfg_bedrock();
        let mut channel = None;
        let origin = BlockInfo::default();
        let f0 = frame!(0xAB, 0, vec![0xDD; 50], false);
        let f1 = frame!(0xAB, 1, vec![0; MAX_RLP_BYTES_PER_CHANNEL_BEDROCK as usize], true);
        let _ = process_frame(&cfg, &mut channel, f0, origin);
        let out = process_frame(&cfg, &mut channel, f1, origin);
        assert!(matches!(out, FrameOutcome::Dropped(FrameDropReason::ChannelSizeExceeded)));
        assert!(channel.is_none());
    }

    #[test]
    fn size_limit_exceeded_drops_channel_fjord() {
        let cfg = cfg_fjord();
        let mut channel = None;
        let origin = BlockInfo::default();
        let f0 = frame!(0xAB, 0, vec![0xDD; 50], false);
        let f1 = frame!(0xAB, 1, vec![0; MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize], true);
        let _ = process_frame(&cfg, &mut channel, f0, origin);
        let out = process_frame(&cfg, &mut channel, f1, origin);
        assert!(matches!(out, FrameOutcome::Dropped(FrameDropReason::ChannelSizeExceeded)));
    }

    #[test]
    fn malformed_subsequent_frame_keeps_channel_alive() {
        // Mirrors stages::channel_assembler::test_assembler_already_built:
        // a malformed frame (different id) is rejected, but the channel
        // built so far stays.
        let cfg = cfg_bedrock();
        let mut channel = None;
        let origin = BlockInfo::default();
        let f0 = frame!(0xAB, 0, vec![0xDD; 50], false);
        let mut f_bad = frame!(0xAB, 1, vec![0xDD; 50], false);
        f_bad.id = Default::default();
        let _ = process_frame(&cfg, &mut channel, f0, origin);
        match process_frame(&cfg, &mut channel, f_bad, origin) {
            FrameOutcome::Dropped(FrameDropReason::AddFrameFailed(_)) => {}
            other => panic!("expected Dropped(AddFrameFailed), got {other:?}"),
        }
        assert!(channel.is_some());
    }
}
