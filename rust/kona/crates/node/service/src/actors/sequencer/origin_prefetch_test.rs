use super::*;
use crate::{L1OriginSelector, OriginSelector};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use rstest::rstest;
use std::sync::{
    Mutex,
    atomic::{AtomicBool, AtomicUsize, Ordering},
};

#[derive(Debug, Clone)]
struct MemoryL1 {
    blocks: Arc<Mutex<Vec<Header>>>,
    calls: Arc<AtomicUsize>,
    receipts: Arc<AtomicUsize>,
    stalled: Arc<AtomicBool>,
    stalled_receipts: Arc<AtomicBool>,
    receipt_latency_ms: Arc<AtomicUsize>,
    stalled_receipt_hash: Arc<Mutex<Option<B256>>>,
    head: watch::Receiver<Option<BlockInfo>>,
    depth: u64,
}

impl MemoryL1 {
    fn info(header: &Header) -> BlockInfo {
        BlockInfo {
            hash: header.hash_slow(),
            parent_hash: header.parent_hash,
            number: header.number,
            timestamp: header.timestamp,
        }
    }

    fn at(&self, number: u64) -> BlockInfo {
        Self::info(&self.blocks.lock().unwrap()[number as usize])
    }

    async fn count_call(&self) {
        self.calls.fetch_add(1, Ordering::SeqCst);
        if self.stalled.load(Ordering::SeqCst) {
            std::future::pending::<()>().await;
        }
    }
}

#[async_trait]
impl L1OriginSelectorProvider for MemoryL1 {
    async fn get_block_by_hash(
        &self,
        hash: B256,
    ) -> Result<Option<BlockInfo>, L1OriginSelectorError> {
        self.count_call().await;
        Ok(self
            .blocks
            .lock()
            .unwrap()
            .iter()
            .find(|header| header.hash_slow() == hash)
            .map(Self::info))
    }

    async fn get_block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockInfo>, L1OriginSelectorError> {
        if number != 0 &&
            self.depth != 0 &&
            !self.head.borrow().is_some_and(|head| {
                head.number.checked_sub(number).is_some_and(|depth| depth >= self.depth)
            })
        {
            return Ok(None);
        }
        self.count_call().await;
        Ok(self.blocks.lock().unwrap().get(number as usize).map(Self::info))
    }
}

#[async_trait]
impl ChainProvider for MemoryL1 {
    type Error = PipelineErrorKind;
    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        self.count_call().await;
        self.blocks
            .lock()
            .unwrap()
            .iter()
            .find(|header| header.hash_slow() == hash)
            .cloned()
            .ok_or_else(|| PipelineError::Provider("missing header".into()).temp())
    }
    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        self.count_call().await;
        self.receipts.fetch_add(1, Ordering::SeqCst);
        let latency = self.receipt_latency_ms.load(Ordering::SeqCst);
        if latency != 0 {
            tokio::time::sleep(Duration::from_millis(latency as u64)).await;
        }
        if self.stalled_receipts.load(Ordering::SeqCst) ||
            *self.stalled_receipt_hash.lock().unwrap() == Some(hash)
        {
            std::future::pending::<()>().await;
        }
        Ok(vec![Receipt::default()])
    }
    async fn block_info_by_number(&mut self, _number: u64) -> Result<BlockInfo, Self::Error> {
        unreachable!()
    }
    async fn block_info_and_transactions_by_hash(
        &mut self,
        _hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        unreachable!()
    }
}

struct Harness {
    worker: OriginPrefetcher<MemoryL1, MemoryL1>,
    selector: L1OriginSelector<PreparedOriginProvider>,
    attributes: PreparedAttributesProvider,
    l1: MemoryL1,
    head: watch::Sender<Option<BlockInfo>>,
    parent: L2BlockInfo,
}

