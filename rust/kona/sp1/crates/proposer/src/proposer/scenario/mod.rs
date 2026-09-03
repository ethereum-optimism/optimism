use std::{
    collections::HashMap,
    sync::{
        Arc, OnceLock,
        atomic::{AtomicBool, Ordering},
    },
    time::Duration,
};

use tokio::sync::Notify;

use super::{CycleResult, Proposer, TaskCompletion, TaskId};

#[derive(Debug, PartialEq, Eq)]
pub(super) enum ScenarioError {
    Cycle(String),
    RunningTask { task_id: TaskId },
    UnknownTask { task_id: TaskId },
    AlreadyFinalized { task_id: TaskId },
    BarrierNotReached { task_id: TaskId, barrier: String },
    BarrierTaskMismatch { task_id: TaskId, barrier: String, reached_by: TaskId },
    SettlementWatchdog { task_ids: Vec<TaskId>, completions: Vec<TaskCompletion> },
}

struct BarrierState {
    name: String,
    reached_by: OnceLock<TaskId>,
    released: AtomicBool,
    reached_notify: Notify,
    release_notify: Notify,
}

#[derive(Clone)]
pub(super) struct NamedBarrier(Arc<BarrierState>);

impl NamedBarrier {
    pub(super) fn new(name: impl Into<String>) -> Self {
        let name = name.into();
        assert!(!name.is_empty(), "barrier name must not be empty");
        Self(Arc::new(BarrierState {
            name,
            reached_by: OnceLock::new(),
            released: AtomicBool::new(false),
            reached_notify: Notify::new(),
            release_notify: Notify::new(),
        }))
    }

    pub(super) async fn park(&self, task_id: TaskId) {
        assert!(
            self.0.reached_by.set(task_id).is_ok(),
            "barrier '{}' cannot be reused",
            self.0.name
        );
        self.0.reached_notify.notify_waiters();
        loop {
            let released = self.0.release_notify.notified();
            if self.0.released.load(Ordering::Acquire) {
                return;
            }
            released.await;
        }
    }

    pub(super) async fn wait_until_reached(&self) {
        loop {
            let reached = self.0.reached_notify.notified();
            if self.0.reached_by.get().is_some() {
                return;
            }
            reached.await;
        }
    }

    pub(super) fn release(&self) {
        self.0.released.store(true, Ordering::Release);
        self.0.release_notify.notify_waiters();
    }
}

pub(super) struct ScenarioControl {
    proposer: Arc<Proposer>,
    parked: HashMap<TaskId, NamedBarrier>,
    settlement_watchdog: Duration,
}

impl ScenarioControl {
    pub(super) fn new(proposer: Arc<Proposer>, settlement_watchdog: Duration) -> Self {
        Self { proposer, parked: HashMap::new(), settlement_watchdog }
    }

    pub(super) async fn tick(&mut self) -> Result<CycleResult, ScenarioError> {
        {
            let tasks = self.proposer.tasks.lock().await;
            for (task_id, (handle, _)) in tasks.iter() {
                let is_parked = self.parked.get(task_id).is_some_and(|barrier| {
                    barrier.0.reached_by.get() == Some(task_id) &&
                        !barrier.0.released.load(Ordering::Acquire)
                });
                if !handle.is_finished() && !is_parked {
                    return Err(ScenarioError::RunningTask { task_id: *task_id });
                }
            }
        }

        let result =
            self.proposer.cycle().await.map_err(|error| ScenarioError::Cycle(error.to_string()))?;
        let tasks = self.proposer.tasks.lock().await;
        self.parked.retain(|task_id, _| tasks.contains_key(task_id));
        Ok(result)
    }

    pub(super) async fn record_parked(
        &mut self,
        task_id: TaskId,
        barrier: &NamedBarrier,
    ) -> Result<(), ScenarioError> {
        let tasks = self.proposer.tasks.lock().await;
        if !tasks.contains_key(&task_id) {
            return Err(self.classify_missing(task_id));
        }
        drop(tasks);
        let Some(&reached_by) = barrier.0.reached_by.get() else {
            return Err(ScenarioError::BarrierNotReached {
                task_id,
                barrier: barrier.0.name.clone(),
            });
        };
        if reached_by != task_id {
            return Err(ScenarioError::BarrierTaskMismatch {
                task_id,
                barrier: barrier.0.name.clone(),
                reached_by,
            });
        }
        self.parked.insert(task_id, barrier.clone());
        Ok(())
    }

    pub(super) async fn settle(
        &mut self,
        task_ids: &[TaskId],
    ) -> Result<Vec<TaskCompletion>, ScenarioError> {
        let mut task_ids = task_ids.to_vec();
        task_ids.sort_unstable();
        task_ids.dedup();
        {
            let tasks = self.proposer.tasks.lock().await;
            if let Some(task_id) = task_ids.iter().find(|task_id| !tasks.contains_key(task_id)) {
                return Err(self.classify_missing(*task_id));
            }
        }

        let completions =
            match self.proposer.finalize_tasks(&task_ids, Some(self.settlement_watchdog)).await {
                Ok(completions) => completions,
                Err(completions) => {
                    return Err(ScenarioError::SettlementWatchdog { task_ids, completions });
                }
            };
        for task_id in &task_ids {
            self.parked.remove(task_id);
        }
        Ok(completions)
    }

    fn classify_missing(&self, task_id: TaskId) -> ScenarioError {
        let next_task_id = self.proposer.next_task_id.load(std::sync::atomic::Ordering::Relaxed);
        if task_id.get() >= next_task_id {
            ScenarioError::UnknownTask { task_id }
        } else {
            ScenarioError::AlreadyFinalized { task_id }
        }
    }
}

#[cfg(test)]
mod tests;
