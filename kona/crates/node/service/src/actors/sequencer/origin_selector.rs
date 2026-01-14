//! The [`L1OriginSelector`].

use alloy_primitives::B256;
use alloy_provider::{Provider, RootProvider};
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::{fmt::Debug, sync::Arc};
use tokio::sync::watch;

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
    /// * `is_recovery_mode` - Whether the sequencer is in recovery mode
    ///
    /// # Returns
    /// The selected L1 origin block information, or an error if selection failed.
    async fn next_l1_origin(
        &mut self,
        unsafe_head: L2BlockInfo,
        is_recovery_mode: bool,
    ) -> Result<BlockInfo, L1OriginSelectorError>;
}

/// The [`L1OriginSelector`] is responsible for selecting the L1 origin block based on the
/// current L2 unsafe head's sequence epoch.
#[derive(Debug)]
pub struct L1OriginSelector<P: L1OriginSelectorProvider> {
    /// The [`RollupConfig`].
    cfg: Arc<RollupConfig>,
    /// The [`L1OriginSelectorProvider`].
    l1: P,
    /// The current L1 origin.
    current: Option<BlockInfo>,
    /// The next L1 origin.
    next: Option<BlockInfo>,
}

#[async_trait]
impl<P: L1OriginSelectorProvider + Send + Sync> OriginSelector for L1OriginSelector<P> {
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
        is_recovery_mode: bool,
    ) -> Result<BlockInfo, L1OriginSelectorError> {
        self.select_origins(&unsafe_head, is_recovery_mode).await?;

        // Start building on the next L1 origin block if the next L2 block's timestamp is
        // greater than or equal to the next L1 origin's timestamp.
        if let Some(next) = self.next {
            if unsafe_head.block_info.timestamp + self.cfg.block_time >= next.timestamp {
                return Ok(next);
            }
        }

        let Some(current) = self.current else {
            unreachable!("Current L1 origin should always be set by `select_origins`");
        };

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

        if self
            .next
            .map(|n| unsafe_head.block_info.timestamp + self.cfg.block_time < n.timestamp)
            .unwrap_or(false)
        {
            // If the next L1 origin is ahead of the next L2 block's timestamp, return the current
            // origin.
            return Ok(current);
        }

        self.next.ok_or(L1OriginSelectorError::NotEnoughData(current))
    }
}

impl<P: L1OriginSelectorProvider> L1OriginSelector<P> {
    /// Creates a new [`L1OriginSelector`].
    pub const fn new(cfg: Arc<RollupConfig>, l1: P) -> Self {
        Self { cfg, l1, current: None, next: None }
    }

    /// Returns the current L1 origin.
    pub const fn current(&self) -> Option<&BlockInfo> {
        self.current.as_ref()
    }

    /// Returns the next L1 origin.
    pub const fn next(&self) -> Option<&BlockInfo> {
        self.next.as_ref()
    }

    /// Selects the current and next L1 origin blocks based on the unsafe head.
    async fn select_origins(
        &mut self,
        unsafe_head: &L2BlockInfo,
        in_recovery_mode: bool,
    ) -> Result<(), L1OriginSelectorError> {
        if in_recovery_mode {
            self.current = self.l1.get_block_by_hash(unsafe_head.l1_origin.hash).await?;
            self.next = self.l1.get_block_by_number(unsafe_head.l1_origin.number + 1).await?;
            return Ok(());
        }

        if self.current.map(|c| c.hash == unsafe_head.l1_origin.hash).unwrap_or(false) {
            // Do nothing; The next L2 block exists in the same epoch as the current L1 origin.
        } else if self.next.map(|n| n.hash == unsafe_head.l1_origin.hash).unwrap_or(false) {
            // Advance the origin.
            self.current = self.next.take();
            self.next = None;
        } else {
            // Find the current origin block, as it is missing.
            let current = self.l1.get_block_by_hash(unsafe_head.l1_origin.hash).await?;

            self.current = current;
            self.next = None;
        }

        self.try_fetch_next_origin().await
    }

