//! Bounded optimistic execution against a frozen committed prefix.
//!
//! Workers read immutable committed snapshots over independently opened base readers, or request
//! reads from the coordinator when no compatible source exists. Both backends support thread-affine
//! providers without sharing a mutable `State` or database transaction across workers. Windows
//! drain before returning; rolling direct execution publishes individual results while other
//! workers continue. The caller validates and commits results in authoritative order.
//! Workers never consume speculative writes. A failed/missing result is a request to run the
//! canonical sequential path, never an authoritative transaction-validation error.

mod database;
mod pipeline;
mod snapshot;
mod workers;
pub use pipeline::ExecutionPipeline;
pub use snapshot::{CommittedSnapshot, ReadWindow, SnapshotSession, StateReadFactory};
#[cfg(test)]
mod tests;
pub use database::{Dependencies, SpeculationError, SpeculativeDatabase};

use rayon::ThreadPoolBuilder;
use revm::Database;
use std::{
    fmt,
    panic::{AssertUnwindSafe, catch_unwind},
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, AtomicU64, Ordering},
        mpsc,
    },
    time::Instant,
};

/// Identity and cancellation signal for a single immutable parent/build generation.
#[derive(Debug, Clone)]
pub struct ExecutionGeneration {
    id: u64,
    cancelled: Arc<AtomicBool>,
}

impl Default for ExecutionGeneration {
    fn default() -> Self {
        static NEXT: AtomicU64 = AtomicU64::new(1);
        Self {
            id: NEXT.fetch_add(1, Ordering::Relaxed),
            cancelled: Arc::new(AtomicBool::new(false)),
        }
    }
}

impl ExecutionGeneration {
    /// Stable identity shared with every worker of this generation.
    pub const fn id(&self) -> u64 {
        self.id
    }
    /// Invalidates all pending outputs and subsequent reads belonging to this generation.
    pub fn cancel(&self) {
        self.cancelled.store(true, Ordering::Release);
    }
    /// Whether this generation has been cancelled.
    pub fn is_cancelled(&self) -> bool {
        self.cancelled.load(Ordering::Acquire)
    }
}

/// Runtime counters, useful for tests, benchmarking, and operational inspection.
#[derive(Debug, Default, Clone, Copy)]
pub struct ParallelStatistics {
    /// Logical reads served without a coordinator request.
    pub direct_reads: u64,
    /// Logical reads served by the coordinator.
    pub broker_reads: u64,
    /// Direct reads satisfied by the committed overlay.
    pub snapshot_hits: u64,
    /// Independent readers opened on workers.
    pub provider_opens: u64,
    /// Transactions dispatched to workers.
    pub attempted: u64,
    /// Worker executions that returned complete bounded outputs.
    pub completed: u64,
    /// Validated outputs used by authoritative execution.
    pub reused: u64,
    /// Outputs invalidated by committed dependencies or environment changes.
    pub conflicts: u64,
    /// Shadow comparisons that matched the reference.
    pub shadow_matches: u64,
    /// Shadow comparisons that differed from the reference.
    pub shadow_mismatches: u64,
}

#[derive(Default)]
struct Counters {
    direct_reads: AtomicU64,
    broker_reads: AtomicU64,
    snapshot_hits: AtomicU64,
    provider_opens: AtomicU64,
    attempted: AtomicU64,
    completed: AtomicU64,
    reused: AtomicU64,
    conflicts: AtomicU64,
    shadow_matches: AtomicU64,
    shadow_mismatches: AtomicU64,
}

/// An owned job accepted by a shared worker pool.
pub type WorkerTask = Box<dyn FnOnce() + Send>;

/// Isolated transaction execution callback.
pub type SpeculativeWorker<Job, Output> =
    dyn Fn(Job, &mut SpeculativeDatabase) -> Result<Output, SpeculationError> + Send + Sync;

type SpawnWorker = dyn Fn(WorkerTask) + Send + Sync;

/// How workers obtain state. Auto prefers independent readers when a source is available.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub enum StateReads {
    /// Prefer direct reads; retain the broker for unsupported integrations.
    #[default]
    Auto,
    /// Always request state from the coordinator, for comparison and fallback.
    Broker,
}

