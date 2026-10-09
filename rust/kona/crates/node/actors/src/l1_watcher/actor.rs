//! L1 watcher construction and execution.

use super::{ActorError, Derivation, Handle, State, worker::Worker};
use alloy_primitives::Address;
use alloy_provider::Provider;
use futures::{Stream, StreamExt};
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use std::{future::Future, sync::Arc};
use tokio::sync::watch;

/// Constructs the handle and task.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    published: watch::Sender<State>,
}

impl Default for Builder {
    fn default() -> Self {
        Self::new()
    }
}

impl Builder {
    /// Creates the builder with its initial state.
    pub fn new() -> Self {
        let (published, state) = watch::channel(State::default());
        Self { handle: Handle::new(state), published }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled.
    /// Dropping the future stops the actor.
    #[allow(clippy::too_many_arguments)]
    pub fn build<
        BlockStream: Stream<Item = BlockInfo> + Unpin + Send + 'static,
        L1Provider: Provider + 'static,
        Derivation_: Derivation + 'static,
    >(
        self,
        l1_provider: L1Provider,
        head_stream: BlockStream,
        finalized_stream: BlockStream,
        safe_stream: BlockStream,
        rollup_config: Arc<RollupConfig>,
        derivation: Derivation_,
        block_signer_sender: watch::Sender<Address>,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let actor = Actor { published: self.published, head_stream, finalized_stream, safe_stream };
        let worker = Worker::new(l1_provider, rollup_config, derivation, block_signer_sender);
        actor.run(worker)
    }
}

/// Observes L1 and publishes the latest head and finality.
#[derive(Debug)]
struct Actor<BlockStream> {
    published: watch::Sender<State>,
    head_stream: BlockStream,
    finalized_stream: BlockStream,
    safe_stream: BlockStream,
}

impl<BlockStream> Actor<BlockStream>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
{
    async fn run<L1Provider: Provider, Derivation_: Derivation>(
        self,
        worker: Worker<L1Provider, Derivation_>,
    ) -> Result<(), ActorError> {
        let state = self.published.subscribe();
        // Coalesce observations while forwarding or signer refresh is stalled.
        tokio::try_join!(self.observe(), worker.run(state))?;
        Ok(())
    }

    async fn observe(mut self) -> Result<(), ActorError> {
        loop {
            tokio::select! {
                head = self.head_stream.next() => {
                    let head = head.ok_or(ActorError::StreamEnded)?;
                    self.published.send_modify(|state| state.head_l1 = Some(head));
                }
                finalized = self.finalized_stream.next() => {
                    let finalized = finalized.ok_or(ActorError::StreamEnded)?;
                    self.published.send_modify(|state| state.finalized_l1 = Some(finalized));
                }
                safe = self.safe_stream.next() => {
                    let safe = safe.ok_or(ActorError::StreamEnded)?;
                    self.published.send_modify(|state| state.safe_l1 = Some(safe));
                }
            }
        }
    }
}

#[cfg(test)]
mod tests;