impl Harness {
    fn new(depth: u64) -> Self {
        let (head, head_rx) = watch::channel(None);
        let mut blocks = Vec::<Header>::new();
        for number in 0..500 {
            blocks.push(Header {
                number,
                timestamp: number * 12,
                parent_hash: blocks.last().map(Header::hash_slow).unwrap_or_default(),
                ..Default::default()
            });
        }
        let l1 = MemoryL1 {
            blocks: Arc::new(Mutex::new(blocks)),
            calls: Arc::default(),
            receipts: Arc::default(),
            stalled: Arc::default(),
            stalled_receipts: Arc::default(),
            receipt_latency_ms: Arc::default(),
            stalled_receipt_hash: Arc::default(),
            head: head_rx.clone(),
            depth,
        };
        let rpc = alloy_provider::RootProvider::new_http("http://127.0.0.1:1".parse().unwrap());
        let OriginPreparation { prefetcher: worker, origins: provider, attributes } =
            OriginPreparation::new(rpc, head_rx, depth, l1.clone());
        let worker = OriginPrefetcher {
            origins: l1.clone(),
            chain: l1.clone(),
            request: worker.request,
            window: worker.window,
            l1_head: worker.l1_head,
            retry: worker.retry,
            fetch_timeout: worker.fetch_timeout,
        };
        let selector = L1OriginSelector::new(
            Arc::new(RollupConfig {
                block_time: 10,
                max_sequencer_drift: 600,
                ..Default::default()
            }),
            provider,
        );
        let parent = L2BlockInfo { l1_origin: l1.at(0).id(), ..Default::default() };
        Self { worker, selector, attributes, l1, head, parent }
    }

    async fn fill(&mut self, recheck: bool) -> Result<(), L1OriginSelectorError> {
        let request = *self.worker.request.borrow_and_update();
        self.worker.fill(request.unwrap(), recheck).await
    }

    async fn drive(&mut self, use_actor: bool, recheck: bool) {
        if !use_actor {
            self.fill(recheck).await.unwrap();
            return;
        }
        // Exercise production notification coalescing and the worker's cancellation select.
        for _ in 0..100 {
            if !self.worker.request.has_changed().unwrap() &&
                !self.worker.l1_head.has_changed().unwrap()
            {
                return;
            }
            self.worker.step().await.unwrap();
        }
        panic!("worker did not consume its notifications");
    }

    async fn initialize(&mut self) {
        assert!(matches!(
            self.selector.next_l1_origin(self.parent, false).await,
            Err(L1OriginSelectorError::Pending)
        ));
        self.fill(true).await.unwrap();
    }
}

#[tokio::test(start_paused = true)]
#[rstest]
#[case::before_build(true, false)]
#[case::after_build(false, false)]
#[case::actor_before_build(true, true)]
#[case::actor_after_build(false, true)]
async fn keeps_up_with_l1_without_build_path_rpc(
    #[case] prefetch_first: bool,
    #[case] use_actor: bool,
) {
    let mut h = Harness::new(4);
    h.head.send_replace(Some(h.l1.at(0)));
    h.initialize().await;
    for timestamp in (0..5400).step_by(10) {
        let l1_head = timestamp / 12;
        h.head.send_replace(Some(h.l1.at(l1_head)));
        h.parent.block_info.timestamp = timestamp;
        if prefetch_first {
            h.drive(use_actor, true).await;
        }
        let calls = h.l1.calls.load(Ordering::SeqCst);
        let origin = h.selector.next_l1_origin(h.parent, false).await.unwrap();
        assert_eq!(
            h.attributes.header_by_hash(origin.hash).await.unwrap().hash_slow(),
            origin.hash
        );
        assert_eq!(h.attributes.receipts_by_hash(origin.hash).await.unwrap().len(), 1);
        assert_eq!(
            h.l1.calls.load(Ordering::SeqCst),
            calls,
            "selection and attributes must not issue RPCs"
        );
        assert!(origin.number <= l1_head.saturating_sub(4));
        assert!(
            l1_head.saturating_sub(origin.number) <= 5,
            "origin lag at {timestamp}: head {l1_head}, origin {}",
            origin.number
        );
        // Adoption asks for the lookahead immediately; no unsafe-head update is delivered.
        h.drive(use_actor, !prefetch_first).await;
        h.parent.l1_origin = origin.id();
        tokio::time::advance(Duration::from_secs(10)).await;
    }
}

