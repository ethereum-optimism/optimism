//! Tracks the canonical L1 chain.
//!
//! The actor polls the L1 head, safe and finalized blocks, and holds the canonical chain from
//! about the finalized block to the head. Each consumer keeps a cursor, the last L1 block it
//! processed, and asks the actor what changed since: the blocks that extend the cursor, or, after a
//! reorg, the common ancestor to roll back to. Every consumer sees the actor's one view of L1. The
//! actor never waits on a consumer, and answers requests while it fetches blocks from L1.

use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_provider::Provider;
use alloy_rpc_client::PollerBuilder;
use alloy_rpc_types_eth::Block;
use alloy_transport::TransportResult;
use futures::{Stream, StreamExt, future::BoxFuture};
use kona_protocol::BlockInfo;
use std::{
    collections::{HashMap, VecDeque},
    future::Future,
    num::NonZeroUsize,
    time::Duration,
};
use thiserror::Error;
use tokio::sync::{mpsc, oneshot};

pub use crate::capacity::{Capacity, InvalidCapacity};

/// What changed on L1 since a consumer's cursor.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Update {
    /// Canonical blocks that extend the cursor, oldest first.
    Advanced(Vec<BlockInfo>),
    /// The cursor was reorged out. The consumer rolls back to `ancestor`, the newest canonical
    /// block on the cursor's chain, then applies `blocks`, oldest first.
    Reorged {
        /// The newest block both on the cursor's chain and on the canonical chain.
        ancestor: BlockInfo,
        /// Canonical blocks that extend the ancestor, oldest first.
        blocks: Vec<BlockInfo>,
    },
    /// The cursor is older than `base`, the oldest block the actor holds, so the actor cannot
    /// tell whether it is canonical. The consumer catches up by number through `base`, checking
    /// parent hashes, and asks again from there.
    ///
    /// `base` is the finalized block once the actor has seen the chain advance past it. It is a
    /// later block after startup, after a reorg or gap deeper than [`Config::window`], and while
    /// finality stalls for longer than the window.
    Behind {
        /// The oldest block the actor holds.
        base: BlockInfo,
    },
    /// The cursor is on no chain the actor knows of, so the consumer resets.
    Unknown,
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

/// How the actor polls L1.
#[derive(Debug, Clone, Copy)]
pub struct Config {
    /// How often to poll the head and safe blocks.
    pub head_interval: Duration,
    /// How often to poll the finalized block.
    pub finalized_interval: Duration,
    /// The most canonical blocks the actor holds. While finality stalls, the oldest are dropped.
    /// A head further ahead than this starts the held chain over rather than fetching the blocks
    /// in between.
    pub window: NonZeroUsize,
}

/// Constructs the handle and the actor.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    messages: mpsc::Receiver<Message>,
}

impl Builder {
    /// Creates the builder with a request queue of the given capacity.
    pub fn new(capacity: Capacity) -> Self {
        let (messages_tx, messages) = mpsc::channel(capacity.get());
        Self { handle: Handle { messages: messages_tx }, messages }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled. Dropping the future stops the actor, and
    /// requests then fail.
    pub fn build<L1Provider: Provider + 'static>(
        self,
        provider: L1Provider,
        config: Config,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, messages } = self;
        Actor {
            provider,
            config,
            messages,
            window: Window::default(),
            safe: BlockInfo::default(),
            finalized: BlockInfo::default(),
            update_waiters: Vec::new(),
            head_waiters: Vec::new(),
            finalized_waiters: Vec::new(),
        }
        .run()
    }
}

/// Requests to the L1 actor.
#[derive(Debug, Clone)]
pub struct Handle {
    messages: mpsc::Sender<Message>,
}

/// The L1 actor stopped.
#[derive(Debug, Clone, Copy, Error)]
#[error("the L1 actor stopped")]
pub struct HandleError;

impl Handle {
    /// Waits until there is something to report after `cursor`, then returns it, with at most
    /// `max` blocks.
    pub async fn updates_since(
        &self,
        cursor: BlockInfo,
        max: NonZeroUsize,
    ) -> Result<Update, HandleError> {
        self.request(|reply| Message::UpdatesSince { cursor, max, reply }).await
    }

    /// Waits until the canonical head is a block other than `seen`, and returns it.
    pub async fn next_head(&self, seen: BlockInfo) -> Result<BlockInfo, HandleError> {
        self.request(|reply| Message::NextHead(ChangeWaiter { seen, reply })).await
    }

    /// Waits until the finalized block is a block other than `seen`, and returns it.
    pub async fn next_finalized(&self, seen: BlockInfo) -> Result<BlockInfo, HandleError> {
        self.request(|reply| Message::NextFinalized(ChangeWaiter { seen, reply })).await
    }