    /// Attempts to fetch the next L1 origin block.
    async fn try_fetch_next_origin(&mut self) -> Result<(), L1OriginSelectorError> {
        // If there is no next L1 origin set, attempt to find it. If it's not yet available, leave
        // it unset.
        if let Some(current) = self.current.as_ref() {
            // If the next L1 origin is already set, do nothing.
            if self.next.is_some() {
                return Ok(());
            }

            // If the next L1 origin is a logical extension of the current L1 chain, set it.
            //
            // Ignore the eventuality that the block is not found, as the next L1 origin fetch is
            // performed on a best-effort basis.
            let next = self.l1.get_block_by_number(current.number + 1).await?;
            if next.map(|n| n.parent_hash == current.hash).unwrap_or(false) {
                self.next = next;
            }
        }

        Ok(())
    }
}

/// An error produced by the [`L1OriginSelector`].
#[derive(Debug, thiserror::Error)]
pub enum L1OriginSelectorError {
    /// An error produced by the [`RootProvider`].
    #[error(transparent)]
    Provider(#[from] RpcError<TransportErrorKind>),
    /// The L1 provider does not have enough data to select the next L1 origin block.
    #[error(
        "Waiting for more L1 data to be available to select the next L1 origin block. Current L1 origin: {0:?}"
    )]
    NotEnoughData(BlockInfo),
}

/// L1 [`BlockInfo`] provider interface for the [`L1OriginSelector`].
#[async_trait]
pub trait L1OriginSelectorProvider: Debug + Sync {
    /// Returns a [`BlockInfo`] by its hash.
    async fn get_block_by_hash(
        &self,
        hash: B256,
    ) -> Result<Option<BlockInfo>, L1OriginSelectorError>;

    /// Returns a [`BlockInfo`] by its number.
    async fn get_block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockInfo>, L1OriginSelectorError>;
}

/// A wrapper around the [`RootProvider`] that delays the view of the L1 chain by a configurable
/// amount of blocks.
#[derive(Debug)]
pub struct DelayedL1OriginSelectorProvider {
    /// The inner [`RootProvider`].
    inner: RootProvider,
    /// The L1 head watch channel.
    l1_head: watch::Receiver<Option<BlockInfo>>,
    /// The confirmation depth to delay the view of the L1 chain.
    confirmation_depth: u64,
}

impl DelayedL1OriginSelectorProvider {
    /// Creates a new [`DelayedL1OriginSelectorProvider`].
    pub const fn new(
        inner: RootProvider,
        l1_head: watch::Receiver<Option<BlockInfo>>,
        confirmation_depth: u64,
    ) -> Self {
        Self { inner, l1_head, confirmation_depth }
    }
}

#[async_trait]
impl L1OriginSelectorProvider for DelayedL1OriginSelectorProvider {
    async fn get_block_by_hash(
        &self,
        hash: B256,
    ) -> Result<Option<BlockInfo>, L1OriginSelectorError> {
        // By-hash lookups are not delayed, as they're direct indexes.
        Ok(Provider::get_block_by_hash(&self.inner, hash).await?.map(Into::into))
    }