/// Scheduling strategy within shadow or parallel execution.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub enum ExecutionScheduler {
    /// Drain each speculative window before committing.
    #[default]
    Window,
    /// Overlap ordered commit with independent-reader execution.
    Rolling,
}

impl fmt::Display for ExecutionScheduler {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            Self::Window => "window",
            Self::Rolling => "rolling",
        })
    }
}

impl core::str::FromStr for ExecutionScheduler {
    type Err = &'static str;
    fn from_str(value: &str) -> Result<Self, Self::Err> {
        match value {
            "window" => Ok(Self::Window),
            "rolling" => Ok(Self::Rolling),
            _ => Err("expected window or rolling"),
        }
    }
}

impl fmt::Display for StateReads {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            Self::Auto => "auto",
            Self::Broker => "broker",
        })
    }
}

impl core::str::FromStr for StateReads {
    type Err = &'static str;
    fn from_str(value: &str) -> Result<Self, Self::Err> {
        match value {
            "auto" => Ok(Self::Auto),
            "broker" => Ok(Self::Broker),
            _ => Err("expected auto or broker"),
        }
    }
}

/// Operational mode. The reference executor remains authoritative in shadow mode.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub enum ExecutionMode {
    /// Execute only through the reference path.
    #[default]
    Sequential,
    /// Speculate and compare against reference execution without committing worker results.
    Shadow,
    /// Commit dependency-validated worker results in canonical order.
    Parallel,
}

impl fmt::Display for ExecutionMode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            Self::Sequential => "sequential",
            Self::Shadow => "shadow",
            Self::Parallel => "parallel",
        })
    }
}

impl core::str::FromStr for ExecutionMode {
    type Err = &'static str;
    fn from_str(value: &str) -> Result<Self, Self::Err> {
        match value {
            "sequential" => Ok(Self::Sequential),
            "shadow" => Ok(Self::Shadow),
            "parallel" => Ok(Self::Parallel),
            _ => Err("expected sequential, shadow, or parallel"),
        }
    }
}

/// Per-runtime concurrency and per-executor resource bounds.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ParallelExecutionConfig {
    /// Window scheduling remains the default; rolling requires independent readers.
    pub scheduler: ExecutionScheduler,
    /// Independent source selection.
    pub state_reads: StateReads,
    /// Estimated retained snapshot versions and dirty-key budget per execution session.
    pub max_snapshot_bytes: usize,
    /// Disabled by default.
    pub mode: ExecutionMode,
    /// Shared worker budget; zero selects half the available CPUs, at least one.
    pub workers: usize,
    /// Maximum speculative transactions retained by a block executor.
    pub max_in_flight: usize,
    /// Sum of declared gas limits allowed across speculation in one block/build.
    pub max_speculative_gas: u64,
    /// Maximum retained dependencies per worker.
    pub max_read_bytes: usize,
    /// Maximum retained output, including policy observations, per worker.
    pub max_output_bytes: usize,
}

impl Default for ParallelExecutionConfig {
    fn default() -> Self {
        Self {
            scheduler: ExecutionScheduler::Window,
            state_reads: StateReads::Auto,
            max_snapshot_bytes: 64 * 1024 * 1024,
            mode: ExecutionMode::Sequential,
            workers: 0,
            max_in_flight: 16,
            max_speculative_gas: 30_000_000,
            max_read_bytes: 4 * 1024 * 1024,
            max_output_bytes: 4 * 1024 * 1024,
        }
    }
}

/// A validated runtime configuration owns a persistent worker pool, shared across blocks/builds.
pub struct ParallelRuntime {
    config: ParallelExecutionConfig,
    workers: Arc<workers::Workers>,
    counters: Arc<Counters>,
}

impl fmt::Debug for ParallelRuntime {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("ParallelRuntime").field("config", &self.config).finish_non_exhaustive()
    }
}

