//! The [`L1OriginSelector`].

use crate::l1;
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::{fmt::Debug, sync::Arc};

/// Trait for selecting the next L1 origin block for sequencing.
///
/// This trait is used by the sequencer to determine which L1 block should be used
/// as the origin for the next L2 block being built.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait OriginSelector: Debug + Send + Sync {
    /// Selects the next L1 origin block for sequencing.
    ///
    /// # Arguments
    /// * `unsafe_head` - The current unsafe head of the L2 chain
    ///
    /// # Returns
    /// The selected L1 origin block information, or an error if selection failed.
    async fn next_l1_origin(
        &mut self,
        unsafe_head: L2BlockInfo,
    ) -> Result<BlockInfo, L1OriginSelectorError>;
}

/// The [`L1OriginSelector`] is responsible for selecting the L1 origin block based on the
/// current L2 unsafe head's sequence epoch.
///
/// It asks the L1 actor for the unsafe head's L1 origin and its canonical child, and takes the
/// child as the next origin only once it is `confirmations` blocks below the L1 head.
#[derive(Debug)]
pub struct L1OriginSelector {
    /// The [`RollupConfig`].
    cfg: Arc<RollupConfig>,
    /// The L1 chain.
    l1: l1::Handle,
    /// How many blocks the next L1 origin must be below the L1 head.
    confirmations: u64,
}

#[async_trait]
impl OriginSelector for L1OriginSelector {
    /// Determines what the next L1 origin block should be, based off of the [`L2BlockInfo`] unsafe
    /// head.
    ///
    /// The L1 origin is selected based off of the sequencing epoch, determined by the next L2
    /// block's timestamp in relation to the current L1 origin's timestamp. If the next L2
    /// block's timestamp is greater than the L2 unsafe head's L1 origin timestamp, the L1
    /// origin is the block following the current L1 origin.
    async fn next_l1_origin(
        &mut self,
        unsafe_head: L2BlockInfo,
    ) -> Result<BlockInfo, L1OriginSelectorError> {
        let (current, next) = self.origins(unsafe_head.l1_origin).await?;

        // Start building on the next L1 origin block if the next L2 block's timestamp is
        // greater than or equal to the next L1 origin's timestamp.
        if let Some(next) = next &&
            unsafe_head.block_info.timestamp + self.cfg.block_time >= next.timestamp
        {
            return Ok(next);
        }

        let max_seq_drift = self.cfg.max_sequencer_drift(current.timestamp);
        let past_seq_drift = unsafe_head.block_info.timestamp + self.cfg.block_time -
            current.timestamp >
            max_seq_drift;

        // If the sequencer drift has not been exceeded, return the current L1 origin.
        if !past_seq_drift {
            return Ok(current);
        }

        warn!(
            target: "l1_origin_selector",
            current_origin_time = current.timestamp,
            unsafe_head_time = unsafe_head.block_info.timestamp,
            max_seq_drift,
            "Next L2 block time is past the sequencer drift"
        );

        if next
            .map(|n| unsafe_head.block_info.timestamp + self.cfg.block_time < n.timestamp)
            .unwrap_or(false)
        {
            // If the next L1 origin is ahead of the next L2 block's timestamp, return the current
            // origin.
            return Ok(current);
        }

        next.ok_or(L1OriginSelectorError::NotEnoughData(current))
    }
}

impl L1OriginSelector {
    /// Creates a new [`L1OriginSelector`], which selects origins from `l1` once they are
    /// `confirmations` blocks below the L1 head.
    pub const fn new(cfg: Arc<RollupConfig>, l1: l1::Handle, confirmations: u64) -> Self {
        Self { cfg, l1, confirmations }
    }

