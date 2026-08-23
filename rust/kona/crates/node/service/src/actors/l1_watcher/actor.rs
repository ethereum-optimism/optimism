//! [`NodeActor`] implementation for an L1 chain watcher that polls for L1 block updates over HTTP
//! RPC.

use crate::{NodeActor, actors::l1_watcher::error::L1WatcherActorError};
use alloy_eips::BlockId;
use alloy_provider::Provider;
use async_trait::async_trait;
use futures::{
    Stream, StreamExt,
    future::{select_all, try_join_all},
};
use kona_protocol::BlockInfo;
use kona_rpc::{L1State, L1WatcherQueries};
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
    /// A query for one chain.
    Query {
        /// The index of the queried chain in the actor's chain vec.
        index: usize,
        /// The query, or `None` if the chain's query channel closed.
        query: Option<L1WatcherQueries>,
    },
}

/// An L1 chain watcher that checks for L1 block updates over RPC.
///
/// A single watcher serves N chains: the L1 head and finalized streams are shared, and every
/// update is fanned out to each chain's derivation actor. The system config log filter and the
/// unsafe block signer updates are per chain. A standalone kona-node runs this with a single
/// chain.
#[derive(Debug)]
pub struct L1WatcherActor<BlockStream, L1Provider, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// The L1 provider.
    l1_provider: L1Provider,
    /// The latest L1 head block.
    latest_head: watch::Sender<Option<BlockInfo>>,
    /// A stream over the latest head.
    head_stream: BlockStream,
    /// A stream over the finalized block accepted as canonical.
    finalized_stream: BlockStream,
    /// The chains served by this watcher, rotated as their queries are served. Never empty.
    chains: Vec<L1WatcherChain<L1WatcherDerivationClient_>>,
}

impl<BlockStream, L1Provider, L1WatcherDerivationClient_>
    L1WatcherActor<BlockStream, L1Provider, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send,
    L1Provider: Provider,
    L1WatcherDerivationClient_: L1WatcherDerivationClient,
{
    /// Instantiate a new [`L1WatcherActor`] serving the given chains.
    ///
    /// # Panics
    /// Panics if `chains` is empty; a watcher without a chain to serve has nothing to do.
    pub fn new(
        l1_provider: L1Provider,
        l1_head_updates_tx: watch::Sender<Option<BlockInfo>>,
        head_stream: BlockStream,
        finalized_stream: BlockStream,
        chains: Vec<L1WatcherChain<L1WatcherDerivationClient_>>,
    ) -> Self {
        assert!(!chains.is_empty(), "the L1 watcher must serve at least one chain");

        Self { l1_provider, latest_head: l1_head_updates_tx, head_stream, finalized_stream, chains }
    }

    /// Reads the L1 view answered by [`L1WatcherQueries::L1State`] from the L1 provider.
    ///
    /// Takes the provider rather than `&self`. `step` sits behind `#[async_trait]`, so a `&Self`
    /// held across an await would need `BlockStream: Sync`, which the two streams do not satisfy.
    async fn l1_state(l1_provider: &L1Provider, current_l1: Option<BlockInfo>) -> L1State {
        let read = move |tag: BlockId, what: &'static str| async move {
            match l1_provider.get_block(tag).await {
                Ok(block) => block,
                Err(e) => {
                    warn!(target: "l1_watcher", error = ?e, tag = what, "failed to query l1 provider for block");
                    None
                }
            }
            .map(|block| block.into_consensus().into())
        };

        // Issued together: this runs inline in `step`. Three serial round trips would park the
        // watcher, and with it every chain's head and finalized fan-out.
        let (head_l1, finalized_l1, safe_l1) = tokio::join!(
            read(BlockId::latest(), "latest"),
            read(BlockId::finalized(), "finalized"),
            read(BlockId::safe(), "safe"),
        );

        L1State { current_l1, current_l1_finalized: finalized_l1, head_l1, safe_l1, finalized_l1 }
    }
}

#[async_trait]
impl<BlockStream, L1Provider, L1WatcherDerivationClient_> NodeActor
    for L1WatcherActor<BlockStream, L1Provider, L1WatcherDerivationClient_>