    /// Returns the latest L1 blocks the actor observed.
    pub async fn status(&self) -> Result<Status, HandleError> {
        self.request(Message::Status).await
    }

    async fn request<T>(
        &self,
        message: impl FnOnce(oneshot::Sender<T>) -> Message,
    ) -> Result<T, HandleError> {
        let (reply, response) = oneshot::channel();
        self.messages.send(message(reply)).await.map_err(|_| HandleError)?;
        response.await.map_err(|_| HandleError)
    }
}

#[cfg(test)]
impl Handle {
    /// Returns a handle whose actor reports `status`, and drops every other request.
    pub(crate) fn reporting(status: tokio::sync::watch::Receiver<Status>) -> Self {
        let (messages, mut requests) = mpsc::channel(1);
        tokio::spawn(async move {
            while let Some(message) = requests.recv().await {
                if let Message::Status(reply) = message {
                    let _ = reply.send(*status.borrow());
                }
            }
        });
        Self { messages }
    }
}

#[derive(Debug)]
enum Message {
    UpdatesSince { cursor: BlockInfo, max: NonZeroUsize, reply: oneshot::Sender<Update> },
    NextHead(ChangeWaiter),
    NextFinalized(ChangeWaiter),
    Status(oneshot::Sender<Status>),
}

/// A fatal error from the L1 actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// A block stream ended.
    #[error("block stream ended")]
    StreamEnded,
}

/// The canonical chain from about the finalized block to the head, and the blocks reorged out of
/// it.
#[derive(Debug, Default)]
struct Window {
    /// Canonical blocks, oldest first. Each block's parent is the block before it.
    blocks: VecDeque<BlockInfo>,
    /// Blocks reorged out of `blocks`, by hash, no older than its oldest block. A consumer only
    /// learns blocks from the actor, so its cursor can be traced through them to the canonical
    /// chain. A block can be both here and canonical again, after the head moves back and then
    /// forward over it; canonical blocks are searched first.
    orphans: HashMap<B256, BlockInfo>,
}

impl Window {
    fn head(&self) -> Option<BlockInfo> {
        self.blocks.back().copied()
    }

    /// Returns the index of the block, if it is canonical and held.
    fn index_of(&self, number: u64, hash: B256) -> Option<usize> {
        let index = usize::try_from(number.checked_sub(self.blocks.front()?.number)?).ok()?;
        (self.blocks.get(index)?.hash == hash).then_some(index)
    }

    /// Returns the index of the block's parent, if it is canonical and held.
    fn parent_index(&self, block: &BlockInfo) -> Option<usize> {
        self.index_of(block.number.checked_sub(1)?, block.parent_hash)
    }

    /// Returns what changed since `cursor`, or `None` if nothing did.
    fn updates_since(&self, cursor: BlockInfo, max: NonZeroUsize) -> Option<Update> {
        let base = *self.blocks.front()?;
        if cursor.number < base.number {
            return Some(Update::Behind { base });
        }
        let after = |index: usize| -> Vec<BlockInfo> {
            self.blocks.iter().skip(index + 1).take(max.get()).copied().collect()
        };
        if let Some(index) = self.index_of(cursor.number, cursor.hash) {
            let blocks = after(index);
            return (!blocks.is_empty()).then_some(Update::Advanced(blocks));
        }
        // Follow the cursor's chain back through the orphans until it rejoins the canonical chain.
        // Each step visits a different orphan, which bounds the walk.
        let mut block = cursor;
        for _ in 0..=self.orphans.len() {
            if let Some(index) = self.parent_index(&block) {
                return Some(Update::Reorged {
                    ancestor: self.blocks[index],
                    blocks: after(index),
                });
            }
            match self.orphans.get(&block.parent_hash) {
                Some(parent) => block = *parent,
                None => break,
            }
        }
        Some(Update::Unknown)
    }

    /// Keeps the blocks up to `index` and orphans the rest, returning how many it orphaned.
    fn truncate_after(&mut self, index: usize) -> usize {
        let orphaned = self.blocks.len().saturating_sub(index + 1);
        for block in self.blocks.drain(index + 1..) {
            self.orphans.insert(block.hash, block);
        }
        orphaned
    }

    /// Orphans every block, returning how many it orphaned.
    fn clear(&mut self) -> usize {
        let orphaned = self.blocks.len();
        for block in self.blocks.drain(..) {
            self.orphans.insert(block.hash, block);
        }
        orphaned
    }

    /// Appends blocks that extend the head, oldest first, keeping at most `max`.
    fn extend(&mut self, blocks: impl IntoIterator<Item = BlockInfo>, max: NonZeroUsize) {
        self.blocks.extend(blocks);
        while self.blocks.len() > max.get() {
            self.blocks.pop_front();
        }
        self.prune_orphans();
    }

