//! Publishes the unsafe block signer on L1 head changes.
//!
//! Failed lookups retain the previous signer until another head arrives.

use alloy_primitives::{Address, B256};
use alloy_provider::Provider;
use alloy_transport::TransportError;
use futures::StreamExt;
use kona_protocol::BlockInfo;
use std::future::Future;
use tokio::sync::watch;
use tokio_stream::wrappers::WatchStream;

/// The signer lookup needed by the updater.
pub trait SignerProvider: Send + Sync {
    /// An error reading the signer.
    type Error: std::error::Error + Send + Sync + 'static;

    /// Reads the signer at a canonical L1 block hash.
    fn signer_at_block(
        &self,
        system_config: Address,
        block_hash: B256,
    ) -> impl Future<Output = Result<Address, Self::Error>> + Send;
}

impl<P: Provider> SignerProvider for P {
    type Error = TransportError;

    async fn signer_at_block(
        &self,
        system_config: Address,
        block_hash: B256,
    ) -> Result<Address, Self::Error> {
        kona_providers_alloy::unsafe_block_signer(self, system_config, block_hash).await
    }
}

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
    pub fn build<L1Provider: SignerProvider + 'static>(
        self,
        provider: L1Provider,
        system_config: Address,
        head: watch::Receiver<BlockInfo>,
    ) -> impl Future<Output = ()> + Send + 'static {
        Actor { provider, system_config, head, published: self.published }.run()
    }
}

#[derive(Debug)]
struct Actor<L1Provider> {
    provider: L1Provider,
    system_config: Address,
    head: watch::Receiver<BlockInfo>,
    published: watch::Sender<Address>,
}