where
    BlockStream: Stream<Item = BlockInfo> + Unpin + Send + 'static,
    L1Provider: Provider + 'static,
    L1WatcherDerivationClient_: L1WatcherDerivationClient + 'static,
{
    type Error = L1WatcherActorError<BlockInfo>;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let event = select! {
            new_head = self.head_stream.next() => L1WatcherEvent::Head(new_head),
            new_finalized = self.finalized_stream.next() => L1WatcherEvent::Finalized(new_finalized),
            // `select_all` panics on an empty iterator, which `new` rules out.
            (query, index, _) = select_all(
                self.chains.iter_mut().map(|chain| Box::pin(chain.inbound_queries.recv()))
            ) => L1WatcherEvent::Query { index, query },
        };

        match event {
            L1WatcherEvent::Head(None) | L1WatcherEvent::Finalized(None) => {
                Err(L1WatcherActorError::StreamEnded)
            }
            L1WatcherEvent::Head(Some(head_block_info)) => {
                // Send the head update event to all consumers.
                self.latest_head.send_replace(Some(head_block_info));

                // Fan out to every chain before the log queries below. Each derivation queue is
                // bounded, so serial awaits let one full queue park the watcher and stall every
                // other chain.
                try_join_all(
                    self.chains.iter().map(|chain| chain.send_new_l1_head(head_block_info)),
                )
                .await?;
                // Fetch every chain's system config logs together, so the latency is one round
                // trip rather than N and one chain's failing request does not stop the others'
                // from being made. The first error still aborts the round, and the signer updates
                // already delivered stay delivered.
                try_join_all(self.chains.iter().map(|chain| {
                    chain.process_system_config_logs(&self.l1_provider, head_block_info)
                }))
                .await?;

                Ok(())
            }
            L1WatcherEvent::Finalized(Some(finalized_block_info)) => {
                // Started together for the same reason as the head fan-out above: one chain whose
                // derivation queue is full must not park the watcher and stall every other chain.
                try_join_all(
                    self.chains
                        .iter()
                        .map(|chain| chain.send_finalized_l1_block(finalized_block_info)),
                )
                .await?;

                Ok(())
            }
            L1WatcherEvent::Query { index, query: Some(query) } => {
                let chain_id = self.chains[index].chain_id();

                match query {
                    L1WatcherQueries::Config(sender) => {
                        if let Err(e) = sender.send((*self.chains[index].rollup_config).clone()) {
                            warn!(target: "l1_watcher", chain_id, error = ?e, "Failed to send L1 config to the query sender");
                        }
                    }
                    L1WatcherQueries::L1State(sender) => {
                        let current_l1 = *self.latest_head.borrow();
                        if let Err(e) =
                            sender.send(Self::l1_state(&self.l1_provider, current_l1).await)
                        {
                            warn!(target: "l1_watcher", chain_id, error = ?e, "Failed to send L1 state to the query sender");
                        }
                    }
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
    use alloy_primitives::{Address, B256, U256};
    use alloy_provider::{ProviderBuilder, mock::Asserter};
    use alloy_rpc_types_eth::{Block as RpcBlock, Log as RpcLog};
    use futures::stream;
    use kona_genesis::{
        CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC, RollupConfig, SystemConfigUpdateKind,
    };
    use kona_rpc::L1WatcherQueries;
    use std::{
        pin::Pin,
        sync::{
            Arc,
            atomic::{AtomicUsize, Ordering},
        },
    };
    use tokio::sync::{mpsc, oneshot};

    /// Both block streams of an actor must have the same type, so tests box them.
    type TestStream = Pin<Box<dyn Stream<Item = BlockInfo> + Send>>;

    /// The channel ends of a test chain that the test itself asserts on.
    struct ChainHandles {
        query_tx: mpsc::Sender<L1WatcherQueries>,
        signer_rx: mpsc::Receiver<Address>,
    }

    fn chain_at(index: u8) -> Address {
        Address::repeat_byte(index)
    }

    fn signer_at(index: u8) -> Address {
        Address::repeat_byte(index + 0x80)
    }

    fn head(number: u64) -> BlockInfo {
        BlockInfo::new(B256::repeat_byte(number as u8), number, B256::ZERO, number * 12)
    }

    /// An `eth_getLogs` result carrying a single unsafe block signer update.
    fn signer_update_log(system_config_address: Address, signer: Address) -> RpcLog {
        let mut data = Vec::with_capacity(96);
        data.extend_from_slice(&U256::from(32).to_be_bytes::<32>());
        data.extend_from_slice(&U256::from(32).to_be_bytes::<32>());
        data.extend_from_slice(signer.into_word().as_slice());

        RpcLog {
            inner: alloy_primitives::Log {
                address: system_config_address,
                data: alloy_primitives::LogData::new_unchecked(
                    vec![
                        CONFIG_UPDATE_TOPIC,
                        CONFIG_UPDATE_EVENT_VERSION_0,
                        B256::from(U256::from(SystemConfigUpdateKind::UnsafeBlockSigner as u64)),
                    ],
                    data.into(),
                ),
            },
            ..Default::default()
        }
    }

    /// An `eth_getBlockByNumber` result whose header fields identify which tag it answered.
    ///
    /// [`BlockInfo`] takes its hash from `hash_slow()` rather than the RPC `hash` field, so the
    /// number, parent hash and timestamp are what a test can pin a response to.
    fn rpc_block(number: u64, parent: u8) -> RpcBlock {
        let mut block: RpcBlock = RpcBlock::default();
        block.header.inner.number = number;
        block.header.inner.parent_hash = B256::repeat_byte(parent);
        block.header.inner.timestamp = number * 12;
        block
    }

    fn test_chain<Client>(index: u8, client: Client) -> (L1WatcherChain<Client>, ChainHandles) {
        let rollup_config = Arc::new(RollupConfig {
            l1_system_config_address: chain_at(index),
            ..Default::default()
        });
        let (signer_tx, signer_rx) = mpsc::channel(16);
        let (query_tx, query_rx) = mpsc::channel(16);

        (
            L1WatcherChain::new(rollup_config, client, signer_tx, query_rx),
            ChainHandles { query_tx, signer_rx },
        )
    }

    /// Builds `count` chains via [`test_chain`], the client of each built by `client`.
    fn test_chains<Client>(
        count: u8,
        mut client: impl FnMut(u8) -> Client,
    ) -> (Vec<L1WatcherChain<Client>>, Vec<ChainHandles>) {
        (0..count).map(|i| test_chain(i, client(i))).unzip()
    }

    /// Builds an actor over `chains` whose head stream yields `heads` and whose finalized stream
    /// yields `finalized`, then pends forever so it never races the branch under test.
    fn actor<Client: L1WatcherDerivationClient>(
        asserter: Asserter,
        heads: Vec<BlockInfo>,
        finalized: Vec<BlockInfo>,
        chains: Vec<L1WatcherChain<Client>>,
    ) -> (L1WatcherActor<TestStream, impl Provider, Client>, watch::Receiver<Option<BlockInfo>>)
    {
        let (latest_head_tx, latest_head_rx) = watch::channel(None);
        let actor = L1WatcherActor::new(
            ProviderBuilder::default().connect_mocked_client(asserter),
            latest_head_tx,
            Box::pin(stream::iter(heads).chain(stream::pending())) as TestStream,
            Box::pin(stream::iter(finalized).chain(stream::pending())) as TestStream,
            chains,
        );
        (actor, latest_head_rx)
    }

    /// A new L1 head reaches every chain's derivation actor, and each chain's system config logs
    /// are routed to that chain's unsafe block signer channel.
    ///
    /// The chains' log requests are issued concurrently, and [`Asserter`] answers them from one
    /// FIFO queue with no per-request routing. `try_join_all` polls its futures in the order it
    /// was given them, so response `i` still answers chain `i`'s request: a chain that read
    /// another chain's response would receive the wrong signer and fail the assertions below.
    #[tokio::test]
    async fn new_head_fans_out_to_every_chain() {
        const CHAINS: u8 = 3;

        let block = head(7);
        let asserter = Asserter::new();
        let (chains, mut handles) = test_chains(CHAINS, |index| {
            let mut client = MockL1WatcherDerivationClient::new();
            client
                .expect_send_new_l1_head()
                .times(1)
                .withf(move |b| *b == block)
                .returning(|_| Ok(()));
            client.expect_send_finalized_l1_block().times(0);

            asserter.push_success(&vec![signer_update_log(chain_at(index), signer_at(index))]);

            client
        });

        let (mut actor, latest_head_rx) = actor(asserter, vec![block], vec![], chains);
        actor.step().await.expect("step");

        assert_eq!(*latest_head_rx.borrow(), Some(block));
        for (index, handle) in handles.iter_mut().enumerate() {
            assert_eq!(
                handle.signer_rx.try_recv().expect("signer update"),
                signer_at(index as u8),
                "chain {index} did not receive its own unsafe block signer update"
            );
        }
    }

    /// A new finalized L1 block reaches every chain's derivation actor.
    #[tokio::test]
    async fn finalized_block_fans_out_to_every_chain() {
        const CHAINS: u8 = 3;

        let block = head(9);
        let (chains, handles) = test_chains(CHAINS, |_| {
            let mut client = MockL1WatcherDerivationClient::new();
            client
                .expect_send_finalized_l1_block()
                .times(1)
                .withf(move |b| *b == block)
                .returning(|_| Ok(()));
            client.expect_send_new_l1_head().times(0);
            client
        });

        let (mut actor, _latest_head_rx) = actor(Asserter::new(), vec![], vec![block], chains);
        actor.step().await.expect("step");

        // Keep the query senders alive for the duration of the step.
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

    impl ConcurrencyProbeClient {
        async fn probe(&self) {
            self.started.fetch_add(1, Ordering::SeqCst);
            tokio::task::yield_now().await;
            self.observed.store(self.started.load(Ordering::SeqCst), Ordering::SeqCst);
        }
    }

    #[async_trait]
    impl L1WatcherDerivationClient for ConcurrencyProbeClient {
        async fn send_finalized_l1_block(&self, _block: BlockInfo) -> DerivationClientResult<()> {
            self.probe().await;
            Ok(())
        }

        async fn send_new_l1_head(&self, _block: BlockInfo) -> DerivationClientResult<()> {
            self.probe().await;
            Ok(())
        }
    }

    /// Steps an actor over `CHAINS` probe chains and asserts that every chain's send was already
    /// under way before any of them finished.
    ///
    /// A fan-out awaited one chain at a time would let a chain whose derivation queue is full park
    /// the watcher, stalling every other chain's L1 updates and queries.
    async fn assert_fan_out_starts_every_chain_together(
        heads: Vec<BlockInfo>,
        finalized: Vec<BlockInfo>,
    ) {
        const CHAINS: u8 = 3;

        let started = Arc::new(AtomicUsize::new(0));
        let asserter = Asserter::new();
        let mut observed = Vec::new();

        let (chains, handles) = test_chains(CHAINS, |_| {
            // Only the head fan-out reaches the log queries; the finalized case leaves these
            // responses unread.
            asserter.push_success(&Vec::<RpcLog>::new());

            let chain_observed = Arc::new(AtomicUsize::new(0));
            observed.push(Arc::clone(&chain_observed));
            ConcurrencyProbeClient { started: Arc::clone(&started), observed: chain_observed }
        });

        let (mut actor, _latest_head_rx) = actor(asserter, heads, finalized, chains);
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

    /// The head fan-out starts every chain's send together.
    #[tokio::test]
    async fn head_fan_out_starts_every_chain_together() {
        assert_fan_out_starts_every_chain_together(vec![head(11)], vec![]).await;
    }

    /// The finalized fan-out starts every chain's send together.
    #[tokio::test]
    async fn finalized_fan_out_starts_every_chain_together() {
        assert_fan_out_starts_every_chain_together(vec![], vec![head(13)]).await;
    }

    /// A chain whose query channel is never empty does not starve the chains after it.
    #[tokio::test]
    async fn query_polling_rotates_across_chains() {
        const CHAINS: u8 = 3;

        let (chains, handles) = test_chains(CHAINS, |_| MockL1WatcherDerivationClient::new());
        let (mut actor, _latest_head_rx) = actor(Asserter::new(), vec![], vec![], chains);

        // Keep chain 0's channel non-empty for the whole test: with a fixed poll order it would
        // win every round and no other chain would ever be reached.
        let mut chain_0_rxs = (0..4)
            .map(|_| {
                let (tx, rx) = oneshot::channel();
                handles[0].query_tx.try_send(L1WatcherQueries::Config(tx)).unwrap();
                rx
            })
            .collect::<Vec<_>>();

        let mut waiting_rxs = [1usize, 2].map(|chain| {
            let (tx, rx) = oneshot::channel();
            handles[chain].query_tx.try_send(L1WatcherQueries::Config(tx)).unwrap();
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
        let (mut actor, _latest_head_rx) = actor(Asserter::new(), vec![], vec![], chains);

        // Drop one chain's query sender; its `recv` then resolves to `None`.
        drop(handles.remove(1));

        assert!(matches!(actor.step().await, Err(L1WatcherActorError::StreamEnded)));
    }

    /// A config query is answered with the rollup config of the chain it was sent to.
    #[tokio::test]
    async fn config_query_is_answered_per_chain() {
        const CHAINS: u8 = 3;

        let (chains, handles) = test_chains(CHAINS, |_| MockL1WatcherDerivationClient::new());
        let (mut actor, _latest_head_rx) = actor(Asserter::new(), vec![], vec![], chains);

        for index in 0..CHAINS {
            let (tx, rx) = oneshot::channel();
            handles[index as usize].query_tx.send(L1WatcherQueries::Config(tx)).await.unwrap();
            actor.step().await.expect("step");
            assert_eq!(rx.await.unwrap().l1_system_config_address, chain_at(index));
        }
    }

    /// An `L1State` query is answered on the querying chain's oneshot, with the actor's current
    /// head and one block per provider read.
    ///
    /// The three `get_block` calls are issued concurrently, so like the head arm's log fan-out
    /// this does depend on [`Asserter`]'s FIFO answering lining up with a set of concurrent
    /// requests. `tokio::join!` polls its futures in the order written on the first poll - its
    /// rotator starts at zero skips - so the reads are issued latest, finalized, safe and
    /// response `i` still answers read `i`.
    ///
    /// What the distinct block numbers pin down is which response lands in which field, and the
    /// drained queue that exactly three reads are made. The tags themselves are *not* covered:
    /// [`Asserter`] answers from its queue without looking at the request, so swapping
    /// `BlockId::finalized()` for `BlockId::safe()` in the arm keeps every response in the same
    /// position and this test still passes.
    #[tokio::test]
    async fn l1_state_query_is_answered_with_the_head_and_every_tag() {
        const CHAINS: u8 = 3;
        const QUERIED: u8 = 1;

        let block = head(7);
        let asserter = Asserter::new();
        let (chains, handles) = test_chains(CHAINS, |_| {
            let mut client = MockL1WatcherDerivationClient::new();
            client.expect_send_new_l1_head().times(1).returning(|_| Ok(()));

            // No config updates: this test is about the query arm, not the log fan-out.
            asserter.push_success(&Vec::<RpcLog>::new());

            client
        });

        let (mut actor, _latest_head_rx) = actor(asserter.clone(), vec![block], vec![], chains);

        // Take the head first, so that the query below has a `current_l1` to report.
        actor.step().await.expect("head step");

        // Answered in the order the arm asks for them: latest, then finalized, then safe.
        asserter.push_success(&rpc_block(100, 0xa1));
        asserter.push_success(&rpc_block(200, 0xa2));
        asserter.push_success(&rpc_block(300, 0xa3));

        let (tx, rx) = oneshot::channel();
        handles[QUERIED as usize].query_tx.send(L1WatcherQueries::L1State(tx)).await.unwrap();
        actor.step().await.expect("query step");

        let state = rx.await.expect("the queried chain's oneshot was never answered");

        assert_eq!(state.current_l1, Some(block), "current_l1 is not the actor's latest head");

        let head_l1 = state.head_l1.expect("head_l1");
        assert_eq!((head_l1.number, head_l1.parent_hash), (100, B256::repeat_byte(0xa1)));

        let finalized_l1 = state.finalized_l1.expect("finalized_l1");
        assert_eq!((finalized_l1.number, finalized_l1.parent_hash), (200, B256::repeat_byte(0xa2)));

        let safe_l1 = state.safe_l1.expect("safe_l1");
        assert_eq!((safe_l1.number, safe_l1.parent_hash), (300, B256::repeat_byte(0xa3)));

        assert_eq!(
            state.current_l1_finalized, state.finalized_l1,
            "current_l1_finalized is documented as matching finalized_l1"
        );

        assert!(
            asserter.read_q().is_empty(),
            "the arm made fewer provider reads than the three responses queued for it"
        );
    }
}
