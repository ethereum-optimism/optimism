//! Background page-cache warming ahead of persistence.
//!
//! Persistence reads its pages one at a time on a single thread, so on network-attached disks it
//! waits on one read at a time. A [`Prefetcher`] reads the same pages earlier, from several
//! read-only transactions at once:
//!
//! - when a block enters the in-memory buffer, the pages its write will touch;
//! - when a batch has been persisted, the pages the prunes of the next two batches will touch, so
//!   that a prune's pages are read a whole batch before it runs.
//!
//! Warming only reads. Work that is dropped, late or failed costs performance, never correctness.

use super::{EngineConfig, service_guard::ServiceGuard};
use crate::{BlockStateDiff, OpProofStoragePruner, OpProofsStore, PrunerError};
use alloy_eips::eip1898::BlockWithParent;
use parking_lot::{Condvar, Mutex};
use reth_provider::{BlockHashReader, StageCheckpointReader};
use std::{
    collections::VecDeque,
    panic,
    sync::{Arc, Weak},
    thread,
    time::Instant,
};
use tracing::{debug, error};

/// Most blocks one prune warming pass reads. This bounds the pass when a prune has a backlog to
/// remove: only the first blocks of the backlog are then warmed.
const MAX_PRUNE_WARM_BLOCKS: u64 = 128;

/// A block in the engine's in-memory buffer.
type BufferedBlock = (BlockWithParent, BlockStateDiff);

/// A unit of warming work.
enum Job {
    /// Warm the pages persisting this block will write, if the block is still buffered.
    Write(Weak<BufferedBlock>),
    /// Warm the pages the next prunes will touch, with this many more blocks stored by then.
    Prune(u64),
}

impl Job {
    const fn kind(&self) -> &'static str {
        match self {
            Self::Write(_) => "write",
            Self::Prune(_) => "prune",
        }
    }
}

/// Warming work waiting for the warming thread.
#[derive(Default)]
struct Pending {
    /// Blocks whose write is still to be warmed, oldest first. The buffer owns the blocks: one
    /// that was persisted or unwound in the meantime is gone, and needs no warming.
    writes: VecDeque<Weak<BufferedBlock>>,
    /// Set while prunes are still to be warmed: the blocks stored before they run.
    prune: Option<u64>,
    /// Set when the prefetcher is dropped.
    closed: bool,
}

/// Hands warming work from the engine thread to the warming thread.
struct Queue {
    pending: Mutex<Pending>,
    /// Signalled when `pending` gains work or closes.
    changed: Condvar,
    /// Most blocks kept in [`Pending::writes`].
    capacity: usize,
}

impl Queue {
    fn new(capacity: usize) -> Self {
        Self { pending: Mutex::default(), changed: Condvar::new(), capacity }
    }

    /// Adds a block whose write is to be warmed. When full, the oldest blocks make room: they
    /// are persisted soonest, so warming them helps least. Returns how many were dropped.
    fn push_write(&self, block: Weak<BufferedBlock>) -> usize {
        let mut pending = self.pending.lock();
        let dropped = (pending.writes.len() + 1).saturating_sub(self.capacity);
        pending.writes.drain(..dropped);
        pending.writes.push_back(block);
        drop(pending);
        self.changed.notify_one();
        dropped
    }

    /// Asks for the next prunes to be warmed, with `incoming` more blocks stored by then. It
    /// replaces a request the warming thread has not taken yet.
    fn request_prune(&self, incoming: u64) {
        self.pending.lock().prune = Some(incoming);
        self.changed.notify_one();
    }

    /// Waits for work and returns it, or `None` once closed. Prunes go first: a prune costs
    /// persistence more than a write, and one pass covers a whole batch.
    fn next(&self) -> Option<Job> {
        let mut pending = self.pending.lock();
        loop {
            if pending.closed {
                return None;
            }
            if let Some(incoming) = pending.prune.take() {
                return Some(Job::Prune(incoming));
            }
            if let Some(block) = pending.writes.pop_front() {
                return Some(Job::Write(block));
            }
            self.changed.wait(&mut pending);
        }
    }

    /// Discards the waiting work and stops the warming thread after its current job.
    fn close(&self) {
        *self.pending.lock() = Pending { closed: true, ..Default::default() };
        self.changed.notify_one();
    }
}