    /// Drops the blocks older than the finalized block, which can no longer reorg. The head is
    /// kept even if the finalized block is ahead of it.
    fn prune(&mut self, finalized: BlockInfo) {
        while self.blocks.len() > 1 &&
            self.blocks.front().is_some_and(|block| block.number < finalized.number)
        {
            self.blocks.pop_front();
        }
        self.prune_orphans();
    }

    /// Drops the orphans older than the oldest block. A cursor that old is answered with
    /// [`Update::Behind`] before the orphans are searched.
    fn prune_orphans(&mut self) {
        if let Some(base) = self.blocks.front().map(|block| block.number) {
            self.orphans.retain(|_, block| block.number >= base);
        }
    }
}

/// A request waiting for the window to change.
#[derive(Debug)]
struct UpdateWaiter {
    cursor: BlockInfo,
    max: NonZeroUsize,
    reply: oneshot::Sender<Update>,
}

/// A request waiting for a block other than `seen`.
#[derive(Debug)]
struct ChangeWaiter {
    seen: BlockInfo,
    reply: oneshot::Sender<BlockInfo>,
}

/// Replies to the waiters whose block changed to `current`, and keeps the others.
fn answer_changes(waiters: &mut Vec<ChangeWaiter>, current: Option<BlockInfo>) {
    for waiter in std::mem::take(waiters) {
        match current {
            _ if waiter.reply.is_closed() => {}
            Some(current) if current != waiter.seen => {
                let _ = waiter.reply.send(current);
            }
            _ => waiters.push(waiter),
        }
    }
}

/// Parks a waiter, dropping the ones whose requester stopped waiting.
fn park<T>(waiters: &mut Vec<T>, waiter: T, abandoned: impl Fn(&T) -> bool) {
    waiters.retain(|waiter| !abandoned(waiter));
    waiters.push(waiter);
}

/// A walk back from a new head to the held chain, waiting for a parent block from L1.
struct Walk {
    /// The blocks between the new head and the held chain, newest first.
    branch: Vec<BlockInfo>,
    /// Fetches the parent of the oldest block in `branch`.
    parent: BoxFuture<'static, TransportResult<Option<Block>>>,
}

#[derive(Debug)]
struct Actor<L1Provider> {
    provider: L1Provider,
    config: Config,
    messages: mpsc::Receiver<Message>,
    window: Window,
    safe: BlockInfo,
    finalized: BlockInfo,
    update_waiters: Vec<UpdateWaiter>,
    head_waiters: Vec<ChangeWaiter>,
    finalized_waiters: Vec<ChangeWaiter>,
}

impl<L1Provider: Provider> Actor<L1Provider> {
    async fn run(mut self) -> Result<(), ActorError> {
        let interval = self.config.head_interval;
        let mut heads = poll(&self.provider, BlockNumberOrTag::Latest, interval);
        let mut safe = poll(&self.provider, BlockNumberOrTag::Safe, interval);
        let interval = self.config.finalized_interval;
        let mut finalized = poll(&self.provider, BlockNumberOrTag::Finalized, interval);
        // Heads are not polled while a walk fetches blocks; the next head is polled after it.
        let mut walk: Option<Walk> = None;
        loop {
            tokio::select! {
                head = heads.next(), if walk.is_none() => {
                    walk = self.start_walk(head.ok_or(ActorError::StreamEnded)?);
                    self.answer_waiters();
                }
                parent = async { walk.as_mut().expect("polled only during a walk").parent.as_mut().await }, if walk.is_some() => {
                    let Walk { mut branch, .. } = walk.take().expect("polled only during a walk");
                    match parent {
                        Ok(Some(parent)) => {
                            branch.push(block_info(&parent));
                            walk = self.walk(branch);
                        }
                        // The parent was reorged out during the walk. The next head tries again.
                        Ok(None) => {}
                        Err(error) => {
                            warn!(target: "l1", ?error, head = ?branch[0], "Failed to fetch the blocks before the L1 head");
                        }
                    }
                    self.answer_waiters();
                }
                block = safe.next() => self.safe = block.ok_or(ActorError::StreamEnded)?,
                block = finalized.next() => {
                    self.finalized = block.ok_or(ActorError::StreamEnded)?;
                    self.window.prune(self.finalized);
                    self.answer_waiters();
                }
                Some(message) = self.messages.recv() => self.handle(message),
            }
        }
    }