#[tokio::test]
async fn receipts_must_be_ready_before_origin_is_published() {
    let mut h = Harness::new(0);
    h.initialize().await;
    let next = h.l1.at(1);
    let calls = h.l1.calls.load(Ordering::SeqCst);
    h.parent.block_info.timestamp = 10;
    let origin = h.selector.next_l1_origin(h.parent, false).await.unwrap();
    assert_eq!(origin, next);
    assert_eq!(h.attributes.header_by_hash(next.hash).await.unwrap().hash_slow(), next.hash);
    assert_eq!(h.attributes.receipts_by_hash(next.hash).await.unwrap().len(), 1);
    assert_eq!(h.l1.calls.load(Ordering::SeqCst), calls);
    // Selection prepares a further lookahead before the unsafe-head origin advances, while
    // retaining the parent in case building on the selected origin fails.
    h.fill(false).await.unwrap();
    assert!(h.attributes.get(h.l1.at(3).hash).is_ok());
    assert!(h.attributes.get(h.parent.l1_origin.hash).is_ok());
}

#[tokio::test(start_paused = true)]
async fn recovery_waits_for_available_successor_below_max_drift() {
    let mut h = Harness::new(4);
    h.head.send_replace(Some(h.l1.at(0)));
    h.initialize().await;
    h.parent.block_info.timestamp = 10;
    h.head.send_replace(Some(h.l1.at(5)));
    // Normal sequencing can reuse the current origin without waiting for preparation.
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap(), h.l1.at(0));
    let origin = tokio::select! {
        result = h.selector.next_l1_origin(h.parent, true) => result.unwrap(),
        _ = async { loop { h.worker.step().await.unwrap(); } } => unreachable!(),
    };
    assert_eq!(origin, h.l1.at(1));
    assert!(h.attributes.get(origin.hash).is_ok());
}

#[tokio::test(start_paused = true)]
async fn recovery_wait_for_stalled_successor_is_bounded_below_max_drift() {
    let mut h = Harness::new(4);
    h.head.send_replace(Some(h.l1.at(0)));
    h.initialize().await;
    h.parent.block_info.timestamp = 10;
    h.head.send_replace(Some(h.l1.at(5)));
    h.l1.stalled.store(true, Ordering::SeqCst);
    let start = tokio::time::Instant::now();
    let result = tokio::select! {
        result = h.selector.next_l1_origin(h.parent, true) => result,
        _ = async { loop { h.worker.step().await.unwrap(); } } => unreachable!(),
    };
    assert!(matches!(result, Err(L1OriginSelectorError::Timeout)));
    assert_eq!(start.elapsed(), Duration::from_millis(500));
}

#[tokio::test]
async fn orphaned_marker_for_another_hash_at_the_same_height_does_not_reject_successor() {
    let mut h = Harness::new(0);
    h.initialize().await;
    h.worker.window.send_modify(|window| {
        window.orphaned = Some(BlockNumHash { number: 0, hash: B256::with_last_byte(99) });
    });
    h.parent.block_info.timestamp = 10;
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap(), h.l1.at(1));
}

#[tokio::test(start_paused = true)]
async fn prefetch_retries_do_not_burst_after_a_stall() {
    let mut h = Harness::new(0);
    tokio::time::advance(Duration::from_secs(10)).await;
    let start = tokio::time::Instant::now();
    h.worker.retry.tick().await;
    assert_eq!(start.elapsed(), Duration::ZERO);
    h.worker.retry.tick().await;
    assert_eq!(start.elapsed(), Duration::from_secs(1));
    h.worker.retry.tick().await;
    assert_eq!(start.elapsed(), Duration::from_secs(2));
}

#[tokio::test(start_paused = true)]
async fn stalled_rpc_is_off_path_and_max_drift_wait_is_bounded() {
    let mut h = Harness::new(4);
    h.head.send_replace(Some(h.l1.at(0)));
    h.initialize().await;
    h.l1.stalled.store(true, Ordering::SeqCst);
    // Cached current header remains usable even with an unavailable L1.
    let start = tokio::time::Instant::now();
    assert!(h.selector.next_l1_origin(h.parent, false).await.is_ok());
    assert_eq!(tokio::time::Instant::now(), start);
    h.parent.block_info.timestamp = 600;
    assert!(matches!(
        h.selector.next_l1_origin(h.parent, false).await,
        Err(L1OriginSelectorError::Timeout)
    ));
    assert_eq!(start.elapsed(), Duration::from_millis(500));
    h.head.send_replace(Some(h.l1.at(10)));
    assert!(matches!(h.fill(true).await, Err(L1OriginSelectorError::Timeout)));
}

