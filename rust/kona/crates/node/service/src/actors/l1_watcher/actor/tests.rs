use super::*;
use crate::{DerivationActorRequest, QueuedL1WatcherDerivationClient};
use alloy_json_rpc::{RequestPacket, Response, ResponsePacket, ResponsePayload};
use alloy_primitives::{Address, B256, U256};
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_transport::{TransportError, TransportErrorKind};
use kona_genesis::RollupConfig;
use kona_rpc::L1State;
use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};
use tokio::{
    sync::{Notify, mpsc},
    time::{self, Duration},
};
use tokio_stream::wrappers::ReceiverStream;

#[derive(Debug, Default)]
struct RpcState {
    fail: AtomicBool,
    stall: AtomicBool,
    started: Notify,
    release: Notify,
    cancelled: Notify,
    safe_stalled: AtomicBool,
    safe_started: Notify,
}

fn provider(state: Arc<RpcState>) -> RootProvider {
    let transport = tower::service_fn(
        move |request: RequestPacket| -> alloy_transport::TransportFut<'static> {
            let state = state.clone();
            Box::pin(async move {
                let RequestPacket::Single(request) = request else { panic!("unexpected batch") };
                let params: serde_json::Value =
                    serde_json::from_str(request.params().unwrap().get()).unwrap();
                let result = match request.method() {
                    "eth_getStorageAt" => {
                        let address: Address = serde_json::from_value(params[0].clone()).unwrap();
                        let hash: B256 =
                            serde_json::from_value(params[2]["blockHash"].clone()).unwrap();
                        assert_eq!(params[2]["requireCanonical"], true);
                        if address == Address::ZERO {
                            state.started.notify_one();
                            if state.stall.load(Ordering::SeqCst) {
                                struct OnDrop<'a>(&'a Notify);
                                impl Drop for OnDrop<'_> {
                                    fn drop(&mut self) {
                                        self.0.notify_one();
                                    }
                                }
                                let _on_drop = OnDrop(&state.cancelled);
                                state.release.notified().await;
                            }
                            if state.fail.load(Ordering::SeqCst) {
                                return Err(TransportErrorKind::custom_str("L1 unavailable"));
                            }
                        }
                        let signer = Address::repeat_byte(hash[0] + address[0]);
                        serde_json::to_string(&U256::from_be_slice(signer.as_slice())).unwrap()
                    }
                    "eth_getBlockByNumber" => {
                        let mut block = alloy_rpc_types_eth::Block::<
                            alloy_rpc_types_eth::Transaction,
                        >::default();
                        let tag = params[0].as_str().unwrap();
                        if tag == "safe" && state.safe_stalled.load(Ordering::SeqCst) {
                            state.safe_started.notify_one();
                            state.release.notified().await;
                        }
                        block.header.inner.number = match tag {
                            "latest" => 100,
                            "finalized" => 80,
                            "safe" => 90,
                            tag => panic!("unexpected tag {tag}"),
                        };
                        serde_json::to_string(&block).unwrap()
                    }
                    method => panic!("unexpected RPC {method}"),
                };
                Ok::<_, TransportError>(ResponsePacket::Single(Response {
                    id: request.id().clone(),
                    payload: ResponsePayload::Success(
                        serde_json::value::RawValue::from_string(result).unwrap(),
                    ),
                }))
            })
        },
    );
    RootProvider::new(RpcClient::new(transport, false))
}

struct ChainHandles {
    signer: watch::Receiver<Address>,
    derivation: mpsc::Receiver<DerivationActorRequest>,
}
impl ChainHandles {
    /// Waits for the next signer the watcher publishes for this chain.
    async fn next_signer(&mut self) -> Address {
        self.signer.changed().await.unwrap();
        *self.signer.borrow_and_update()
    }
}
struct Harness {
    heads: mpsc::Sender<BlockInfo>,
    safe: mpsc::Sender<BlockInfo>,
    finalized: mpsc::Sender<BlockInfo>,
    state: watch::Receiver<L1State>,
    chains: Vec<ChainHandles>,
    tasks: JoinSet<Result<(), L1WatcherActorError>>,
    rpc: Arc<RpcState>,
}
fn head(number: u64) -> BlockInfo {
    BlockInfo::new(B256::repeat_byte(number as u8), number, B256::ZERO, number * 12)
}
impl Harness {
    fn new(count: u8) -> Self {
        let rpc = Arc::new(RpcState::default());
        let mut configs = Vec::new();
        let mut chains = Vec::new();
        for index in 0..count {
            let (signer_tx, signer) = watch::channel(Address::ZERO);
            let (derivation_tx, derivation) = mpsc::channel(1);
            configs.push(L1WatcherChain::new(
                Arc::new(RollupConfig {
                    l2_chain_id: u64::from(index).into(),
                    l1_system_config_address: Address::repeat_byte(index),
                    ..Default::default()
                }),
                QueuedL1WatcherDerivationClient { derivation_actor_request_tx: derivation_tx },
                signer_tx,
            ));
            chains.push(ChainHandles { signer, derivation });
        }
        let (heads, head_rx) = mpsc::channel(4);
        let (safe, safe_rx) = mpsc::channel(4);
        let (finalized, finalized_rx) = mpsc::channel(4);
        let (latest, state) = watch::channel(L1State::default());
        let mut actor = L1WatcherActor::new(
            provider(rpc.clone()),
            latest,
            ReceiverStream::new(head_rx),
            ReceiverStream::new(safe_rx),
            ReceiverStream::new(finalized_rx),
            configs,
        );
        let mut tasks = JoinSet::new();
        tasks.spawn(async move {
            loop {
                actor.step().await?;
            }
            #[allow(unreachable_code)]
            Ok(())
        });
        Self { heads, safe, finalized, state, chains, tasks, rpc }
    }

