//! The finality actor, which finalizes L2 blocks once the L1 blocks they were derived from
//! finalize.
//!
//! The engine drives it through the [`Handle`], which implements
//! [`engine::Finality`](crate::engine::Finality): it gives the actor each safe head with the L1
//! block it was derived from, and waits for a block to finalize. The actor follows L1's finalized
//! block, and decides the highest safe head whose source that block covers. Only derivation from L1
//! runs it: trusted derivation finalizes the delegate's blocks.

mod actor;

mod handle;
pub use handle::Handle;

use crate::engine::Safe;
use async_trait::async_trait;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::{fmt::Debug, future::Future, sync::Arc};
use thiserror::Error;
use tokio::sync::watch;

/// The L1 chain, as the finality actor follows it.
#[async_trait]
pub trait L1: Debug + Send + Sync {
    /// Waits for a finalized L1 block other than the last one returned, and returns it. Returns
    /// `None` once no more finalized blocks will arrive. Cancelling the wait loses no block.
    async fn next_finalized(&mut self) -> Option<BlockInfo>;
}

/// An error that stops the finality actor.
#[derive(Error, Debug)]
pub enum ActorError {
    /// The L1 actor stopped.
    #[error("the L1 actor stopped")]
    L1Stopped,
    /// Every handle was dropped.
    #[error("every handle was dropped")]
    HandlesDropped,
}

/// The finality actor stopped.
#[derive(Debug, Clone, Copy, Error)]
#[error("the finality actor stopped")]
pub struct Stopped;

/// Constructs the handle and the actor.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    safe_rx: watch::Receiver<Option<Safe>>,
    finalized_tx: watch::Sender<Option<L2BlockInfo>>,
}

impl Builder {
    /// Creates the builder, with nothing to finalize yet.
    pub fn new() -> Self {
        let (safe_tx, safe_rx) = watch::channel(None);
        let (finalized_tx, finalized_rx) = watch::channel(None);
        Self { handle: Handle::new(Arc::new(safe_tx), finalized_rx), safe_rx, finalized_tx }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies the dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// The actor follows the finalized L1 block through `l1`. Runtime work begins when the future
    /// is polled. Dropping the future stops the actor.
    pub fn build<L1_: L1 + 'static>(
        self,
        l1: L1_,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, safe_rx, finalized_tx } = self;
        actor::Actor::new(safe_rx, finalized_tx, l1).run()
    }
}

impl Default for Builder {
    fn default() -> Self {
        Self::new()
    }
}