/// Warms the page cache ahead of persistence on a background thread.
pub(crate) struct Prefetcher {
    queue: Arc<Queue>,
    /// Blocks in a persistence batch while the buffer is not backed up.
    persistence_threshold: u64,
    #[cfg(feature = "metrics")]
    metrics: metrics::PrefetchMetrics,
    /// Joins the warming thread on drop, once it has finished its current job.
    _worker: ServiceGuard,
}

impl Prefetcher {
    /// Spawns the warming thread. Each of its jobs reads from `config.prefetch_threads` parallel
    /// transactions.
    pub(crate) fn spawn<S, H>(
        storage: S,
        pruner: OpProofStoragePruner<S, H>,
        config: &EngineConfig,
    ) -> Self
    where
        S: OpProofsStore + 'static,
        H: BlockHashReader + StageCheckpointReader + Send + 'static,
    {
        // No more blocks than the backpressure threshold can be waiting to be persisted.
        let queue = Arc::new(Queue::new(config.backpressure_threshold.max(1) as usize));
        #[cfg(feature = "metrics")]
        let metrics = metrics::PrefetchMetrics::default();
        let worker = Worker {
            queue: queue.clone(),
            storage,
            pruner,
            threads: config.prefetch_threads,
            prune_warmed_through: 0,
            #[cfg(feature = "metrics")]
            metrics: metrics.clone(),
        };
        let worker = thread::Builder::new()
            .name("proofs-prefetch".into())
            .spawn(move || worker.run())
            .expect("failed to spawn proofs-prefetch thread");

        Self {
            queue,
            persistence_threshold: config.persistence_threshold,
            #[cfg(feature = "metrics")]
            metrics,
            _worker: ServiceGuard::new(worker),
        }
    }

    /// Queues warming for the pages persisting `block` will write.
    pub(crate) fn warm_write(&self, block: &Arc<BufferedBlock>) {
        let dropped = self.queue.push_write(Arc::downgrade(block));
        #[cfg(feature = "metrics")]
        self.metrics.dropped_jobs_total.increment(dropped as u64);
        #[cfg(not(feature = "metrics"))]
        let _ = dropped;
    }

    /// Queues warming for the next prunes, once a batch is persisted and `buffered` blocks wait
    /// for the next one.
    pub(crate) fn on_persisted(&self, buffered: u64) {
        // The next batch stores the buffered blocks, or a full threshold of blocks if fewer are
        // waiting. Its prune's pages were read a batch ago, so cover the batch after it too.
        self.queue.request_prune(buffered.max(self.persistence_threshold).saturating_mul(2));
    }
}

impl Drop for Prefetcher {
    fn drop(&mut self) {
        self.queue.close();
    }
}

/// The warming thread.
struct Worker<S, H> {
    queue: Arc<Queue>,
    storage: S,
    /// Tells which blocks the next prunes remove.
    pruner: OpProofStoragePruner<S, H>,
    /// Parallel readers per job.
    threads: usize,
    /// Last block whose prune has been warmed.
    prune_warmed_through: u64,
    #[cfg(feature = "metrics")]
    metrics: metrics::PrefetchMetrics,
}

impl<S, H> Worker<S, H>
where
    S: OpProofsStore,
    H: BlockHashReader + StageCheckpointReader,
{
    fn run(mut self) {
        while let Some(job) = self.queue.next() {
            let start = Instant::now();
            // A panic in one job must not end warming for the rest of the process's life.
            match panic::catch_unwind(panic::AssertUnwindSafe(|| self.warm(&job))) {
                Ok(Ok(0)) => {}
                Ok(Ok(keys)) => {
                    #[cfg(feature = "metrics")]
                    self.metrics.record(&job, keys, start.elapsed());
                    #[cfg(not(feature = "metrics"))]
                    let _ = (keys, start);
                }
                Ok(Err(error)) => {
                    #[cfg(feature = "metrics")]
                    self.metrics.failed_jobs_total.increment(1);
                    debug!(target: "trie::engine::prefetch", job = job.kind(), %error, "Page warming failed");
                }
                Err(_) => {
                    #[cfg(feature = "metrics")]
                    self.metrics.failed_jobs_total.increment(1);
                    error!(target: "trie::engine::prefetch", job = job.kind(), "Page warming panicked");
                }
            }
        }
    }

    /// Runs `job` and returns the number of keys it read.
    fn warm(&mut self, job: &Job) -> Result<usize, PrunerError> {
        match job {
            Job::Write(block) => {
                // The block left the buffer before its turn: warming fell behind.
                let Some(block) = block.upgrade() else {
                    #[cfg(feature = "metrics")]
                    self.metrics.dropped_jobs_total.increment(1);
                    return Ok(0);
                };
                Ok(self.storage.warm_for_write(&block.1, self.threads)?)
            }
            Job::Prune(incoming) => {
                let Some(blocks) = self.pruner.upcoming_prune_range(*incoming)? else {
                    return Ok(0);
                };
                // Earlier passes read up to `prune_warmed_through`. The window's earliest block
                // only moves forward, so no block at or below that one needs reading again.
                let first = (*blocks.start()).max(self.prune_warmed_through.saturating_add(1));
                let last = (*blocks.end()).min(first.saturating_add(MAX_PRUNE_WARM_BLOCKS - 1));
                if first > last {
                    return Ok(0);
                }
                let keys = self.storage.warm_for_prune(first..=last, self.threads)?;
                self.prune_warmed_through = last;
                Ok(keys)
            }
        }
    }
}