    async fn get_block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockInfo>, L1OriginSelectorError> {
        let Some(l1_head) = *self.l1_head.borrow() else {
            // If the L1 head is not available, do not enforce a confirmation delay.
            return Ok(Provider::get_block_by_number(&self.inner, number.into())
                .await?
                .map(Into::into));
        };

        if number == 0 ||
            self.confirmation_depth == 0 ||
            number + self.confirmation_depth <= l1_head.number
        {
            Ok(Provider::get_block_by_number(&self.inner, number.into()).await?.map(Into::into))
        } else {
            Ok(None)
        }
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use alloy_eips::NumHash;
    use rstest::rstest;
    use std::collections::HashSet;

    /// A mock [`OriginSelectorProvider`] with a local set of [`BlockInfo`]s available.
    #[derive(Default, Debug, Clone)]
    struct MockOriginSelectorProvider {
        blocks: HashSet<BlockInfo>,
    }

    impl MockOriginSelectorProvider {
        /// Creates a new [`MockOriginSelectorProvider`].
        pub(crate) fn with_block(&mut self, block: BlockInfo) {
            self.blocks.insert(block);
        }
    }

    #[async_trait]
    impl L1OriginSelectorProvider for MockOriginSelectorProvider {
        async fn get_block_by_hash(
            &self,
            hash: B256,
        ) -> Result<Option<BlockInfo>, L1OriginSelectorError> {
            Ok(self.blocks.iter().find(|b| b.hash == hash).copied())
        }

        async fn get_block_by_number(
            &self,
            number: u64,
        ) -> Result<Option<BlockInfo>, L1OriginSelectorError> {
            Ok(self.blocks.iter().find(|b| b.number == number).copied())
        }
    }

    #[tokio::test]
    #[rstest]
    #[case::single_epoch(1)]
    #[case::many_epochs(12)]
    async fn test_next_l1_origin_several_epochs(#[case] num_epochs: usize) {
        // Assume an L1 slot time of 12 seconds.
        const L1_SLOT_TIME: u64 = 12;
        // Assume an L2 block time of 2 seconds.
        const L2_BLOCK_TIME: u64 = 2;

        // Initialize the rollup configuration with a block time of 2 seconds and a sequencer drift
        // of 600 seconds.
        let cfg = Arc::new(RollupConfig {
            block_time: L2_BLOCK_TIME,
            max_sequencer_drift: 600,
            ..Default::default()
        });

        // Initialize the provider with mock L1 blocks, equal to the number of epochs + 1
        // (such that the next logical origin is always available.)
        let mut provider = MockOriginSelectorProvider::default();
        for i in 0..num_epochs + 1 {
            provider.with_block(BlockInfo {
                parent_hash: B256::with_last_byte(i.saturating_sub(1) as u8),
                hash: B256::with_last_byte(i as u8),
                number: i as u64,
                timestamp: i as u64 * L1_SLOT_TIME,
            });
        }

        let mut selector = L1OriginSelector::new(cfg.clone(), provider);

        // Ensure all L1 origin blocks are produced correctly for each L2 block within all available
        // epochs.
        for i in 0..(num_epochs as u64 * (L1_SLOT_TIME / cfg.block_time)) {
            let current_epoch = (i * cfg.block_time) / L1_SLOT_TIME;
            let unsafe_head = L2BlockInfo {
                block_info: BlockInfo {
                    hash: B256::ZERO,
                    number: i,
                    timestamp: i * cfg.block_time,
                    ..Default::default()
                },
                l1_origin: NumHash {
                    number: current_epoch,
                    hash: B256::with_last_byte(current_epoch as u8),
                },
                seq_num: 0,
            };
            let next = selector.next_l1_origin(unsafe_head, false).await.unwrap();

            // The expected L1 origin block is the one corresponding to the epoch of the current L2
            // block.
            let expected_epoch = ((i + 1) * cfg.block_time) / L1_SLOT_TIME;
            assert_eq!(next.hash, B256::with_last_byte(expected_epoch as u8));
            assert_eq!(next.number, expected_epoch);
        }
    }

    #[tokio::test]
    #[rstest]
    #[case::not_available(false)]
    #[case::is_available(true)]
    async fn test_next_l1_origin_next_maybe_available(#[case] next_l1_origin_available: bool) {
        // Assume an L2 block time of 2 seconds.
        const L2_BLOCK_TIME: u64 = 2;

        // Initialize the rollup configuration with a block time of 2 seconds and a sequencer drift
        // of 600 seconds.
        let cfg = Arc::new(RollupConfig {
            block_time: L2_BLOCK_TIME,
            max_sequencer_drift: 600,
            ..Default::default()
        });

        // Initialize the provider with a single L1 block.
        let mut provider = MockOriginSelectorProvider::default();
        provider.with_block(BlockInfo {
            parent_hash: B256::ZERO,
            hash: B256::ZERO,
            number: 0,
            timestamp: 0,
        });

        if next_l1_origin_available {
            // If the next L1 origin is available, add it to the provider.
            provider.with_block(BlockInfo {
                parent_hash: B256::ZERO,
                hash: B256::with_last_byte(1),
                number: 1,
                timestamp: cfg.block_time,
            });
        }

        let mut selector = L1OriginSelector::new(cfg.clone(), provider);

        let current_epoch = 0;
        let unsafe_head = L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::ZERO,
                number: 5,
                timestamp: 5 * cfg.block_time,
                ..Default::default()
            },
            l1_origin: NumHash {
                number: current_epoch,
                hash: B256::with_last_byte(current_epoch as u8),
            },
            seq_num: 0,
        };
        let next = selector.next_l1_origin(unsafe_head, false).await.unwrap();

