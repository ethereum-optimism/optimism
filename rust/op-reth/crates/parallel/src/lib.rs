//! Bounded optimistic execution against a frozen committed prefix.
//!
//! The coordinator services state reads while its canonical database is frozen. This supports
//! reth's thread-affine providers without sharing a mutable `State` or database transaction across
//! workers. Once the window finishes, results are validated and committed by the caller in order.
//! Workers never consume speculative writes. A failed/missing result is a request to run the
//! canonical sequential path, never an authoritative transaction-validation error.

mod database;
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
    spawn: Box<SpawnWorker>,
    window: Mutex<()>,
    counters: Counters,
}

impl fmt::Debug for ParallelRuntime {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("ParallelRuntime").field("config", &self.config).finish_non_exhaustive()
    }
}

impl ParallelRuntime {
    /// Creates a bounded worker pool. Callers avoid constructing one in sequential mode.
    pub fn new(config: ParallelExecutionConfig) -> Result<Self, SpeculationError> {
        if config.max_in_flight == 0 ||
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
            config,
            spawn: Box::new(spawn),
            window: Mutex::new(()),
            counters: Counters::default(),
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
        // One admitted window across concurrent validation/build callers bounds queued work and
        // retained memory as well as active threads. A panic cannot publish any worker output.
        let _window = self.window.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        if jobs.len() > self.config.max_in_flight {
            return jobs.into_iter().map(|_| Err(SpeculationError::Limit)).collect();
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
            (self.spawn)(Box::new(move || {
                loop {
                    let Some((index, job)) =
                        jobs.lock().unwrap_or_else(std::sync::PoisonError::into_inner).next()
                    else {
                        break;
                    };
                    let mut db =
                        SpeculativeDatabase::new(events.clone(), limit, generation.clone());
                    let started = Instant::now();
                    let result = catch_unwind(AssertUnwindSafe(|| {
                        if generation.is_cancelled() {
                            return Err(SpeculationError::Cancelled);
                        }
                        execute(job, &mut db)
                    }))
                    .map_err(|_| SpeculationError::Worker("speculative worker panicked".into()))
                    .and_then(core::convert::identity);
                    let elapsed = started.elapsed();
                    let read_wait = db.read_wait();
                    metrics::histogram!("optimism_parallel.worker_seconds")
                        .record(elapsed.as_secs_f64());
                    metrics::histogram!("optimism_parallel.evm_seconds")
                        .record(elapsed.saturating_sub(read_wait).as_secs_f64());
                    metrics::histogram!("optimism_parallel.read_wait_seconds")
                        .record(read_wait.as_secs_f64());
                    let result = result.and_then(|output| db.finish().map(|reads| (output, reads)));
                    let _ = events.send(database::Event::Finished(index, result));
                }
            }));
        }
        drop(events);
        let mut remaining = count;
        while remaining != 0 {
            match receiver.recv() {
                Ok(database::Event::Read(request)) => request.respond(database),
                Ok(database::Event::Finished(index, result)) => {
                    if let Ok((_, reads)) = &result {
                        self.counters.completed.fetch_add(1, Ordering::Relaxed);
                        metrics::histogram!("optimism_parallel.read_bytes")
                            .record(reads.size_bytes() as f64);
                    } else {
                        metrics::counter!("optimism_parallel.worker_failures").increment(1);
                    }
                    results[index] = Some(result);
                    remaining -= 1;
                }
                Err(_) => break,
            }
        }
        metrics::histogram!("optimism_parallel.window_seconds")
            .record(started.elapsed().as_secs_f64());
        results
            .into_iter()
            .map(|result| {
                if generation.is_cancelled() {
                    return Err(SpeculationError::Cancelled);
                }
                result
                    .unwrap_or_else(|| Err(SpeculationError::Worker("worker disconnected".into())))
            })
            .collect()
    }
}