#[cfg(feature = "metrics")]
mod metrics {
    use super::Job;
    use metrics::{Counter, Histogram};
    use reth_metrics::Metrics;
    use std::time::Duration;

    /// Page warming metrics.
    #[derive(Metrics, Clone)]
    #[metrics(scope = "optimism_trie.engine.prefetch")]
    pub(super) struct PrefetchMetrics {
        /// Time to warm one block's write pages, in seconds.
        pub(super) write_duration_seconds: Histogram,
        /// Time to warm the next prunes' pages, in seconds.
        pub(super) prune_duration_seconds: Histogram,
        /// Keys read while warming write pages.
        pub(super) write_keys_total: Counter,
        /// Keys read while warming prune pages, counted once for each block that changed them.
        pub(super) prune_keys_total: Counter,
        /// Block writes that were not warmed because warming fell behind persistence.
        pub(super) dropped_jobs_total: Counter,
        /// Warming jobs that failed.
        pub(super) failed_jobs_total: Counter,
    }

    impl PrefetchMetrics {
        /// Records a `job` that read `keys` keys.
        pub(super) fn record(&self, job: &Job, keys: usize, duration: Duration) {
            let (durations, total) = match job {
                Job::Write(_) => (&self.write_duration_seconds, &self.write_keys_total),
                Job::Prune(_) => (&self.prune_duration_seconds, &self.prune_keys_total),
            };
            durations.record(duration);
            total.increment(keys as u64);
        }
    }
}

/// A store that reports the warming it is asked for.
#[cfg(test)]
pub(super) mod test_utils {
    use crate::{BlockStateDiff, OpProofsStorageResult, OpProofsStore};
    use crossbeam_channel::Sender;
    use std::{
        ops::RangeInclusive,
        sync::{
            Arc,
            atomic::{AtomicBool, Ordering},
        },
    };

    /// A warming request.
    #[derive(Debug, PartialEq, Eq)]
    pub(in crate::engine) enum Warmed {
        Write,
        Prune(RangeInclusive<u64>),
    }

    /// Wraps a store and reports each warming request on `warmed` before forwarding it.
    #[derive(Debug, Clone)]
    pub(in crate::engine) struct NotifyingStore<S> {
        pub(in crate::engine) inner: S,
        pub(in crate::engine) warmed: Sender<Warmed>,
        /// Makes the next write warming panic instead.
        pub(in crate::engine) panic_next_write: Arc<AtomicBool>,
    }

    impl<S> NotifyingStore<S> {
        pub(in crate::engine) fn new(inner: S, warmed: Sender<Warmed>) -> Self {
            Self { inner, warmed, panic_next_write: Arc::default() }
        }
    }

    impl<S: OpProofsStore> OpProofsStore for NotifyingStore<S> {
        type ProviderRO<'a>
            = S::ProviderRO<'a>
        where
            Self: 'a;
        type ProviderRw<'a>
            = S::ProviderRw<'a>
        where
            Self: 'a;
        type Initializer<'a>
            = S::Initializer<'a>
        where
            Self: 'a;

        fn provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRO<'a>> {
            self.inner.provider_ro()
        }

