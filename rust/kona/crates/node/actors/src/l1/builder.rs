//! Constructs the L1 actor and its handle.

use super::{ActorError, Handle, actor::Actor, chain::Chain};
use alloy_provider::RootProvider;
use kona_protocol::BlockInfo;
use std::{future::Future, time::Duration};
use tokio::sync::watch;

/// How the actor polls L1.
#[derive(Debug, Clone, Copy)]
pub struct Config {
    /// How often to poll the head and safe blocks.
    pub head_interval: Duration,
    /// How often to poll the finalized block.
    pub finalized_interval: Duration,
}

/// Constructs the handle and the actor.
#[derive(Debug)]
pub struct Builder {
    provider: RootProvider,
    chain: watch::Sender<Chain>,
}

impl Builder {
    /// Creates the builder with the chain starting at `finalized`, a block that cannot reorg, such
    /// as the one [`finalized_block`](super::finalized_block) reads.
    pub fn new(provider: RootProvider, finalized: BlockInfo) -> Self {
        Self { provider, chain: watch::channel(Chain::new(finalized)).0 }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        Handle::new(self.chain.subscribe(), self.provider.clone())
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled. Dropping the future stops the actor, and
    /// questions to its handles then fail.
    pub fn build(
        self,
        config: Config,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { provider, chain } = self;
        Actor { provider, config, chain }.run()
    }
}
