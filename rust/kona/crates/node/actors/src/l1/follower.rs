//! Follows the L1 head.

use super::{
    rpc::block_info,
    view::{Extension, View},
};
use alloy_primitives::B256;
use alloy_provider::Provider;
use alloy_transport::TransportResult;
use futures::{StreamExt, future::BoxFuture, stream::FuturesUnordered};
use kona_protocol::BlockInfo;
use std::collections::HashSet;

/// Follows the L1 head, moving the view to the latest head whenever a new head or a fetched block
/// lets it, and fetching the ancestors the view needs to link the head.
pub(super) struct Follower<L1Provider> {
    provider: L1Provider,
    /// The latest head.
    head: BlockInfo,
    /// The hashes of the blocks being fetched.
    fetching: HashSet<B256>,
    /// The fetches of those blocks.
    fetches: FuturesUnordered<BoxFuture<'static, Fetched>>,
}

/// A fetched ancestor of a head.
#[derive(Debug)]
pub(super) struct Fetched {
    /// The hash of the block.
    hash: B256,
    /// The block, or `None` if L1 no longer has it.
    block: TransportResult<Option<BlockInfo>>,
}

impl<L1Provider: Provider> Follower<L1Provider> {
    /// Creates a follower starting at `head`, that fetches the ancestors new heads need from
    /// `provider`.
    pub(super) fn new(provider: L1Provider, head: BlockInfo) -> Self {
        Self { provider, head, fetching: HashSet::new(), fetches: FuturesUnordered::new() }
    }

    /// Waits for a fetch to finish. Returns `None` at once if nothing is being fetched. Cancelling
    /// the wait loses nothing.
    pub(super) async fn next_fetched(&mut self) -> Option<Fetched> {
        self.fetches.next().await
    }

    /// Moves `view` to a new head, or fetches the ancestor it needs first.
    pub(super) fn on_head(&mut self, head: BlockInfo, view: &mut View) {
        self.head = head;
        view.insert(head);
        self.follow(view);
    }

    /// Adds a fetched ancestor to `view`, and moves the view to the latest head if it now can.
    pub(super) fn on_fetched(&mut self, Fetched { hash, block }: Fetched, view: &mut View) {
        self.fetching.remove(&hash);
        match block {
            Ok(Some(block)) => {
                view.insert(block);
                self.follow(view);
            }
            // The block was reorged out. The next head tries again.
            Ok(None) => {}
            Err(error) => {
                warn!(target: "l1", ?error, %hash, "Failed to fetch an ancestor of the L1 head");
            }
        }
    }

    /// Moves `view` to the latest head, or fetches the ancestor it needs first.
    fn follow(&mut self, view: &mut View) {
        let head = self.head;
        match view.extend(head) {
            Extension::Moved { orphaned: 0 } => {}
            Extension::Moved { orphaned } => info!(target: "l1", ?head, orphaned, "L1 reorg"),
            Extension::Rejected => {
                warn!(target: "l1", ?head, "L1 head does not descend from the finalized block");
            }
            Extension::NeedsParent(hash) => self.fetch(hash),
        }
    }

    /// Fetches the block with `hash`, unless it is being fetched already.
    fn fetch(&mut self, hash: B256) {
        if !self.fetching.insert(hash) {
            return;
        }
        let provider = self.provider.root().clone();
        self.fetches.push(Box::pin(async move {
            let block = provider.get_block_by_hash(hash).await;
            Fetched { hash, block: block.map(|block| block.as_ref().map(block_info)) }
        }));
    }
}
