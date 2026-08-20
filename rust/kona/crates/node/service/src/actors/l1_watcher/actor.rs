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
    /// A query for the chain at the given index, or `None` if its query channel closed.
    Query(usize, Option<L1WatcherQueries>),
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
    /// The chains served by this watcher. Never empty.
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
            (query, chain_index, _) = select_all(
                self.chains.iter_mut().map(|chain| Box::pin(chain.inbound_queries.recv()))
            ) => L1WatcherEvent::Query(chain_index, query),
        };

        match event {
            L1WatcherEvent::Head(None) | L1WatcherEvent::Finalized(None) => {
                Err(L1WatcherActorError::StreamEnded)
            }
            L1WatcherEvent::Head(Some(head_block_info)) => {
                // Send the head update event to all consumers.
                self.latest_head.send_replace(Some(head_block_info));

                // Fan the head out to every chain before the log queries below, so that a slow
                // query for one chain does not hold up derivation on the others.
                for chain in &self.chains {
                    chain.send_new_l1_head(head_block_info).await?;
                }
                // Fetch every chain's system config logs together, so that the round-trip
                // latency is one request rather than N and one chain's failing request does not
                // stop the others' from being made. The first error still aborts the round, and
                // the signer updates already delivered stay delivered.
                try_join_all(self.chains.iter().map(|chain| {
                    chain.process_system_config_logs(&self.l1_provider, head_block_info)
                }))
                .await?;

                Ok(())
            }
            L1WatcherEvent::Finalized(Some(finalized_block_info)) => {
                for chain in &self.chains {
                    chain.send_finalized_l1_block(finalized_block_info).await?;
                }

                Ok(())
            }
            L1WatcherEvent::Query(chain_index, Some(query)) => {
                match query {
                    L1WatcherQueries::Config(sender) => {
                        if let Err(e) =
                            sender.send((*self.chains[chain_index].rollup_config).clone())
                        {
                            warn!(target: "l1_watcher", error = ?e, "Failed to send L1 config to the query sender");
                        }
                    }
                    L1WatcherQueries::L1State(sender) => {
                        let current_l1 = *self.latest_head.borrow();
                        if let Err(e) = sender.send(l1_state(&self.l1_provider, current_l1).await) {
                            warn!(target: "l1_watcher", error = ?e, "Failed to send L1 state to the query sender");
                        }
                    }
                }

                Ok(())
            }
            L1WatcherEvent::Query(_, None) => {
                error!(target: "l1_watcher", "L1 watcher query channel closed unexpectedly, exiting query processor task.");
                Err(L1WatcherActorError::StreamEnded)
            }
        }
    }
}

/// Reads the L1 view answered by [`L1WatcherQueries::L1State`] from the L1 provider.
async fn l1_state(l1_provider: &impl Provider, current_l1: Option<BlockInfo>) -> L1State {
    let head_l1 = match l1_provider.get_block(BlockId::latest()).await {
            Ok(block) => block,
            Err(e) => {
                warn!(target: "l1_watcher", error = ?e, "failed to query l1 provider for latest head block");
                None
            }}.map(|block| block.into_consensus().into());

    let finalized_l1 = match l1_provider.get_block(BlockId::finalized()).await {
            Ok(block) => block,
            Err(e) => {
                warn!(target: "l1_watcher", error = ?e, "failed to query l1 provider for latest finalized block");
                None
            }}.map(|block| block.into_consensus().into());

    let safe_l1 = match l1_provider.get_block(BlockId::safe()).await {
            Ok(block) => block,
            Err(e) => {
                warn!(target: "l1_watcher", error = ?e, "failed to query l1 provider for latest safe block");
                None
            }}.map(|block| block.into_consensus().into());

    L1State { current_l1, current_l1_finalized: finalized_l1, head_l1, safe_l1, finalized_l1 }
}

#[cfg(test)]
mod tests {
    use super::{super::client::MockL1WatcherDerivationClient, *};
    use alloy_primitives::{Address, B256, U256};
    use alloy_provider::{ProviderBuilder, mock::Asserter};
    use alloy_rpc_types_eth::Log as RpcLog;
    use futures::stream;
    use kona_genesis::{
        CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC, RollupConfig, SystemConfigUpdateKind,
    };
    use kona_rpc::L1WatcherQueries;
    use std::{pin::Pin, sync::Arc};
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

    fn test_chain(
        index: u8,
        client: MockL1WatcherDerivationClient,
    ) -> (L1WatcherChain<MockL1WatcherDerivationClient>, ChainHandles) {
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

    /// Builds an actor over `chains` whose head stream yields `heads` and whose finalized stream
    /// yields `finalized`, then pends forever so it never races the branch under test.
    fn actor(
        asserter: Asserter,
        heads: Vec<BlockInfo>,
        finalized: Vec<BlockInfo>,
        chains: Vec<L1WatcherChain<MockL1WatcherDerivationClient>>,
    ) -> (
        L1WatcherActor<TestStream, impl Provider, MockL1WatcherDerivationClient>,
        watch::Receiver<Option<BlockInfo>>,
    ) {
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
        let mut chains = Vec::new();
        let mut handles = Vec::new();

        for index in 0..CHAINS {
            let mut client = MockL1WatcherDerivationClient::new();
            client
                .expect_send_new_l1_head()
                .times(1)
                .withf(move |b| *b == block)
                .returning(|_| Ok(()));
            client.expect_send_finalized_l1_block().times(0);

            asserter.push_success(&vec![signer_update_log(chain_at(index), signer_at(index))]);

            let (chain, handle) = test_chain(index, client);
            chains.push(chain);
            handles.push(handle);
        }

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
        let mut chains = Vec::new();
        let mut handles = Vec::new();

        for index in 0..CHAINS {
            let mut client = MockL1WatcherDerivationClient::new();
            client
                .expect_send_finalized_l1_block()
                .times(1)
                .withf(move |b| *b == block)
                .returning(|_| Ok(()));
            client.expect_send_new_l1_head().times(0);

            let (chain, handle) = test_chain(index, client);
            chains.push(chain);
            handles.push(handle);
        }

        let (mut actor, _latest_head_rx) = actor(Asserter::new(), vec![], vec![block], chains);
        actor.step().await.expect("step");

        // Keep the query senders alive for the duration of the step.
        drop(handles);
    }

    /// A config query is answered with the rollup config of the chain it was sent to.
    #[tokio::test]
    async fn config_query_is_answered_per_chain() {
        const CHAINS: u8 = 3;

        let mut chains = Vec::new();
        let mut handles = Vec::new();
        for index in 0..CHAINS {
            let (chain, handle) = test_chain(index, MockL1WatcherDerivationClient::new());
            chains.push(chain);
            handles.push(handle);
        }

        let (mut actor, _latest_head_rx) = actor(Asserter::new(), vec![], vec![], chains);

        for index in 0..CHAINS {
            let (tx, rx) = oneshot::channel();
            handles[index as usize].query_tx.send(L1WatcherQueries::Config(tx)).await.unwrap();
            actor.step().await.expect("step");
            assert_eq!(rx.await.unwrap().l1_system_config_address, chain_at(index));
        }
    }
}