    /// Starts moving the window to `head`. Returns the walk if it needs blocks from L1.
    fn start_walk(&mut self, head: BlockInfo) -> Option<Walk> {
        if let Some(index) = self.window.index_of(head.number, head.hash) {
            // Unchanged, or the head moved back to one of its ancestors.
            let orphaned = self.window.truncate_after(index);
            if orphaned > 0 {
                info!(target: "l1", ?head, orphaned, "L1 head moved back");
            }
            return None;
        }
        let window = self.config.window;
        let far_ahead = self
            .window
            .head()
            .is_some_and(|held| head.number.saturating_sub(held.number) >= window.get() as u64);
        if far_ahead {
            // The blocks in between would push every held block out of the window.
            let orphaned = self.window.clear();
            warn!(target: "l1", ?head, orphaned, "L1 head is too far ahead of the held chain");
            self.window.extend([head], window);
            return None;
        }
        self.walk(vec![head])
    }

    /// Walks back from the oldest block in `branch` until it joins the window, then moves the
    /// window to the branch. Returns the walk if it needs a parent from L1.
    fn walk(&mut self, mut branch: Vec<BlockInfo>) -> Option<Walk> {
        let head = branch[0];
        loop {
            let oldest = *branch.last().expect("the branch starts with the head");
            if let Some(index) = self.window.parent_index(&oldest) {
                let orphaned = self.window.truncate_after(index);
                if orphaned > 0 {
                    info!(target: "l1", ?head, orphaned, "L1 reorg");
                }
                break;
            }
            let below_window =
                self.window.blocks.front().is_none_or(|base| oldest.number <= base.number);
            if below_window || branch.len() >= self.config.window.get() {
                // The window is empty, or the reorg is deeper than it: start over from the branch.
                let orphaned = self.window.clear();
                if orphaned > 0 {
                    warn!(target: "l1", ?head, orphaned, "L1 head does not extend the held chain");
                }
                break;
            }
            // A parent reorged out earlier needs no fetch.
            if let Some(parent) = self.window.orphans.get(&oldest.parent_hash) {
                branch.push(*parent);
                continue;
            }
            let provider = self.provider.root().clone();
            let hash = oldest.parent_hash;
            let parent = Box::pin(async move { provider.get_block_by_hash(hash).await });
            return Some(Walk { branch, parent });
        }
        self.window.extend(branch.into_iter().rev(), self.config.window);
        None
    }

    fn handle(&mut self, message: Message) {
        match message {
            Message::UpdatesSince { cursor, max, reply } => {
                match self.window.updates_since(cursor, max) {
                    Some(update) => {
                        let _ = reply.send(update);
                    }
                    None => park(
                        &mut self.update_waiters,
                        UpdateWaiter { cursor, max, reply },
                        |waiter| waiter.reply.is_closed(),
                    ),
                }
            }
            Message::NextHead(waiter) => {
                park(&mut self.head_waiters, waiter, |waiter| waiter.reply.is_closed());
                answer_changes(&mut self.head_waiters, self.window.head());
            }
            Message::NextFinalized(waiter) => {
                park(&mut self.finalized_waiters, waiter, |waiter| waiter.reply.is_closed());
                answer_changes(&mut self.finalized_waiters, Some(self.finalized));
            }
            Message::Status(reply) => {
                let _ = reply.send(Status {
                    head: self.window.head().unwrap_or_default(),
                    safe: self.safe,
                    finalized: self.finalized,
                });
            }
        }
    }

    /// Answers the waiting requests that now have something to report.
    fn answer_waiters(&mut self) {
        for waiter in std::mem::take(&mut self.update_waiters) {
            if waiter.reply.is_closed() {
                continue;
            }
            match self.window.updates_since(waiter.cursor, waiter.max) {
                Some(update) => {
                    let _ = waiter.reply.send(update);
                }
                None => self.update_waiters.push(waiter),
            }
        }
        answer_changes(&mut self.head_waiters, self.window.head());
        answer_changes(&mut self.finalized_waiters, Some(self.finalized));
    }
}

/// Polls the block with the given tag.
fn poll(
    provider: &impl Provider,
    tag: BlockNumberOrTag,
    interval: Duration,
) -> impl Stream<Item = BlockInfo> + Send + Unpin + 'static {
    PollerBuilder::<(BlockNumberOrTag, bool), Block>::new(
        provider.weak_client(),
        "eth_getBlockByNumber",
        (tag, false),
    )
    .with_poll_interval(interval)
    .into_stream()
    .map(|block| block_info(&block))
}

