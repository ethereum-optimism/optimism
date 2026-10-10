//! Shared L1 observation with independent per-chain consumers.

use super::{L1WatcherActorError, L1WatcherChain, L1WatcherDerivationClient};
use crate::NodeActor;
use alloy_provider::Provider;
use async_trait::async_trait;
use futures::{Stream, StreamExt};
use kona_protocol::BlockInfo;
use tokio::{sync::watch, task::JoinSet};

#[derive(Debug)]
struct ChainExit {
    chain_id: u64,
    result: Result<(), L1WatcherActorError<BlockInfo>>,
}

/// Observes L1 once and distributes the latest head and finality to independent chain workers.
///
/// Head notifications are already allowed to skip blocks. Keeping only the latest observation
/// lets a slow chain catch up without either blocking its peers or growing a queue indefinitely.
/// Workers and their in-flight RPCs are cancelled when the watcher is dropped.
#[derive(Debug)]
pub struct L1WatcherActor<BlockStream, L1Provider, Client>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider: Provider + Clone,
    Client: L1WatcherDerivationClient,
{
    l1_provider: L1Provider,
    latest_head: watch::Sender<Option<BlockInfo>>,
    latest_finalized: watch::Sender<Option<BlockInfo>>,
    head_stream: BlockStream,
    finalized_stream: BlockStream,
    /// Moved into workers on the first step, so construction does not require a runtime.
    chains: Vec<L1WatcherChain<Client>>,
    workers: JoinSet<ChainExit>,
}

impl<BlockStream, L1Provider, Client> L1WatcherActor<BlockStream, L1Provider, Client>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider: Provider + Clone,
    Client: L1WatcherDerivationClient,
{
    /// Constructs a watcher. Workers are started by the first call to `step`.
    ///
    /// # Panics
    /// Panics when there are no chains to serve.
    pub fn new(
        l1_provider: L1Provider,
        latest_head: watch::Sender<Option<BlockInfo>>,
        head_stream: BlockStream,
        finalized_stream: BlockStream,
        chains: Vec<L1WatcherChain<Client>>,
    ) -> Self {
        assert!(!chains.is_empty(), "the L1 watcher must serve at least one chain");
        let (latest_finalized, _) = watch::channel(None);
        Self {
            l1_provider,
            latest_head,
            latest_finalized,
            head_stream,
            finalized_stream,
            chains,
            workers: JoinSet::new(),
        }
    }
}

#[async_trait]
impl<BlockStream, L1Provider, Client> NodeActor for L1WatcherActor<BlockStream, L1Provider, Client>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send + 'static,
    L1Provider: Provider + Clone + 'static,
    Client: L1WatcherDerivationClient + 'static,
{
    type Error = L1WatcherActorError<BlockInfo>;

    async fn step(&mut self) -> Result<(), Self::Error> {
        for chain in self.chains.drain(..) {
            let provider = self.l1_provider.clone();
            let head = self.latest_head.subscribe();
            let finalized = self.latest_finalized.subscribe();
            self.workers.spawn(async move {
                let chain_id = chain.chain_id();
                ChainExit { chain_id, result: chain.run(provider, head, finalized).await }
            });
        }
        if self.workers.is_empty() {
            return Err(L1WatcherActorError::StreamEnded);
        }
        tokio::select! {
            head = self.head_stream.next() => {
                self.latest_head.send_replace(Some(head.ok_or(L1WatcherActorError::StreamEnded)?));
            }
            finalized = self.finalized_stream.next() => {
                self.latest_finalized.send_replace(Some(finalized.ok_or(L1WatcherActorError::StreamEnded)?));
            }
            Some(exit) = self.workers.join_next() => {
                match exit {
                    Ok(exit) => error!(target: "l1_watcher", chain_id = exit.chain_id, result = ?exit.result, "Chain detached from L1 watcher"),
                    Err(err) => error!(target: "l1_watcher", ?err, "L1 chain worker failed"),
                }
                // A dead chain cannot be revived by retrying its closed channels. Detach it;
                // the owning chain supervisor is responsible for rebuilding that chain.
                if self.workers.is_empty() {
                    return Err(L1WatcherActorError::StreamEnded);
                }
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests;
