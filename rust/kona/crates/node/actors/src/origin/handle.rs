//! Answers the sequencer's question: which L1 origin to build on.

use super::window::{Origins, Window};
use alloy_eips::BlockNumHash;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::watch;

/// Why the origin actor gave no L1 origin.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
pub enum OriginError {
    /// The next L2 block would be past the sequencer drift on its parent's origin, and L1 has no
    /// confirmed next origin yet.
    #[error("the next L2 block reaches the maximum sequencer drift, and L1 has no next origin yet")]
    MaxDriftReached,
    /// L1 reorged out the parent's L1 origin.
    #[error("L1 reorged out the L1 origin {0:?}")]
    Reorged(BlockNumHash),
    /// The origin actor stopped.
    #[error("the origin actor stopped")]
    Stopped,
}

/// Selects the L1 origin of the sequencer's next L2 block.
#[derive(Debug, Clone)]
pub struct Handle {
    config: Arc<RollupConfig>,
    window: watch::Receiver<Window>,
}

impl Handle {
    /// Creates a handle that selects origins under `config` from `window`.
    pub(super) const fn new(config: Arc<RollupConfig>, window: watch::Receiver<Window>) -> Self {
        Self { config, window }
    }

    /// Returns the L1 origin of the child of `parent`.
    ///
    /// The child stays on its parent's origin until its timestamp reaches the next L1 block, and
    /// moves to the next L1 block then, once L1 has it confirmed. Past the sequencer drift, it must
    /// move to the next L1 block if it can.
    ///
    /// This answers at once while the origin actor holds the parent's origin, as it does in normal
    /// operation, and waits for the actor otherwise, such as right after an engine reset.
    pub async fn next(&self, parent: L2BlockInfo) -> Result<BlockInfo, OriginError> {
        let mut window = self.window.clone();
        loop {
            let origins = window.borrow_and_update().origins(parent.l1_origin);
            match origins {
                Some(Origins::Held { current, next }) => {
                    return select(&self.config, parent, current, next);
                }
                Some(Origins::Reorged) => return Err(OriginError::Reorged(parent.l1_origin)),
                None => window.changed().await.map_err(|_| OriginError::Stopped)?,
            }
        }
    }
}