impl ParallelRuntime {
    /// Creates a bounded worker pool. Callers avoid constructing one in sequential mode.
    pub fn new(config: ParallelExecutionConfig) -> Result<Self, SpeculationError> {
        if config.max_snapshot_bytes == 0 ||
            config.max_in_flight == 0 ||
            config.max_speculative_gas == 0 ||
            config.max_read_bytes == 0 ||
            config.max_output_bytes == 0
        {
            return Err(SpeculationError::Configuration(
                "parallel resource limits must be positive",
            ));
        }
        let workers = if config.workers == 0 {
            std::thread::available_parallelism().map_or(1, |n| (n.get() / 2).max(1))
        } else {
            config.workers
        };
        let pool = ThreadPoolBuilder::new()
            .num_threads(workers)
            .thread_name(|i| format!("op-speculate-{i}"))
            .build()
            .map_err(|error| SpeculationError::Worker(error.to_string()))?;
        Self::with_spawner(config, workers, move |job| pool.spawn(job))
    }

    /// Shares an existing execution/prewarming pool. `max_workers` is its physical CPU budget;
    /// the configured worker count limits active speculative jobs within that budget.
    pub fn with_spawner(
        mut config: ParallelExecutionConfig,
        max_workers: usize,
        spawn: impl Fn(WorkerTask) + Send + Sync + 'static,
    ) -> Result<Self, SpeculationError> {
        if max_workers == 0 ||
            config.max_snapshot_bytes == 0 ||
            config.max_in_flight == 0 ||
            config.max_speculative_gas == 0 ||
            config.max_read_bytes == 0 ||
            config.max_output_bytes == 0
        {
            return Err(SpeculationError::Configuration(
                "parallel resource limits must be positive",
            ));
        }
        let requested = if config.workers == 0 {
            std::thread::available_parallelism().map_or(1, |n| (n.get() / 2).max(1))
        } else {
            config.workers
        };
        config.workers = requested.min(max_workers);
        Ok(Self {
            workers: workers::Workers::new(config.workers, Box::new(spawn)),
            config,
            counters: Arc::default(),
        })
    }

    /// The configured resource bounds.
    pub const fn config(&self) -> &ParallelExecutionConfig {
        &self.config
    }

    /// Snapshot cumulative execution counters shared by all callers of this runtime.
    pub fn statistics(&self) -> ParallelStatistics {
        let counters = &self.counters;
        ParallelStatistics {
            direct_reads: counters.direct_reads.load(Ordering::Relaxed),
            broker_reads: counters.broker_reads.load(Ordering::Relaxed),
            snapshot_hits: counters.snapshot_hits.load(Ordering::Relaxed),
            provider_opens: counters.provider_opens.load(Ordering::Relaxed),
            attempted: counters.attempted.load(Ordering::Relaxed),
            completed: counters.completed.load(Ordering::Relaxed),
            reused: counters.reused.load(Ordering::Relaxed),
            conflicts: counters.conflicts.load(Ordering::Relaxed),
            shadow_matches: counters.shadow_matches.load(Ordering::Relaxed),
            shadow_mismatches: counters.shadow_mismatches.load(Ordering::Relaxed),
        }
    }

    /// Records a dependency-validated authoritative reuse.
    pub fn record_reuse(&self) {
        self.counters.reused.fetch_add(1, Ordering::Relaxed);
        metrics::counter!("optimism_parallel.reused").increment(1);
    }
    /// Records a dependency conflict requiring a canonical retry.
    pub fn record_conflict(&self) {
        self.counters.conflicts.fetch_add(1, Ordering::Relaxed);
        metrics::counter!("optimism_parallel.conflicts").increment(1);
    }
    /// Records a comparison against canonical sequential output.
    pub fn record_shadow(&self, equal: bool) {
        if equal { &self.counters.shadow_matches } else { &self.counters.shadow_mismatches }
            .fetch_add(1, Ordering::Relaxed);
        metrics::counter!("optimism_parallel.shadow_comparisons", "result" => if equal { "match" } else { "mismatch" }).increment(1);
    }