        fn provider_rw<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRw<'a>> {
            self.inner.provider_rw()
        }

        fn initialization_provider<'a>(&'a self) -> OpProofsStorageResult<Self::Initializer<'a>> {
            self.inner.initialization_provider()
        }

        fn warm_for_write(
            &self,
            diff: &BlockStateDiff,
            threads: usize,
        ) -> OpProofsStorageResult<usize> {
            assert!(!self.panic_next_write.swap(false, Ordering::SeqCst), "warming failed");
            let _ = self.warmed.send(Warmed::Write);
            self.inner.warm_for_write(diff, threads)
        }

        fn warm_for_prune(
            &self,
            blocks: RangeInclusive<u64>,
            threads: usize,
        ) -> OpProofsStorageResult<usize> {
            let _ = self.warmed.send(Warmed::Prune(blocks.clone()));
            self.inner.warm_for_prune(blocks, threads)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{
        test_utils::{NotifyingStore, Warmed},
        *,
    };
    use crate::{MdbxProofsStorage, OpProofsInitProvider, OpProofsProviderRw};
    use alloy_eips::{BlockNumHash, NumHash};
    use alloy_primitives::{B256, U256};
    use crossbeam_channel::Receiver;
    use reth_primitives_traits::Account;
    use reth_provider::{
        ProviderFactory, StageCheckpointWriter,
        test_utils::{MockNodeTypesWithDB, create_test_provider_factory},
    };
    use reth_stages_types::{StageCheckpoint, StageId};
    use reth_trie::HashedPostState;
    use std::{ops::RangeInclusive, sync::atomic::Ordering, time::Duration};
    use tempfile::TempDir;

    type Store = NotifyingStore<Arc<MdbxProofsStorage>>;
    type Pruner = OpProofStoragePruner<Store, ProviderFactory<MockNodeTypesWithDB>>;

    /// Longer than warming ever takes: a test waits this long only when it fails.
    const TIMEOUT: Duration = Duration::from_secs(30);

    /// Batches of 1 block, a buffer of 2, and 2 readers per job.
    const CONFIG: EngineConfig =
        EngineConfig { persistence_threshold: 1, backpressure_threshold: 2, prefetch_threads: 2 };

    fn hash(block: u64) -> B256 {
        B256::from(U256::from(block))
    }

    /// Block `number`, changing one account.
    fn block(number: u64) -> Arc<BufferedBlock> {
        let mut state = HashedPostState::default();
        state.accounts.insert(hash(number), Some(Account::default()));
        let diff = BlockStateDiff { sorted_post_state: state.into_sorted(), ..Default::default() };
        Arc::new((BlockWithParent::new(hash(number - 1), NumHash::new(number, hash(number))), diff))
    }

    fn store_blocks(store: &impl OpProofsStore, blocks: RangeInclusive<u64>) {
        let blocks = blocks.map(|number| Arc::unwrap_or_clone(block(number))).collect();
        let provider = store.provider_rw().unwrap();
        provider.store_trie_updates_batch(blocks).unwrap();
        OpProofsProviderRw::commit(provider).unwrap();
    }

    /// A store holding blocks `0..=latest`, and a pruner that keeps 2 blocks of proofs on a
    /// chain reth has persisted up to block 1000.
    fn store_and_pruner(latest: u64) -> (TempDir, Store, Pruner, Receiver<Warmed>) {
        let dir = TempDir::new().unwrap();
        let (warmed_tx, warmed) = crossbeam_channel::unbounded();
        let store =
            NotifyingStore::new(Arc::new(MdbxProofsStorage::new(dir.path()).unwrap()), warmed_tx);
        let init = store.initialization_provider().unwrap();
        init.set_initial_state_anchor(BlockNumHash::new(0, hash(0))).unwrap();
        init.commit_initial_state().unwrap();
        OpProofsInitProvider::commit(init).unwrap();
        store_blocks(&store, 1..=latest);

        let chain = create_test_provider_factory();
        let provider = chain.provider_rw().unwrap();
        provider.save_stage_checkpoint(StageId::Finish, StageCheckpoint::new(1000)).unwrap();
        provider.commit().unwrap();

        let pruner = OpProofStoragePruner::new(store.clone(), chain, 2);
        (dir, store, pruner, warmed)
    }

    /// A warming thread's state, to run jobs on the test's thread.
    fn worker(store: Store, pruner: Pruner) -> Worker<Store, ProviderFactory<MockNodeTypesWithDB>> {
        Worker {
            queue: Arc::new(Queue::new(1)),
            storage: store,
            pruner,
            threads: CONFIG.prefetch_threads,
            prune_warmed_through: 0,
            #[cfg(feature = "metrics")]
            metrics: metrics::PrefetchMetrics::default(),
        }
    }

    #[test]
    fn queue_keeps_the_newest_blocks_and_serves_prunes_first() {
        let queue = Queue::new(2);
        let blocks: Vec<_> = (1..=3).map(block).collect();
        let dropped: Vec<_> =
            blocks.iter().map(|block| queue.push_write(Arc::downgrade(block))).collect();
        assert_eq!(dropped, [0, 0, 1]);
        queue.request_prune(10);
        // A newer request replaces one the warming thread has not taken yet.
        queue.request_prune(12);

        assert!(matches!(queue.next(), Some(Job::Prune(12))));
        // Block 1, the oldest, made room for block 3.
        let Some(Job::Write(next)) = queue.next() else { panic!("expected a write job") };
        assert_eq!(next.upgrade().unwrap().0.block.number, 2);

        queue.close();
        assert!(queue.next().is_none());
    }

    #[test]
    fn write_jobs_skip_blocks_that_left_the_buffer() {
        let (_dir, store, pruner, warmed) = store_and_pruner(1);
        let mut worker = worker(store, pruner);

        let buffered = block(2);
        let job = Job::Write(Arc::downgrade(&buffered));
        assert_eq!(worker.warm(&job).unwrap(), 1);
        assert_eq!(warmed.try_recv(), Ok(Warmed::Write));

        // Persisting or unwinding the block removes it from the buffer.
        drop(buffered);
        assert_eq!(worker.warm(&job).unwrap(), 0);
        assert!(warmed.try_recv().is_err());
    }

    #[test]
    fn prune_passes_are_bounded_and_continue_where_the_last_one_stopped() {
        // The window holds 200 blocks more than the pruner keeps.
        let (_dir, store, pruner, warmed) = store_and_pruner(202);
        let mut worker = worker(store, pruner);
        let job = Job::Prune(0);

        worker.warm(&job).unwrap();
        assert_eq!(warmed.try_recv(), Ok(Warmed::Prune(1..=MAX_PRUNE_WARM_BLOCKS)));
        worker.warm(&job).unwrap();
        assert_eq!(warmed.try_recv(), Ok(Warmed::Prune(MAX_PRUNE_WARM_BLOCKS + 1..=200)));
        // Nothing is left to read.
        worker.warm(&job).unwrap();
        assert!(warmed.try_recv().is_err());

        // A range that ends below the last block read was read by an earlier pass.
        worker.prune_warmed_through = 500;
        worker.warm(&job).unwrap();
        assert!(warmed.try_recv().is_err());

        // Once the prune has passed the last block read, the next range is read from its start.
        worker.prune_warmed_through = 150;
        let provider = worker.storage.provider_rw().unwrap();
        provider.prune_earliest_state(block(200).0).unwrap();
        OpProofsProviderRw::commit(provider).unwrap();
        store_blocks(&worker.storage, 203..=212);
        worker.warm(&job).unwrap();
        assert_eq!(warmed.try_recv(), Ok(Warmed::Prune(201..=210)));
    }

    #[test]
    fn warms_buffered_blocks_and_the_next_prunes() {
        let (_dir, store, pruner, warmed) = store_and_pruner(10);
        let prefetcher = Prefetcher::spawn(store, pruner, &CONFIG);

        let buffered = block(11);
        prefetcher.warm_write(&buffered);
        assert_eq!(warmed.recv_timeout(TIMEOUT), Ok(Warmed::Write));

        // Two batches of 1 block make the window 0..=12, and their prunes keep 2 blocks.
        prefetcher.on_persisted(0);
        assert_eq!(warmed.recv_timeout(TIMEOUT), Ok(Warmed::Prune(1..=10)));
    }

    #[test]
    fn keeps_warming_after_a_job_panics() {
        let (_dir, store, pruner, warmed) = store_and_pruner(10);
        let prefetcher = Prefetcher::spawn(store.clone(), pruner, &CONFIG);

        // Block 11's job panics, and block 12's still runs after it.
        let buffered = [block(11), block(12)];
        store.panic_next_write.store(true, Ordering::SeqCst);
        prefetcher.warm_write(&buffered[0]);
        prefetcher.warm_write(&buffered[1]);
        assert_eq!(warmed.recv_timeout(TIMEOUT), Ok(Warmed::Write));
        assert!(!store.panic_next_write.load(Ordering::SeqCst));
    }
}
