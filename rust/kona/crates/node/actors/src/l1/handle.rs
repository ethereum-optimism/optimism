//! Questions about L1, answered from the actor's chain.

use super::{
    chain::{Chain, Height},
    rpc::block_info,
};
use alloy_eips::BlockNumHash;
use alloy_provider::{Provider, RootProvider};
use alloy_transport::{TransportError, TransportErrorKind};
use kona_protocol::BlockInfo;
use thiserror::Error;
use tokio::sync::watch;

/// What follows a block on L1.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Child {
    /// The block is canonical, and this is its canonical child.
    Block(BlockInfo),
    /// The block is not canonical: it was reorged out.
    Reorged,
}

/// The latest L1 blocks the actor observed.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Status {
    /// The canonical head.
    pub head: BlockInfo,
    /// The safe block.
    pub safe: BlockInfo,
    /// The finalized block.
    pub finalized: BlockInfo,
}

/// A failed question to the L1 actor.
#[derive(Debug, Error)]
pub enum HandleError {
    /// The L1 actor stopped.
    #[error("the L1 actor stopped")]
    Stopped,
    /// A block before the finalized block could not be read from L1.
    #[error("failed to read a finalized L1 block")]
    Rpc(#[from] TransportError),
}

/// Answers questions about L1 from the actor's chain.
///
/// The actor holds the canonical chain from the latest finalized block to the head. Blocks before
/// the finalized block cannot reorg, so the handle reads them from L1 by number. Methods that wait
/// return when they can answer, and cancelling them loses nothing.
#[derive(Debug, Clone)]
pub struct Handle {
    chain: watch::Receiver<Chain>,
    history: RootProvider,
}

impl Handle {
    /// Creates a handle that answers from `chain`, and reads blocks before the finalized block from
    /// `history`.
    pub(super) const fn new(chain: watch::Receiver<Chain>, history: RootProvider) -> Self {
        Self { chain, history }
    }

    /// Returns the latest L1 blocks the actor observed.
    pub fn status(&self) -> Result<Status, HandleError> {
        let chain = self.chain()?;
        Ok(Status { head: chain.head(), safe: chain.safe(), finalized: chain.finalized() })
    }

    /// Waits until the canonical head is a block other than `seen`, and returns it.
    pub async fn next_head(&self, seen: BlockInfo) -> Result<BlockInfo, HandleError> {
        self.wait_for(|chain| Some(chain.head()).filter(|head| *head != seen)).await
    }

    /// Waits until the finalized block is a block other than `seen`, and returns it.
    pub async fn next_finalized(&self, seen: BlockInfo) -> Result<BlockInfo, HandleError> {
        self.wait_for(|chain| Some(chain.finalized()).filter(|finalized| *finalized != seen)).await
    }

    /// Returns the canonical block at `number`, or `None` if it is after the head.
    pub async fn canonical(&self, number: u64) -> Result<Option<BlockInfo>, HandleError> {
        let height = self.chain()?.at(number);
        match height {
            Height::Final => self.final_block(number).await.map(Some),
            Height::Held(block) => Ok(Some(block)),
            Height::Ahead => Ok(None),
        }
    }

    /// Waits until L1 has a canonical child of `block`, or `block` is reorged out, and returns
    /// which.
    pub async fn child_of(&self, block: BlockNumHash) -> Result<Child, HandleError> {
        let mut chain = self.chain.clone();
        loop {
            // Mark the current chain seen before answering from it, so that a change while
            // answering ends the wait below.
            chain.mark_unchanged();
            if let Some(child) = self.try_child_of(block, 0).await? {
                return Ok(child);
            }
            chain.changed().await.map_err(|_| HandleError::Stopped)?;
        }
    }

