//! L1 block polling and publication.

use alloy_eips::BlockNumberOrTag;
use alloy_provider::Provider;
use alloy_rpc_client::PollerBuilder;
use alloy_rpc_types_eth::Block;
use futures::StreamExt;
use kona_protocol::BlockInfo;
use std::{future::Future, time::Duration};
use thiserror::Error;
use tokio::sync::watch;

/// The L1 block tag to observe.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Tag {
    /// The latest L1 head.
    Head,
    /// The safe L1 head.
    Safe,
    /// The finalized L1 head.
    Finalized,
}

impl From<Tag> for BlockNumberOrTag {
    fn from(tag: Tag) -> Self {
        match tag {
            Tag::Head => Self::Latest,
            Tag::Safe => Self::Safe,
            Tag::Finalized => Self::Finalized,
        }
    }
}

/// Constructs the state publisher and task.
#[derive(Debug)]
pub struct Builder {
    published: watch::Sender<BlockInfo>,
}

impl Builder {
    /// Creates the builder with its initial block.
    pub fn new(initial: BlockInfo) -> Self {
        let (published, _) = watch::channel(initial);
        Self { published }
    }

    /// Subscribes to the latest block before the actor starts.
    ///
    /// Observations may skip intermediate blocks.
    pub fn handle(&self) -> watch::Receiver<BlockInfo> {
        self.published.subscribe()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled.
    /// Dropping the future stops the actor and closes its publisher.
    pub fn build<L1Provider: Provider + 'static>(
        self,
        provider: L1Provider,
        tag: Tag,
        interval: Duration,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        Actor { provider, tag, interval, published: self.published }.run()
    }
}

/// A fatal error from the L1 actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// The block stream ended.
    #[error("block stream ended")]
    StreamEnded,
}

#[derive(Debug)]
struct Actor<L1Provider> {
    provider: L1Provider,
    tag: Tag,
    interval: Duration,
    published: watch::Sender<BlockInfo>,
}

