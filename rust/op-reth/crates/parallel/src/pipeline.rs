//! Owned direct-reader work, independent of the coordinator's database borrow.

use crate::{
    Counters, Dependencies, ExecutionGeneration, ParallelExecutionConfig, ParallelRuntime,
    ReadWindow, SpeculationError, SpeculativeDatabase, SpeculativeWorker, StateReadFactory,
    open_reader, run_attempt, workers::Workers,
};
use revm::primitives::B256;
use std::{
    collections::{BTreeMap, VecDeque},
    panic::{AssertUnwindSafe, catch_unwind},
    sync::{Arc, Condvar, Mutex, atomic::Ordering},
    time::Instant,
};

type Outcome<Output> = Result<(Output, Dependencies), SpeculationError>;

enum Status<Output> {
    Queued,
    Running,
    Ready(Outcome<Output>),
    /// The coordinator owns the result, but has not yet committed or rejected it.
    Prepared,
    /// A cancelled running attempt continues occupying its slot until execution ends.
    Retired,
}

struct Attempt<Output> {
    cancel: ExecutionGeneration,
    status: Status<Output>,
}

struct JobInput<Job> {
    hash: B256,
    job: Job,
    reads: ReadWindow,
    admitted: Instant,
}

struct State<Job, Output> {
    jobs: VecDeque<JobInput<Job>>,
    attempts: BTreeMap<B256, Attempt<Output>>,
    tasks: usize,
    closed: bool,
    source_failed: bool,
}

impl<Job, Output> Default for State<Job, Output> {
    fn default() -> Self {
        Self {
            jobs: VecDeque::new(),
            attempts: BTreeMap::new(),
            tasks: 0,
            closed: false,
            source_failed: false,
        }
    }
}

struct Shared<Job, Output> {
    state: Mutex<State<Job, Output>>,
    changed: Condvar,
    generation: ExecutionGeneration,
    source_generation: ExecutionGeneration,
    admission_cancel: ExecutionGeneration,
    factory: Arc<dyn StateReadFactory>,
    execute: Arc<SpeculativeWorker<Job, Output>>,
    config: ParallelExecutionConfig,
    workers: Arc<Workers>,
    counters: Arc<Counters>,
}

/// Bounded, operation-owned speculation. Results can be consumed before other attempts finish.
///
/// A taken result retains its slot until `retire` reports commit or rejection. Dropping this
/// handle cancels and drains all work, including destruction of thread-affine readers.
pub struct ExecutionPipeline<Job: Send + 'static, Output: Send + 'static> {
    shared: Arc<Shared<Job, Output>>,
}

impl<Job: Send + 'static, Output: Send + 'static> std::fmt::Debug
    for ExecutionPipeline<Job, Output>
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("ExecutionPipeline")
            .field("generation", &self.shared.generation.id())
            .field("outstanding", &self.outstanding())
            .finish_non_exhaustive()
    }
}

impl<Job: Send + 'static, Output: Send + 'static> ExecutionPipeline<Job, Output> {
    /// Creates a pipeline bound to one source and build generation. The initial snapshot is
    /// not retained; every submission supplies its own committed version.
    pub fn new(
        runtime: &ParallelRuntime,
        execute: Arc<SpeculativeWorker<Job, Output>>,
        generation: ExecutionGeneration,
        source: &ReadWindow,
    ) -> Self {
        Self {
            shared: Arc::new(Shared {
                state: Mutex::default(),
                changed: Condvar::new(),
                generation,
                source_generation: source.generation.clone(),
                admission_cancel: ExecutionGeneration::default(),
                factory: source.factory.clone(),
                execute,
                config: runtime.config.clone(),
                workers: runtime.workers.clone(),
                counters: runtime.counters.clone(),
            }),
        }
    }

    /// Admits one transaction. Duplicate identities and exhausted slots are rejected.
    pub fn submit(&self, hash: B256, job: Job, reads: ReadWindow) -> Result<(), SpeculationError> {
        self.submit_batch(vec![(hash, job, reads)])
    }