        // The expected L1 origin block is the one corresponding to the epoch of the current L2
        // block. Assuming the next L1 origin block is not available from the eyes of the
        // provider (_and_ it is not past the sequencer drift), the current L1 origin block
        // will be re-used.
        let expected_epoch =
            if next_l1_origin_available { current_epoch + 1 } else { current_epoch };
        assert_eq!(next.hash, B256::with_last_byte(expected_epoch as u8));
        assert_eq!(next.number, expected_epoch);
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
        // Assume an L2 block time of 2 seconds.
        const L2_BLOCK_TIME: u64 = 2;

        // Initialize the rollup configuration with a block time of 2 seconds and a sequencer drift
        // of 600 seconds.
        let cfg = Arc::new(RollupConfig {
            block_time: L2_BLOCK_TIME,
            max_sequencer_drift: 600,
            ..Default::default()
        });

        // Initialize the provider with a single L1 block.
        let mut provider = MockOriginSelectorProvider::default();
        provider.with_block(BlockInfo {
            parent_hash: B256::ZERO,
            hash: B256::ZERO,
            number: 0,
            timestamp: 0,
        });

        if next_available {
            // If the next L1 origin is to be available, add it to the provider.
            provider.with_block(BlockInfo {
                parent_hash: B256::ZERO,
                hash: B256::with_last_byte(1),
                number: 1,
                timestamp: if next_ahead_of_unsafe {
                    cfg.max_sequencer_drift + cfg.block_time * 2
                } else {
                    cfg.block_time
                },
            });
        }

        let mut selector = L1OriginSelector::new(cfg.clone(), provider);

        let current_epoch = 0;
        let unsafe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: cfg.max_sequencer_drift, ..Default::default() },
            l1_origin: NumHash {
                number: current_epoch,
                hash: B256::with_last_byte(current_epoch as u8),
            },
            seq_num: 0,
        };

        if next_available {
            if next_ahead_of_unsafe {
                // If the next L1 origin is available and ahead of the unsafe head, the L1 origin
                // should not change.
                let next = selector.next_l1_origin(unsafe_head, false).await.unwrap();
                assert_eq!(next.hash, B256::ZERO);
                assert_eq!(next.number, 0);
            } else {
                // If the next L1 origin is available and behind the unsafe head, the L1 origin
                // should advance.
                let next = selector.next_l1_origin(unsafe_head, false).await.unwrap();
                assert_eq!(next.hash, B256::with_last_byte(1));
                assert_eq!(next.number, 1);
            }
        } else {
            // If we're past the sequencer drift, and the next L1 block is not available, a
            // `NotEnoughData` error should be returned signifying that we cannot
            // proceed with the next L1 origin until the block is present.
            let next_err = selector.next_l1_origin(unsafe_head, false).await.unwrap_err();
            assert!(matches!(next_err, L1OriginSelectorError::NotEnoughData(_)));
        }
    }
}