    /// Returns the canonical child of `block` if L1 has one with at least `confirmations` blocks
    /// after it, [`Child::Reorged`] if `block` is reorged out, or `None` if L1 cannot tell yet.
    pub async fn try_child_of(
        &self,
        block: BlockNumHash,
        confirmations: u64,
    ) -> Result<Option<Child>, HandleError> {
        let (at_block, at_child, head) = {
            let chain = self.chain()?;
            (chain.at(block.number), chain.at(block.number + 1), chain.head())
        };
        let canonical = match at_block {
            Height::Final => self.final_block(block.number).await?,
            Height::Held(canonical) => canonical,
            Height::Ahead => return Ok(None),
        };
        if canonical.hash != block.hash {
            return Ok(Some(Child::Reorged));
        }
        let child = match at_child {
            Height::Final => self.final_block(block.number + 1).await?,
            Height::Held(child) => child,
            Height::Ahead => return Ok(None),
        };
        Ok((child.number + confirmations <= head.number).then_some(Child::Block(child)))
    }

    /// Returns the actor's chain, or an error if the actor stopped.
    fn chain(&self) -> Result<watch::Ref<'_, Chain>, HandleError> {
        self.chain.has_changed().map_err(|_| HandleError::Stopped)?;
        Ok(self.chain.borrow())
    }

    /// Waits until `answer` answers from the actor's chain, and returns its answer.
    async fn wait_for<T>(&self, answer: impl Fn(&Chain) -> Option<T>) -> Result<T, HandleError> {
        let mut chain = self.chain.clone();
        loop {
            let answered = answer(&chain.borrow_and_update());
            if let Some(answer) = answered {
                return Ok(answer);
            }
            chain.changed().await.map_err(|_| HandleError::Stopped)?;
        }
    }