    /// Atomically admits a bounded batch and wakes workers once, amortizing reader setup.
    pub fn submit_batch(&self, jobs: Vec<(B256, Job, ReadWindow)>) -> Result<(), SpeculationError> {
        let count = jobs.len();
        if jobs.iter().any(|(_, _, reads)| {
            !Arc::ptr_eq(&reads.factory, &self.shared.factory) ||
                reads.generation.id() != self.shared.source_generation.id()
        }) {
            return Err(SpeculationError::Configuration("pipeline source changed"));
        }
        {
            let mut state =
                self.shared.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
            if state.source_failed {
                return Err(SpeculationError::Source("pipeline source failed".into()));
            }
            if state.closed || self.shared.cancelled() {
                return Err(SpeculationError::Cancelled);
            }
            let identities: std::collections::BTreeSet<_> =
                jobs.iter().map(|(hash, _, _)| *hash).collect();
            if count > self.shared.config.max_in_flight.saturating_sub(state.attempts.len()) ||
                identities.len() != count ||
                identities.iter().any(|hash| state.attempts.contains_key(hash))
            {
                return Err(SpeculationError::Limit);
            }
            for (hash, job, reads) in jobs {
                state.attempts.insert(
                    hash,
                    Attempt { cancel: ExecutionGeneration::default(), status: Status::Queued },
                );
                state.jobs.push_back(JobInput { hash, job, reads, admitted: Instant::now() });
            }
            metrics::histogram!("optimism_parallel.in_flight").record(state.attempts.len() as f64);
        }
        self.shared.counters.attempted.fetch_add(count as u64, Ordering::Relaxed);
        metrics::counter!("optimism_parallel.attempts").increment(count as u64);
        self.shared.schedule();
        Ok(())
    }

    /// Gives the selected transaction precedence over queued hints and waits only for it.
    /// `None` means this identity has no reusable attempt.
    pub fn take(&self, hash: B256) -> Option<Outcome<Output>> {
        let started = Instant::now();
        let mut state = self.shared.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        if let Some(position) = state.jobs.iter().position(|job| job.hash == hash) {
            let job = state.jobs.remove(position).expect("queued transaction");
            state.jobs.push_front(job);
        }
        let result = loop {
            if state.source_failed {
                break Some(Err(SpeculationError::Source("pipeline source failed".into())));
            }
            if state.closed || self.shared.cancelled() {
                break Some(Err(SpeculationError::Cancelled));
            }
            let Some(attempt) = state.attempts.get_mut(&hash) else { break None };
            match &attempt.status {
                Status::Ready(_) => {
                    let Status::Ready(result) =
                        std::mem::replace(&mut attempt.status, Status::Prepared)
                    else {
                        unreachable!()
                    };
                    break Some(result);
                }
                Status::Retired | Status::Prepared => break None,
                Status::Queued | Status::Running => {
                    state = self
                        .shared
                        .changed
                        .wait(state)
                        .unwrap_or_else(std::sync::PoisonError::into_inner);
                }
            }
        };
        metrics::histogram!("optimism_parallel.head_wait_seconds")
            .record(started.elapsed().as_secs_f64());
        result
    }

    /// Releases a committed/rejected attempt, or cancels a hint that will not be selected.
    pub fn retire(&self, hash: B256) {
        let mut state = self.shared.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        let Some(attempt) = state.attempts.get_mut(&hash) else { return };
        attempt.cancel.cancel();
        if matches!(attempt.status, Status::Running | Status::Retired) {
            attempt.status = Status::Retired;
        } else {
            state.attempts.remove(&hash);
            state.jobs.retain(|job| job.hash != hash);
        }
        self.shared.changed.notify_all();
    }

    /// All retained attempts, including running cancellations and prepared output.
    pub fn outstanding(&self) -> usize {
        self.shared.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner).attempts.len()
    }

    /// Whether an attempt still owns its slot (including cancellation awaiting completion).
    pub fn contains(&self, hash: B256) -> bool {
        self.shared
            .state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .attempts
            .contains_key(&hash)
    }

    /// A failure disables direct reads for the owning snapshot session.
    pub fn source_failed(&self) -> bool {
        self.shared.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner).source_failed
    }

    /// Cancels all attempts and waits for worker-local reader destruction. Safe to repeat.
    pub fn cancel_and_drain(&self) {
        let started = Instant::now();
        let mut state = self.shared.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        state.closed = true;
        self.shared.admission_cancel.cancel();
        Shared::<Job, Output>::discard(&mut state);
        drop(state);
        self.shared.workers.cancel_queued(self.shared.generation.id());
        let mut state = self.shared.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        while state.tasks != 0 {
            state =
                self.shared.changed.wait(state).unwrap_or_else(std::sync::PoisonError::into_inner);
        }
        state.attempts.clear();
        metrics::histogram!("optimism_parallel.pipeline_drain_seconds")
            .record(started.elapsed().as_secs_f64());
    }
}

impl<Job: Send + 'static, Output: Send + 'static> Drop for ExecutionPipeline<Job, Output> {
    fn drop(&mut self) {
        self.cancel_and_drain();
    }
}

