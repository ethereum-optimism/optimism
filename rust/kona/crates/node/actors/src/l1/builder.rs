//! Constructs the L1 actor and its handle.

use super::{Handle, actor::Actor, chain::Chain};
use kona_protocol::BlockInfo;
use kona_sources::RetryingRpc;
use std::{convert::Infallible, future::Future, time::Duration};
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
    rpc: RetryingRpc,
    chain: watch::Sender<Chain>,
}

impl Builder {
    /// Creates the builder with the chain starting at `finalized`, a block that cannot reorg, such
    /// as the one [`finalized_block`](super::finalized_block) reads.
    ///
    /// The actor and its handles read L1 through `rpc`, and wait for as long as L1 cannot be read.
    pub fn new(rpc: RetryingRpc, finalized: BlockInfo) -> Self {
        Self { rpc, chain: watch::channel(Chain::new(finalized)).0 }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        Handle::new(self.chain.subscribe(), self.rpc.clone())
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled. The actor runs until the future is dropped,
    /// and questions to its handles then fail.
    pub fn build(self, config: Config) -> impl Future<Output = Infallible> + Send + 'static {
        let Self { rpc, chain } = self;
        Actor { rpc, config, chain }.run()
    }
}