#[tokio::test]
async fn orphaned_current_resets_before_drift_limit_including_recovery() {
    for recovery in [false, true] {
        let mut h = Harness::new(0);
        h.initialize().await;
        h.l1.blocks.lock().unwrap()[1].parent_hash = B256::with_last_byte(99);
        h.fill(true).await.unwrap();
        assert!(matches!(
            h.selector.next_l1_origin(h.parent, recovery).await,
            Err(L1OriginSelectorError::NextOriginOrphaned)
        ));
    }
}

#[tokio::test]
async fn orphaned_lookahead_is_not_adopted_and_reset_replaces_window() {
    let mut h = Harness::new(0);
    h.initialize().await;
    h.l1.blocks.lock().unwrap()[2].parent_hash = B256::with_last_byte(99);
    h.fill(true).await.unwrap();
    h.parent.block_info.timestamp = 10;
    assert!(matches!(
        h.selector.next_l1_origin(h.parent, false).await,
        Err(L1OriginSelectorError::NextOriginOrphaned)
    ));
    // An engine reset to a different canonical parent can prepare a new window.
    h.parent.l1_origin = h.l1.at(2).id();
    assert!(matches!(
        h.selector.next_l1_origin(h.parent, false).await,
        Err(L1OriginSelectorError::Pending)
    ));
    h.fill(true).await.unwrap();
    // The mock block 3 still points at the old block 2, so that successor is orphaned too.
    assert_eq!(h.worker.window.borrow().blocks[0].info.id(), h.parent.l1_origin);
}

#[tokio::test]
async fn confirmation_depth_waits_for_head_and_handles_head_regression() {
    let mut h = Harness::new(4);
    h.initialize().await;
    h.parent.block_info.timestamp = 10;
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap().number, 0);
    h.head.send_replace(Some(h.l1.at(5)));
    h.fill(true).await.unwrap();
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap().number, 1);
    h.head.send_replace(Some(h.l1.at(4)));
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap().number, 0);
}

#[tokio::test]
async fn worker_failure_is_terminal_for_selector() {
    let mut h = Harness::new(0);
    h.initialize().await;
    drop(h.worker);
    assert!(matches!(
        h.selector.next_l1_origin(h.parent, false).await,
        Err(L1OriginSelectorError::Closed)
    ));
}

#[tokio::test(start_paused = true)]
async fn reset_cancels_stalled_fetch_before_stale_publication() {
    let mut h = Harness::new(0);
    h.initialize().await;
    // Discard initial notifications, then make a new request for an uncached origin.
    h.worker.request.borrow_and_update();
    h.worker.l1_head.borrow_and_update();
    h.worker.retry.tick().await;
    h.parent.l1_origin = h.l1.at(10).id();
    assert!(matches!(
        h.selector.next_l1_origin(h.parent, false).await,
        Err(L1OriginSelectorError::Pending)
    ));
    h.l1.stalled.store(true, Ordering::SeqCst);
    let mut step = Box::pin(h.worker.step());
    assert!(futures::poll!(&mut step).is_pending());
    h.parent.l1_origin = h.l1.at(20).id();
    assert!(matches!(
        h.selector.next_l1_origin(h.parent, false).await,
        Err(L1OriginSelectorError::Pending)
    ));
    step.await.unwrap();
    assert!(h.attributes.get(h.l1.at(10).hash).is_err());
    h.l1.stalled.store(false, Ordering::SeqCst);
    h.fill(true).await.unwrap();
    assert_eq!(h.worker.window.borrow().blocks[0].info.id(), h.parent.l1_origin);
}

#[tokio::test(start_paused = true)]
async fn slow_receipts_are_never_exposed_as_a_ready_origin() {
    let mut h = Harness::new(4);
    h.head.send_replace(Some(h.l1.at(4)));
    h.initialize().await;
    h.head.send_replace(Some(h.l1.at(5)));
    h.parent.block_info.timestamp = 10;
    h.l1.stalled_receipts.store(true, Ordering::SeqCst);
    let request = (*h.worker.request.borrow_and_update()).unwrap();
    let mut fill = Box::pin(h.worker.fill(request, true));
    assert!(futures::poll!(&mut fill).is_pending());
    let calls = h.l1.calls.load(Ordering::SeqCst);
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap().number, 0);
    assert!(h.attributes.get(h.l1.at(1).hash).is_err());
    assert_eq!(h.l1.calls.load(Ordering::SeqCst), calls);
    assert!(matches!(fill.await, Err(L1OriginSelectorError::Timeout)));
    h.l1.stalled_receipts.store(false, Ordering::SeqCst);
    h.fill(true).await.unwrap();
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap().number, 1);
}