impl<Job: Send + 'static, Output: Send + 'static> Shared<Job, Output> {
    fn cancelled(&self) -> bool {
        self.generation.is_cancelled() || self.source_generation.is_cancelled()
    }

    fn discard(state: &mut State<Job, Output>) {
        state.jobs.clear();
        state.attempts.retain(|_, attempt| {
            attempt.cancel.cancel();
            if matches!(attempt.status, Status::Running | Status::Retired) {
                attempt.status = Status::Retired;
                true
            } else {
                false
            }
        });
    }

    fn fail_source(&self) {
        let mut state = self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        state.source_failed = true;
        self.admission_cancel.cancel();
        Self::discard(&mut state);
        self.changed.notify_all();
    }

    fn schedule(self: &Arc<Self>) {
        let count = {
            let mut state = self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
            if state.closed || state.source_failed || self.cancelled() {
                return;
            }
            let count = self.config.workers.saturating_sub(state.tasks).min(state.jobs.len());
            state.tasks += count;
            count
        };
        for _ in 0..count {
            let task = Task { shared: self.clone(), started: false };
            self.workers.submit(
                self.generation.id(),
                Some(&self.admission_cancel),
                Box::new(move || task.run()),
            );
        }
    }

    fn next(&self) -> Option<(JobInput<Job>, ExecutionGeneration)> {
        let mut state = self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        if state.closed || state.source_failed || self.cancelled() {
            return None;
        }
        let job = state.jobs.pop_front()?;
        let attempt = state.attempts.get_mut(&job.hash).expect("admitted attempt");
        attempt.status = Status::Running;
        Some((job, attempt.cancel.clone()))
    }

    fn run(&self) {
        let Some(mut next) = self.next() else { return };
        let reader = match open_reader(self.factory.as_ref(), &self.counters) {
            Ok(reader) => reader,
            Err(_) => {
                self.fail_source();
                // No EVM was started for this reserved slot.
                self.state
                    .lock()
                    .unwrap_or_else(std::sync::PoisonError::into_inner)
                    .attempts
                    .remove(&next.0.hash);
                return;
            }
        };
        for index in 0..self.config.max_in_flight {
            let (JobInput { hash, job, reads, admitted }, cancel) = next;
            metrics::histogram!("optimism_parallel.queue_wait_seconds")
                .record(admitted.elapsed().as_secs_f64());
            let db = SpeculativeDatabase::from_reader(
                reader.clone(),
                reads.snapshot.clone(),
                self.config.max_read_bytes,
                self.generation.clone(),
                reads.generation.clone(),
            )
            .with_attempt_generation(cancel);
            // The job and database release their leases before the outcome becomes visible.
            drop(reads);
            let result = run_attempt(self.execute.as_ref(), job, db, &self.counters, true);
            if matches!(result, Err(SpeculationError::Source(_))) {
                self.fail_source();
            }
            {
                let mut state =
                    self.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
                if state.closed ||
                    state.source_failed ||
                    self.cancelled() ||
                    state
                        .attempts
                        .get(&hash)
                        .is_some_and(|attempt| matches!(attempt.status, Status::Retired))
                {
                    state.attempts.remove(&hash);
                } else if let Some(attempt) = state.attempts.get_mut(&hash) {
                    attempt.status = Status::Ready(result);
                }
                self.changed.notify_all();
            }
            if index + 1 == self.config.max_in_flight ||
                self.workers.other_waiting(self.generation.id())
            {
                break;
            }
            match self.next() {
                Some(job) => next = job,
                None => break,
            }
        }
        if catch_unwind(AssertUnwindSafe(|| drop(reader))).is_err() {
            self.fail_source();
        }
    }
}

// A spawner may discard queued work during shutdown. Its ownership still carries completion,
// so cancellation cannot wait forever for a closure that will never run.
struct Task<Job: Send + 'static, Output: Send + 'static> {
    shared: Arc<Shared<Job, Output>>,
    started: bool,
}

impl<Job: Send + 'static, Output: Send + 'static> Task<Job, Output> {
    fn run(mut self) {
        self.started = true;
        if catch_unwind(AssertUnwindSafe(|| self.shared.run())).is_err() {
            self.shared.fail_source();
        }
    }
}

impl<Job: Send + 'static, Output: Send + 'static> Drop for Task<Job, Output> {
    fn drop(&mut self) {
        {
            let mut state =
                self.shared.state.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
            if !self.started && !state.closed && !state.source_failed {
                state.source_failed = true;
                self.shared.admission_cancel.cancel();
                Shared::<Job, Output>::discard(&mut state);
            }
            state.tasks -= 1;
            self.shared.changed.notify_all();
        }
        self.shared.schedule();
    }
}
