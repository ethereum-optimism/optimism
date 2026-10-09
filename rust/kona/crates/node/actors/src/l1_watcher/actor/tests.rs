use super::{super::derivation::MockDerivation, *};
use alloy_json_rpc::{RequestPacket, Response, ResponsePacket, ResponsePayload};
use alloy_primitives::{Address, B256, U256};
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_transport::{TransportError, TransportErrorKind};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};
use tokio::{
    sync::{Notify, mpsc},
    task::JoinSet,
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
                        assert_eq!(address, Address::ZERO);
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
                        let signer = Address::repeat_byte(hash[0]);
                        serde_json::to_string(&U256::from_be_slice(signer.as_slice())).unwrap()
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

#[derive(Debug)]
enum Observation {
    Head(BlockInfo),
    Finalized(BlockInfo),
}

#[derive(Debug)]
struct TestDerivation(mpsc::Sender<Observation>);

#[async_trait]
impl Derivation for TestDerivation {
    type Error = mpsc::error::SendError<Observation>;

    async fn send_finalized_l1_block(&self, block: BlockInfo) -> Result<(), Self::Error> {
        self.0.send(Observation::Finalized(block)).await
    }

    async fn send_new_l1_head(&self, block: BlockInfo) -> Result<(), Self::Error> {
        self.0.send(Observation::Head(block)).await
    }
}

struct Harness {
    heads: mpsc::Sender<BlockInfo>,
    finalized: mpsc::Sender<BlockInfo>,
    safe: mpsc::Sender<BlockInfo>,
    state: watch::Receiver<State>,
    signer: watch::Receiver<Address>,
    derivation: mpsc::Receiver<Observation>,
    tasks: JoinSet<Result<(), ActorError>>,
    rpc: Arc<RpcState>,
}
fn head(number: u64) -> BlockInfo {
    BlockInfo::new(B256::repeat_byte(number as u8), number, B256::ZERO, number * 12)
}
impl Harness {
    fn new() -> Self {
        let rpc = Arc::new(RpcState::default());
        let (signer_tx, signer) = watch::channel(Address::ZERO);
        let (derivation_tx, derivation) = mpsc::channel(1);
        let (heads, head_rx) = mpsc::channel(4);
        let (finalized, finalized_rx) = mpsc::channel(4);
        let (safe, safe_rx) = mpsc::channel(4);
        let builder = Builder::new();
        let state = builder.handle().state_receiver();
        let lifetime = builder.build(
            provider(rpc.clone()),
            ReceiverStream::new(head_rx),
            ReceiverStream::new(finalized_rx),
            ReceiverStream::new(safe_rx),
            Arc::new(RollupConfig::default()),
            TestDerivation(derivation_tx),
            signer_tx,
        );
        let mut tasks = JoinSet::new();
        tasks.spawn(lifetime);
        Self { heads, finalized, safe, state, signer, derivation, tasks, rpc }
    }

    /// Waits for the next signer the watcher publishes.
    async fn next_signer(&mut self) -> Address {
        self.signer.changed().await.unwrap();
        *self.signer.borrow_and_update()
    }

    async fn send_head_and_receive(&mut self, block: BlockInfo) {
        self.heads.send(block).await.unwrap();
        match time::timeout(Duration::from_secs(1), self.derivation.recv()).await.unwrap().unwrap()
        {
            Observation::Head(received) => {
                assert_eq!(received, block)
            }
            other => panic!("unexpected request {other:?}"),
        }
    }
}

#[tokio::test(start_paused = true)]
async fn full_derivation_queue_does_not_block_observation_or_signer_refresh() {
    let mut h = Harness::new();
    // Leave derivation paused while observations and signer updates continue.
    for number in 1..=5 {
        let block = head(number);
        h.heads.send(block).await.unwrap();
        h.state.wait_for(|state| state.head_l1 == Some(block)).await.unwrap();
        assert_eq!(h.next_signer().await, Address::repeat_byte(number as u8));
    }
    h.finalized.send(head(4)).await.unwrap();
    h.safe.send(head(3)).await.unwrap();
    h.state
        .wait_for(|state| {
            state.head_l1 == Some(head(5)) &&
                state.finalized_l1 == Some(head(4)) &&
                state.safe_l1 == Some(head(3))
        })
        .await
        .unwrap();
    // Derivation eventually receives the latest head and finality when it resumes.
    time::timeout(Duration::from_secs(1), async {
        let mut received_latest_head = false;
        loop {
            match h.derivation.recv().await.unwrap() {
                Observation::Head(block) if block == head(5) => {
                    received_latest_head = true;
                }
                Observation::Finalized(block) => {
                    assert_eq!(block, head(4));
                    assert!(received_latest_head);
                    break;
                }
                _ => {}
            }
        }
    })
    .await
    .unwrap();
}

#[tokio::test(start_paused = true)]
async fn stalled_rpc_is_isolated_and_retried_without_a_new_head() {
    let mut h = Harness::new();
    h.rpc.stall.store(true, Ordering::SeqCst);
    h.send_head_and_receive(head(7)).await;
    h.rpc.started.notified().await;
    h.send_head_and_receive(head(8)).await;
    assert_eq!(h.state.borrow().head_l1, Some(head(8)));
    assert!(!h.signer.has_changed().unwrap());
    h.rpc.stall.store(false, Ordering::SeqCst);
    time::advance(Duration::from_secs(10)).await;
    assert_eq!(
        time::timeout(Duration::from_secs(1), h.next_signer()).await.unwrap(),
        Address::repeat_byte(8)
    );
}

#[tokio::test(start_paused = true)]
async fn failed_snapshot_recovers_and_same_height_reorg_restores_signer() {
    let mut h = Harness::new();
    h.rpc.fail.store(true, Ordering::SeqCst);
    h.send_head_and_receive(head(7)).await;
    h.rpc.started.notified().await;
    assert!(!h.signer.has_changed().unwrap());
    h.rpc.fail.store(false, Ordering::SeqCst);
    time::advance(Duration::from_secs(10)).await;
    assert_eq!(h.next_signer().await, Address::repeat_byte(7));
    let mut reorg = head(7);
    reorg.hash = B256::repeat_byte(3);
    h.send_head_and_receive(reorg).await;
    assert_eq!(h.next_signer().await, Address::repeat_byte(3));
}

#[tokio::test(start_paused = true)]
async fn publishes_safe_and_finalized_without_new_heads() {
    let mut h = Harness::new();
    h.send_head_and_receive(head(100)).await;
    h.state.borrow_and_update();
    h.safe.send(head(90)).await.unwrap();
    h.state.changed().await.unwrap();
    assert_eq!(h.state.borrow_and_update().safe_l1, Some(head(90)));
    assert!(h.derivation.try_recv().is_err());
    h.finalized.send(head(80)).await.unwrap();
    assert!(
        matches!(h.derivation.recv().await.unwrap(), Observation::Finalized(block) if block == head(80))
    );
    assert_eq!(
        *h.state.borrow(),
        State { head_l1: Some(head(100)), safe_l1: Some(head(90)), finalized_l1: Some(head(80)) }
    );
}

#[tokio::test(start_paused = true)]
async fn obsolete_in_flight_snapshot_cannot_overwrite_new_head() {
    let mut h = Harness::new();
    h.rpc.stall.store(true, Ordering::SeqCst);
    h.send_head_and_receive(head(7)).await;
    h.rpc.started.notified().await;
    h.send_head_and_receive(head(8)).await;
    h.rpc.stall.store(false, Ordering::SeqCst);
    h.rpc.release.notify_one();
    assert_eq!(h.next_signer().await, Address::repeat_byte(8));
}

#[tokio::test(start_paused = true)]
async fn closed_derivation_receiver_stops_watcher() {
    let mut h = Harness::new();
    h.derivation.close();
    h.heads.send(head(1)).await.unwrap();
    assert!(matches!(
        time::timeout(Duration::from_secs(1), h.tasks.join_next()).await.unwrap().unwrap().unwrap(),
        Err(ActorError::Derivation(_))
    ));
    assert!(h.state.has_changed().is_err());
}

#[tokio::test(start_paused = true)]
async fn unchanged_signer_does_not_notify_subscribers() {
    let mut h = Harness::new();
    h.send_head_and_receive(head(1)).await;
    h.rpc.started.notified().await;
    assert_eq!(h.next_signer().await, Address::repeat_byte(1));
    // A later head whose state holds the same signer is read, but publishes nothing.
    let mut same_signer = head(2);
    same_signer.hash = B256::repeat_byte(1);
    h.send_head_and_receive(same_signer).await;
    h.rpc.started.notified().await;
    time::sleep(Duration::from_millis(10)).await;
    assert!(!h.signer.has_changed().unwrap());
}

#[tokio::test(start_paused = true)]
async fn watcher_shutdown_cancels_outstanding_rpc() {
    let mut h = Harness::new();
    h.rpc.stall.store(true, Ordering::SeqCst);
    h.send_head_and_receive(head(1)).await;
    h.rpc.started.notified().await;
    h.tasks.abort_all();
    assert!(h.tasks.join_next().await.unwrap().unwrap_err().is_cancelled());
    time::timeout(Duration::from_secs(1), h.rpc.cancelled.notified()).await.unwrap();
    assert!(h.state.has_changed().is_err());
}

#[test]
fn builds_without_a_runtime_and_closes_on_drop() {
    let builder = Builder::new();
    let state = builder.handle().state_receiver();
    assert_eq!(*state.borrow(), State::default());
    let (signer_tx, signer) = watch::channel(Address::ZERO);
    let lifetime = builder.build(
        provider(Arc::new(RpcState::default())),
        futures::stream::pending(),
        futures::stream::pending(),
        futures::stream::pending(),
        Arc::new(RollupConfig::default()),
        MockDerivation::new(),
        signer_tx,
    );
    assert_eq!(*state.borrow(), State::default());
    drop(lifetime);
    assert!(state.has_changed().is_err());
    assert!(signer.has_changed().is_err());
}

#[tokio::test(start_paused = true)]
async fn observation_stream_end_is_fatal() {
    let mut h = Harness::new();
    drop(h.heads);
    assert!(matches!(h.tasks.join_next().await.unwrap().unwrap(), Err(ActorError::StreamEnded)));
    assert!(h.state.has_changed().is_err());
    assert!(h.signer.has_changed().is_err());
}

#[tokio::test]
async fn dependency_error_is_fatal() {
    let builder = Builder::new();
    let state = builder.handle().state_receiver();
    let (signer_tx, _signer) = watch::channel(Address::ZERO);
    let mut derivation = MockDerivation::new();
    derivation
        .expect_send_new_l1_head()
        .once()
        .returning(|_| Err(std::io::Error::other("unavailable")));
    let lifetime = builder.build(
        provider(Arc::new(RpcState::default())),
        futures::stream::iter([head(1)]).chain(futures::stream::pending()).boxed(),
        futures::stream::pending().boxed(),
        futures::stream::pending().boxed(),
        Arc::new(RollupConfig::default()),
        derivation,
        signer_tx,
    );
    assert!(matches!(lifetime.await, Err(ActorError::Derivation(error)) if error == "unavailable"));
    assert!(state.has_changed().is_err());
}