    /// Returns the current L1 origin, and its confirmed canonical child if L1 has one.
    async fn origins(
        &self,
        origin: BlockNumHash,
    ) -> Result<(BlockInfo, Option<BlockInfo>), L1OriginSelectorError> {
        let current = match self.l1.canonical(origin.number).await? {
            Some(current) if current.hash == origin.hash => current,
            Some(_) => return Err(L1OriginSelectorError::OriginReorged(origin)),
            None => return Err(L1OriginSelectorError::OriginAhead(origin)),
        };
        let next = match self.l1.try_child_of(origin, self.confirmations).await? {
            Some(l1::Child::Block(next)) => Some(next),
            Some(l1::Child::Reorged) => return Err(L1OriginSelectorError::OriginReorged(origin)),
            None => None,
        };
        Ok((current, next))
    }
}

/// An error produced by the [`L1OriginSelector`].
#[derive(Debug, thiserror::Error)]
pub enum L1OriginSelectorError {
    /// The L1 provider does not have enough data to select the next L1 origin block.
    #[error(
        "Waiting for more L1 data to be available to select the next L1 origin block. Current L1 origin: {0:?}"
    )]
    NotEnoughData(BlockInfo),
    /// L1 reorged out the unsafe head's L1 origin.
    #[error("L1 reorged out the L1 origin {0:?}")]
    OriginReorged(BlockNumHash),
    /// The unsafe head's L1 origin is after the L1 head.
    #[error("the L1 origin {0:?} is after the L1 head")]
    OriginAhead(BlockNumHash),
    /// The L1 actor could not answer.
    #[error(transparent)]
    L1(#[from] l1::HandleError),
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::l1::test_utils::{TestChain, block, chain};
    use rstest::rstest;

    /// The L1 slot time of the test L1 blocks.
    const L1_SLOT_TIME: u64 = 12;

    /// An L2 block time of 2 seconds, and a sequencer drift of 600 seconds.
    fn rollup_config() -> Arc<RollupConfig> {
        Arc::new(RollupConfig { block_time: 2, max_sequencer_drift: 600, ..Default::default() })
    }

    /// Returns an unsafe head with the given timestamp, built on the given L1 origin.
    fn unsafe_head(timestamp: u64, l1_origin: BlockInfo) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { timestamp, ..Default::default() },
            l1_origin: l1_origin.id(),
            seq_num: 0,
        }
    }

    #[tokio::test]
    #[rstest]
    #[case::single_epoch(1)]
    #[case::many_epochs(12)]
    async fn test_next_l1_origin_several_epochs(#[case] num_epochs: u64) {
        let cfg = rollup_config();
        // One more L1 block than epochs, such that the next logical origin is always available.
        let l1 = TestChain::new(&chain(0..=num_epochs, 1));
        let mut selector = L1OriginSelector::new(cfg.clone(), l1.handle(), 0);

        // Ensure all L1 origin blocks are produced correctly for each L2 block within all available
        // epochs.
        for i in 0..(num_epochs * (L1_SLOT_TIME / cfg.block_time)) {
            let current_epoch = (i * cfg.block_time) / L1_SLOT_TIME;
            let head = unsafe_head(i * cfg.block_time, block(current_epoch, 1, 1));
            let next = selector.next_l1_origin(head).await.unwrap();

            // The expected L1 origin block is the one corresponding to the epoch of the current L2
            // block.
            let expected_epoch = ((i + 1) * cfg.block_time) / L1_SLOT_TIME;
            assert_eq!(next, block(expected_epoch, 1, 1));
        }
    }

    #[tokio::test]
    #[rstest]
    #[case::not_available(false)]
    #[case::is_available(true)]
    async fn test_next_l1_origin_next_maybe_available(#[case] next_l1_origin_available: bool) {
        let l1 = TestChain::new(&chain(0..=u64::from(next_l1_origin_available), 1));
        let mut selector = L1OriginSelector::new(rollup_config(), l1.handle(), 0);

        // The next L2 block's timestamp is that of L1 block 1.
        let next = selector.next_l1_origin(unsafe_head(10, block(0, 1, 1))).await.unwrap();

        // Assuming the next L1 origin block is not available (_and_ it is not past the sequencer
        // drift), the current L1 origin block will be re-used.
        let expected_epoch = u64::from(next_l1_origin_available);
        assert_eq!(next, block(expected_epoch, 1, 1));
    }

    #[tokio::test]
    #[rstest]
    #[case::next_not_available(false, false)]
    #[case::next_available_but_behind(true, false)]
    #[case::next_available_and_ahead(true, true)]
    async fn test_next_l1_origin_next_past_seq_drift(
        #[case] next_available: bool,
        #[case] next_ahead_of_unsafe: bool,
    ) {
        let cfg = rollup_config();
        let mut blocks = vec![block(0, 1, 1)];
        if next_available {
            let timestamp = if next_ahead_of_unsafe {
                cfg.max_sequencer_drift + cfg.block_time * 2
            } else {
                cfg.block_time
            };
            blocks.push(BlockInfo { timestamp, ..block(1, 1, 1) });
        }
        let l1 = TestChain::new(&blocks);
        let mut selector = L1OriginSelector::new(cfg.clone(), l1.handle(), 0);
        let head = unsafe_head(cfg.max_sequencer_drift, block(0, 1, 1));

        if next_available {
            let next = selector.next_l1_origin(head).await.unwrap();
            if next_ahead_of_unsafe {
                // If the next L1 origin is available and ahead of the unsafe head, the L1 origin
                // should not change.
                assert_eq!(next, block(0, 1, 1));
            } else {
                // If the next L1 origin is available and behind the unsafe head, the L1 origin
                // should advance.
                assert_eq!(next, blocks[1]);
            }
        } else {
            // If we're past the sequencer drift, and the next L1 block is not available, a
            // `NotEnoughData` error should be returned signifying that we cannot
            // proceed with the next L1 origin until the block is present.
            let next_err = selector.next_l1_origin(head).await.unwrap_err();
            assert!(matches!(next_err, L1OriginSelectorError::NotEnoughData(_)));
        }
    }

    #[tokio::test]
    async fn the_next_origin_needs_its_confirmations() {
        let l1 = TestChain::new(&chain(0..=3, 1));
        let mut selector = L1OriginSelector::new(rollup_config(), l1.handle(), 2);
        // Block 1 is two blocks below the head, but block 2 is not.
        assert_eq!(
            selector.next_l1_origin(unsafe_head(10, block(0, 1, 1))).await.unwrap(),
            block(1, 1, 1)
        );
        assert_eq!(
            selector.next_l1_origin(unsafe_head(22, block(1, 1, 1))).await.unwrap(),
            block(1, 1, 1)
        );
        l1.set_head(&[block(4, 1, 1)]);
        assert_eq!(
            selector.next_l1_origin(unsafe_head(22, block(1, 1, 1))).await.unwrap(),
            block(2, 1, 1)
        );
    }

    #[tokio::test]
    async fn a_reorged_out_origin_is_an_error() {
        let l1 = TestChain::new(&chain(0..=2, 1));
        let mut selector = L1OriginSelector::new(rollup_config(), l1.handle(), 0);
        let origin = block(1, 2, 1);
        assert!(matches!(
            selector.next_l1_origin(unsafe_head(10, origin)).await,
            Err(L1OriginSelectorError::OriginReorged(reorged)) if reorged == origin.id()
        ));
    }

    #[tokio::test]
    async fn an_origin_after_the_l1_head_is_an_error() {
        let l1 = TestChain::new(&chain(0..=2, 1));
        let mut selector = L1OriginSelector::new(rollup_config(), l1.handle(), 0);
        let origin = block(3, 1, 1);
        assert!(matches!(
            selector.next_l1_origin(unsafe_head(36, origin)).await,
            Err(L1OriginSelectorError::OriginAhead(ahead)) if ahead == origin.id()
        ));
    }
}