#[tokio::test(start_paused = true)]
async fn failed_head_recheck_is_retried_without_another_head_update() {
    let mut h = Harness::new(0);
    h.head.send_replace(Some(h.l1.at(4)));
    h.initialize().await;
    h.worker.l1_head.borrow_and_update();
    h.worker.request.borrow_and_update();
    h.worker.retry.tick().await;
    // Reorg successors while preserving the canonical current origin.
    {
        let mut blocks = h.l1.blocks.lock().unwrap();
        blocks[1].extra_data = vec![1].into();
        blocks[2].parent_hash = blocks[1].hash_slow();
        blocks[3].parent_hash = blocks[2].hash_slow();
        blocks[4].parent_hash = blocks[3].hash_slow();
    }
    h.head.send_replace(Some(h.l1.at(4)));
    h.l1.stalled.store(true, Ordering::SeqCst);
    h.worker.step().await.unwrap(); // canonical recheck times out
    h.l1.stalled.store(false, Ordering::SeqCst);
    h.worker.step().await.unwrap(); // retry tick must still recheck
    h.parent.block_info.timestamp = 10;
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap(), h.l1.at(1));
}

#[tokio::test(start_paused = true)]
async fn background_receipts_can_exceed_the_max_drift_wait_budget() {
    let mut h = Harness::new(0);
    h.l1.receipt_latency_ms.store(600, Ordering::SeqCst);
    let start = tokio::time::Instant::now();
    h.initialize().await;
    assert!(start.elapsed() >= Duration::from_millis(1800));
    h.parent.block_info.timestamp = 10;
    let calls = h.l1.calls.load(Ordering::SeqCst);
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap().number, 1);
    assert_eq!(h.attributes.receipts_by_hash(h.l1.at(1).hash).await.unwrap().len(), 1);
    assert_eq!(h.l1.calls.load(Ordering::SeqCst), calls);
}

#[tokio::test(start_paused = true)]
async fn canceled_head_recheck_is_retried_without_another_head_update() {
    let mut h = Harness::new(0);
    h.head.send_replace(Some(h.l1.at(4)));
    h.initialize().await;
    h.worker.l1_head.borrow_and_update();
    h.worker.request.borrow_and_update();
    h.worker.retry.tick().await;
    {
        let mut blocks = h.l1.blocks.lock().unwrap();
        blocks[1].parent_hash = B256::with_last_byte(99);
        blocks[2].parent_hash = blocks[1].hash_slow();
    }
    h.head.send_replace(Some(h.l1.at(2)));
    h.l1.stalled.store(true, Ordering::SeqCst);
    let mut step = Box::pin(h.worker.step());
    assert!(futures::poll!(&mut step).is_pending());
    // A recovery-mode change cancels this generation, even though the parent is unchanged.
    h.selector.next_l1_origin(h.parent, true).await.unwrap();
    step.await.unwrap();
    h.l1.stalled.store(false, Ordering::SeqCst);
    h.worker.step().await.unwrap();
    assert!(matches!(
        h.selector.next_l1_origin(h.parent, true).await,
        Err(L1OriginSelectorError::NextOriginOrphaned)
    ));
}