    /// Reads the canonical block at `number`, before the finalized block, from L1.
    async fn final_block(&self, number: u64) -> Result<BlockInfo, HandleError> {
        let block = self.history.get_block_by_number(number.into()).await?.ok_or_else(|| {
            TransportErrorKind::custom_str(&format!("L1 has no finalized block {number}"))
        })?;
        Ok(block_info(&block))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::{TestChain, block, chain as blocks};
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::Block;
    use alloy_transport::mock::Asserter;
    use std::time::Duration;
    use tokio::time::timeout;

    const WAIT: Duration = Duration::from_secs(1);

    /// Returns a provider that answers with `blocks`, in order, and the RPC block for `info`.
    fn history(blocks: &[BlockInfo]) -> RootProvider {
        let responses = Asserter::new();
        for info in blocks {
            let mut block: Block = Block::default();
            block.header.hash = info.hash;
            block.header.inner.number = info.number;
            block.header.inner.parent_hash = info.parent_hash;
            block.header.inner.timestamp = info.timestamp;
            responses.push_success(&block);
        }
        RootProvider::new(RpcClient::mocked(responses))
    }

    #[tokio::test(start_paused = true)]
    async fn answers_from_the_held_chain() {
        let chain = TestChain::new(&blocks(3..=6, 1));
        let handle = chain.handle();
        assert_eq!(
            handle.status().unwrap(),
            Status { head: block(6, 1, 1), safe: block(3, 1, 1), finalized: block(3, 1, 1) }
        );
        assert_eq!(handle.canonical(4).await.unwrap(), Some(block(4, 1, 1)));
        assert_eq!(handle.canonical(7).await.unwrap(), None);
        assert_eq!(
            handle.try_child_of(block(4, 1, 1).id(), 0).await.unwrap(),
            Some(Child::Block(block(5, 1, 1)))
        );
        // A block that is not canonical, at or after the finalized block.
        assert_eq!(
            handle.try_child_of(block(5, 2, 1).id(), 0).await.unwrap(),
            Some(Child::Reorged)
        );
        assert_eq!(
            handle.try_child_of(block(3, 2, 2).id(), 0).await.unwrap(),
            Some(Child::Reorged)
        );
        // Nothing after the head, nor after a block after the head.
        assert_eq!(handle.try_child_of(block(6, 1, 1).id(), 0).await.unwrap(), None);
        assert_eq!(handle.try_child_of(block(9, 2, 2).id(), 0).await.unwrap(), None);
    }

    #[tokio::test(start_paused = true)]
    async fn a_child_needs_its_confirmations() {
        let chain = TestChain::new(&blocks(3..=6, 1));
        let handle = chain.handle();
        assert_eq!(
            handle.try_child_of(block(3, 1, 1).id(), 2).await.unwrap(),
            Some(Child::Block(block(4, 1, 1)))
        );
        assert_eq!(handle.try_child_of(block(4, 1, 1).id(), 2).await.unwrap(), None);
    }

    #[tokio::test(start_paused = true)]
    async fn reads_blocks_before_the_finalized_block_from_l1() {
        let chain = TestChain::new(&blocks(10..=12, 1));
        let handle = chain.handle_with_history(history(&[
            block(4, 1, 1),
            block(4, 1, 1),
            block(5, 1, 1),
            block(4, 1, 1),
        ]));
        assert_eq!(handle.canonical(4).await.unwrap(), Some(block(4, 1, 1)));
        assert_eq!(
            handle.try_child_of(block(4, 1, 1).id(), 0).await.unwrap(),
            Some(Child::Block(block(5, 1, 1)))
        );
        // A block that lost to a block L1 then finalized.
        assert_eq!(
            handle.try_child_of(block(4, 2, 2).id(), 0).await.unwrap(),
            Some(Child::Reorged)
        );
        // The child of the block before the finalized block is the finalized block.
        let handle = chain.handle_with_history(history(&[block(9, 1, 1)]));
        assert_eq!(
            handle.try_child_of(block(9, 1, 1).id(), 0).await.unwrap(),
            Some(Child::Block(block(10, 1, 1)))
        );
    }

    #[tokio::test(start_paused = true)]
    async fn fails_when_l1_lacks_a_finalized_block() {
        let chain = TestChain::new(&blocks(10..=12, 1));
        let responses = Asserter::new();
        responses.push_success(&Option::<Block>::None);
        responses.push_failure_msg("L1 unavailable");
        let handle = chain.handle_with_history(RootProvider::new(RpcClient::mocked(responses)));
        assert!(matches!(handle.canonical(4).await, Err(HandleError::Rpc(_))));
        assert!(matches!(handle.canonical(4).await, Err(HandleError::Rpc(_))));
    }

    #[tokio::test(start_paused = true)]
    async fn waits_for_a_child_or_a_reorg() {
        let chain = TestChain::new(&blocks(3..=5, 1));
        let handle = chain.handle();
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.child_of(block(5, 1, 1).id()).await.unwrap() }
        });
        assert!(timeout(WAIT, handle.child_of(block(5, 1, 1).id())).await.is_err());
        chain.set_head(&[block(6, 1, 1)]);
        assert_eq!(waiting.await.unwrap(), Child::Block(block(6, 1, 1)));

        // A block after the head waits until the head reaches it.
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.child_of(block(7, 2, 2).id()).await.unwrap() }
        });
        chain.set_head(&[block(7, 1, 1)]);
        assert_eq!(waiting.await.unwrap(), Child::Reorged);
    }

    #[tokio::test(start_paused = true)]
    async fn waits_for_a_new_head_or_finalized_block() {
        let chain = TestChain::new(&blocks(3..=5, 1));
        let handle = chain.handle();
        assert_eq!(handle.next_head(BlockInfo::default()).await.unwrap(), block(5, 1, 1));
        assert_eq!(handle.next_finalized(BlockInfo::default()).await.unwrap(), block(3, 1, 1));
        assert!(timeout(WAIT, handle.next_head(block(5, 1, 1))).await.is_err());

        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_head(block(5, 1, 1)).await.unwrap() }
        });
        // A head that moves back is a new head too.
        chain.set_head(&[block(4, 1, 1)]);
        assert_eq!(next.await.unwrap(), block(4, 1, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn fails_once_the_actor_stops() {
        let chain = TestChain::new(&blocks(3..=5, 1));
        let handle = chain.handle();
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.child_of(block(5, 1, 1).id()).await }
        });
        tokio::task::yield_now().await;
        drop(chain);
        assert!(matches!(waiting.await.unwrap(), Err(HandleError::Stopped)));
        assert!(matches!(handle.status(), Err(HandleError::Stopped)));
        assert!(matches!(
            handle.try_child_of(block(4, 1, 1).id(), 0).await,
            Err(HandleError::Stopped)
        ));
        assert!(matches!(handle.next_head(block(5, 1, 1)).await, Err(HandleError::Stopped)));
    }
}
