//! Tracks the canonical L1 chain.
//!
//! The actor polls the L1 head, safe and finalized blocks, and holds the canonical chain from
//! about the finalized block to the head. Each consumer keeps a cursor, the last L1 block it
//! processed, and asks the actor what changed since: the blocks that extend the cursor, or, after a
//! reorg, the common ancestor to roll back to. Every consumer sees the actor's one view of L1. The
//! actor never waits on a consumer, and answers requests while it fetches blocks from L1.

mod actor;
pub use actor::ActorError;

mod builder;
pub use builder::{Builder, Config};

mod handle;
pub use handle::{Handle, HandleError, Status, Update};

mod follower;

mod non_empty;

mod rpc;

mod view;

#[cfg(test)]
mod test_utils;

pub use crate::capacity::{Capacity, InvalidCapacity};