#[tokio::test(start_paused = true)]
async fn partial_reorg_window_is_rechecked_even_when_the_old_head_returns() {
    let mut h = Harness::new(0);
    let old_blocks = h.l1.blocks.lock().unwrap().clone();
    let old_head = h.l1.at(4);
    h.head.send_replace(Some(old_head));
    h.initialize().await;
    h.worker.l1_head.borrow_and_update();
    h.worker.request.borrow_and_update();
    h.worker.retry.tick().await;
    {
        let mut blocks = h.l1.blocks.lock().unwrap();
        blocks[1].extra_data = vec![1].into();
        blocks[2].parent_hash = blocks[1].hash_slow();
        blocks[3].parent_hash = blocks[2].hash_slow();
        blocks[4].parent_hash = blocks[3].hash_slow();
    }
    let fork_origin = h.l1.at(1);
    *h.l1.stalled_receipt_hash.lock().unwrap() = Some(h.l1.at(2).hash);
    h.head.send_replace(Some(h.l1.at(4)));
    let mut step = Box::pin(h.worker.step());
    assert!(futures::poll!(&mut step).is_pending());
    assert!(
        h.attributes.get(fork_origin.hash).is_ok(),
        "first new origin was published before the stalled lookahead"
    );
    *h.l1.blocks.lock().unwrap() = old_blocks;
    h.head.send_replace(Some(old_head));
    step.await.unwrap(); // cancel the partially published fork
    h.l1.stalled.store(true, Ordering::SeqCst);
    h.worker.step().await.unwrap(); // consume and fail the return-to-old-head recheck
    h.l1.stalled.store(false, Ordering::SeqCst);
    h.worker.step().await.unwrap(); // retry without another head notification
    h.parent.block_info.timestamp = 10;
    assert_eq!(h.selector.next_l1_origin(h.parent, false).await.unwrap(), h.l1.at(1));
}

#[tokio::test(start_paused = true)]
async fn max_drift_wait_can_receive_a_prepared_origin_from_the_worker() {
    let mut h = Harness::new(4);
    h.head.send_replace(Some(h.l1.at(0)));
    h.initialize().await;
    h.worker.l1_head.borrow_and_update();
    h.worker.request.borrow_and_update();
    h.worker.retry.tick().await;
    h.parent.block_info.timestamp = 600;
    h.head.send_replace(Some(h.l1.at(5)));
    let origin = tokio::select! {
        result = h.selector.next_l1_origin(h.parent, false) => result.unwrap(),
        _ = async { loop { h.worker.step().await.unwrap(); } } => unreachable!(),
    };
    assert_eq!(origin, h.l1.at(1));
}

#[tokio::test]
async fn missing_origin_resets_even_if_the_selector_remembers_it_from_before_a_reset() {
    let mut h = Harness::new(0);
    h.initialize().await;
    h.selector.next_l1_origin(h.parent, false).await.unwrap();
    let original = h.parent.l1_origin;
    h.parent.l1_origin = h.l1.at(10).id();
    assert!(matches!(
        h.selector.next_l1_origin(h.parent, false).await,
        Err(L1OriginSelectorError::Pending)
    ));
    h.fill(true).await.unwrap();
    h.l1.blocks.lock().unwrap()[0].extra_data = vec![1].into();
    h.parent.l1_origin = original;
    h.selector.next_l1_origin(h.parent, false).await.unwrap();
    h.fill(true).await.unwrap();
    assert!(
        matches!(h.selector.next_l1_origin(h.parent, false).await, Err(L1OriginSelectorError::OriginNotFound(hash)) if hash == original.hash)
    );
}

#[tokio::test(start_paused = true)]
async fn repeated_max_drift_requests_do_not_cancel_the_same_inflight_fetch() {
    let mut h = Harness::new(4);
    h.head.send_replace(Some(h.l1.at(0)));
    h.initialize().await;
    h.worker.l1_head.borrow_and_update();
    h.worker.request.borrow_and_update();
    h.worker.retry.tick().await;
    h.parent.block_info.timestamp = 600;
    h.head.send_replace(Some(h.l1.at(5)));
    h.l1.receipt_latency_ms.store(3000, Ordering::SeqCst);
    let origin = tokio::select! {
        result = async {
            for _ in 0..10 {
                match h.selector.next_l1_origin(h.parent, false).await {
                    Ok(origin) => return origin,
                    Err(L1OriginSelectorError::Timeout) => {},
                    Err(error) => panic!("unexpected selection error: {error}"),
                }
                tokio::time::sleep(Duration::from_millis(500)).await;
            }
            panic!("repeated drift waits starved background receipts preparation");
        } => result,
        _ = async { loop { h.worker.step().await.unwrap(); } } => unreachable!(),
    };
    assert_eq!(origin, h.l1.at(1));
    // Genesis was prepared once and the slow successor was not repeatedly restarted.
    assert_eq!(h.l1.receipts.load(Ordering::SeqCst), 2);
}
