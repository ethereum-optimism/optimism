//! Questions about L1, answered from the actor's chain.

use super::{
    chain::{Chain, Height},
    rpc::block_info,
};
use alloy_eips::BlockNumHash;
use kona_protocol::BlockInfo;
use kona_sources::RetryingRpc;
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

/// The L1 actor stopped.
#[derive(Debug, Clone, Copy, Error)]
#[error("the L1 actor stopped")]
pub struct HandleError;

/// Answers questions about L1 from the actor's chain.
///
/// The actor holds the canonical chain from the latest finalized block to the head. Blocks before
/// the finalized block cannot reorg, so the handle reads them from L1 by number, waiting for as
/// long as L1 cannot be read. Methods that wait return when they can answer, and cancelling them
/// loses nothing.
#[derive(Debug, Clone)]
pub struct Handle {
    chain: watch::Receiver<Chain>,
    history: RetryingRpc,
}

impl Handle {
    /// Creates a handle that answers from `chain`, and reads blocks before the finalized block from
    /// `history`.
    pub(super) const fn new(chain: watch::Receiver<Chain>, history: RetryingRpc) -> Self {
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
    ///
    /// While the actor is linking a head it polled, a height after the chain's head waits until it
    /// has: L1 may already have a block there.
    pub async fn canonical(&self, number: u64) -> Result<Option<BlockInfo>, HandleError> {
        let height = self.settled(number).await?;
        match height {
            Height::Final => Ok(Some(self.final_block(number).await)),
            Height::Held(block) => Ok(Some(block)),
            Height::Ahead => Ok(None),
        }
    }

    /// Returns the L1 block `block`, or `None` if L1 does not have it. A block the chain does not
    /// hold, such as one after the head, is read from L1 by hash, and may not be canonical.
    ///
    /// While the actor is linking a head it polled, a block after the chain's head waits until it
    /// has.
    pub async fn block(&self, block: BlockNumHash) -> Result<Option<BlockInfo>, HandleError> {
        let height = self.settled(block.number).await?;
        if let Height::Held(held) = height &&
            held.hash == block.hash
        {
            return Ok(Some(held));
        }
        let read = self.history.header_by_hash(block.hash).await.map(|header| block_info(&header));
        Ok(read.filter(|read| read.number == block.number))
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
            chain.changed().await.map_err(|_| HandleError)?;
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
            Height::Final => self.final_block(block.number).await,
            Height::Held(canonical) => canonical,
            Height::Ahead => return Ok(None),
        };
        if canonical.hash != block.hash {
            return Ok(Some(Child::Reorged));
        }
        let child = match at_child {
            Height::Final => self.final_block(block.number + 1).await,
            Height::Held(child) => child,
            Height::Ahead => return Ok(None),
        };
        Ok((child.number + confirmations <= head.number).then_some(Child::Block(child)))
    }

