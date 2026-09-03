//! [`NodeActor`] implementation for an L1 chain watcher that polls for L1 block updates over HTTP
//! RPC.

use crate::{NodeActor, actors::l1_watcher::error::L1WatcherActorError};
use async_trait::async_trait;
use futures::{
    Stream, StreamExt,
    future::{select_all, try_join_all},
};
use kona_protocol::BlockInfo;
use kona_rpc::L1WatcherQueries;
use tokio::{select, sync::watch};

use super::{L1WatcherChain, L1WatcherDerivationClient};

/// The L1 update that a single step of the [`L1WatcherActor`] observed.
///
/// The `select!` arms only classify the update; all handling happens afterwards, once the borrows
/// the arms hold on the actor's streams and query receivers have been released.
#[derive(Debug)]
enum L1WatcherEvent {
    /// A new L1 head block, or `None` if the head stream ended.
    Head(Option<BlockInfo>),
    /// A new finalized L1 block, or `None` if the finalized stream ended.
    Finalized(Option<BlockInfo>),
    /// A new safe L1 block, or `None` if the safe stream ended.
    Safe(Option<BlockInfo>),
    /// A query for one chain.
    Query {
        /// The index of the queried chain in the actor's chain vec.
        index: usize,
        /// The query, or `None` if the chain's query channel closed.
        query: Option<L1WatcherQueries>,
    },
}

/// The latest value of each L1 tag the watcher polls, as watch receivers. `None` until the
/// first poll of that tag lands.
#[derive(Debug, Clone)]
pub struct L1Watches {
    /// The latest L1 head.
    pub head: watch::Receiver<Option<BlockInfo>>,
    /// The latest finalized L1 block.
    pub finalized: watch::Receiver<Option<BlockInfo>>,
    /// The latest safe L1 block.
    pub safe: watch::Receiver<Option<BlockInfo>>,
}

/// An L1 chain watcher that polls the L1 head, safe and finalized tags over RPC.
///
/// Every polled tag is published on a watch channel ([`L1Watches`]); the chain view walks the
/// canonical chain from those. A single watcher serves N chains: each new head is also fanned
/// out to every chain's derivation actor, and each chain answers its own config queries. A
/// standalone kona-node runs this with a single chain.
#[derive(Debug)]
pub struct L1WatcherActor<BlockStream, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// The latest L1 head block.
    latest_head: watch::Sender<Option<BlockInfo>>,
    /// The latest finalized L1 block.
    latest_finalized: watch::Sender<Option<BlockInfo>>,
    /// The latest safe L1 block.
    latest_safe: watch::Sender<Option<BlockInfo>>,
    /// A stream over the latest head.
    head_stream: BlockStream,
    /// A stream over the finalized block accepted as canonical.
    finalized_stream: BlockStream,
    /// A stream over the safe block.
    safe_stream: BlockStream,
    /// The chains served by this watcher, rotated as their queries are served. Never empty.
    chains: Vec<L1WatcherChain<L1WatcherDerivationClient_>>,
}

impl<BlockStream, L1WatcherDerivationClient_>
    L1WatcherActor<BlockStream, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// Instantiate a new [`L1WatcherActor`] serving the given chains, with the receivers of the
    /// tags it publishes.
    ///
    /// # Panics
    /// Panics if `chains` is empty; a watcher without a chain to serve has nothing to do.
    pub fn new(
        head_stream: BlockStream,
        finalized_stream: BlockStream,
        safe_stream: BlockStream,
        chains: Vec<L1WatcherChain<L1WatcherDerivationClient_>>,
    ) -> (Self, L1Watches) {
        assert!(!chains.is_empty(), "the L1 watcher must serve at least one chain");

        let (latest_head, head) = watch::channel(None);
        let (latest_finalized, finalized) = watch::channel(None);
        let (latest_safe, safe) = watch::channel(None);
        let actor = Self {
            latest_head,
            latest_finalized,
            latest_safe,
            head_stream,
            finalized_stream,
            safe_stream,
            chains,
        };
        (actor, L1Watches { head, finalized, safe })
    }
}

