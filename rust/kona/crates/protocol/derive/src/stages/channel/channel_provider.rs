//! This module contains the [ChannelProvider] stage.

use super::{ChannelAssembler, ChannelBank, ChannelReaderProvider, NextFrameProvider};
use crate::{
    errors::PipelineError,
    traits::{OriginAdvancer, OriginProvider, SignalReceiver},
    types::{PipelineResult, Signal},
};
use alloc::{boxed::Box, sync::Arc};
use alloy_primitives::Bytes;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;

/// The [`ChannelProvider`] stage is a mux between the [`ChannelBank`] and [`ChannelAssembler`]
/// stages.
///
/// Rules:
/// When Holocene is not active, the [`ChannelBank`] is used.
/// When Holocene is active, the [`ChannelAssembler`] is used.
#[derive(Debug)]
pub struct ChannelProvider<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
    /// The previous stage of the derivation pipeline.
    ///
    /// If this is set to [`None`], the multiplexer has been activated and the active stage
    /// owns the previous stage.
    ///
    /// Must be [`None`] if `channel_bank` or `channel_assembler` is [`Some`].
    pub prev: Option<P>,
    /// The channel bank stage of the provider.
    ///
    /// Must be [`None`] if `prev` or `channel_assembler` is [`Some`].
    pub channel_bank: Option<ChannelBank<P>>,
    /// The channel assembler stage of the provider.
    ///
    /// Must be [`None`] if `prev` or `channel_bank` is [`Some`].
    pub channel_assembler: Option<ChannelAssembler<P>>,
}

impl<P> ChannelProvider<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// Creates a new [`ChannelProvider`] with the given configuration and previous stage.
    pub const fn new(cfg: Arc<RollupConfig>, prev: P) -> Self {
        Self { cfg, prev: Some(prev), channel_bank: None, channel_assembler: None }
    }

    /// Attempts to update the active stage of the mux.
    pub(crate) fn attempt_update(&mut self) -> PipelineResult<()> {
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        if let Some(prev) = self.prev.take() {
            // On the first call to `attempt_update`, we need to determine the active stage to
            // initialize the mux with.
            if self.cfg.is_holocene_active(origin.timestamp) {
                self.channel_assembler = Some(ChannelAssembler::new(self.cfg.clone(), prev));
            } else {
                self.channel_bank = Some(ChannelBank::new(self.cfg.clone(), prev));
            }
        } else if self.channel_bank.is_some() && self.cfg.is_holocene_active(origin.timestamp) {
            // If the channel bank is active and Holocene is also active, transition to the channel
            // assembler.
            let channel_bank = self.channel_bank.take().expect("Must have channel bank");
            self.channel_assembler =
                Some(ChannelAssembler::new(self.cfg.clone(), channel_bank.prev));
        } else if self.channel_assembler.is_some() && !self.cfg.is_holocene_active(origin.timestamp)
        {
            // If the channel assembler is active, and Holocene is not active, it indicates an L1
            // reorg around Holocene activation. Transition back to the channel bank
            // until Holocene re-activates.
            let channel_assembler =
                self.channel_assembler.take().expect("Must have channel assembler");
            self.channel_bank = Some(ChannelBank::new(self.cfg.clone(), channel_assembler.prev));
        }
        Ok(())
    }
}

#[async_trait]
impl<P> OriginAdvancer for ChannelProvider<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.attempt_update()?;

        if let Some(channel_assembler) = self.channel_assembler.as_mut() {
            channel_assembler.advance_origin().await
        } else if let Some(channel_bank) = self.channel_bank.as_mut() {
            channel_bank.advance_origin().await
        } else {
            Err(PipelineError::NotEnoughData.temp())
        }
    }
}

impl<P> OriginProvider for ChannelProvider<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.channel_assembler.as_ref().map_or_else(
            || {
                self.channel_bank.as_ref().map_or_else(
                    || self.prev.as_ref().and_then(|prev| prev.origin()),
                    |channel_bank| channel_bank.origin(),
                )
            },
            |channel_assembler| channel_assembler.origin(),
        )
    }
}