impl<L1Provider: SignerProvider> Actor<L1Provider> {
    async fn run(self) {
        let mut heads = WatchStream::from_changes(self.head);
        while let Some(head) = heads.next().await {
            match self.provider.signer_at_block(self.system_config, head.hash).await {
                Ok(signer) => {
                    self.published
                        .send_if_modified(|current| std::mem::replace(current, signer) != signer);
                }
                Err(error) => {
                    warn!(target: "l1_signer_updater", ?error, "signer lookup unavailable")
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::U256;
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::mock::Asserter;
    use std::{
        io,
        sync::{
            Arc,
            atomic::{AtomicBool, AtomicUsize, Ordering},
        },
    };
    use tokio::{
        sync::Notify,
        task::JoinSet,
        time::{self, Duration},
    };

    const SYSTEM_CONFIG: Address = Address::repeat_byte(11);

    #[derive(Debug, Default)]
    struct ProviderState {
        calls: AtomicUsize,
        fail: AtomicBool,
        stall: AtomicBool,
        started: Notify,
        release: Notify,
        cancelled: Notify,
    }

    struct OnDrop<'a>(&'a Notify);

    impl Drop for OnDrop<'_> {
        fn drop(&mut self) {
            self.0.notify_one();
        }
    }

    struct TestProvider(Arc<ProviderState>);

    impl SignerProvider for TestProvider {
        type Error = io::Error;

        async fn signer_at_block(
            &self,
            system_config: Address,
            block_hash: B256,
        ) -> Result<Address, Self::Error> {
            let state = &self.0;
            assert_eq!(system_config, SYSTEM_CONFIG);
            state.calls.fetch_add(1, Ordering::SeqCst);
            state.started.notify_one();
            if state.stall.load(Ordering::SeqCst) {
                let _on_drop = OnDrop(&state.cancelled);
                state.release.notified().await;
            }
            if state.fail.load(Ordering::SeqCst) {
                return Err(io::Error::other("signer unavailable"));
            }
            Ok(Address::repeat_byte(block_hash[0]))
        }
    }

    struct Harness {
        heads: watch::Sender<BlockInfo>,
        signer: watch::Receiver<Address>,
        tasks: JoinSet<()>,
        provider: Arc<ProviderState>,
    }

    fn head(number: u64) -> BlockInfo {
        BlockInfo::new(B256::repeat_byte(number as u8), number, B256::ZERO, number * 12)
    }

    impl Harness {
        fn new() -> Self {
            let provider = Arc::new(ProviderState::default());
            let (heads, head_rx) = watch::channel(BlockInfo::default());
            let builder = Builder::new(Address::repeat_byte(9));
            let signer = builder.handle();
            let lifetime = builder.build(TestProvider(provider.clone()), SYSTEM_CONFIG, head_rx);
            let mut tasks = JoinSet::new();
            tasks.spawn(lifetime);
            Self { heads, signer, tasks, provider }
        }

        async fn next_signer(&mut self) -> Address {
            self.signer.changed().await.unwrap();
            *self.signer.borrow_and_update()
        }
    }

    #[tokio::test]
    async fn alloy_provider_returns_signer_and_rpc_errors() {
        let responses = Asserter::new();
        let provider = RootProvider::new(RpcClient::mocked(responses.clone()));
        let signer = Address::repeat_byte(7);
        responses.push_success(&U256::from_be_slice(signer.as_slice()));
        assert_eq!(provider.signer_at_block(SYSTEM_CONFIG, head(7).hash).await.unwrap(), signer);
        responses.push_failure_msg("signer unavailable");
        assert!(matches!(
            provider.signer_at_block(SYSTEM_CONFIG, head(7).hash).await,
            Err(TransportError::ErrorResp(_))
        ));
    }

    #[tokio::test(start_paused = true)]
    async fn queries_only_after_head_changes() {
        let mut h = Harness::new();
        time::advance(Duration::from_secs(30)).await;
        assert_eq!(h.provider.calls.load(Ordering::SeqCst), 0);
        assert_eq!(*h.signer.borrow(), Address::repeat_byte(9));
        h.heads.send_replace(head(8));
        assert_eq!(h.next_signer().await, Address::repeat_byte(8));
        time::advance(Duration::from_secs(30)).await;
        assert_eq!(h.provider.calls.load(Ordering::SeqCst), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn failed_snapshot_recovers_on_next_head_and_reorgs_restore_signer() {
        let mut h = Harness::new();
        h.provider.fail.store(true, Ordering::SeqCst);
        h.heads.send_replace(head(7));
        h.provider.started.notified().await;
        assert_eq!(*h.signer.borrow(), Address::repeat_byte(9));
        assert!(!h.signer.has_changed().unwrap());
        time::advance(Duration::from_secs(30)).await;
        assert_eq!(h.provider.calls.load(Ordering::SeqCst), 1);
        h.provider.fail.store(false, Ordering::SeqCst);
        h.heads.send_replace(head(8));
        assert_eq!(h.next_signer().await, Address::repeat_byte(8));
        let mut reorg = head(8);
        reorg.hash = B256::repeat_byte(3);
        h.heads.send_replace(reorg);
        assert_eq!(h.next_signer().await, Address::repeat_byte(3));
        h.heads.send_replace(head(4));
        assert_eq!(h.next_signer().await, Address::repeat_byte(4));
    }

    #[tokio::test(start_paused = true)]
    async fn completed_snapshot_is_published_before_catching_up() {
        let mut h = Harness::new();
        h.provider.stall.store(true, Ordering::SeqCst);
        h.heads.send_replace(head(7));
        h.provider.started.notified().await;
        h.heads.send_replace(head(8));
        h.provider.release.notify_one();
        h.provider.started.notified().await;
        assert_eq!(h.next_signer().await, Address::repeat_byte(7));
        h.provider.stall.store(false, Ordering::SeqCst);
        h.provider.release.notify_one();
        assert_eq!(h.next_signer().await, Address::repeat_byte(8));
    }

    #[tokio::test(start_paused = true)]
    async fn unchanged_signer_does_not_notify_subscribers() {
        let mut h = Harness::new();
        h.heads.send_replace(head(7));
        assert_eq!(h.next_signer().await, Address::repeat_byte(7));
        h.provider.started.notified().await;
        let mut same_signer = head(8);
        same_signer.hash[0] = 7;
        h.heads.send_replace(same_signer);
        h.provider.started.notified().await;
        assert!(!h.signer.has_changed().unwrap());
        time::advance(Duration::from_secs(30)).await;
        assert_eq!(h.provider.calls.load(Ordering::SeqCst), 2);
        assert!(!h.signer.has_changed().unwrap());
    }

    #[tokio::test(start_paused = true)]
    async fn dropping_lifetime_cancels_outstanding_lookup() {
        let mut h = Harness::new();
        h.provider.stall.store(true, Ordering::SeqCst);
        h.heads.send_replace(head(7));
        h.provider.started.notified().await;
        h.tasks.abort_all();
        assert!(h.tasks.join_next().await.unwrap().unwrap_err().is_cancelled());
        h.provider.cancelled.notified().await;
        assert!(h.signer.has_changed().is_err());
    }

    #[test]
    fn builds_without_a_runtime_and_closes_on_drop() {
        let initial = Address::repeat_byte(9);
        let builder = Builder::new(initial);
        let signer = builder.handle();
        assert_eq!(*signer.borrow(), initial);
        let (_heads, head_rx) = watch::channel(head(7));
        let lifetime =
            builder.build(TestProvider(Arc::new(ProviderState::default())), SYSTEM_CONFIG, head_rx);
        assert_eq!(*signer.borrow(), initial);
        drop(lifetime);
        assert!(signer.has_changed().is_err());
    }

    #[tokio::test(start_paused = true)]
    async fn closed_head_publisher_stops_updater() {
        let mut h = Harness::new();
        h.heads.send_replace(head(7));
        h.next_signer().await;
        drop(h.heads);
        h.tasks.join_next().await.unwrap().unwrap();
        assert!(h.signer.has_changed().is_err());
    }
}
