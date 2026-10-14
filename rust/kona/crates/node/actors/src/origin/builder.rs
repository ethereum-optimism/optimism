//! Constructs the origin actor and its handle.

use super::{ActorError, Handle, actor::Actor, window::Window};
use crate::{engine::EngineState, l1};
use kona_genesis::RollupConfig;
use std::{future::Future, sync::Arc};
use tokio::sync::watch;

/// Constructs the handle and the actor.
#[derive(Debug)]
pub struct Builder {
    config: Arc<RollupConfig>,
    confirmations: u64,
    window: watch::Sender<Window>,
}

impl Builder {
    /// Creates the builder. The handle selects origins under `config`, and an L1 block is a next
    /// origin once it is `confirmations` blocks below the L1 head.
    pub fn new(config: Arc<RollupConfig>, confirmations: u64) -> Self {
        Self { config, confirmations, window: watch::channel(Window::default()).0 }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        Handle::new(self.config.clone(), self.window.subscribe())
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// The actor follows the unsafe head in `engine_state`, and reads L1 through `l1`. Runtime
    /// work begins when the future is polled. Dropping the future stops the actor, and its handles
    /// then report [`OriginError::Stopped`](super::OriginError::Stopped).
    pub fn build(
        self,
        l1: l1::Handle,
        engine_state: watch::Receiver<EngineState>,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { config: _, confirmations, window } = self;
        Actor { l1, engine_state, confirmations, window }.run()
    }
}
