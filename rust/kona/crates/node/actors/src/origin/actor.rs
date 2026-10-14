//! The origin actor.

use super::window::{Origins, Window};
use crate::{
    engine::EngineState,
    l1::{self, Child},
};
use alloy_eips::BlockNumHash;
use kona_protocol::BlockInfo;
use std::collections::VecDeque;
use thiserror::Error;
use tokio::sync::watch;

/// How many L1 blocks after the origin the window holds at most. The sequencer moves to the next
/// origin at most once per L2 block, so a few blocks keep the next origin ready even while the
/// actor catches up with the engine.
const LOOKAHEAD: usize = 4;

/// An error that stops the origin actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// The L1 actor stopped.
    #[error("the L1 actor stopped")]
    L1Stopped,
    /// The engine's state publisher closed.
    #[error("engine state channel closed")]
    EngineStateClosed,
}

impl From<l1::HandleError> for ActorError {
    fn from(_: l1::HandleError) -> Self {
        Self::L1Stopped
    }
}

/// Keeps the window of L1 blocks the sequencer may build on next.
#[derive(Debug)]
pub(super) struct Actor {
    pub(super) l1: l1::Handle,
    pub(super) engine_state: watch::Receiver<EngineState>,
    /// How many blocks below the L1 head a block must be to be the next origin.
    pub(super) confirmations: u64,
    pub(super) window: watch::Sender<Window>,
}

impl Actor {
    /// Updates the window whenever the engine's unsafe head or the L1 head changes.
    pub(super) async fn run(mut self) -> Result<(), ActorError> {
        let mut l1_head = BlockInfo::default();
        loop {
            let origin = self.engine_state.borrow_and_update().unsafe_head().l1_origin;
            // Before its first reset, the engine has no heads.
            if origin != BlockNumHash::default() {
                self.update(origin).await?;
            }
            tokio::select! {
                changed = self.engine_state.changed() => {
                    changed.map_err(|_| ActorError::EngineStateClosed)?;
                }
                head = self.l1.next_head(l1_head) => l1_head = head?,
            }
        }
    }

    /// Makes the window start at `origin`, the L1 origin of the engine's unsafe head, and hold the
    /// canonical, confirmed blocks after it.
    async fn update(&self, origin: BlockNumHash) -> Result<(), ActorError> {
        let window = self.window.borrow().clone();
        // Keep the blocks from the origin that L1 has finalized, which cannot reorg.
        let finalized = self.l1.status()?.finalized.number;
        let mut blocks: VecDeque<BlockInfo> = window
            .blocks
            .iter()
            .copied()
            .skip_while(|block| block.number < origin.number)
            .take_while(|block| block.number < finalized)
            .collect();
        if blocks.front().map(BlockInfo::id) != Some(origin) {
            blocks.clear();
            match self.first(origin, &window).await? {
                Some(first) => blocks.push_back(first),
                None => {
                    warn!(target: "origin", ?origin, "L1 reorged out the L1 origin of the unsafe head");
                    self.publish(Window { blocks, reorged: Some(origin) });
                    return Ok(());
                }
            }
        }
        while blocks.len() <= LOOKAHEAD {
            let last = blocks[blocks.len() - 1];
            match self.l1.try_child_of(last.id(), self.confirmations).await? {
                Some(Child::Block(child)) => blocks.push_back(child),
                // L1 changed under the window; the next update follows it.
                Some(Child::Reorged) | None => break,
            }
        }
        self.publish(Window { blocks, reorged: None });
        Ok(())
    }

    /// Returns the L1 block `origin`, or `None` if L1 reorged it out.
    async fn first(
        &self,
        origin: BlockNumHash,
        window: &Window,
    ) -> Result<Option<BlockInfo>, ActorError> {
        match self.l1.canonical(origin.number).await? {
            Some(canonical) => Ok((canonical.hash == origin.hash).then_some(canonical)),
            // After the L1 head, where it may yet be canonical. The unsafe chain already builds on
            // it, so the sequencer may too.
            None => match window.origins(origin) {
                Some(Origins::Held { current, .. }) => Ok(Some(current)),
                _ => Ok(self.l1.block(origin).await?),
            },
        }
    }