    /// Runs one bounded window against a frozen canonical database.
    ///
    /// All provider access stays on the calling thread. Each job receives a fresh tracking
    /// database; output order always matches input order, regardless of completion order.
    pub fn execute<DB, Job, Output>(
        &self,
        database: &mut DB,
        jobs: Vec<Job>,
        execute: Arc<SpeculativeWorker<Job, Output>>,
    ) -> Vec<Result<(Output, Dependencies), SpeculationError>>
    where
        DB: Database,
        Job: Send + 'static,
        Output: Send + 'static,
    {
        self.execute_in_generation(database, jobs, execute, &ExecutionGeneration::default())
    }

    /// Runs a window belonging to a specific build generation. Cancellation discards every
    /// output even if a transaction completed before the cancellation signal arrived.
    pub fn execute_in_generation<DB, Job, Output>(
        &self,
        database: &mut DB,
        jobs: Vec<Job>,
        execute: Arc<SpeculativeWorker<Job, Output>>,
        generation: &ExecutionGeneration,
    ) -> Vec<Result<(Output, Dependencies), SpeculationError>>
    where
        DB: Database,
        Job: Send + 'static,
        Output: Send + 'static,
    {
        self.execute_with_reads(database, jobs, execute, generation, None)
    }

    /// Runs a window with an optional immutable snapshot and independent parent reader factory.
    /// Providers are created and destroyed on their worker. The return barrier includes teardown.
    pub fn execute_with_reads<DB, Job, Output>(
        &self,
        database: &mut DB,
        jobs: Vec<Job>,
        execute: Arc<SpeculativeWorker<Job, Output>>,
        generation: &ExecutionGeneration,
        reads: Option<ReadWindow>,
    ) -> Vec<Result<(Output, Dependencies), SpeculationError>>
    where
        DB: Database,
        Job: Send + 'static,
        Output: Send + 'static,
    {
        if jobs.len() > self.config.max_in_flight {
            return jobs.into_iter().map(|_| Err(SpeculationError::Limit)).collect();
        }
        if generation.is_cancelled() ||
            reads.as_ref().is_some_and(|reads| reads.generation.is_cancelled())
        {
            return jobs.into_iter().map(|_| Err(SpeculationError::Cancelled)).collect();
        }
        if reads.is_none() {
            metrics::counter!("optimism_parallel.source_fallbacks", "reason" => if self.config.state_reads == StateReads::Broker { "forced_broker" } else { "source_unavailable" }).increment(1);
        }
        let count = jobs.len();
        let started = Instant::now();
        self.counters.attempted.fetch_add(count as u64, Ordering::Relaxed);
        metrics::counter!("optimism_parallel.attempts").increment(count as u64);
        let (events, receiver) = mpsc::sync_channel(count.max(1));
        let mut results: Vec<_> = (0..count).map(|_| None).collect();
        let jobs = Arc::new(Mutex::new(jobs.into_iter().enumerate()));
        for _ in 0..count.min(self.config.workers) {
            let jobs = jobs.clone();
            let execute = execute.clone();
            let events = events.clone();
            let limit = self.config.max_read_bytes;
            let generation = generation.clone();
            let reads = reads.clone();
            let counters = self.counters.clone();
            self.workers.submit(
                generation.id(),
                None,
                Box::new(move || {
                    let reader =
                        reads.as_ref().map(|reads| open_reader(reads.factory.as_ref(), &counters));
                    loop {
                        let Some((index, job)) =
                            jobs.lock().unwrap_or_else(std::sync::PoisonError::into_inner).next()
                        else {
                            break;
                        };
                        let db = match (&reader, &reads) {
                            (Some(Ok(reader)), Some(reads)) => SpeculativeDatabase::from_reader(
                                reader.clone(),
                                reads.snapshot.clone(),
                                limit,
                                generation.clone(),
                                reads.generation.clone(),
                            ),
                            (Some(Err(error)), _) => {
                                let _ = events
                                    .send(database::Event::Finished(index, Err(error.clone())));
                                continue;
                            }
                            _ => {
                                SpeculativeDatabase::new(events.clone(), limit, generation.clone())
                            }
                        };
                        let result =
                            run_attempt(execute.as_ref(), job, db, &counters, reads.is_some());
                        let _ = events.send(database::Event::Finished(index, result));
                    }
                    // Drop thread-affine readers before disconnecting the completion channel.
                    if catch_unwind(AssertUnwindSafe(|| drop(reader))).is_err() {
                        let _ = events.send(database::Event::SourceFailed);
                    }
                }),
            );
        }
        drop(events);
        let mut source_failed = false;
        loop {
            match receiver.recv() {
                Ok(database::Event::Read(request)) => request.respond(database),
                Ok(database::Event::Finished(index, result)) => {
                    results[index] = Some(result);
                }
                Ok(database::Event::SourceFailed) => source_failed = true,
                Err(_) => break,
            }
        }
        metrics::histogram!("optimism_parallel.window_seconds")
            .record(started.elapsed().as_secs_f64());
        results
            .into_iter()
            .map(|result| {
                if generation.is_cancelled() ||
                    reads.as_ref().is_some_and(|reads| reads.generation.is_cancelled())
                {
                    return Err(SpeculationError::Cancelled);
                }
                if source_failed {
                    return Err(SpeculationError::Source("reader teardown panicked".into()));
                }
                result
                    .unwrap_or_else(|| Err(SpeculationError::Worker("worker disconnected".into())))
            })
            .collect()
    }
}

