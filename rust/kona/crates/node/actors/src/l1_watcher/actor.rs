//! L1 observation with independent derivation forwarding and signer refresh.

use super::{L1WatcherActorError, L1WatcherDerivationClient, worker::Worker};
use crate::{L1State, NodeActor};
use alloy_primitives::Address;
use alloy_provider::Provider;
use async_trait::async_trait;
use futures::{Stream, StreamExt};
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use std::sync::Arc;
use tokio::{sync::watch, task::JoinSet};

/// Observes L1 and publishes the latest head and finality.
///
/// Head notifications are already allowed to skip blocks. Keeping only the latest observation
/// lets derivation catch up without blocking observation or growing a queue indefinitely.
/// Forwarding and signer refresh are cancelled when the watcher is dropped.
#[derive(Debug)]
pub struct L1WatcherActor<BlockStream, L1Provider, Client>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider: Provider,
    Client: L1WatcherDerivationClient,
{
    state: watch::Sender<L1State>,
    head_stream: BlockStream,
    finalized_stream: BlockStream,
    safe_stream: BlockStream,
    /// Started on the first step, so construction does not require a runtime.
    worker: Option<Worker<L1Provider, Client>>,
    task: JoinSet<Result<(), L1WatcherActorError<BlockInfo>>>,
}

impl<BlockStream, L1Provider, Client> L1WatcherActor<BlockStream, L1Provider, Client>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider: Provider,
    Client: L1WatcherDerivationClient,
{
    /// Constructs a watcher. Forwarding and signer refresh start on the first step.
    pub fn new(
        l1_provider: L1Provider,
        head_stream: BlockStream,
        finalized_stream: BlockStream,
        safe_stream: BlockStream,
        rollup_config: Arc<RollupConfig>,
        derivation_client: Client,
        block_signer_sender: watch::Sender<Address>,
    ) -> Self {
        let (state, _) = watch::channel(L1State::default());
        Self {
            state,
            head_stream,
            finalized_stream,
            safe_stream,
            worker: Some(Worker::new(
                l1_provider,
                rollup_config,
                derivation_client,
                block_signer_sender,
            )),
            task: JoinSet::new(),
        }
    }

    /// Subscribes to the watcher's latest L1 observations.
    pub fn state_receiver(&self) -> watch::Receiver<L1State> {
        self.state.subscribe()
    }
}

#[async_trait]
impl<BlockStream, L1Provider, Client> NodeActor for L1WatcherActor<BlockStream, L1Provider, Client>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send + 'static,
    L1Provider: Provider + 'static,
    Client: L1WatcherDerivationClient + 'static,
{
    type Error = L1WatcherActorError<BlockInfo>;

    async fn step(&mut self) -> Result<(), Self::Error> {
        if let Some(worker) = self.worker.take() {
            let state = self.state.subscribe();
            self.task.spawn(worker.run(state));
        }
        tokio::select! {
            head = self.head_stream.next() => {
                let head = head.ok_or(L1WatcherActorError::StreamEnded)?;
                self.state.send_modify(|state| state.head_l1 = Some(head));
            }
            finalized = self.finalized_stream.next() => {
                let finalized = finalized.ok_or(L1WatcherActorError::StreamEnded)?;
                self.state.send_modify(|state| state.finalized_l1 = Some(finalized));
            }
            safe = self.safe_stream.next() => {
                let safe = safe.ok_or(L1WatcherActorError::StreamEnded)?;
                self.state.send_modify(|state| state.safe_l1 = Some(safe));
            }
            Some(result) = self.task.join_next() => {
                match result {
                    Ok(result) => error!(target: "l1_watcher", ?result, "L1 watcher worker exited"),
                    Err(err) => error!(target: "l1_watcher", ?err, "L1 watcher worker failed"),
                }
                return Err(L1WatcherActorError::StreamEnded);
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests;
