//! Blocks and chains for tests that use the L1 actor.

use super::{Handle, chain::Chain};
use alloy_primitives::B256;
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_transport::mock::Asserter;
use kona_protocol::BlockInfo;
use tokio::sync::watch;

/// Returns block `number` of the chain `fork`, whose parent is block `number - 1` of the chain
/// `parent_fork`.
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

/// Returns a chain of `blocks`, oldest first; the oldest is finalized.
pub(super) fn chain_of(blocks: &[BlockInfo]) -> Chain {
    let mut chain = Chain::new(blocks[0]);
    chain.set_head(blocks[1..].to_vec());
    chain
}

/// An L1 chain that a test moves, in place of the L1 actor.
#[derive(Debug)]
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
        Handle::new(self.0.subscribe(), history)
    }

    /// Returns a handle to the chain, which must not read blocks before the finalized block.
    pub(crate) fn handle(&self) -> Handle {
        self.handle_with_history(RootProvider::new(RpcClient::mocked(Asserter::new())))
    }
}