    async fn send_head_and_receive(&mut self, block: BlockInfo, chain: usize) {
        self.heads.send(block).await.unwrap();
        match time::timeout(Duration::from_secs(1), self.chains[chain].derivation.recv())
            .await
            .unwrap()
            .unwrap()
        {
            DerivationActorRequest::ProcessL1HeadUpdateRequest(received) => {
                assert_eq!(*received, block)
            }
            other => panic!("unexpected request {other:?}"),
        }
    }
}

#[tokio::test(start_paused = true)]
async fn slow_chain_does_not_block_subsequent_heads_finality_or_observations() {
    let mut h = Harness::new(2);
    // Chain 0's derivation queue fills. Healthy chain 1 still receives later heads.
    for number in 1..=5 {
        h.send_head_and_receive(head(number), 1).await;
        assert_eq!(h.chains[1].next_signer().await, Address::repeat_byte(number as u8 + 1));
    }
    h.finalized.send(head(4)).await.unwrap();
    assert!(
        matches!(h.chains[1].derivation.recv().await.unwrap(), DerivationActorRequest::ProcessFinalizedL1Block(block) if *block == head(4))
    );
    assert_eq!(h.state.borrow().head, Some(head(5)));
    assert_eq!(h.state.borrow().finalized, Some(head(4)));
    // When it resumes, the slow chain eventually receives the latest observation.
    time::timeout(Duration::from_secs(1), async {
        loop {
            if matches!(h.chains[0].derivation.recv().await.unwrap(), DerivationActorRequest::ProcessL1HeadUpdateRequest(block) if *block == head(5)) { break; }
        }
    }).await.unwrap();
}

#[tokio::test(start_paused = true)]
async fn stalled_rpc_is_isolated_and_retried_without_a_new_head() {
    let mut h = Harness::new(2);
    h.rpc.stall.store(true, Ordering::SeqCst);
    h.send_head_and_receive(head(7), 1).await;
    h.rpc.started.notified().await;
    assert_eq!(h.chains[1].next_signer().await, Address::repeat_byte(8));
    h.send_head_and_receive(head(8), 1).await;
    assert_eq!(h.chains[1].next_signer().await, Address::repeat_byte(9));
    assert_eq!(h.state.borrow().head, Some(head(8)));
    h.rpc.stall.store(false, Ordering::SeqCst);
    time::advance(Duration::from_secs(10)).await;
    assert_eq!(
        time::timeout(Duration::from_secs(1), h.chains[0].next_signer()).await.unwrap(),
        Address::repeat_byte(8)
    );
}

#[tokio::test(start_paused = true)]
async fn failed_snapshot_recovers_and_same_height_reorg_restores_signer() {
    let mut h = Harness::new(1);
    h.rpc.fail.store(true, Ordering::SeqCst);
    h.send_head_and_receive(head(7), 0).await;
    h.rpc.started.notified().await;
    assert!(!h.chains[0].signer.has_changed().unwrap());
    h.rpc.fail.store(false, Ordering::SeqCst);
    time::advance(Duration::from_secs(10)).await;
    assert_eq!(h.chains[0].next_signer().await, Address::repeat_byte(7));
    let mut reorg = head(7);
    reorg.hash = B256::repeat_byte(3);
    h.send_head_and_receive(reorg, 0).await;
    assert_eq!(h.chains[0].next_signer().await, Address::repeat_byte(3));
}

