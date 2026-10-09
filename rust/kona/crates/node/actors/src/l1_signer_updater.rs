//! Publishes the unsafe block signer from L1.

use alloy_primitives::Address;
use alloy_provider::Provider;
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use std::{future::Future, sync::Arc};
use thiserror::Error;
use tokio::{
    sync::watch,
    time::{self, Duration},
};

/// Constructs the signer publisher and task.
#[derive(Debug)]
pub struct Builder {
    published: watch::Sender<Address>,
}

impl Builder {
    /// Creates the builder with its initial signer.
    pub fn new(initial: Address) -> Self {
        let (published, _) = watch::channel(initial);
        Self { published }
    }

    /// Subscribes to the latest signer before the actor starts.
    pub fn handle(&self) -> watch::Receiver<Address> {
        self.published.subscribe()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled.
    /// Dropping the future stops the actor and closes its publisher.
    pub fn build<L1Provider: Provider + 'static>(
        self,
        provider: L1Provider,
        config: Arc<RollupConfig>,
        head: watch::Receiver<BlockInfo>,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        Actor { provider, config, head, published: self.published }.run()
    }
}

/// A fatal error from the signer updater.
#[derive(Debug, Error)]
pub enum ActorError {
    /// A watch channel closed.
    #[error("channel closed")]
    ChannelClosed,
}

#[derive(Debug)]
struct Actor<L1Provider> {
    provider: L1Provider,
    config: Arc<RollupConfig>,
    head: watch::Receiver<BlockInfo>,
    published: watch::Sender<Address>,
}