#[async_trait]
impl<P> SignalReceiver for ChannelProvider<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        self.attempt_update()?;

        if let Some(channel_assembler) = self.channel_assembler.as_mut() {
            channel_assembler.signal(signal).await
        } else if let Some(channel_bank) = self.channel_bank.as_mut() {
            channel_bank.signal(signal).await
        } else {
            Err(PipelineError::NotEnoughData.temp())
        }
    }
}

#[async_trait]
impl<P> ChannelReaderProvider for ChannelProvider<P>
where
    P: NextFrameProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn next_data(&mut self) -> PipelineResult<Option<Bytes>> {
        self.attempt_update()?;

        if let Some(channel_assembler) = self.channel_assembler.as_mut() {
            channel_assembler.next_data().await
        } else if let Some(channel_bank) = self.channel_bank.as_mut() {
            channel_bank.next_data().await
        } else {
            Err(PipelineError::NotEnoughData.temp())
        }
    }
}

#[cfg(test)]
mod test {
    use crate::{
        ChannelProvider, ChannelReaderProvider, OriginProvider, PipelineError, ResetSignal,
        SignalReceiver, test_utils::TestNextFrameProvider,
    };
    use alloc::{sync::Arc, vec};
    use kona_genesis::{HardForkConfig, RollupConfig};
    use kona_protocol::BlockInfo;

    #[test]
    fn test_channel_provider_assembler_active() {
        let provider = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let mut channel_provider = ChannelProvider::new(cfg, provider);

        assert!(channel_provider.attempt_update().is_ok());
        assert!(channel_provider.prev.is_none());
        assert!(channel_provider.channel_bank.is_none());
        assert!(channel_provider.channel_assembler.is_some());
    }

    #[test]
    fn test_channel_provider_bank_active() {
        let provider = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_provider = ChannelProvider::new(cfg, provider);

        assert!(channel_provider.attempt_update().is_ok());
        assert!(channel_provider.prev.is_none());
        assert!(channel_provider.channel_bank.is_some());
        assert!(channel_provider.channel_assembler.is_none());
    }

    #[test]
    fn test_channel_provider_retain_current_bank() {
        let provider = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_provider = ChannelProvider::new(cfg, provider);

        // Assert the multiplexer hasn't been initialized.
        assert!(channel_provider.channel_bank.is_none());
        assert!(channel_provider.channel_assembler.is_none());
        assert!(channel_provider.prev.is_some());

        // Load in the active stage.
        channel_provider.attempt_update().unwrap();
        assert!(channel_provider.channel_bank.is_some());
        assert!(channel_provider.channel_assembler.is_none());
        assert!(channel_provider.prev.is_none());
        // Ensure the active stage is retained on the second call.
        channel_provider.attempt_update().unwrap();
        assert!(channel_provider.channel_bank.is_some());
        assert!(channel_provider.channel_assembler.is_none());
        assert!(channel_provider.prev.is_none());
    }

    #[test]
    fn test_channel_provider_retain_current_assembler() {
        let provider = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let mut channel_provider = ChannelProvider::new(cfg, provider);

        // Assert the multiplexer hasn't been initialized.
        assert!(channel_provider.channel_bank.is_none());
        assert!(channel_provider.channel_assembler.is_none());
        assert!(channel_provider.prev.is_some());

        // Load in the active stage.
        channel_provider.attempt_update().unwrap();
        assert!(channel_provider.channel_bank.is_none());
        assert!(channel_provider.channel_assembler.is_some());
        assert!(channel_provider.prev.is_none());
        // Ensure the active stage is retained on the second call.
        channel_provider.attempt_update().unwrap();
        assert!(channel_provider.channel_bank.is_none());
        assert!(channel_provider.channel_assembler.is_some());
        assert!(channel_provider.prev.is_none());
    }