#[tokio::test(start_paused = true)]
async fn safe_and_finalized_observations_update_without_a_new_head() {
    let mut h = Harness::new(1);
    h.send_head_and_receive(head(7), 0).await;
    h.safe.send(head(5)).await.unwrap();
    h.finalized.send(head(4)).await.unwrap();
    time::timeout(
        Duration::from_secs(1),
        h.state.wait_for(|state| state.safe == Some(head(5)) && state.finalized == Some(head(4))),
    )
    .await
    .unwrap()
    .unwrap();
    assert_eq!(h.state.borrow().head, Some(head(7)));
    // A safe-only publication does not fabricate another new-head notification.
    assert!(matches!(h.chains[0].derivation.recv().await.unwrap(),
        DerivationActorRequest::ProcessFinalizedL1Block(block) if *block == head(4)));
    assert!(h.chains[0].derivation.try_recv().is_err());
}

#[tokio::test(start_paused = true)]
async fn blocked_safe_poll_does_not_hold_back_head_or_finalized_and_recovers_without_new_head() {
    use crate::BlockStream;
    use alloy_eips::BlockNumberOrTag;
    let rpc = Arc::new(RpcState::default());
    rpc.safe_stalled.store(true, Ordering::SeqCst);
    let provider = provider(rpc.clone());
    let streams = [BlockNumberOrTag::Latest, BlockNumberOrTag::Safe, BlockNumberOrTag::Finalized]
        .map(|tag| {
            BlockStream::new_as_stream(provider.clone(), tag, Duration::from_secs(4)).unwrap()
        });
    let [head, safe, finalized] = streams;
    let (state_tx, mut state) = watch::channel(L1State::default());
    let (signer_tx, _signer) = watch::channel(Address::ZERO);
    let (requests, _requests) = mpsc::channel(1);
    let chain = L1WatcherChain::new(
        Arc::new(RollupConfig::default()),
        QueuedL1WatcherDerivationClient { derivation_actor_request_tx: requests },
        signer_tx,
    );
    let mut actor = L1WatcherActor::new(provider, state_tx, head, safe, finalized, vec![chain]);
    let task = tokio::spawn(async move {
        loop {
            actor.step().await.unwrap();
        }
    });
    rpc.safe_started.notified().await;
    time::timeout(
        Duration::from_secs(1),
        state.wait_for(|state| {
            state.head.is_some_and(|head| head.number == 100) &&
                state.finalized.is_some_and(|head| head.number == 80)
        }),
    )
    .await
    .unwrap()
    .unwrap();
    assert!(state.borrow().safe.is_none());
    rpc.safe_stalled.store(false, Ordering::SeqCst);
    rpc.release.notify_one();
    time::timeout(Duration::from_secs(1), state.wait_for(|state| state.safe.is_some()))
        .await
        .unwrap()
        .unwrap();
    assert_eq!(state.borrow().safe.unwrap().number, 90);
    task.abort();
    let _ = task.await;
}

#[tokio::test(start_paused = true)]
async fn obsolete_in_flight_snapshot_cannot_overwrite_new_head() {
    let mut h = Harness::new(1);
    h.rpc.stall.store(true, Ordering::SeqCst);
    h.send_head_and_receive(head(7), 0).await;
    h.rpc.started.notified().await;
    h.send_head_and_receive(head(8), 0).await;
    h.rpc.stall.store(false, Ordering::SeqCst);
    h.rpc.release.notify_one();
    assert_eq!(h.chains[0].next_signer().await, Address::repeat_byte(8));
}

#[tokio::test(start_paused = true)]
async fn closed_derivation_receiver_detaches_only_that_chain() {
    let mut h = Harness::new(2);
    h.chains[0].derivation.close();
    for number in 1..=3 {
        h.send_head_and_receive(head(number), 1).await;
        assert_eq!(h.chains[1].next_signer().await, Address::repeat_byte(number as u8 + 1));
    }
    assert_eq!(h.state.borrow().head, Some(head(3)));
}

#[tokio::test(start_paused = true)]
async fn unchanged_signer_does_not_notify_subscribers() {
    let mut h = Harness::new(1);
    h.send_head_and_receive(head(1), 0).await;
    h.rpc.started.notified().await;
    assert_eq!(h.chains[0].next_signer().await, Address::repeat_byte(1));
    // A later head whose state holds the same signer is read, but publishes nothing.
    let mut same_signer = head(2);
    same_signer.hash = B256::repeat_byte(1);
    h.send_head_and_receive(same_signer, 0).await;
    h.rpc.started.notified().await;
    time::sleep(Duration::from_millis(10)).await;
    assert!(!h.chains[0].signer.has_changed().unwrap());
}

#[tokio::test(start_paused = true)]
async fn watcher_shutdown_cancels_outstanding_chain_rpc() {
    let mut h = Harness::new(1);
    h.rpc.stall.store(true, Ordering::SeqCst);
    h.send_head_and_receive(head(1), 0).await;
    h.rpc.started.notified().await;
    h.tasks.abort_all();
    assert!(h.tasks.join_next().await.unwrap().unwrap_err().is_cancelled());
    time::timeout(Duration::from_secs(1), h.rpc.cancelled.notified()).await.unwrap();
}