impl<L1Provider: Provider> Actor<L1Provider> {
    async fn run(mut self) -> Result<(), ActorError> {
        let mut retry = time::interval(Duration::from_secs(10));
        retry.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        let mut last_head = None;
        loop {
            let head_changed = tokio::select! {
                result = self.head.changed() => {
                    result.map_err(|_| ActorError::ChannelClosed)?;
                    true
                }
                _ = retry.tick() => false,
                _ = self.published.closed() => return Err(ActorError::ChannelClosed),
            };
            let target = *self.head.borrow_and_update();
            if head_changed && last_head == Some(target) {
                continue;
            }
            last_head = Some(target);
            let read = kona_providers_alloy::unsafe_block_signer(
                &self.provider,
                self.config.l1_system_config_address,
                target.hash,
            );
            match time::timeout(Duration::from_secs(10), read).await {
                // Head changes supersede outstanding reads, including same-height reorgs.
                Ok(Ok(signer)) if *self.head.borrow() == target => {
                    self.published
                        .send_if_modified(|current| std::mem::replace(current, signer) != signer);
                }
                Ok(Ok(_)) => {}
                result => {
                    warn!(target: "l1_signer_updater", chain_id = self.config.l2_chain_id.id(), ?result, "unsafe block signer refresh unavailable; retaining previous value")
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::{RequestPacket, Response, ResponsePacket, ResponsePayload};
    use alloy_primitives::{Address, B256, U256};
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::{TransportError, TransportErrorKind};
    use kona_genesis::RollupConfig;
    use std::sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    };
    use tokio::{
        sync::Notify,
        task::JoinSet,
        time::{self, Duration},
    };

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
                    let RequestPacket::Single(request) = request else {
                        panic!("unexpected batch")
                    };
                    let params: serde_json::Value =
                        serde_json::from_str(request.params().unwrap().get()).unwrap();
                    let result = match request.method() {
                        "eth_getStorageAt" => {
                            let address: Address =
                                serde_json::from_value(params[0].clone()).unwrap();
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

    struct Harness {
        heads: watch::Sender<BlockInfo>,
        signer: watch::Receiver<Address>,
        tasks: JoinSet<Result<(), ActorError>>,
        rpc: Arc<RpcState>,
    }

    fn head(number: u64) -> BlockInfo {
        BlockInfo::new(B256::repeat_byte(number as u8), number, B256::ZERO, number * 12)
    }

    impl Harness {
        fn new() -> Self {
            let rpc = Arc::new(RpcState::default());
            let (heads, head_rx) = watch::channel(head(7));
            let builder = Builder::new(Address::repeat_byte(9));
            let signer = builder.handle();
            let lifetime =
                builder.build(provider(rpc.clone()), Arc::new(RollupConfig::default()), head_rx);
            let mut tasks = JoinSet::new();
            tasks.spawn(lifetime);
            Self { heads, signer, tasks, rpc }
        }

        async fn next_signer(&mut self) -> Address {
            self.signer.changed().await.unwrap();
            *self.signer.borrow_and_update()
        }
    }

    #[tokio::test(start_paused = true)]
    async fn stalled_rpc_is_retried_at_the_latest_head() {
        let mut h = Harness::new();
        h.rpc.stall.store(true, Ordering::SeqCst);
        h.rpc.started.notified().await;
        h.heads.send_replace(head(8));
        assert_eq!(*h.signer.borrow(), Address::repeat_byte(9));
        h.rpc.stall.store(false, Ordering::SeqCst);
        time::advance(Duration::from_secs(10)).await;
        assert_eq!(h.next_signer().await, Address::repeat_byte(8));
        h.rpc.cancelled.notified().await;
    }

    #[tokio::test(start_paused = true)]
    async fn failed_snapshot_recovers_and_reorgs_restore_signer() {
        let mut h = Harness::new();
        h.rpc.fail.store(true, Ordering::SeqCst);
        h.rpc.started.notified().await;
        assert_eq!(*h.signer.borrow(), Address::repeat_byte(9));
        assert!(!h.signer.has_changed().unwrap());
        h.rpc.fail.store(false, Ordering::SeqCst);
        time::advance(Duration::from_secs(10)).await;
        assert_eq!(h.next_signer().await, Address::repeat_byte(7));
        let mut reorg = head(7);
        reorg.hash = B256::repeat_byte(3);
        h.heads.send_replace(reorg);
        assert_eq!(h.next_signer().await, Address::repeat_byte(3));
        h.heads.send_replace(head(4));
        assert_eq!(h.next_signer().await, Address::repeat_byte(4));
    }

    #[tokio::test(start_paused = true)]
    async fn obsolete_in_flight_snapshot_cannot_overwrite_new_head() {
        let mut h = Harness::new();
        h.rpc.stall.store(true, Ordering::SeqCst);
        h.rpc.started.notified().await;
        h.heads.send_replace(head(8));
        h.rpc.stall.store(false, Ordering::SeqCst);
        h.rpc.release.notify_one();
        assert_eq!(h.next_signer().await, Address::repeat_byte(8));
    }

    #[tokio::test(start_paused = true)]
    async fn unchanged_signer_does_not_notify_subscribers() {
        let mut h = Harness::new();
        assert_eq!(h.next_signer().await, Address::repeat_byte(7));
        h.rpc.started.notified().await;
        let mut same_signer = head(8);
        same_signer.hash = head(7).hash;
        h.heads.send_replace(same_signer);
        h.rpc.started.notified().await;
        assert!(!h.signer.has_changed().unwrap());
        time::advance(Duration::from_secs(10)).await;
        h.rpc.started.notified().await;
        assert!(!h.signer.has_changed().unwrap());
    }

    #[tokio::test(start_paused = true)]
    async fn dropping_lifetime_cancels_outstanding_rpc() {
        let mut h = Harness::new();
        h.rpc.stall.store(true, Ordering::SeqCst);
        h.rpc.started.notified().await;
        h.tasks.abort_all();
        assert!(h.tasks.join_next().await.unwrap().unwrap_err().is_cancelled());
        h.rpc.cancelled.notified().await;
        assert!(h.signer.has_changed().is_err());
    }

    #[test]
    fn builds_without_a_runtime_and_closes_on_drop() {
        let initial = Address::repeat_byte(9);
        let builder = Builder::new(initial);
        let signer = builder.handle();
        assert_eq!(*signer.borrow(), initial);
        let (_heads, head_rx) = watch::channel(head(7));
        let lifetime = builder.build(
            provider(Arc::new(RpcState::default())),
            Arc::new(RollupConfig::default()),
            head_rx,
        );
        assert_eq!(*signer.borrow(), initial);
        drop(lifetime);
        assert!(signer.has_changed().is_err());
    }

    #[tokio::test(start_paused = true)]
    async fn closed_head_publisher_stops_updater() {
        let mut h = Harness::new();
        h.next_signer().await;
        drop(h.heads);
        assert!(matches!(
            h.tasks.join_next().await.unwrap().unwrap(),
            Err(ActorError::ChannelClosed)
        ));
        assert!(h.signer.has_changed().is_err());
    }

    #[tokio::test(start_paused = true)]
    async fn closed_signer_subscribers_stop_updater() {
        let mut h = Harness::new();
        h.next_signer().await;
        drop(h.signer);
        assert!(matches!(
            h.tasks.join_next().await.unwrap().unwrap(),
            Err(ActorError::ChannelClosed)
        ));
    }
}
