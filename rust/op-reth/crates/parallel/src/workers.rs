//! Shared admission for window and rolling execution.

use crate::{ExecutionGeneration, SpawnWorker, WorkerTask};
use std::{
    collections::VecDeque,
    sync::{Arc, Mutex},
};

struct Queued {
    operation: u64,
    task: WorkerTask,
}

#[derive(Default)]
struct Admission {
    active: usize,
    waiting: VecDeque<Queued>,
}

/// A permit covers execution and reader teardown, including unwinding.
pub(crate) struct Workers {
    spawn: Box<SpawnWorker>,
    limit: usize,
    admission: Mutex<Admission>,
}

impl Workers {
    pub(crate) fn new(limit: usize, spawn: Box<SpawnWorker>) -> Arc<Self> {
        Arc::new(Self { spawn, limit, admission: Mutex::default() })
    }

    pub(crate) fn submit(
        self: &Arc<Self>,
        operation: u64,
        cancelled: Option<&ExecutionGeneration>,
        task: WorkerTask,
    ) {
        let next = {
            let mut state =
                self.admission.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
            // Cancellation and queue removal synchronize through this lock. A task reserved by
            // a racing refill cannot enter the queue after cancel-and-drain removed its peers.
            if cancelled.is_some_and(ExecutionGeneration::is_cancelled) {
                drop(state);
                drop(task);
                return;
            }
            state.waiting.push_back(Queued { operation, task });
            if state.active < self.limit {
                state.active += 1;
                state.waiting.pop_front()
            } else {
                None
            }
        };
        if let Some(next) = next {
            self.spawn(next);
        }
    }

    pub(crate) fn cancel_queued(&self, operation: u64) {
        let cancelled = {
            let mut state =
                self.admission.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
            let mut cancelled = Vec::new();
            let mut kept = VecDeque::new();
            for job in state.waiting.drain(..) {
                if job.operation == operation {
                    cancelled.push(job);
                } else {
                    kept.push_back(job);
                }
            }
            state.waiting = kept;
            cancelled
        };
        // Completion guards may wake coordinators or admit other work. Never drop them while
        // holding the global admission lock.
        drop(cancelled);
    }

    pub(crate) fn other_waiting(&self, operation: u64) -> bool {
        self.admission
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .waiting
            .iter()
            .any(|job| job.operation != operation)
    }

    fn spawn(self: &Arc<Self>, next: Queued) {
        let permit = Permit(self.clone());
        (self.spawn)(Box::new(move || {
            let _permit = permit;
            let _active = ActiveWorker(std::time::Instant::now());
            metrics::gauge!("optimism_parallel.active_workers").increment(1.0);
            (next.task)();
        }));
    }

    fn release(self: &Arc<Self>) {
        let next = {
            let mut state =
                self.admission.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
            let next = state.waiting.pop_front();
            if next.is_none() {
                state.active -= 1;
            }
            next
        };
        if let Some(next) = next {
            self.spawn(next);
        }
    }
}

struct Permit(Arc<Workers>);

impl Drop for Permit {
    fn drop(&mut self) {
        self.0.release();
    }
}

struct ActiveWorker(std::time::Instant);
impl Drop for ActiveWorker {
    fn drop(&mut self) {
        metrics::gauge!("optimism_parallel.active_workers").decrement(1.0);
        metrics::histogram!("optimism_parallel.worker_lease_seconds")
            .record(self.0.elapsed().as_secs_f64());
    }
}