    #[test]
    fn test_channel_provider_transition_stage() {
        let provider = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(2), ..Default::default() },
            ..Default::default()
        });
        let mut channel_provider = ChannelProvider::new(cfg, provider);

        channel_provider.attempt_update().unwrap();

        // Update the L1 origin to Holocene activation.
        let Some(ref mut stage) = channel_provider.channel_bank else {
            panic!("Expected ChannelBank");
        };
        stage.prev.block_info = Some(BlockInfo { number: 1, timestamp: 2, ..Default::default() });

        // Transition to the ChannelAssembler stage.
        channel_provider.attempt_update().unwrap();
        assert!(channel_provider.channel_bank.is_none());
        assert!(channel_provider.channel_assembler.is_some());

        assert_eq!(channel_provider.origin().unwrap().number, 1);
    }

    #[test]
    fn test_channel_provider_transition_stage_backwards() {
        let provider = TestNextFrameProvider::new(vec![]);
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(2), ..Default::default() },
            ..Default::default()
        });
        let mut channel_provider = ChannelProvider::new(cfg, provider);

        channel_provider.attempt_update().unwrap();

        // Update the L1 origin to Holocene activation.
        let Some(ref mut stage) = channel_provider.channel_bank else {
            panic!("Expected ChannelBank");
        };
        stage.prev.block_info = Some(BlockInfo { number: 1, timestamp: 2, ..Default::default() });

        // Transition to the ChannelAssembler stage.
        channel_provider.attempt_update().unwrap();
        assert!(channel_provider.channel_bank.is_none());
        assert!(channel_provider.channel_assembler.is_some());

        // Update the L1 origin to before Holocene activation, to simulate a re-org.
        let Some(ref mut stage) = channel_provider.channel_assembler else {
            panic!("Expected ChannelAssembler");
        };
        stage.prev.block_info = Some(BlockInfo::default());

        channel_provider.attempt_update().unwrap();
        assert!(channel_provider.channel_bank.is_some());
        assert!(channel_provider.channel_assembler.is_none());
    }

    #[tokio::test]
    async fn test_channel_provider_reset_bank() {
        let frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let provider = TestNextFrameProvider::new(frames.into_iter().rev().map(Ok).collect());
        let cfg = Arc::new(RollupConfig::default());
        let mut channel_provider = ChannelProvider::new(cfg.clone(), provider);

        // Load in the first frame.
        assert_eq!(
            channel_provider.next_data().await.unwrap_err(),
            PipelineError::NotEnoughData.temp()
        );
        let Some(channel_bank) = channel_provider.channel_bank.as_mut() else {
            panic!("Expected ChannelBank");
        };
        // Ensure a channel is in the queue.
        assert!(channel_bank.channel_queue.len() == 1);

        // Reset the channel provider.
        channel_provider.signal(ResetSignal::default().signal()).await.unwrap();

        // Ensure the channel queue is empty after reset.
        let Some(channel_bank) = channel_provider.channel_bank.as_mut() else {
            panic!("Expected ChannelBank");
        };
        assert!(channel_bank.channel_queue.is_empty());
    }

    #[tokio::test]
    async fn test_channel_provider_reset_assembler() {
        let frames = [
            crate::frame!(0xFF, 0, vec![0xDD; 50], false),
            crate::frame!(0xFF, 1, vec![0xDD; 50], true),
        ];
        let provider = TestNextFrameProvider::new(frames.into_iter().rev().map(Ok).collect());
        let cfg = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let mut channel_provider = ChannelProvider::new(cfg.clone(), provider);

        // Load in the first frame.
        assert_eq!(
            channel_provider.next_data().await.unwrap_err(),
            PipelineError::NotEnoughData.temp()
        );
        let Some(channel_assembler) = channel_provider.channel_assembler.as_mut() else {
            panic!("Expected ChannelAssembler");
        };
        // Ensure a channel is being built.
        assert!(channel_assembler.channel.is_some());

        // Reset the channel provider.
        channel_provider.signal(ResetSignal::default().signal()).await.unwrap();

        // Ensure the channel assembler is empty after reset.
        let Some(channel_assembler) = channel_provider.channel_assembler.as_mut() else {
            panic!("Expected ChannelAssembler");
        };
        assert!(channel_assembler.channel.is_none());
    }
}
