//! The finality actor.

use super::{ActorError, L1};
use crate::engine::Safe;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::collections::BTreeMap;
use tokio::sync::watch;

/// An L1 block number.
type L1BlockNumber = u64;

/// Finalizes L2 blocks once the L1 blocks they were derived from finalize.
///
/// The engine's safe heads coalesce, so the actor may miss some. It then finalizes the highest one
/// it saw, and catches up when the next L1 block finalizes.
#[derive(Debug)]
pub(super) struct Actor<L1_> {
    /// The engine's latest safe head, which handles publish.
    safe_rx: watch::Receiver<Option<Safe>>,
    /// Publishes the block to finalize.
    finalized_tx: watch::Sender<Option<L2BlockInfo>>,
    /// The L1 chain, whose finalized blocks the actor follows.
    l1: L1_,
    /// The highest safe L2 block derived from each L1 block, awaiting finalization.
    safe_heads: BTreeMap<L1BlockNumber, L2BlockInfo>,
    /// The latest finalized L1 block.
    l1_finalized: Option<BlockInfo>,
}

impl<L1_: L1> Actor<L1_> {
    /// Creates the actor.
    pub(super) const fn new(
        safe_rx: watch::Receiver<Option<Safe>>,
        finalized_tx: watch::Sender<Option<L2BlockInfo>>,
        l1: L1_,
    ) -> Self {
        Self { safe_rx, finalized_tx, l1, safe_heads: BTreeMap::new(), l1_finalized: None }
    }

    pub(super) async fn run(mut self) -> Result<(), ActorError> {
        loop {
            tokio::select! {
                changed = self.safe_rx.changed() => {
                    changed.map_err(|_| ActorError::HandlesDropped)?;
                    let safe = *self.safe_rx.borrow_and_update();
                    if let Some(safe) = safe {
                        self.record(safe);
                    }
                }
                finalized = self.l1.next_finalized() => {
                    self.l1_finalized = Some(finalized.ok_or(ActorError::L1Stopped)?);
                }
            }
            self.finalize();
        }
    }

    /// Records the safe head and the L1 block it was derived from.
    ///
    /// The engine's safe head only moves forward, except when the engine resets. So a safe head at
    /// or below one recorded follows a reset, which an L1 reorg may have caused: the actor forgets
    /// the safe heads from its height up. Those below it are its ancestors, older than any reorg
    /// the reset walked back from.
    fn record(&mut self, safe: Safe) {
        let number = safe.head.block_info.number;
        if self.safe_heads.values().any(|head| head.block_info.number >= number) {
            self.safe_heads.retain(|_, head| head.block_info.number < number);
        }
        let highest = self.safe_heads.entry(safe.source.number).or_insert(safe.head);
        if highest.block_info.number < number {
            *highest = safe.head;
        }
    }

    /// Publishes the highest recorded safe head whose source the finalized L1 block covers, and
    /// forgets the safe heads it covers.
    fn finalize(&mut self) {
        let Some(l1_finalized) = self.l1_finalized else { return };
        let covered = self.safe_heads.range(..=l1_finalized.number).next_back();
        let Some((_, &finalized)) = covered else { return };
        self.safe_heads.retain(|&source, _| source > l1_finalized.number);
        self.finalized_tx.send_replace(Some(finalized));
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        engine::Finality,
        finality::{Builder, Handle},
    };
    use async_trait::async_trait;
    use std::future::Future;
    use tokio::time::{self, Duration};

    #[async_trait]
    impl L1 for watch::Receiver<BlockInfo> {
        async fn next_finalized(&mut self) -> Option<BlockInfo> {
            self.changed().await.ok()?;
            Some(*self.borrow_and_update())
        }
    }

    /// Runs an actor while `script` drives it through its handle and the finalized L1 block, and
    /// returns the actor's error if it stops first.
    async fn run_during<Script: Future<Output = ()>>(
        script: impl FnOnce(Handle, watch::Sender<BlockInfo>) -> Script,
    ) -> Option<ActorError> {
        let builder = Builder::new();
        let handle = builder.handle();
        let (finalized, finalized_rx) = watch::channel(BlockInfo::default());
        tokio::select! {
            result = builder.build(finalized_rx) => Some(result.unwrap_err()),
            () = script(handle, finalized) => None,
        }
    }

    fn l1_block(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    fn l2_block(number: u64) -> L2BlockInfo {
        L2BlockInfo { block_info: l1_block(number), ..Default::default() }
    }

    /// Returns the L2 block with the given number as a safe head, derived from the L1 block with
    /// the given number.
    fn safe(head: u64, source: u64) -> Safe {
        Safe { head: l2_block(head), source: l1_block(source) }
    }

    /// Gives the actor `safe`, and returns the block it decides to finalize, if any within a while.
    async fn next(handle: &mut Handle, safe: Safe) -> Option<L2BlockInfo> {
        time::timeout(Duration::from_secs(10), handle.next(safe)).await.ok().map(Result::unwrap)
    }

    /// Each finalized L1 block finalizes the highest safe head derived from it or an earlier L1
    /// block, once.
    #[tokio::test(start_paused = true)]
    async fn finalizes_the_highest_safe_head_a_finalized_l1_block_covers() {
        let stopped = run_during(async |mut handle, finalized| {
            for (head, source) in [(10, 5), (12, 6), (13, 6)] {
                assert_eq!(next(&mut handle, safe(head, source)).await, None);
            }
            finalized.send_replace(l1_block(4));
            assert_eq!(next(&mut handle, safe(15, 8)).await, None);
            finalized.send_replace(l1_block(6));
            assert_eq!(next(&mut handle, safe(15, 8)).await, Some(l2_block(13)));
            finalized.send_replace(l1_block(7));
            assert_eq!(next(&mut handle, safe(15, 8)).await, None);
            finalized.send_replace(l1_block(8));
            assert_eq!(next(&mut handle, safe(15, 8)).await, Some(l2_block(15)));
        })
        .await;
        assert!(stopped.is_none());
    }

    /// A safe head at or below one recorded follows an engine reset: the actor forgets the safe
    /// heads from its height up.
    #[tokio::test(start_paused = true)]
    async fn a_safe_head_moving_back_forgets_the_safe_heads_above_it() {
        let stopped = run_during(async |mut handle, finalized| {
            assert_eq!(next(&mut handle, safe(8, 4)).await, None);
            assert_eq!(next(&mut handle, safe(10, 5)).await, None);
            // The engine reset to block 8, and derived block 9 from L1 block 6.
            assert_eq!(next(&mut handle, safe(9, 6)).await, None);
            finalized.send_replace(l1_block(5));
            assert_eq!(next(&mut handle, safe(9, 6)).await, Some(l2_block(8)));
            finalized.send_replace(l1_block(6));
            assert_eq!(next(&mut handle, safe(9, 6)).await, Some(l2_block(9)));
        })
        .await;
        assert!(stopped.is_none());
    }

    /// The actor stops once every handle is dropped, or the L1 actor stops.
    #[rstest::rstest]
    #[case::handles(true)]
    #[case::l1(false)]
    #[tokio::test]
    async fn stops_when_its_publishers_stop(#[case] handles: bool) {
        let builder = Builder::new();
        let handle = builder.handle();
        let (finalized, finalized_rx) = watch::channel(BlockInfo::default());
        if handles {
            drop(handle);
        } else {
            drop(finalized);
        }
        let result = builder.build(finalized_rx).await;
        if handles {
            assert!(matches!(result, Err(ActorError::HandlesDropped)));
        } else {
            assert!(matches!(result, Err(ActorError::L1Stopped)));
        }
    }
}