fn open_reader(
    factory: &dyn StateReadFactory,
    counters: &Counters,
) -> Result<database::LocalReader, SpeculationError> {
    let started = Instant::now();
    let result = catch_unwind(AssertUnwindSafe(|| factory.open()))
        .map_err(|_| SpeculationError::Source("reader initialization panicked".into()))
        .and_then(core::convert::identity)
        .map_err(|error| SpeculationError::Source(error.to_string()))
        .map(|reader| std::rc::Rc::new(std::cell::RefCell::new(reader)));
    metrics::histogram!("optimism_parallel.provider_open_seconds")
        .record(started.elapsed().as_secs_f64());
    if result.is_ok() {
        counters.provider_opens.fetch_add(1, Ordering::Relaxed);
        metrics::counter!("optimism_parallel.provider_opens").increment(1);
    }
    result
}

fn run_attempt<Job, Output>(
    execute: &SpeculativeWorker<Job, Output>,
    job: Job,
    mut db: SpeculativeDatabase,
    counters: &Counters,
    direct: bool,
) -> Result<(Output, Dependencies), SpeculationError> {
    let started = Instant::now();
    let result = catch_unwind(AssertUnwindSafe(|| {
        db.check_generation()?;
        execute(job, &mut db)
    }))
    .map_err(|_| {
        if direct {
            SpeculationError::Source("speculative worker panicked".into())
        } else {
            SpeculationError::Worker("speculative worker panicked".into())
        }
    })
    .and_then(core::convert::identity);
    let elapsed = started.elapsed();
    let read_wait = db.read_wait();
    metrics::histogram!("optimism_parallel.worker_seconds").record(elapsed.as_secs_f64());
    metrics::histogram!("optimism_parallel.evm_seconds")
        .record(elapsed.saturating_sub(read_wait).as_secs_f64());
    metrics::histogram!("optimism_parallel.read_wait_seconds").record(read_wait.as_secs_f64());
    counters.direct_reads.fetch_add(db.direct_reads.get(), Ordering::Relaxed);
    counters.broker_reads.fetch_add(db.broker_reads.get(), Ordering::Relaxed);
    counters.snapshot_hits.fetch_add(db.snapshot_hits.get(), Ordering::Relaxed);
    metrics::counter!("optimism_parallel.direct_reads").increment(db.direct_reads.get());
    metrics::counter!("optimism_parallel.broker_reads").increment(db.broker_reads.get());
    metrics::counter!("optimism_parallel.snapshot_hits").increment(db.snapshot_hits.get());
    if direct {
        metrics::histogram!("optimism_parallel.provider_read_seconds")
            .record(read_wait.as_secs_f64());
    }
    let result = result.and_then(|output| db.finish().map(|reads| (output, reads)));
    if let Ok((_, reads)) = &result {
        counters.completed.fetch_add(1, Ordering::Relaxed);
        metrics::histogram!("optimism_parallel.read_bytes").record(reads.size_bytes() as f64);
    } else {
        metrics::counter!("optimism_parallel.worker_failures").increment(1);
    }
    result
}