    fn publish(&self, window: Window) {
        self.window.send_if_modified(|published| {
            if *published == window {
                return false;
            }
            *published = window;
            true
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        engine::HeadsUpdate,
        l1::test_utils::{TestChain, block, chain, child, genesis, rpc_block},
    };
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::mock::Asserter;
    use kona_protocol::L2BlockInfo;
    use std::time::Duration;
    use tokio::{task::JoinHandle, time};

    /// Returns an engine state whose unsafe head is built on `l1_origin`.
    fn engine_state(l1_origin: BlockInfo) -> EngineState {
        let unsafe_head = L2BlockInfo { l1_origin: l1_origin.id(), ..Default::default() };
        EngineState::default()
            .apply_update(HeadsUpdate { unsafe_head: Some(unsafe_head), ..Default::default() })
    }

    /// A running origin actor, and what it reads and publishes.
    struct Running {
        engine_state: watch::Sender<EngineState>,
        window: watch::Receiver<Window>,
        actor: JoinHandle<Result<(), ActorError>>,
    }

    impl Running {
        /// Runs an actor on `l1` that confirms blocks two blocks below the L1 head.
        fn new(l1: l1::Handle, state: EngineState) -> Self {
            let (engine_state, engine_state_rx) = watch::channel(state);
            let (window_tx, window) = watch::channel(Window::default());
            let actor =
                Actor { l1, engine_state: engine_state_rx, confirmations: 2, window: window_tx };
            Self { engine_state, window, actor: tokio::spawn(actor.run()) }
        }

        /// Lets the actor handle what changed, and returns the blocks of its window.
        async fn blocks(&self) -> Vec<BlockInfo> {
            time::sleep(Duration::from_secs(1)).await;
            self.window.borrow().blocks.iter().copied().collect()
        }
    }

    #[tokio::test(start_paused = true)]
    async fn holds_the_origin_and_the_confirmed_blocks_after_it() {
        let l1 = TestChain::new(&chain(0..=6, 1));
        let running = Running::new(l1.handle(), engine_state(block(2, 1, 1)));
        assert_eq!(running.blocks().await, chain(2..=4, 1));

        // L1 grows, and confirms more blocks, up to the lookahead.
        l1.set_head(&chain(7..=12, 1));
        assert_eq!(running.blocks().await, chain(2..=6, 1));

        // The engine's unsafe head moves to the next origin.
        running.engine_state.send_replace(engine_state(block(3, 1, 1)));
        assert_eq!(running.blocks().await, chain(3..=7, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn follows_l1_reorgs() {
        let l1 = TestChain::new(&chain(0..=6, 1));
        let running = Running::new(l1.handle(), engine_state(block(2, 1, 1)));
        assert_eq!(running.blocks().await, chain(2..=4, 1));

        // L1 replaces block 4 and the blocks after it.
        l1.set_head(&[block(4, 2, 1), block(5, 2, 2), block(6, 2, 2)]);
        assert_eq!(running.blocks().await, vec![block(2, 1, 1), block(3, 1, 1), block(4, 2, 1)]);

        // L1 replaces the origin.
        l1.set_head(&[block(2, 3, 1), block(3, 3, 3), block(4, 3, 3), block(5, 3, 3)]);
        assert_eq!(running.blocks().await, vec![]);
        assert_eq!(running.window.borrow().reorged, Some(block(2, 1, 1).id()));
    }

    #[tokio::test(start_paused = true)]
    async fn reads_an_origin_after_the_l1_head_from_l1() {
        let blocks: Vec<_> =
            std::iter::successors(Some(genesis()), |block| Some(child(block, 1))).take(7).collect();
        let l1 = TestChain::new(&blocks[..2]);
        let responses = Asserter::new();
        responses.push_success(&rpc_block(&blocks[3]));
        let history = RootProvider::new(RpcClient::mocked(responses.clone()));
        let running = Running::new(l1.handle_with_history(history), engine_state(blocks[3]));
        assert_eq!(running.blocks().await, vec![blocks[3]]);

        // L1 reaches the origin, and confirms the block after it.
        l1.set_head(&blocks[2..]);
        assert_eq!(running.blocks().await, blocks[3..5].to_vec());
    }

    #[tokio::test(start_paused = true)]
    async fn waits_for_the_engine_to_have_heads() {
        let l1 = TestChain::new(&chain(0..=6, 1));
        let running = Running::new(l1.handle(), EngineState::default());
        assert_eq!(running.blocks().await, vec![]);
        assert_eq!(*running.window.borrow(), Window::default());
    }

    #[tokio::test(start_paused = true)]
    async fn stops_when_l1_or_the_engine_stops() {
        let l1 = TestChain::new(&chain(0..=6, 1));
        let running = Running::new(l1.handle(), engine_state(block(2, 1, 1)));
        running.blocks().await;
        drop(l1);
        assert!(matches!(running.actor.await.unwrap(), Err(ActorError::L1Stopped)));

        let l1 = TestChain::new(&chain(0..=6, 1));
        let running = Running::new(l1.handle(), engine_state(block(2, 1, 1)));
        running.blocks().await;
        drop(running.engine_state);
        assert!(matches!(running.actor.await.unwrap(), Err(ActorError::EngineStateClosed)));
    }
}