    /// Returns the actor's chain, or an error if the actor stopped.
    fn chain(&self) -> Result<watch::Ref<'_, Chain>, HandleError> {
        self.chain.has_changed().map_err(|_| HandleError)?;
        Ok(self.chain.borrow())
    }

    /// Returns where `number` falls relative to the chain, waiting while it is after the head and
    /// the actor is linking a head it polled.
    async fn settled(&self, number: u64) -> Result<Height, HandleError> {
        self.chain()?;
        self.wait_for(|chain| {
            let height = chain.at(number);
            (height != Height::Ahead || !chain.linking()).then_some(height)
        })
        .await
    }

    /// Waits until `answer` answers from the actor's chain, and returns its answer.
    async fn wait_for<T>(&self, answer: impl Fn(&Chain) -> Option<T>) -> Result<T, HandleError> {
        let mut chain = self.chain.clone();
        loop {
            let answered = answer(&chain.borrow_and_update());
            if let Some(answer) = answered {
                return Ok(answer);
            }
            chain.changed().await.map_err(|_| HandleError)?;
        }
    }

    /// Reads the canonical block at `number`, before the finalized block, from L1.
    async fn final_block(&self, number: u64) -> BlockInfo {
        block_info(&self.history.header_at(number).await)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::{
        TestChain, block, chain as blocks, child, descendants, genesis, rpc_block,
    };
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::Block;
    use alloy_transport::mock::Asserter;
    use std::time::Duration;
    use tokio::time::timeout;

    const WAIT: Duration = Duration::from_secs(1);

    /// Returns the blocks of an L1 chain from genesis, which an RPC can serve, and a function
    /// returning block `number` of them.
    fn l1(count: usize) -> (Vec<BlockInfo>, impl Fn(u64) -> BlockInfo) {
        let blocks = std::iter::once(genesis()).chain(descendants(&genesis(), count, 1));
        let blocks: Vec<_> = blocks.collect();
        (blocks.clone(), move |number| blocks[number as usize])
    }

    /// Returns a handle to `chain` that reads L1 from `responses`.
    fn reading(chain: &TestChain, responses: &Asserter) -> Handle {
        chain.handle_with_history(RootProvider::new(RpcClient::mocked(responses.clone())))
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
        let (blocks, block) = l1(12);
        let chain = TestChain::new(&blocks[10..]);
        let responses = Asserter::new();
        let handle = reading(&chain, &responses);
        responses.push_success(&rpc_block(&block(4)));
        assert_eq!(handle.canonical(4).await.unwrap(), Some(block(4)));
        responses.push_success(&rpc_block(&block(4)));
        responses.push_success(&rpc_block(&block(5)));
        assert_eq!(
            handle.try_child_of(block(4).id(), 0).await.unwrap(),
            Some(Child::Block(block(5)))
        );
        // A block that lost to a block L1 then finalized.
        responses.push_success(&rpc_block(&block(4)));
        let lost = child(&block(3), 2);
        assert_eq!(handle.try_child_of(lost.id(), 0).await.unwrap(), Some(Child::Reorged));
        // The child of the block before the finalized block is the finalized block.
        responses.push_success(&rpc_block(&block(9)));
        assert_eq!(
            handle.try_child_of(block(9).id(), 0).await.unwrap(),
            Some(Child::Block(block(10)))
        );
    }

    #[tokio::test(start_paused = true)]
    async fn returns_a_block_from_the_chain_or_else_from_l1_by_hash() {
        let (blocks, block) = l1(12);
        let chain = TestChain::new(&blocks[10..]);
        let responses = Asserter::new();
        let handle = reading(&chain, &responses);
        assert_eq!(handle.block(block(11).id()).await.unwrap(), Some(block(11)));
        // A block after the head.
        let ahead = child(&block(12), 1);
        responses.push_success(&rpc_block(&ahead));
        assert_eq!(handle.block(ahead.id()).await.unwrap(), Some(ahead));
        // A block L1 does not have.
        responses.push_success(&Option::<Block>::None);
        assert_eq!(handle.block(child(&block(11), 2).id()).await.unwrap(), None);
    }

    #[tokio::test(start_paused = true)]
    async fn waits_for_l1_to_serve_a_block_before_the_finalized_block() {
        let (blocks, block) = l1(12);
        let chain = TestChain::new(&blocks[10..]);
        let responses = Asserter::new();
        let handle = reading(&chain, &responses);
        // L1 is unreachable, then does not have the block, then serves a block that is not it.
        responses.push_failure_msg("L1 unavailable");
        responses.push_success(&Option::<Block>::None);
        responses.push_success(&rpc_block(&block(5)));
        responses.push_success(&rpc_block(&block(4)));
        assert_eq!(handle.canonical(4).await.unwrap(), Some(block(4)));
    }

    #[tokio::test(start_paused = true)]
    async fn waits_to_answer_after_the_head_while_the_actor_links_a_head() {
        let chain = TestChain::new(&blocks(3..=5, 1));
        chain.set_linking(true);
        let handle = chain.handle();
        // Heights the chain holds answer at once.
        assert_eq!(handle.canonical(4).await.unwrap(), Some(block(4, 1, 1)));
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.canonical(7).await.unwrap() }
        });
        assert!(timeout(WAIT, handle.block(block(7, 1, 1).id())).await.is_err());
        chain.set_head(&blocks(6..=8, 1));
        chain.set_linking(false);
        assert_eq!(waiting.await.unwrap(), Some(block(7, 1, 1)));
        // Once linked, a height after the head has no block yet.
        assert_eq!(handle.canonical(9).await.unwrap(), None);
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
        assert!(matches!(waiting.await.unwrap(), Err(HandleError)));
        assert!(matches!(handle.status(), Err(HandleError)));
        assert!(matches!(handle.try_child_of(block(4, 1, 1).id(), 0).await, Err(HandleError)));
        assert!(matches!(handle.next_head(block(5, 1, 1)).await, Err(HandleError)));
    }
}
