//! Derivation delegated to a trusted source.
//!
//! The actor polls an external derivation delegate for its sync status, checks the L1 blocks it
//! reports against the canonical L1 chain, and publishes the delegate's safe and finalized heads
//! for the engine to adopt. The engine reads safe heads through [`Handle::next_safe`], and
//! finalized heads through the [`Handle`]'s [`engine::Finality`](crate::engine::Finality).

mod actor;

mod client;
pub use client::{
    DerivationDelegateClient, DerivationDelegateClientError, DerivationDelegateProvider,
};

mod handle;
pub use handle::Handle;

use crate::engine::Safe;
use kona_derive::ChainProvider;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::{convert::Infallible, future::Future};
use tokio::sync::watch;

/// The heads the delegate reported.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct Reported {
    /// The delegate's safe head, with the L1 block it derived up to standing in for its source.
    safe: Safe,
    /// The delegate's finalized head.
    finalized: L2BlockInfo,
}

/// Constructs the handle and the actor.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    reported_tx: watch::Sender<Option<Reported>>,
    current_l1_tx: watch::Sender<BlockInfo>,
}

impl Builder {
    /// Creates the builder, with no heads from the delegate yet.
    pub fn new() -> Self {
        let (reported_tx, reported_rx) = watch::channel(None);
        let (current_l1_tx, current_l1_rx) = watch::channel(BlockInfo::default());
        Self { handle: Handle::new(reported_rx, current_l1_rx), reported_tx, current_l1_tx }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies the dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// The actor polls `delegate`, and checks the L1 blocks it reports with `l1`. Runtime work
    /// begins when the future is polled. Dropping the future stops the actor.
    pub fn build<Delegate, L1Provider>(
        self,
        delegate: Delegate,
        l1: L1Provider,
    ) -> impl Future<Output = Infallible> + Send + 'static
    where
        Delegate: DerivationDelegateProvider + 'static,
        L1Provider: ChainProvider + Send + 'static,
    {
        let Self { handle: _, reported_tx, current_l1_tx } = self;
        async move { actor::Actor::new(delegate, l1, reported_tx, current_l1_tx).run().await }
    }
}

impl Default for Builder {
    fn default() -> Self {
        Self::new()
    }
}
