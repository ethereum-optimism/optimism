//! Constructs the L1 actor and its handle.

use super::{ActorError, Capacity, Handle, actor, handle::Message};
use alloy_provider::Provider;
use std::{future::Future, time::Duration};
use tokio::sync::mpsc;

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
    handle: Handle,
    messages: mpsc::Receiver<Message>,
}

impl Builder {
    /// Creates the builder with a request queue of the given capacity.
    pub fn new(capacity: Capacity) -> Self {
        let (messages_tx, messages) = mpsc::channel(capacity.get());
        Self { handle: Handle::new(messages_tx), messages }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// The actor holds L1 from the latest finalized block, so requests wait until L1 reports one.
    /// Runtime work begins when the future is polled. Dropping the future stops the actor, and
    /// requests then fail.
    pub fn build<L1Provider: Provider + 'static>(
        self,
        provider: L1Provider,
        config: Config,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, messages } = self;
        actor::run(provider, config, messages)
    }
}