fn block_info(block: &Block) -> BlockInfo {
    let header = &block.header;
    BlockInfo::new(header.hash, header.number, header.parent_hash, header.timestamp)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::{ErrorPayload, RequestPacket, Response, ResponsePacket, ResponsePayload};
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::TransportErrorKind;
    use std::sync::{
        Arc, Mutex,
        atomic::{AtomicBool, AtomicUsize, Ordering},
    };
    use tokio::{
        sync::Notify,
        task::JoinHandle,
        time::{self, timeout},
    };

    const INTERVAL: Duration = Duration::from_secs(4);
    const MAX: NonZeroUsize = NonZeroUsize::new(16).unwrap();

    /// Returns block `number` of the chain `fork`, whose parent is block `number - 1` of the chain
    /// `parent_fork`.
    fn block(number: u64, fork: u8, parent_fork: u8) -> BlockInfo {
        let hash = |number: u64, fork: u8| {
            let mut hash = B256::repeat_byte(fork);
            hash[..8].copy_from_slice(&number.to_be_bytes());
            hash
        };
        let parent_hash =
            number.checked_sub(1).map_or(B256::ZERO, |parent| hash(parent, parent_fork));
        BlockInfo::new(hash(number, fork), number, parent_hash, number * 12)
    }

    /// Returns blocks `numbers` of the chain `fork`, all children of the same chain.
    fn chain(numbers: std::ops::RangeInclusive<u64>, fork: u8) -> Vec<BlockInfo> {
        numbers.map(|number| block(number, fork, fork)).collect()
    }

    fn window(blocks: Vec<BlockInfo>) -> Window {
        let mut window = Window::default();
        window.extend(blocks, MAX);
        window
    }

    #[test]
    fn advances_from_a_canonical_cursor() {
        let window = window(chain(1..=5, 1));
        assert_eq!(
            window.updates_since(block(2, 1, 1), MAX),
            Some(Update::Advanced(chain(3..=5, 1)))
        );
        assert_eq!(
            window.updates_since(block(2, 1, 1), NonZeroUsize::MIN),
            Some(Update::Advanced(vec![block(3, 1, 1)]))
        );
        // Nothing after the head.
        assert_eq!(window.updates_since(block(5, 1, 1), MAX), None);
    }

    #[test]
    fn traces_a_reorged_cursor_to_the_common_ancestor() {
        let mut window = window(chain(1..=5, 1));
        // Blocks 4 and 5 are replaced.
        window.truncate_after(2);
        window.extend([block(4, 2, 1), block(5, 2, 2), block(6, 2, 2)], MAX);
        assert_eq!(
            window.updates_since(block(5, 1, 1), MAX),
            Some(Update::Reorged {
                ancestor: block(3, 1, 1),
                blocks: vec![block(4, 2, 1), block(5, 2, 2), block(6, 2, 2)],
            })
        );
        // A cursor before the fork only advances.
        assert_eq!(
            window.updates_since(block(3, 1, 1), MAX),
            Some(Update::Advanced(vec![block(4, 2, 1), block(5, 2, 2), block(6, 2, 2)]))
        );

        // A second reorg replaces block 5 again. A cursor on the first fork still finds its
        // ancestor through both.
        window.truncate_after(3);
        window.extend([block(5, 3, 2)], MAX);
        assert_eq!(
            window.updates_since(block(5, 1, 1), MAX),
            Some(Update::Reorged {
                ancestor: block(3, 1, 1),
                blocks: vec![block(4, 2, 1), block(5, 3, 2)]
            })
        );
        assert_eq!(
            window.updates_since(block(6, 2, 2), MAX),
            Some(Update::Reorged { ancestor: block(4, 2, 1), blocks: vec![block(5, 3, 2)] })
        );
    }

    #[test]
    fn a_head_that_moves_back_reorgs_cursors_past_it() {
        let mut window = window(chain(1..=5, 1));
        window.truncate_after(2);
        assert_eq!(
            window.updates_since(block(5, 1, 1), MAX),
            Some(Update::Reorged { ancestor: block(3, 1, 1), blocks: vec![] })
        );
        assert_eq!(window.updates_since(block(3, 1, 1), MAX), None);
    }

    #[test]
    fn old_and_unknown_cursors() {
        let mut window = window(chain(3..=5, 1));
        assert_eq!(
            window.updates_since(block(2, 1, 1), MAX),
            Some(Update::Behind { base: block(3, 1, 1) })
        );
        assert_eq!(window.updates_since(block(4, 9, 9), MAX), Some(Update::Unknown));
        // The default cursor, before any block, has no parent to trace.
        window.prune(block(0, 1, 1));
        assert_eq!(
            window.updates_since(BlockInfo::default(), MAX),
            Some(Update::Behind { base: block(3, 1, 1) })
        );
    }

    #[test]
    fn finality_ahead_of_the_head_keeps_the_head() {
        let mut window = window(chain(1..=5, 1));
        window.prune(block(8, 1, 1));
        assert_eq!(window.blocks, VecDeque::from(vec![block(5, 1, 1)]));
        assert_eq!(
            window.updates_since(block(4, 1, 1), MAX),
            Some(Update::Behind { base: block(5, 1, 1) })
        );
    }

    #[test]
    fn finality_and_the_size_limit_drop_the_oldest_blocks_and_their_orphans() {
        let mut window = window(chain(1..=5, 1));
        window.truncate_after(1);
        window.extend([block(3, 2, 1), block(4, 2, 2), block(5, 2, 2)], MAX);
        window.prune(block(4, 2, 2));
        assert_eq!(window.blocks, VecDeque::from(vec![block(4, 2, 2), block(5, 2, 2)]));
        // The orphans older than the window are gone, so a cursor on them is behind.
        assert!(window.orphans.values().all(|block| block.number >= 4));
        assert_eq!(
            window.updates_since(block(3, 1, 1), MAX),
            Some(Update::Behind { base: block(4, 2, 2) })
        );

        let three = NonZeroUsize::new(3).unwrap();
        let mut window = Window::default();
        window.extend(chain(1..=5, 1), three);
        assert_eq!(window.blocks, VecDeque::from(chain(3..=5, 1)));
        window.extend([block(6, 1, 1)], three);
        assert_eq!(window.blocks, VecDeque::from(chain(4..=6, 1)));
    }

    /// An L1 RPC serving a block tree and the blocks its tags point at.
    #[derive(Default)]
    struct RpcState {
        blocks: Mutex<HashMap<B256, BlockInfo>>,
        tags: Mutex<HashMap<&'static str, BlockInfo>>,
        by_hash_calls: AtomicUsize,
        terminal: AtomicBool,
        stall: AtomicBool,
        cancelled: Notify,
        /// Fails polls for blocks by tag.
        fail_polls: AtomicBool,
        /// Fails fetches of blocks by hash.
        fail_by_hash: AtomicBool,
        /// Holds fetches of blocks by hash until `release`.
        hold_by_hash: AtomicBool,
        release: Notify,
    }

    impl RpcState {
        fn new(head: BlockInfo) -> Arc<Self> {
            let state = Arc::new(Self::default());
            state.set_head(head);
            state.set("safe", head);
            state.set("finalized", head);
            state
        }

        /// Makes `block` the head, and adds it to the tree.
        fn set_head(&self, block: BlockInfo) {
            self.set("latest", block);
        }

        fn set(&self, tag: &'static str, block: BlockInfo) {
            self.add([block]);
            self.tags.lock().unwrap().insert(tag, block);
        }

        fn add(&self, blocks: impl IntoIterator<Item = BlockInfo>) {
            let mut tree = self.blocks.lock().unwrap();
            tree.extend(blocks.into_iter().map(|block| (block.hash, block)));
        }
    }

    fn rpc_block(info: BlockInfo) -> Block {
        let mut block: Block = Block::default();
        block.header.hash = info.hash;
        block.header.inner.number = info.number;
        block.header.inner.parent_hash = info.parent_hash;
        block.header.inner.timestamp = info.timestamp;
        block
    }

    fn provider(state: Arc<RpcState>) -> RootProvider {
        let transport = tower::service_fn(
            move |request: RequestPacket| -> alloy_transport::TransportFut<'static> {
                let state = state.clone();
                Box::pin(async move {
                    let RequestPacket::Single(request) = request else {
                        panic!("unexpected batch")
                    };
                    let params: serde_json::Value =
                        serde_json::from_str(request.params().unwrap().get()).unwrap();
                    if state.stall.load(Ordering::SeqCst) {
                        struct OnDrop<'a>(&'a Notify);
                        impl Drop for OnDrop<'_> {
                            fn drop(&mut self) {
                                self.0.notify_one();
                            }
                        }
                        let _on_drop = OnDrop(&state.cancelled);
                        std::future::pending::<()>().await;
                    }
                    let block = match request.method() {
                        "eth_getBlockByNumber" => {
                            if state.fail_polls.load(Ordering::SeqCst) {
                                return Err(TransportErrorKind::custom_str("L1 unavailable"));
                            }
                            let tag = params[0].as_str().unwrap();
                            state.tags.lock().unwrap().get(tag).copied()
                        }
                        "eth_getBlockByHash" => {
                            state.by_hash_calls.fetch_add(1, Ordering::SeqCst);
                            if state.hold_by_hash.load(Ordering::SeqCst) {
                                state.release.notified().await;
                            }
                            if state.fail_by_hash.load(Ordering::SeqCst) {
                                return Err(TransportErrorKind::custom_str("L1 unavailable"));
                            }
                            let hash: B256 = serde_json::from_value(params[0].clone()).unwrap();
                            state.blocks.lock().unwrap().get(&hash).copied()
                        }
                        method => panic!("unexpected method {method}"),
                    };
                    let payload = if state.terminal.load(Ordering::SeqCst) {
                        ResponsePayload::Failure(ErrorPayload {
                            code: -32000,
                            message: "filter not found".into(),
                            data: None,
                        })
                    } else {
                        ResponsePayload::Success(
                            serde_json::value::to_raw_value(&block.map(rpc_block)).unwrap(),
                        )
                    };
                    Ok::<_, alloy_transport::TransportError>(ResponsePacket::Single(Response {
                        id: request.id().clone(),
                        payload,
                    }))
                })
            },
        );
        RootProvider::new(RpcClient::new(transport, false))
    }

    const CONFIG: Config =
        Config { head_interval: INTERVAL, finalized_interval: INTERVAL, window: MAX };

    fn spawn(state: Arc<RpcState>) -> (Handle, JoinHandle<Result<(), ActorError>>) {
        let builder = Builder::new(Capacity::try_from(16).unwrap());
        let handle = builder.handle();
        (handle, tokio::spawn(builder.build(provider(state), CONFIG)))
    }

    #[tokio::test(start_paused = true)]
    async fn reports_advances_including_gaps_and_waits_for_new_heads() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        assert_eq!(head, block(10, 1, 1));

        // Nothing after the head yet.
        let pending = handle.updates_since(head, MAX);
        assert!(timeout(INTERVAL * 3, pending).await.is_err());

        // The head jumps two blocks; the actor fetches the one between.
        state.add([block(11, 1, 1)]);
        state.set_head(block(12, 1, 1));
        assert_eq!(
            handle.updates_since(head, MAX).await.unwrap(),
            Update::Advanced(chain(11..=12, 1))
        );
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn reports_reorgs_to_waiting_and_late_consumers_alike() {
        let state = RpcState::new(block(10, 1, 1));
        state.add([block(9, 1, 1)]);
        let (handle, _actor) = spawn(state.clone());
        let mut cursor = handle.next_head(BlockInfo::default()).await.unwrap();
        state.set_head(block(11, 1, 1));
        cursor = match handle.updates_since(cursor, MAX).await.unwrap() {
            Update::Advanced(blocks) => *blocks.last().unwrap(),
            update => panic!("unexpected {update:?}"),
        };
        assert_eq!(cursor, block(11, 1, 1));

        // A waiting consumer and one that asks later both see the reorg of block 11.
        let waiting = tokio::spawn({
            let handle = handle.clone();
            async move { handle.updates_since(cursor, MAX).await.unwrap() }
        });
        state.add([block(11, 2, 1)]);
        state.set_head(block(12, 2, 2));
        let expected = Update::Reorged {
            ancestor: block(10, 1, 1),
            blocks: vec![block(11, 2, 1), block(12, 2, 2)],
        };
        assert_eq!(waiting.await.unwrap(), expected);
        assert_eq!(handle.updates_since(cursor, MAX).await.unwrap(), expected);
    }

    #[tokio::test(start_paused = true)]
    async fn reports_status_and_finalized_changes_and_prunes_finalized_blocks() {
        let state = RpcState::new(block(10, 1, 1));
        state.set("safe", block(8, 1, 1));
        state.set("finalized", block(6, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let finalized = handle.next_finalized(BlockInfo::default()).await.unwrap();
        assert_eq!(finalized, block(6, 1, 1));
        handle.next_head(BlockInfo::default()).await.unwrap();
        assert_eq!(
            handle.status().await.unwrap(),
            Status { head: block(10, 1, 1), safe: block(8, 1, 1), finalized: block(6, 1, 1) }
        );

        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_finalized(finalized).await.unwrap() }
        });
        state.set("finalized", block(10, 1, 1));
        assert_eq!(next.await.unwrap(), block(10, 1, 1));
        // The window now starts at the finalized block.
        assert_eq!(
            handle.updates_since(block(6, 1, 1), MAX).await.unwrap(),
            Update::Behind { base: block(10, 1, 1) }
        );
    }

    #[tokio::test(start_paused = true)]
    async fn answers_requests_while_fetching_blocks() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        state.hold_by_hash.store(true, Ordering::SeqCst);
        state.add([block(11, 1, 1)]);
        state.set_head(block(12, 1, 1));
        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_head(head).await.unwrap() }
        });
        // The actor is waiting for block 11, but still answers.
        time::sleep(INTERVAL * 3).await;
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 1);
        assert_eq!(handle.status().await.unwrap().head, head);
        assert!(!next.is_finished());

        state.release.notify_one();
        assert_eq!(next.await.unwrap(), block(12, 1, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn a_head_far_ahead_starts_over_without_fetching() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        let far = block(10 + MAX.get() as u64, 1, 1);
        state.set_head(far);
        assert_eq!(handle.next_head(head).await.unwrap(), far);
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 0);
        assert_eq!(handle.updates_since(head, MAX).await.unwrap(), Update::Behind { base: far });
    }

    #[rstest::rstest]
    #[case::missing(false)]
    #[case::failed(true)]
    #[tokio::test(start_paused = true)]
    async fn an_unavailable_parent_keeps_the_held_chain_until_a_later_head(#[case] fail: bool) {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        state.fail_by_hash.store(fail, Ordering::SeqCst);
        // Block 11 is missing from the RPC, or fetching it fails.
        state.set_head(block(12, 1, 1));
        time::sleep(INTERVAL * 3).await;
        assert!(state.by_hash_calls.load(Ordering::SeqCst) > 0);
        assert_eq!(handle.status().await.unwrap().head, head);

        state.fail_by_hash.store(false, Ordering::SeqCst);
        state.add([block(11, 1, 1)]);
        assert_eq!(
            handle.updates_since(head, MAX).await.unwrap(),
            Update::Advanced(chain(11..=12, 1))
        );
    }

    #[tokio::test(start_paused = true)]
    async fn a_polled_head_that_moves_back_reorgs_later_cursors() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let head = handle.next_head(BlockInfo::default()).await.unwrap();
        state.set_head(block(11, 1, 1));
        let cursor = handle.next_head(head).await.unwrap();
        state.set_head(head);
        assert_eq!(
            handle.updates_since(cursor, MAX).await.unwrap(),
            Update::Reorged { ancestor: head, blocks: vec![] }
        );
        assert_eq!(state.by_hash_calls.load(Ordering::SeqCst), 0);
    }

    #[tokio::test(start_paused = true)]
    async fn retries_failed_polls() {
        let state = RpcState::new(block(10, 1, 1));
        state.fail_polls.store(true, Ordering::SeqCst);
        let (handle, _actor) = spawn(state.clone());
        let next = tokio::spawn({
            let handle = handle.clone();
            async move { handle.next_head(BlockInfo::default()).await.unwrap() }
        });
        time::sleep(INTERVAL * 3).await;
        assert!(!next.is_finished());
        assert_eq!(handle.status().await.unwrap(), Status::default());
        state.fail_polls.store(false, Ordering::SeqCst);
        assert_eq!(next.await.unwrap(), block(10, 1, 1));
    }

    #[tokio::test(start_paused = true)]
    async fn a_cancelled_wait_loses_no_block() {
        let state = RpcState::new(block(10, 1, 1));
        let (handle, _actor) = spawn(state.clone());
        let finalized = handle.next_finalized(BlockInfo::default()).await.unwrap();
        tokio::select! {
            _ = handle.next_finalized(finalized) => panic!("the finalized block did not change"),
            () = time::sleep(INTERVAL * 3) => {}
        }
        state.set("finalized", block(11, 1, 1));
        assert_eq!(handle.next_finalized(finalized).await.unwrap(), block(11, 1, 1));
    }

    #[tokio::test]
    async fn dropping_the_lifetime_cancels_rpcs_and_fails_requests() {
        let state = RpcState::new(block(1, 1, 1));
        state.stall.store(true, Ordering::SeqCst);
        let builder = Builder::new(Capacity::try_from(1).unwrap());
        let handle = builder.handle();
        let mut lifetime = Box::pin(builder.build(provider(state.clone()), CONFIG));
        assert!(futures::poll!(lifetime.as_mut()).is_pending());
        drop(lifetime);
        state.cancelled.notified().await;
        assert!(handle.status().await.is_err());
    }

    #[tokio::test]
    async fn a_terminal_stream_error_stops_the_actor() {
        let state = RpcState::new(block(1, 1, 1));
        state.terminal.store(true, Ordering::SeqCst);
        let (handle, actor) = spawn(state);
        assert!(matches!(actor.await.unwrap(), Err(ActorError::StreamEnded)));
        assert!(handle.status().await.is_err());
    }

    #[test]
    fn builds_without_a_runtime() {
        let builder = Builder::new(Capacity::try_from(1).unwrap());
        let state = RpcState::new(block(1, 1, 1));
        drop(builder.build(provider(state), CONFIG));
    }

    #[tokio::test]
    async fn a_reporting_handle_serves_the_status() {
        let status = Status { head: block(3, 1, 1), ..Default::default() };
        let (_published, receiver) = tokio::sync::watch::channel(status);
        assert_eq!(Handle::reporting(receiver).status().await.unwrap(), status);
    }
}