/// Selects the L1 origin of the child of `parent`, whose origin is `current`, given the next L1
/// block, if L1 has it confirmed.
fn select(
    config: &RollupConfig,
    parent: L2BlockInfo,
    current: BlockInfo,
    next: Option<BlockInfo>,
) -> Result<BlockInfo, OriginError> {
    let timestamp = parent.block_info.timestamp + config.block_time;

    // Move to the next L1 origin once the child's timestamp reaches it.
    if let Some(next) = next &&
        timestamp >= next.timestamp
    {
        return Ok(next);
    }

    let max_seq_drift = config.max_sequencer_drift(current.timestamp);
    if timestamp - current.timestamp <= max_seq_drift {
        return Ok(current);
    }

    warn!(
        target: "origin",
        current_origin_time = current.timestamp,
        unsafe_head_time = parent.block_info.timestamp,
        max_seq_drift,
        "Next L2 block time is past the sequencer drift"
    );

    // The next L1 origin is after the child's timestamp, so the child cannot move to it, despite
    // the drift.
    if next.is_some_and(|next| timestamp < next.timestamp) {
        return Ok(current);
    }

    next.ok_or(OriginError::MaxDriftReached)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::block;
    use rstest::rstest;
    use std::{collections::VecDeque, time::Duration};
    use tokio::time::timeout;

    /// The L1 slot time of the test L1 blocks.
    const L1_SLOT_TIME: u64 = 12;

    /// An L2 block time of 2 seconds, and a sequencer drift of 600 seconds.
    fn config() -> Arc<RollupConfig> {
        Arc::new(RollupConfig { block_time: 2, max_sequencer_drift: 600, ..Default::default() })
    }

    /// Returns an L2 block with the given timestamp, built on the given L1 origin.
    fn parent(timestamp: u64, l1_origin: BlockInfo) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { timestamp, ..Default::default() },
            l1_origin: l1_origin.id(),
            seq_num: 0,
        }
    }

    /// Returns a handle answering from a window of `blocks`, and the window's publisher.
    fn handle(blocks: Vec<BlockInfo>) -> (Handle, watch::Sender<Window>) {
        let (window, receiver) =
            watch::channel(Window { blocks: VecDeque::from(blocks), reorged: None });
        (Handle::new(config(), receiver), window)
    }

    #[tokio::test]
    #[rstest]
    #[case::single_epoch(1)]
    #[case::many_epochs(12)]
    async fn moves_to_each_l1_block_as_l2_time_reaches_it(#[case] epochs: u64) {
        let config = config();
        // One more L1 block than epochs, such that the next origin is always held.
        let (handle, _window) = handle((0..=epochs).map(|number| block(number, 1, 1)).collect());
        for number in 0..(epochs * (L1_SLOT_TIME / config.block_time)) {
            let epoch = (number * config.block_time) / L1_SLOT_TIME;
            let parent = parent(number * config.block_time, block(epoch, 1, 1));
            let expected = ((number + 1) * config.block_time) / L1_SLOT_TIME;
            assert_eq!(handle.next(parent).await.unwrap(), block(expected, 1, 1));
        }
    }

    #[tokio::test]
    #[rstest]
    #[case::not_held(false)]
    #[case::held(true)]
    async fn stays_on_the_origin_until_l1_has_the_next(#[case] next_held: bool) {
        let blocks = (0..=u64::from(next_held)).map(|number| block(number, 1, 1)).collect();
        let (handle, _window) = handle(blocks);
        // The child's timestamp is that of L1 block 1.
        let origin = handle.next(parent(10, block(0, 1, 1))).await.unwrap();
        assert_eq!(origin, block(u64::from(next_held), 1, 1));
    }

    #[tokio::test]
    #[rstest]
    #[case::next_not_held(false, false)]
    #[case::next_held_and_reached(true, false)]
    #[case::next_held_and_ahead(true, true)]
    async fn past_the_drift_moves_to_the_next_origin_if_it_can(
        #[case] next_held: bool,
        #[case] next_ahead: bool,
    ) {
        let config = config();
        let mut blocks = vec![block(0, 1, 1)];
        let timestamp =
            if next_ahead { config.max_sequencer_drift + config.block_time * 2 } else { 2 };
        let next = BlockInfo { timestamp, ..block(1, 1, 1) };
        if next_held {
            blocks.push(next);
        }
        let (handle, _window) = handle(blocks);
        let origin = handle.next(parent(config.max_sequencer_drift, block(0, 1, 1))).await;
        match (next_held, next_ahead) {
            (false, _) => assert_eq!(origin, Err(OriginError::MaxDriftReached)),
            // The next origin is after the child's timestamp, so the child stays on its origin.
            (true, true) => assert_eq!(origin, Ok(block(0, 1, 1))),
            (true, false) => assert_eq!(origin, Ok(next)),
        }
    }

    #[tokio::test]
    async fn reports_a_reorged_out_origin() {
        let (handle, _window) = handle(vec![block(0, 1, 1), block(1, 1, 1)]);
        let origin = block(1, 2, 1);
        assert_eq!(handle.next(parent(12, origin)).await, Err(OriginError::Reorged(origin.id())));
    }

    #[tokio::test(start_paused = true)]
    async fn waits_until_the_window_holds_the_origin() {
        let (handle, window) = handle(vec![block(0, 1, 1)]);
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next(parent(36, block(3, 1, 1))).await }
        });
        assert!(
            timeout(Duration::from_secs(1), handle.next(parent(36, block(3, 1, 1)))).await.is_err()
        );
        window.send_replace(Window {
            blocks: VecDeque::from([block(3, 1, 1), block(4, 1, 1)]),
            reorged: None,
        });
        assert_eq!(waiting.await.unwrap(), Ok(block(3, 1, 1)));

        drop(window);
        assert_eq!(handle.next(parent(0, block(9, 1, 1))).await, Err(OriginError::Stopped));
    }
}
