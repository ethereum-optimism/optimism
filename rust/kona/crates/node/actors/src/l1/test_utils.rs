//! Blocks and chains for tests that use the L1 actor.

use super::{Handle, chain::Chain, rpc::block_info};
use alloy_consensus::{Header, Sealable};
use alloy_primitives::B256;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_eth::Block;
use alloy_transport::mock::Asserter;
use kona_protocol::BlockInfo;
use kona_sources::UntrustedRpc;
use std::time::Duration;
use tokio::sync::watch;

/// How long the test handles wait before reading L1 again.
pub(crate) const RETRY_DELAY: Duration = Duration::from_secs(1);

/// Returns block `number` of the chain `fork`, whose parent is block `number - 1` of the chain
/// `parent_fork`.
///
/// Its hash is made up, so an RPC cannot serve it: tests that read blocks over RPC build them with
/// [`child`] instead.
pub(crate) fn block(number: u64, fork: u8, parent_fork: u8) -> BlockInfo {
    let hash = |number: u64, fork: u8| {
        let mut hash = B256::repeat_byte(fork);
        hash[..8].copy_from_slice(&number.to_be_bytes());
        hash
    };
    let parent_hash = number.checked_sub(1).map_or(B256::ZERO, |parent| hash(parent, parent_fork));
    BlockInfo::new(hash(number, fork), number, parent_hash, number * 12)
}

/// Returns blocks `numbers` of the chain `fork`, all children of the same chain.
pub(crate) fn chain(numbers: std::ops::RangeInclusive<u64>, fork: u8) -> Vec<BlockInfo> {
    numbers.map(|number| block(number, fork, fork)).collect()
}

/// Returns an L1 genesis block whose hash is that of its header, as an RPC serves it.
pub(crate) fn genesis() -> BlockInfo {
    header_block(Header::default())
}

/// Returns the child of `parent` on the chain `fork`, whose hash is that of its header, as an RPC
/// serves it. Children on different forks differ in their timestamps.
pub(crate) fn child(parent: &BlockInfo, fork: u8) -> BlockInfo {
    let number = parent.number + 1;
    header_block(Header {
        number,
        parent_hash: parent.hash,
        timestamp: number * 12 + u64::from(fork),
        ..Default::default()
    })
}

/// Returns `count` blocks on the chain `fork`, each the [`child`] of the one before, the first a
/// child of `parent`.
pub(crate) fn descendants(parent: &BlockInfo, count: usize, fork: u8) -> Vec<BlockInfo> {
    std::iter::successors(Some(*parent), |block| Some(child(block, fork)))
        .skip(1)
        .take(count)
        .collect()
}

/// Returns the RPC block for `info`, a block built with [`genesis`] or [`child`].
pub(crate) fn rpc_block(info: &BlockInfo) -> Block {
    let mut block: Block = Block::default();
    block.header.inner = Header {
        number: info.number,
        parent_hash: info.parent_hash,
        timestamp: info.timestamp,
        ..Default::default()
    };
    block.header.hash = info.hash;
    block
}

fn header_block(header: Header) -> BlockInfo {
    block_info(&header.seal_slow())
}

/// Returns a chain of `blocks`, oldest first; the oldest is finalized.
pub(super) fn chain_of(blocks: &[BlockInfo]) -> Chain {
    let mut chain = Chain::new(blocks[0]);
    chain.set_head(blocks[1..].to_vec());
    chain.set_linking(false);
    chain
}

/// An L1 chain that a test moves, in place of the L1 actor.
#[derive(Debug, Clone)]
pub(crate) struct TestChain(watch::Sender<Chain>);

impl TestChain {
    /// Creates the chain of `blocks`, oldest first; the oldest is finalized.
    pub(crate) fn new(blocks: &[BlockInfo]) -> Self {
        Self(watch::channel(chain_of(blocks)).0)
    }

    /// Makes the last of `blocks` the head. `blocks` is oldest first, and the first is held or a
    /// child of a held block.
    pub(crate) fn set_head(&self, blocks: &[BlockInfo]) {
        self.0.send_modify(|chain| {
            chain.set_head(blocks.to_vec());
        });
    }

    /// Returns a handle to the chain, which reads blocks before the finalized block from
    /// `history`.
    pub(crate) fn handle_with_history(&self, history: RootProvider) -> Handle {
        Handle::new(self.0.subscribe(), UntrustedRpc::new(history).retrying(RETRY_DELAY))
    }

    /// Sets whether the actor is linking a head it polled.
    pub(crate) fn set_linking(&self, linking: bool) {
        self.0.send_modify(|chain| {
            chain.set_linking(linking);
        });
    }

    /// Returns a handle to a chain of `blocks` that never changes, and never stops. The handle must
    /// not read blocks before the finalized block.
    pub(crate) fn fixed(blocks: &[BlockInfo]) -> Handle {
        let chain = Self::new(blocks);
        let handle = chain.handle();
        std::mem::forget(chain);
        handle
    }

    /// Returns a handle to the chain, which must not read blocks before the finalized block.
    pub(crate) fn handle(&self) -> Handle {
        self.handle_with_history(RootProvider::new(RpcClient::mocked(Asserter::new())))
    }
}