impl<L1Provider: Provider> Actor<L1Provider> {
    async fn run(self) -> Result<(), ActorError> {
        let mut blocks = PollerBuilder::<(BlockNumberOrTag, bool), Block>::new(
            self.provider.weak_client(),
            "eth_getBlockByNumber",
            (self.tag.into(), false),
        )
        .with_poll_interval(self.interval)
        .into_stream();
        while let Some(block) = blocks.next().await {
            let block: BlockInfo = block.into_consensus().into();
            // Publish reorgs as well as advances, and suppress unchanged observations.
            self.published.send_if_modified(|current| std::mem::replace(current, block) != block);
        }
        Err(ActorError::StreamEnded)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::{ErrorPayload, RequestPacket, Response, ResponsePacket, ResponsePayload};
    use alloy_primitives::B256;
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::{TransportError, TransportErrorKind};
    use std::sync::{
        Arc, Mutex,
        atomic::{AtomicBool, AtomicUsize, Ordering},
    };
    use tokio::{sync::Notify, time};

    const INTERVAL: Duration = Duration::from_secs(4);

    #[derive(Default)]
    struct RpcState {
        block: Mutex<Block>,
        calls: AtomicUsize,
        fail: AtomicBool,
        terminal: AtomicBool,
        stall: AtomicBool,
        cancelled: Notify,
    }

    fn block(number: u64, parent_hash_byte: u8) -> Block {
        let mut block: Block = Block::default();
        block.header.inner.number = number;
        block.header.inner.timestamp = number * 12;
        block.header.inner.parent_hash = B256::repeat_byte(parent_hash_byte);
        block.header.hash = block.header.inner.hash_slow();
        block
    }

    fn info(block: Block) -> BlockInfo {
        block.into_consensus().into()
    }

    fn provider(state: Arc<RpcState>, tag: &'static str) -> RootProvider {
        let transport = tower::service_fn(
            move |request: RequestPacket| -> alloy_transport::TransportFut<'static> {
                let state = state.clone();
                Box::pin(async move {
                    let RequestPacket::Single(request) = request else {
                        panic!("unexpected batch")
                    };
                    assert_eq!(request.method(), "eth_getBlockByNumber");
                    let params: serde_json::Value =
                        serde_json::from_str(request.params().unwrap().get()).unwrap();
                    assert_eq!(params, serde_json::json!([tag, false]));
                    state.calls.fetch_add(1, Ordering::SeqCst);
                    if state.stall.load(Ordering::SeqCst) {
                        struct OnDrop<'a>(&'a Notify);
                        impl Drop for OnDrop<'_> {
                            fn drop(&mut self) {
                                self.0.notify_one();
                            }
                        }
                        let _on_drop = OnDrop(&state.cancelled);
                        std::future::pending::<()>().await;
                    }
                    if state.fail.load(Ordering::SeqCst) {
                        return Err(TransportErrorKind::custom_str("L1 unavailable"));
                    }
                    let payload = if state.terminal.load(Ordering::SeqCst) {
                        ResponsePayload::Failure(ErrorPayload {
                            code: -32000,
                            message: "filter not found".into(),
                            data: None,
                        })
                    } else {
                        ResponsePayload::Success(
                            serde_json::value::to_raw_value(&*state.block.lock().unwrap()).unwrap(),
                        )
                    };
                    Ok::<_, TransportError>(ResponsePacket::Single(Response {
                        id: request.id().clone(),
                        payload,
                    }))
                })
            },
        );
        RootProvider::new(RpcClient::new(transport, false))
    }

    #[test]
    fn supplies_initial_state_and_builds_without_a_runtime() {
        let initial = info(block(42, 42));
        let builder = Builder::new(initial);
        let receiver = builder.handle();
        assert_eq!(*receiver.borrow(), initial);
        let rpc = Arc::new(RpcState::default());
        let lifetime = builder.build(provider(rpc.clone(), "latest"), Tag::Head, INTERVAL);
        assert_eq!(rpc.calls.load(Ordering::SeqCst), 0);
        assert_eq!(*receiver.borrow(), initial);
        drop(lifetime);
        assert!(receiver.has_changed().is_err());
    }

    #[rstest::rstest]
    #[case(Tag::Head, "latest")]
    #[case(Tag::Safe, "safe")]
    #[case(Tag::Finalized, "finalized")]
    #[tokio::test(start_paused = true)]
    async fn publishes_advances_and_reorgs_without_waiting_for_subscribers(
        #[case] tag: Tag,
        #[case] rpc_tag: &'static str,
    ) {
        let initial = block(10, 10);
        let rpc = Arc::new(RpcState { block: Mutex::new(initial.clone()), ..Default::default() });
        let builder = Builder::new(info(initial));
        let mut receiver = builder.handle();
        let mut slow = builder.handle();
        // The actor owns the provider; no external clone keeps the RPC client alive.
        let mut lifetime = Box::pin(builder.build(provider(rpc.clone(), rpc_tag), tag, INTERVAL));
        assert!(futures::poll!(lifetime.as_mut()).is_pending());
        assert_eq!(rpc.calls.load(Ordering::SeqCst), 1);
        assert!(!receiver.has_changed().unwrap());

        for observed in [block(11, 11), block(11, 3), block(9, 9)] {
            *rpc.block.lock().unwrap() = observed.clone();
            time::advance(INTERVAL).await;
            assert!(futures::poll!(lifetime.as_mut()).is_pending());
            receiver.changed().await.unwrap();
            assert_eq!(*receiver.borrow_and_update(), info(observed));
        }
        // A subscriber that never drained updates still sees the latest observation.
        slow.changed().await.unwrap();
        assert_eq!(*slow.borrow_and_update(), info(block(9, 9)));
        time::advance(INTERVAL).await;
        assert!(futures::poll!(lifetime.as_mut()).is_pending());
        assert_eq!(rpc.calls.load(Ordering::SeqCst), 5);
        assert!(!receiver.has_changed().unwrap());
        drop(lifetime);
        assert!(receiver.has_changed().is_err());
        assert!(slow.has_changed().is_err());
    }

    #[tokio::test(start_paused = true)]
    async fn retries_rpc_failures_and_retains_the_previous_block() {
        let initial = info(block(1, 1));
        let rpc = Arc::new(RpcState {
            block: Mutex::new(block(2, 2)),
            fail: AtomicBool::new(true),
            ..Default::default()
        });
        let builder = Builder::new(initial);
        let mut receiver = builder.handle();
        let mut lifetime =
            Box::pin(builder.build(provider(rpc.clone(), "latest"), Tag::Head, INTERVAL));
        assert!(futures::poll!(lifetime.as_mut()).is_pending());
        assert_eq!(*receiver.borrow(), initial);
        assert!(!receiver.has_changed().unwrap());
        rpc.fail.store(false, Ordering::SeqCst);
        time::advance(INTERVAL).await;
        assert!(futures::poll!(lifetime.as_mut()).is_pending());
        receiver.changed().await.unwrap();
        assert_eq!(*receiver.borrow(), info(block(2, 2)));
        assert_eq!(rpc.calls.load(Ordering::SeqCst), 2);
    }

    #[tokio::test]
    async fn dropping_lifetime_cancels_an_outstanding_rpc() {
        let rpc = Arc::new(RpcState { stall: AtomicBool::new(true), ..Default::default() });
        let builder = Builder::new(BlockInfo::default());
        let receiver = builder.handle();
        let mut lifetime =
            Box::pin(builder.build(provider(rpc.clone(), "latest"), Tag::Head, INTERVAL));
        assert!(futures::poll!(lifetime.as_mut()).is_pending());
        drop(lifetime);
        rpc.cancelled.notified().await;
        assert!(receiver.has_changed().is_err());
    }

    #[tokio::test]
    async fn terminal_stream_error_closes_the_publisher() {
        let rpc = Arc::new(RpcState { terminal: AtomicBool::new(true), ..Default::default() });
        let builder = Builder::new(BlockInfo::default());
        let receiver = builder.handle();
        let result = builder.build(provider(rpc, "latest"), Tag::Head, INTERVAL).await;
        assert!(matches!(result, Err(ActorError::StreamEnded)));
        assert!(receiver.has_changed().is_err());
    }
}
