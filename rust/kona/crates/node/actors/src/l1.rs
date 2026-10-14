//! Tracks the canonical L1 chain.
//!
//! The actor polls the L1 head, safe and finalized blocks, and holds the canonical chain from the
//! latest finalized block to the head. Consumers ask its [`Handle`] precise questions: what the
//! latest blocks are, which block is canonical at a height, and what follows a block they
//! processed, its canonical child, or the news that it was reorged out. Every consumer sees the
//! actor's one view of L1. Blocks before the finalized block cannot reorg, so the handle reads them
//! from L1.
//!
//! The actor and its handles read L1 through a [`RetryingRpc`](kona_sources::RetryingRpc): every
//! block is checked to hash to its header and to be the block asked for, and a read that fails is
//! retried until it succeeds. The node can do little without L1, so the actor waits for it rather
//! than failing.

mod actor;

mod builder;
pub use builder::{Builder, Config};

mod chain;

mod handle;
pub use handle::{Child, Handle, HandleError, Status};

mod rpc;
pub use rpc::finalized_block;

#[cfg(test)]
pub(crate) mod test_utils;