#[async_trait]
impl<BlockStream, L1WatcherDerivationClient_> NodeActor
    for L1WatcherActor<BlockStream, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send + 'static,
    L1WatcherDerivationClient_: L1WatcherDerivationClient + 'static,
{
    type Error = L1WatcherActorError<BlockInfo>;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let event = select! {
            new_head = self.head_stream.next() => L1WatcherEvent::Head(new_head),
            new_finalized = self.finalized_stream.next() => L1WatcherEvent::Finalized(new_finalized),
            new_safe = self.safe_stream.next() => L1WatcherEvent::Safe(new_safe),
            // `select_all` panics on an empty iterator, which `new` rules out.
            (query, index, _) = select_all(
                self.chains.iter_mut().map(|chain| Box::pin(chain.inbound_queries.recv()))
            ) => L1WatcherEvent::Query { index, query },
        };

        match event {
            L1WatcherEvent::Head(None) |
            L1WatcherEvent::Finalized(None) |
            L1WatcherEvent::Safe(None) => Err(L1WatcherActorError::StreamEnded),
            L1WatcherEvent::Head(Some(head_block_info)) => {
                // Send the head update event to all consumers.
                self.latest_head.send_replace(Some(head_block_info));

                // Fan out to every chain together. Each derivation queue is bounded, so serial
                // awaits would let one full queue park the watcher and stall every other chain.
                try_join_all(
                    self.chains.iter().map(|chain| chain.send_new_l1_head(head_block_info)),
                )
                .await?;

                Ok(())
            }
            L1WatcherEvent::Finalized(Some(finalized_block_info)) => {
                self.latest_finalized.send_replace(Some(finalized_block_info));
                Ok(())
            }
            L1WatcherEvent::Safe(Some(safe_block_info)) => {
                self.latest_safe.send_replace(Some(safe_block_info));
                Ok(())
            }
            L1WatcherEvent::Query { index, query: Some(query) } => {
                let chain_id = self.chains[index].chain_id();

                let L1WatcherQueries::Config(sender) = query;
                if let Err(e) = sender.send((*self.chains[index].rollup_config).clone()) {
                    warn!(target: "l1_watcher", chain_id, error = ?e, "Failed to send L1 config to the query sender");
                }

                // The chain just served is polled last next time.
                self.chains.rotate_left(index + 1);

                Ok(())
            }
            L1WatcherEvent::Query { index, query: None } => {
                // This stops the watcher for every chain, so name the chain that caused it.
                error!(target: "l1_watcher", chain_id = self.chains[index].chain_id(), "L1 watcher query channel closed unexpectedly, exiting query processor task.");
                Err(L1WatcherActorError::StreamEnded)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{super::client::MockL1WatcherDerivationClient, *};
    use crate::DerivationClientResult;
    use alloy_primitives::{Address, B256};
    use futures::stream;
    use kona_genesis::RollupConfig;
    use kona_rpc::L1WatcherQueries;
    use std::{
        pin::Pin,
        sync::{
            Arc,
            atomic::{AtomicUsize, Ordering},
        },
    };
    use tokio::sync::{mpsc, oneshot};

    /// The block streams of an actor must have the same type, so tests box them.
    type TestStream = Pin<Box<dyn Stream<Item = BlockInfo> + Send>>;

    fn chain_at(index: u8) -> Address {
        Address::repeat_byte(index)
    }

    fn head(number: u64) -> BlockInfo {
        BlockInfo::new(B256::repeat_byte(number as u8), number, B256::ZERO, number * 12)
    }

    fn test_chain<Client>(
        index: u8,
        client: Client,
    ) -> (L1WatcherChain<Client>, mpsc::Sender<L1WatcherQueries>) {
        let rollup_config = Arc::new(RollupConfig {
            l1_system_config_address: chain_at(index),
            ..Default::default()
        });
        let (query_tx, query_rx) = mpsc::channel(16);
        (L1WatcherChain::new(rollup_config, client, query_rx), query_tx)
    }

    /// Builds `count` chains via [`test_chain`], the client of each built by `client`.
    fn test_chains<Client>(
        count: u8,
        mut client: impl FnMut(u8) -> Client,
    ) -> (Vec<L1WatcherChain<Client>>, Vec<mpsc::Sender<L1WatcherQueries>>) {
        (0..count).map(|i| test_chain(i, client(i))).unzip()
    }

    fn test_stream(blocks: Vec<BlockInfo>) -> TestStream {
        Box::pin(stream::iter(blocks).chain(stream::pending()))
    }

    /// Builds an actor over `chains` whose streams yield the given blocks, then pend forever so
    /// they never race the branch under test.
    fn actor<Client: L1WatcherDerivationClient>(
        heads: Vec<BlockInfo>,
        finalized: Vec<BlockInfo>,
        safe: Vec<BlockInfo>,
        chains: Vec<L1WatcherChain<Client>>,
    ) -> (L1WatcherActor<TestStream, Client>, L1Watches) {
        L1WatcherActor::new(test_stream(heads), test_stream(finalized), test_stream(safe), chains)
    }

    /// A new L1 head reaches every chain's derivation actor and the head watch.
    #[tokio::test]
    async fn new_head_fans_out_to_every_chain() {
        const CHAINS: u8 = 3;

        let block = head(7);
        let (chains, handles) = test_chains(CHAINS, |_| {
            let mut client = MockL1WatcherDerivationClient::new();
            client
                .expect_send_new_l1_head()
                .times(1)
                .withf(move |b| *b == block)
                .returning(|_| Ok(()));
            client
        });

        let (mut actor, watches) = actor(vec![block], vec![], vec![], chains);
        actor.step().await.expect("step");

        assert_eq!(*watches.head.borrow(), Some(block));
        drop(handles);
    }

    /// Finalized and safe L1 blocks only update their watches; derivation is not told.
    #[tokio::test]
    async fn finalized_and_safe_blocks_update_their_watches() {
        const CHAINS: u8 = 2;

        let finalized = head(9);
        let safe = head(11);
        let (chains, handles) = test_chains(CHAINS, |_| {
            let mut client = MockL1WatcherDerivationClient::new();
            client.expect_send_new_l1_head().times(0);
            client
        });

        let (mut actor, watches) = actor(vec![], vec![finalized], vec![safe], chains);
        actor.step().await.expect("step");
        actor.step().await.expect("step");

        assert_eq!(*watches.finalized.borrow(), Some(finalized));
        assert_eq!(*watches.safe.borrow(), Some(safe));
        drop(handles);
    }

    /// A derivation client that records how many of the watcher's sends had been started by the
    /// time its own send finished.
    ///
    /// Every send bumps the shared `started` counter, yields once so its peers get a chance to
    /// run, then records what `started` had reached. Sends that the watcher starts together all
    /// observe every chain; sends the watcher awaits one at a time observe 1, 2, ... in turn.
    #[derive(Debug)]
    struct ConcurrencyProbeClient {
        started: Arc<AtomicUsize>,
        observed: Arc<AtomicUsize>,
    }

    #[async_trait]
    impl L1WatcherDerivationClient for ConcurrencyProbeClient {
        async fn send_new_l1_head(&self, _block: BlockInfo) -> DerivationClientResult<()> {
            self.started.fetch_add(1, Ordering::SeqCst);
            tokio::task::yield_now().await;
            self.observed.store(self.started.load(Ordering::SeqCst), Ordering::SeqCst);
            Ok(())
        }
    }

    /// The head fan-out starts every chain's send together: a fan-out awaited one chain at a
    /// time would let a chain whose derivation queue is full park the watcher, stalling every
    /// other chain's L1 updates and queries.
    #[tokio::test]
    async fn head_fan_out_starts_every_chain_together() {
        const CHAINS: u8 = 3;

        let started = Arc::new(AtomicUsize::new(0));
        let mut observed = Vec::new();

        let (chains, handles) = test_chains(CHAINS, |_| {
            let chain_observed = Arc::new(AtomicUsize::new(0));
            observed.push(Arc::clone(&chain_observed));
            ConcurrencyProbeClient { started: Arc::clone(&started), observed: chain_observed }
        });

        let (mut actor, _watches) = actor(vec![head(11)], vec![], vec![], chains);
        actor.step().await.expect("step");

        for (index, chain_observed) in observed.iter().enumerate() {
            assert_eq!(
                chain_observed.load(Ordering::SeqCst),
                CHAINS as usize,
                "chain {index} finished its send before its peers had started theirs, so the \
                 fan-out is serialised and one chain can block the others"
            );
        }

        // Keep the query senders alive for the duration of the step.
        drop(handles);
    }

    /// A chain whose query channel is never empty does not starve the chains after it.
    #[tokio::test]
    async fn query_polling_rotates_across_chains() {
        const CHAINS: u8 = 3;

        let (chains, handles) = test_chains(CHAINS, |_| MockL1WatcherDerivationClient::new());
        let (mut actor, _watches) = actor(vec![], vec![], vec![], chains);

        // Keep chain 0's channel non-empty for the whole test: with a fixed poll order it would
        // win every round and no other chain would ever be reached.
        let mut chain_0_rxs = (0..4)
            .map(|_| {
                let (tx, rx) = oneshot::channel();
                handles[0].try_send(L1WatcherQueries::Config(tx)).unwrap();
                rx
            })
            .collect::<Vec<_>>();

        let mut waiting_rxs = [1usize, 2].map(|chain| {
            let (tx, rx) = oneshot::channel();
            handles[chain].try_send(L1WatcherQueries::Config(tx)).unwrap();
            rx
        });

        // Chain 0 takes the first round; the rotation must then hand one round to chain 1 and
        // one to chain 2 before chain 0 gets another turn. The last two steps go back to chain 0:
        // by then it sits at the end of the rotated order, so serving it exercises the
        // `rotate_left(len)` edge.
        for _ in 0..5 {
            actor.step().await.expect("step");
        }

        for (index, rx) in waiting_rxs.iter_mut().enumerate() {
            let chain = index as u8 + 1;
            assert_eq!(
                rx.try_recv()
                    .unwrap_or_else(|_| panic!(
                        "chain {chain}'s query was starved by chain 0's non-empty channel"
                    ))
                    .l1_system_config_address,
                chain_at(chain),
            );
        }

        // Steps 1, 4 and 5 each served one of chain 0's queries; the fourth is still queued.
        let answered =
            chain_0_rxs.iter_mut().map(|rx| rx.try_recv().is_ok()).filter(|&ok| ok).count();
        assert_eq!(answered, 3, "chain 0 was not served on exactly the rounds it was due");
    }

    /// A closed query channel stops the watcher: `step` returns
    /// [`L1WatcherActorError::StreamEnded`].
    #[tokio::test]
    async fn closed_query_channel_ends_the_stream() {
        const CHAINS: u8 = 3;

        let (chains, mut handles) = test_chains(CHAINS, |_| MockL1WatcherDerivationClient::new());
        let (mut actor, _watches) = actor(vec![], vec![], vec![], chains);

        // Drop one chain's query sender; its `recv` then resolves to `None`.
        drop(handles.remove(1));

        assert!(matches!(actor.step().await, Err(L1WatcherActorError::StreamEnded)));
    }

    /// A config query is answered with the rollup config of the chain it was sent to.
    #[tokio::test]
    async fn config_query_is_answered_per_chain() {
        const CHAINS: u8 = 3;

        let (chains, handles) = test_chains(CHAINS, |_| MockL1WatcherDerivationClient::new());
        let (mut actor, _watches) = actor(vec![], vec![], vec![], chains);

        for index in 0..CHAINS {
            let (tx, rx) = oneshot::channel();
            handles[index as usize].send(L1WatcherQueries::Config(tx)).await.unwrap();
            actor.step().await.expect("step");
            assert_eq!(rx.await.unwrap().l1_system_config_address, chain_at(index));
        }
    }
}
