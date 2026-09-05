//! Sequential in-process canary scheduling.

use std::{
    future::Future,
    num::NonZeroU64,
    pin::Pin,
    sync::atomic::{AtomicU64, Ordering},
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use alloy_primitives::B256;
use anyhow::{Result, ensure};
use tokio::{sync::watch, time::Instant};

use crate::{
    artifact::ValidatedRangeArtifact,
    config::CanaryConfig,
    execution::{
        ExecutionActivity, ExecutionOutcome, ExecutionResult, StageOutcome, execute_snapshot,
    },
    source::{SnapshotSource, ValidatedSnapshot},
};

const MAX_RUN_DETAIL_BYTES: usize = 4096;
static JITTER_COUNTER: AtomicU64 = AtomicU64::new(1);

/// Exhaustive terminal outcome for one attempted fingerprint.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum RunOutcome {
    /// Both guest modes accepted and matched the canonical snapshot.
    Valid,
    /// At least one guest rejected the trusted input.
    GuestRejected,
    /// At least one guest output differed from the canonical expected output.
    OutputMismatch,
    /// Canonical input, witness, or RPC data was unavailable.
    InputError,
    /// Guest execution reached its configured cycle ceiling.
    CycleLimitExceeded,
    /// The attempt deadline elapsed during a cancellable stage.
    Timeout,
}

impl RunOutcome {
    const fn is_correctness_failure(self) -> bool {
        matches!(self, Self::GuestRejected | Self::OutputMismatch)
    }
}

/// One completed attempted execution, including data needed by logs and metrics.
#[derive(Clone, Debug, PartialEq)]
pub struct AttemptResult {
    /// Selected fingerprint, absent when input selection failed.
    pub fingerprint: Option<B256>,
    /// Finalized target timestamp, absent when input selection failed.
    pub target_timestamp: Option<u64>,
    /// Number of timestamps in the selected span.
    pub span_length: Option<u64>,
    /// Number of configured chains in the selected snapshot.
    pub chain_count: Option<usize>,
    /// Whether this was the one confirmation execution for an identical correctness failure.
    pub confirmation: bool,
    /// Exhaustive terminal classification.
    pub outcome: RunOutcome,
    /// In-process execution result, retained whenever execution began.
    pub execution: Option<ExecutionResult>,
    /// Time spent selecting canonical input.
    pub input_selection_seconds: f64,
    /// Total monotonic attempt duration.
    pub total_seconds: f64,
    /// Bounded diagnostic detail; never suitable for a metric label.
    pub detail: Option<String>,
}

/// Observable result of one sequential scheduler iteration.
#[derive(Clone, Debug, PartialEq)]
#[allow(clippy::large_enum_variant)]
pub enum RunnerEvent {
    /// The sequential scheduler began a new selection cycle.
    SchedulerHeartbeat {
        /// Best-effort Unix timestamp; zero means the wall clock was unavailable.
        unix_time: u64,
    },
    /// The full selection/execution attempt became active or inactive.
    RunActive {
        /// Whether an attempt is currently active.
        active: bool,
    },
    /// An execution or transient input attempt completed.
    Attempt(AttemptResult),
    /// An identical fingerprint already completed successfully.
    SkippedSuccessful {
        /// Previously successful fingerprint.
        fingerprint: B256,
        /// Current finalized target carried by that fingerprint.
        target_timestamp: u64,
    },
    /// An identical correctness failure already received its confirmation attempt.
    ConfirmedFailure {
        /// Confirmed failing fingerprint.
        fingerprint: B256,
        /// Current finalized target carried by that fingerprint.
        target_timestamp: u64,
        /// Confirmed correctness outcome.
        outcome: RunOutcome,
    },
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum SchedulerDecision {
    Run { confirmation: bool },
    SkipSuccessful,
    HoldConfirmedFailure(RunOutcome),
}

#[derive(Default)]
struct SchedulerState {
    last_successful: Option<B256>,
    pending_confirmation: Option<B256>,
    confirmed_failure: Option<(B256, RunOutcome)>,
}

impl SchedulerState {
    fn decision(&self, fingerprint: B256) -> SchedulerDecision {
        if self.last_successful == Some(fingerprint) {
            return SchedulerDecision::SkipSuccessful;
        }
        if let Some((confirmed, outcome)) = self.confirmed_failure &&
            confirmed == fingerprint
        {
            return SchedulerDecision::HoldConfirmedFailure(outcome);
        }
        SchedulerDecision::Run { confirmation: self.pending_confirmation == Some(fingerprint) }
    }

    fn record(&mut self, fingerprint: B256, outcome: RunOutcome) {
        if outcome == RunOutcome::Valid {
            self.last_successful = Some(fingerprint);
            if self.pending_confirmation == Some(fingerprint) {
                self.pending_confirmation = None;
            }
            if self.confirmed_failure.is_some_and(|(failed, _)| failed == fingerprint) {
                self.confirmed_failure = None;
            }
            return;
        }
        if !outcome.is_correctness_failure() {
            return;
        }
        if self.pending_confirmation == Some(fingerprint) {
            self.pending_confirmation = None;
            self.confirmed_failure = Some((fingerprint, outcome));
        } else {
            self.pending_confirmation = Some(fingerprint);
        }
    }
}

#[derive(Clone, Copy, Debug)]
struct RunnerSettings {
    cadence: Duration,
    max_jitter: Duration,
    attempt_deadline: Duration,
}

trait SnapshotIdentity {
    fn fingerprint(&self) -> B256;
    fn target_timestamp(&self) -> u64;
    fn span_length(&self) -> u64;
    fn chain_count(&self) -> usize;
}

impl SnapshotIdentity for ValidatedSnapshot {
    fn fingerprint(&self) -> B256 {
        self.fingerprint()
    }

    fn target_timestamp(&self) -> u64 {
        self.span().end
    }

    fn span_length(&self) -> u64 {
        let span = self.span();
        span.end - span.start + 1
    }

    fn chain_count(&self) -> usize {
        self.chain_ids().len()
    }
}

trait AttemptStages {
    type Snapshot: SnapshotIdentity;

    async fn select(&mut self, now: u64) -> Result<Self::Snapshot>;
    async fn execute(
        &mut self,
        snapshot: &Self::Snapshot,
        deadline: Instant,
        execution_activity: &ExecutionActivity,
    ) -> Result<ExecutionResult>;
}

struct InProcessStages {
    source: SnapshotSource,
    artifact: ValidatedRangeArtifact,
    cycle_limit: NonZeroU64,
    memory_limit: NonZeroU64,
}

impl std::fmt::Debug for InProcessStages {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("InProcessStages")
            .field("source", &self.source)
            .field("artifact", &self.artifact)
            .field("cycle_limit", &self.cycle_limit)
            .field("memory_limit", &self.memory_limit)
            .finish()
    }
}

impl AttemptStages for InProcessStages {
    type Snapshot = ValidatedSnapshot;

    async fn select(&mut self, now: u64) -> Result<Self::Snapshot> {
        self.source.select_finalized(now).await
    }

    async fn execute(
        &mut self,
        snapshot: &Self::Snapshot,
        deadline: Instant,
        execution_activity: &ExecutionActivity,
    ) -> Result<ExecutionResult> {
        let synthesized = snapshot.synthesize_execution()?;
        Ok(execute_snapshot(
            snapshot.host_inputs(),
            &synthesized,
            &self.artifact,
            self.cycle_limit,
            self.memory_limit,
            deadline,
            execution_activity,
        )
        .await)
    }
}

struct Scheduler<S> {
    stages: S,
    settings: RunnerSettings,
    state: SchedulerState,
    execution_activity: ExecutionActivity,
}

impl<S> Scheduler<S>
where
    S: AttemptStages,
{
    fn execution_activity(&self) -> watch::Receiver<bool> {
        self.execution_activity.subscribe()
    }

    async fn run_one(
        &mut self,
        shutdown: &mut watch::Receiver<bool>,
    ) -> Result<Option<RunnerEvent>> {
        if shutdown_requested(shutdown) {
            return Ok(None);
        }
        let started = Instant::now();
        let Some(deadline) = started.checked_add(self.settings.attempt_deadline) else {
            return Ok(Some(RunnerEvent::Attempt(selection_failure(
                started,
                RunOutcome::InputError,
                "attempt deadline exceeds monotonic clock range",
            ))));
        };
        let now = match unix_now() {
            Ok(now) => now,
            Err(error) => {
                return Ok(Some(RunnerEvent::Attempt(selection_failure(
                    started,
                    RunOutcome::InputError,
                    error,
                ))));
            }
        };
        let selection_started = Instant::now();
        let selection = tokio::select! {
            _ = wait_for_shutdown(shutdown) => return Ok(None),
            result = tokio::time::timeout_at(deadline, self.stages.select(now)) => result,
        };
        let snapshot = match selection {
            Ok(Ok(snapshot)) => snapshot,
            Ok(Err(error)) => {
                return Ok(Some(RunnerEvent::Attempt(selection_failure_with_duration(
                    started,
                    selection_started.elapsed(),
                    RunOutcome::InputError,
                    error,
                ))));
            }
            Err(_) => {
                return Ok(Some(RunnerEvent::Attempt(selection_failure_with_duration(
                    started,
                    selection_started.elapsed(),
                    RunOutcome::Timeout,
                    "attempt deadline elapsed during input selection",
                ))));
            }
        };
        let selection_duration = selection_started.elapsed();
        let fingerprint = snapshot.fingerprint();
        let target_timestamp = snapshot.target_timestamp();
        let confirmation = match self.state.decision(fingerprint) {
            SchedulerDecision::SkipSuccessful => {
                return Ok(Some(RunnerEvent::SkippedSuccessful { fingerprint, target_timestamp }));
            }
            SchedulerDecision::HoldConfirmedFailure(outcome) => {
                return Ok(Some(RunnerEvent::ConfirmedFailure {
                    fingerprint,
                    target_timestamp,
                    outcome,
                }));
            }
            SchedulerDecision::Run { confirmation } => confirmation,
        };
        let base = AttemptBase::new(&snapshot, confirmation, selection_duration);

        // The binary sends shutdown only outside the activity guard around uncancellable SP1 work.
        // The execution unit applies the deadline only to witness collection.
        if shutdown_requested(shutdown) {
            return Ok(None);
        }
        let execution = tokio::select! {
            biased;
            _ = wait_for_shutdown(shutdown) => return Ok(None),
            result = self.stages.execute(&snapshot, deadline, &self.execution_activity) => result,
        };
        let execution = match execution {
            Ok(execution) => execution,
            Err(error) => {
                let result = base.finish(
                    started,
                    RunOutcome::InputError,
                    None,
                    Some(format!("execution setup failed: {error:#}")),
                );
                self.state.record(fingerprint, result.outcome);
                return Ok(Some(RunnerEvent::Attempt(result)));
            }
        };

        if execution.outcome == ExecutionOutcome::TimedOut {
            let result = base.finish(
                started,
                RunOutcome::Timeout,
                Some(execution),
                Some("attempt deadline elapsed during witness collection"),
            );
            self.state.record(fingerprint, result.outcome);
            return Ok(Some(RunnerEvent::Attempt(result)));
        }

        let outcome = run_outcome(execution.outcome);
        let detail = execution_detail(&execution);
        let result = base.finish(started, outcome, Some(execution), detail);
        self.state.record(fingerprint, result.outcome);
        Ok(Some(RunnerEvent::Attempt(result)))
    }

    async fn run<F>(&mut self, mut shutdown: watch::Receiver<bool>, mut observe: F) -> Result<()>
    where
        F: FnMut(&RunnerEvent),
    {
        loop {
            if shutdown_requested(&shutdown) {
                return Ok(());
            }
            observe(&RunnerEvent::SchedulerHeartbeat { unix_time: unix_now().unwrap_or_default() });
            observe(&RunnerEvent::RunActive { active: true });
            let completed = self.run_one(&mut shutdown).await;
            observe(&RunnerEvent::RunActive { active: false });
            let Some(event) = completed? else { return Ok(()) };
            observe(&event);

            let delay =
                self.settings.cadence.saturating_add(bounded_jitter(self.settings.max_jitter));
            tokio::select! {
                _ = tokio::time::sleep(delay) => {}
                _ = wait_for_shutdown(&mut shutdown) => return Ok(()),
            }
        }
    }
}

/// One-network sequential canary lifecycle.
pub struct Runner {
    scheduler: Scheduler<InProcessStages>,
}

impl std::fmt::Debug for Runner {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Runner")
            .field("stages", &self.scheduler.stages)
            .field("settings", &self.scheduler.settings)
            .finish_non_exhaustive()
    }
}

impl Runner {
    /// Builds the sequential lifecycle around the in-memory authenticated artifact.
    pub fn new(config: &CanaryConfig, artifact: &ValidatedRangeArtifact) -> Result<Self> {
        ensure!(!config.cadence.is_zero(), "runner cadence must be non-zero");
        ensure!(!config.attempt_deadline.is_zero(), "attempt deadline must be non-zero");
        ensure!(config.max_jitter <= config.cadence, "runner jitter must not exceed cadence");
        ensure!(
            artifact.identity() == config.artifact.identity,
            "runner artifact does not match configured release identity"
        );
        Ok(Self {
            scheduler: Scheduler {
                stages: InProcessStages {
                    source: SnapshotSource::new(config)?,
                    artifact: artifact.clone(),
                    cycle_limit: config.guest_cycle_limit,
                    memory_limit: config.memory_limit,
                },
                settings: RunnerSettings {
                    cadence: config.cadence,
                    max_jitter: config.max_jitter,
                    attempt_deadline: config.attempt_deadline,
                },
                state: SchedulerState::default(),
                execution_activity: ExecutionActivity::new(),
            },
        })
    }

    /// Runs one immediate iteration and returns `None` when shutdown interrupts a cancellable
    /// stage.
    pub fn run_one<'a>(
        &'a mut self,
        shutdown: &'a mut watch::Receiver<bool>,
    ) -> Pin<Box<dyn Future<Output = Result<Option<RunnerEvent>>> + 'a>> {
        Box::pin(self.scheduler.run_one(shutdown))
    }

    /// Reports whether the runner is awaiting uncancellable SP1 emulator work.
    pub fn execution_activity(&self) -> watch::Receiver<bool> {
        self.scheduler.execution_activity()
    }

    /// Runs immediately, then waits one completion-anchored cadence before each next iteration.
    pub fn run<'a, F>(
        &'a mut self,
        shutdown: watch::Receiver<bool>,
        observe: F,
    ) -> Pin<Box<dyn Future<Output = Result<()>> + 'a>>
    where
        F: FnMut(&RunnerEvent) + 'a,
    {
        Box::pin(self.scheduler.run(shutdown, observe))
    }
}

struct AttemptBase {
    fingerprint: B256,
    target_timestamp: u64,
    span_length: u64,
    chain_count: usize,
    confirmation: bool,
    input_selection_seconds: f64,
}

impl AttemptBase {
    fn new<S: SnapshotIdentity>(
        snapshot: &S,
        confirmation: bool,
        selection_duration: Duration,
    ) -> Self {
        Self {
            fingerprint: snapshot.fingerprint(),
            target_timestamp: snapshot.target_timestamp(),
            span_length: snapshot.span_length(),
            chain_count: snapshot.chain_count(),
            confirmation,
            input_selection_seconds: selection_duration.as_secs_f64(),
        }
    }

    fn finish(
        &self,
        started: Instant,
        outcome: RunOutcome,
        execution: Option<ExecutionResult>,
        detail: Option<impl std::fmt::Display>,
    ) -> AttemptResult {
        AttemptResult {
            fingerprint: Some(self.fingerprint),
            target_timestamp: Some(self.target_timestamp),
            span_length: Some(self.span_length),
            chain_count: Some(self.chain_count),
            confirmation: self.confirmation,
            outcome,
            execution,
            input_selection_seconds: self.input_selection_seconds,
            total_seconds: started.elapsed().as_secs_f64(),
            detail: detail.map(|detail| bounded_detail(&detail.to_string())),
        }
    }
}

const fn run_outcome(outcome: ExecutionOutcome) -> RunOutcome {
    match outcome {
        ExecutionOutcome::Valid => RunOutcome::Valid,
        ExecutionOutcome::GuestRejected => RunOutcome::GuestRejected,
        ExecutionOutcome::OutputMismatch => RunOutcome::OutputMismatch,
        ExecutionOutcome::CycleLimitExceeded => RunOutcome::CycleLimitExceeded,
        ExecutionOutcome::TimedOut => RunOutcome::Timeout,
        ExecutionOutcome::InfrastructureFailure => RunOutcome::InputError,
    }
}

fn execution_detail(execution: &ExecutionResult) -> Option<String> {
    let determining_outcome = match execution.outcome {
        ExecutionOutcome::Valid => return None,
        ExecutionOutcome::GuestRejected => StageOutcome::GuestRejected,
        ExecutionOutcome::OutputMismatch => StageOutcome::OutputMismatch,
        ExecutionOutcome::CycleLimitExceeded => StageOutcome::CycleLimitExceeded,
        ExecutionOutcome::TimedOut => StageOutcome::TimedOut,
        ExecutionOutcome::InfrastructureFailure => StageOutcome::InfrastructureFailure,
    };
    [&execution.range, &execution.consolidation]
        .into_iter()
        .find(|stage| stage.outcome == determining_outcome)
        .and_then(|stage| stage.error.as_deref())
        .map(bounded_detail)
}

fn selection_failure(
    started: Instant,
    outcome: RunOutcome,
    detail: impl std::fmt::Display,
) -> AttemptResult {
    selection_failure_with_duration(started, Duration::ZERO, outcome, detail)
}

fn selection_failure_with_duration(
    started: Instant,
    selection_duration: Duration,
    outcome: RunOutcome,
    detail: impl std::fmt::Display,
) -> AttemptResult {
    AttemptResult {
        fingerprint: None,
        target_timestamp: None,
        span_length: None,
        chain_count: None,
        confirmation: false,
        outcome,
        execution: None,
        input_selection_seconds: selection_duration.as_secs_f64(),
        total_seconds: started.elapsed().as_secs_f64(),
        detail: Some(bounded_detail(&detail.to_string())),
    }
}

fn bounded_jitter(maximum: Duration) -> Duration {
    if maximum.is_zero() {
        return Duration::ZERO;
    }
    let counter = JITTER_COUNTER.fetch_add(1, Ordering::Relaxed);
    let nanos = SystemTime::now().duration_since(UNIX_EPOCH).map_or(0, |value| value.as_nanos());
    let maximum_nanos = maximum.as_nanos();
    duration_from_nanos((nanos ^ u128::from(counter)) % maximum_nanos.saturating_add(1))
}

fn duration_from_nanos(nanos: u128) -> Duration {
    let seconds = (nanos / 1_000_000_000).min(u128::from(u64::MAX)) as u64;
    let subsecond = if seconds == u64::MAX { 999_999_999 } else { (nanos % 1_000_000_000) as u32 };
    Duration::new(seconds, subsecond)
}

fn unix_now() -> Result<u64> {
    Ok(SystemTime::now().duration_since(UNIX_EPOCH)?.as_secs())
}

fn shutdown_requested(shutdown: &watch::Receiver<bool>) -> bool {
    *shutdown.borrow()
}

async fn wait_for_shutdown(shutdown: &mut watch::Receiver<bool>) {
    loop {
        if shutdown_requested(shutdown) || shutdown.changed().await.is_err() {
            return;
        }
    }
}

fn bounded_detail(detail: &str) -> String {
    if detail.len() <= MAX_RUN_DETAIL_BYTES {
        return detail.to_string();
    }
    let mut boundary = MAX_RUN_DETAIL_BYTES;
    while !detail.is_char_boundary(boundary) {
        boundary -= 1;
    }
    detail[..boundary].to_string()
}

#[cfg(test)]
mod tests {
    use std::{
        collections::VecDeque,
        sync::{
            Arc, Mutex,
            atomic::{AtomicUsize, Ordering},
        },
        time::Duration,
    };

    use crate::execution::{ExecutionMode, StageOutcome, StageResult};

    use super::*;

    #[derive(Clone)]
    struct FakeSnapshot {
        fingerprint: B256,
        target: u64,
    }

    impl SnapshotIdentity for FakeSnapshot {
        fn fingerprint(&self) -> B256 {
            self.fingerprint
        }

        fn target_timestamp(&self) -> u64 {
            self.target
        }

        fn span_length(&self) -> u64 {
            1
        }

        fn chain_count(&self) -> usize {
            1
        }
    }

    struct FakeStages {
        selection_delay: Duration,
        execution_delay: Duration,
        execution_delays: VecDeque<Duration>,
        selections: Arc<AtomicUsize>,
        active: Arc<AtomicUsize>,
        max_active: Arc<AtomicUsize>,
        completions: Arc<AtomicUsize>,
        snapshots: VecDeque<FakeSnapshot>,
        executions: VecDeque<ExecutionResult>,
        shutdown_after_selection: Option<watch::Sender<bool>>,
        shutdown_after_execution: Option<(usize, watch::Sender<bool>)>,
    }

    impl FakeStages {
        fn immediate() -> Self {
            Self {
                selection_delay: Duration::ZERO,
                execution_delay: Duration::ZERO,
                execution_delays: VecDeque::new(),
                selections: Arc::new(AtomicUsize::new(0)),
                active: Arc::new(AtomicUsize::new(0)),
                max_active: Arc::new(AtomicUsize::new(0)),
                completions: Arc::new(AtomicUsize::new(0)),
                snapshots: VecDeque::new(),
                executions: VecDeque::new(),
                shutdown_after_selection: None,
                shutdown_after_execution: None,
            }
        }
    }

    impl AttemptStages for FakeStages {
        type Snapshot = FakeSnapshot;

        async fn select(&mut self, _now: u64) -> Result<Self::Snapshot> {
            tokio::time::sleep(self.selection_delay).await;
            let selected = self.selections.fetch_add(1, Ordering::SeqCst) + 1;
            if let Some(shutdown) = &self.shutdown_after_selection {
                let _ = shutdown.send(true);
            }
            Ok(self.snapshots.pop_front().unwrap_or_else(|| FakeSnapshot {
                fingerprint: B256::with_last_byte(selected as u8),
                target: selected as u64,
            }))
        }

        async fn execute(
            &mut self,
            _snapshot: &Self::Snapshot,
            _deadline: Instant,
            execution_activity: &ExecutionActivity,
        ) -> Result<ExecutionResult> {
            let _execution_active = execution_activity.enter();
            let active = self.active.fetch_add(1, Ordering::SeqCst) + 1;
            self.max_active.fetch_max(active, Ordering::SeqCst);
            let delay = self.execution_delays.pop_front().unwrap_or(self.execution_delay);
            tokio::time::sleep(delay).await;
            self.active.fetch_sub(1, Ordering::SeqCst);
            let completed = self.completions.fetch_add(1, Ordering::SeqCst) + 1;
            if let Some((limit, shutdown)) = &self.shutdown_after_execution &&
                completed == *limit
            {
                let _ = shutdown.send(true);
            }
            Ok(self.executions.pop_front().unwrap_or_else(valid_execution))
        }
    }

    fn valid_execution() -> ExecutionResult {
        let stage = |mode| StageResult {
            mode,
            outcome: StageOutcome::Valid,
            report: None,
            witness_seconds: 0.0,
            execute_seconds: None,
            error: None,
        };
        ExecutionResult {
            outcome: ExecutionOutcome::Valid,
            range: stage(ExecutionMode::Range),
            consolidation: stage(ExecutionMode::Consolidation),
        }
    }

    fn execution_with_outcome(outcome: ExecutionOutcome) -> ExecutionResult {
        let stage_outcome = match outcome {
            ExecutionOutcome::Valid => StageOutcome::Valid,
            ExecutionOutcome::GuestRejected => StageOutcome::GuestRejected,
            ExecutionOutcome::OutputMismatch => StageOutcome::OutputMismatch,
            ExecutionOutcome::CycleLimitExceeded => StageOutcome::CycleLimitExceeded,
            ExecutionOutcome::TimedOut => StageOutcome::TimedOut,
            ExecutionOutcome::InfrastructureFailure => StageOutcome::InfrastructureFailure,
        };
        let stage = |mode| StageResult {
            mode,
            outcome: stage_outcome,
            report: None,
            witness_seconds: 0.0,
            execute_seconds: None,
            error: None,
        };
        ExecutionResult {
            outcome,
            range: stage(ExecutionMode::Range),
            consolidation: if outcome == ExecutionOutcome::InfrastructureFailure {
                StageResult {
                    mode: ExecutionMode::Consolidation,
                    outcome: StageOutcome::NotRun,
                    report: None,
                    witness_seconds: 0.0,
                    execute_seconds: None,
                    error: None,
                }
            } else {
                stage(ExecutionMode::Consolidation)
            },
        }
    }

    fn scheduler(stages: FakeStages, deadline: Duration) -> Scheduler<FakeStages> {
        Scheduler {
            stages,
            settings: RunnerSettings {
                cadence: Duration::from_secs(1),
                max_jitter: Duration::ZERO,
                attempt_deadline: deadline,
            },
            state: SchedulerState::default(),
            execution_activity: ExecutionActivity::new(),
        }
    }

    #[test]
    fn scheduler_state_retries_and_deduplicates_by_fingerprint() {
        let first = B256::repeat_byte(1);
        let newer = B256::repeat_byte(2);
        let mut state = SchedulerState::default();

        assert_eq!(state.decision(first), SchedulerDecision::Run { confirmation: false });
        state.record(first, RunOutcome::InputError);
        assert_eq!(state.decision(first), SchedulerDecision::Run { confirmation: false });
        state.record(first, RunOutcome::GuestRejected);
        assert_eq!(state.decision(first), SchedulerDecision::Run { confirmation: true });
        state.record(first, RunOutcome::GuestRejected);
        assert_eq!(
            state.decision(first),
            SchedulerDecision::HoldConfirmedFailure(RunOutcome::GuestRejected)
        );
        assert_eq!(state.decision(newer), SchedulerDecision::Run { confirmation: false });
        state.record(newer, RunOutcome::Valid);
        assert_eq!(state.decision(newer), SchedulerDecision::SkipSuccessful);
        assert!(bounded_jitter(Duration::from_millis(10)) <= Duration::from_millis(10));
    }

    #[test]
    fn execution_detail_describes_the_determining_outcome() {
        let mut execution = execution_with_outcome(ExecutionOutcome::OutputMismatch);
        execution.range.outcome = StageOutcome::InfrastructureFailure;
        execution.range.error = Some("range infrastructure failure".to_string());
        execution.consolidation.error = Some("consolidation output mismatch".to_string());

        assert_eq!(execution_detail(&execution).as_deref(), Some("consolidation output mismatch"),);
    }

    #[tokio::test(start_paused = true)]
    async fn retries_failed_target_and_skips_successful_duplicate() {
        let (shutdown_tx, shutdown) = watch::channel(false);
        let mut stages = FakeStages::immediate();
        let first = B256::repeat_byte(1);
        let changed = B256::repeat_byte(2);
        stages.snapshots = VecDeque::from([
            FakeSnapshot { fingerprint: first, target: 100 },
            FakeSnapshot { fingerprint: first, target: 100 },
            FakeSnapshot { fingerprint: first, target: 100 },
            FakeSnapshot { fingerprint: first, target: 100 },
            FakeSnapshot { fingerprint: changed, target: 100 },
            FakeSnapshot { fingerprint: changed, target: 100 },
        ]);
        stages.executions = VecDeque::from([
            execution_with_outcome(ExecutionOutcome::InfrastructureFailure),
            execution_with_outcome(ExecutionOutcome::GuestRejected),
            execution_with_outcome(ExecutionOutcome::GuestRejected),
            execution_with_outcome(ExecutionOutcome::Valid),
        ]);
        let selections = stages.selections.clone();
        let completions = stages.completions.clone();
        let observed = Arc::new(Mutex::new(Vec::new()));
        let observed_events = observed.clone();
        let started = Instant::now();

        scheduler(stages, Duration::from_secs(60))
            .run(shutdown, move |event| {
                if matches!(
                    event,
                    RunnerEvent::Attempt(_) |
                        RunnerEvent::ConfirmedFailure { .. } |
                        RunnerEvent::SkippedSuccessful { .. }
                ) {
                    let mut events = observed_events.lock().unwrap();
                    events.push(event.clone());
                    if events.len() == 6 {
                        let _ = shutdown_tx.send(true);
                    }
                }
            })
            .await
            .unwrap();

        let observed = observed.lock().unwrap();
        assert!(matches!(
            &observed[0],
            RunnerEvent::Attempt(result)
                if result.outcome == RunOutcome::InputError && !result.confirmation
        ));
        assert!(matches!(
            &observed[1],
            RunnerEvent::Attempt(result)
                if result.outcome == RunOutcome::GuestRejected && !result.confirmation
        ));
        assert!(matches!(
            &observed[2],
            RunnerEvent::Attempt(result)
                if result.outcome == RunOutcome::GuestRejected && result.confirmation
        ));
        assert!(matches!(
            &observed[3],
            RunnerEvent::ConfirmedFailure { fingerprint, target_timestamp: 100, outcome: RunOutcome::GuestRejected }
                if *fingerprint == first
        ));
        assert!(matches!(
            &observed[4],
            RunnerEvent::Attempt(result)
                if result.outcome == RunOutcome::Valid && result.fingerprint == Some(changed)
        ));
        assert!(matches!(
            &observed[5],
            RunnerEvent::SkippedSuccessful { fingerprint, target_timestamp: 100 }
                if *fingerprint == changed
        ));
        assert_eq!(selections.load(Ordering::SeqCst), 6);
        assert_eq!(completions.load(Ordering::SeqCst), 4);
        assert!(started.elapsed() >= Duration::from_secs(5));
    }

    #[tokio::test(start_paused = true)]
    async fn runs_never_overlap() {
        let (shutdown_tx, shutdown) = watch::channel(false);
        let mut stages = FakeStages::immediate();
        stages.execution_delay = Duration::from_secs(10);
        stages.shutdown_after_execution = Some((2, shutdown_tx));
        let selections = stages.selections.clone();
        let max_active = stages.max_active.clone();
        scheduler(stages, Duration::from_secs(60)).run(shutdown, |_| {}).await.unwrap();

        assert_eq!(selections.load(Ordering::SeqCst), 2);
        assert_eq!(max_active.load(Ordering::SeqCst), 1);
    }

    #[tokio::test(start_paused = true)]
    async fn selection_deadline_records_timeout() {
        let (_shutdown_tx, mut shutdown) = watch::channel(false);
        let mut stages = FakeStages::immediate();
        stages.selection_delay = Duration::from_secs(2);
        let mut scheduler = scheduler(stages, Duration::from_secs(1));

        let event = scheduler.run_one(&mut shutdown).await.unwrap().unwrap();
        let RunnerEvent::Attempt(result) = event else { panic!("expected attempted run") };
        assert_eq!(result.outcome, RunOutcome::Timeout);
        assert!(result.execution.is_none());
    }

    #[tokio::test(start_paused = true)]
    async fn deadline_on_cancellable_stages_records_timeout() {
        let (shutdown_tx, shutdown) = watch::channel(false);
        let mut stages = FakeStages::immediate();
        stages.execution_delays = VecDeque::from([Duration::from_secs(2), Duration::ZERO]);
        stages.executions = VecDeque::from([
            execution_with_outcome(ExecutionOutcome::TimedOut),
            execution_with_outcome(ExecutionOutcome::Valid),
        ]);
        let selections = stages.selections.clone();
        let outcomes = Arc::new(Mutex::new(Vec::new()));
        let observed_outcomes = outcomes.clone();

        scheduler(stages, Duration::from_secs(1))
            .run(shutdown, move |event| {
                if let RunnerEvent::Attempt(result) = event {
                    let mut outcomes = observed_outcomes.lock().unwrap();
                    outcomes.push(result.outcome);
                    if outcomes.len() == 2 {
                        let _ = shutdown_tx.send(true);
                    }
                }
            })
            .await
            .unwrap();

        assert_eq!(outcomes.lock().unwrap().as_slice(), &[RunOutcome::Timeout, RunOutcome::Valid],);
        assert_eq!(selections.load(Ordering::SeqCst), 2);
    }

    #[tokio::test(start_paused = true)]
    async fn shutdown_stops_scheduling() {
        let (shutdown_tx, shutdown) = watch::channel(false);
        let mut stages = FakeStages::immediate();
        stages.execution_delay = Duration::from_secs(10);
        stages.shutdown_after_execution = Some((1, shutdown_tx));
        let selections = stages.selections.clone();
        let completions = stages.completions.clone();
        scheduler(stages, Duration::from_secs(60)).run(shutdown, |_| {}).await.unwrap();

        assert_eq!(selections.load(Ordering::SeqCst), 1);
        assert_eq!(completions.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn shutdown_after_selection_never_starts_execution() {
        let (shutdown_tx, mut shutdown) = watch::channel(false);
        let mut stages = FakeStages::immediate();
        stages.shutdown_after_selection = Some(shutdown_tx);
        let completions = stages.completions.clone();
        let mut scheduler = scheduler(stages, Duration::from_secs(60));

        assert!(scheduler.run_one(&mut shutdown).await.unwrap().is_none());
        assert_eq!(completions.load(Ordering::SeqCst), 0);
    }

    #[tokio::test(start_paused = true)]
    async fn execution_activity_covers_only_the_uncancellable_stage() {
        let (_shutdown_tx, mut shutdown) = watch::channel(false);
        let mut stages = FakeStages::immediate();
        stages.execution_delay = Duration::from_secs(10);
        let mut scheduler = scheduler(stages, Duration::from_secs(60));
        let mut execution_active = scheduler.execution_activity();
        let mut run = Box::pin(scheduler.run_one(&mut shutdown));

        tokio::select! {
            changed = execution_active.changed() => changed.unwrap(),
            result = &mut run => panic!("attempt completed before execution: {result:?}"),
        }
        assert!(*execution_active.borrow());

        tokio::time::advance(Duration::from_secs(10)).await;
        assert!(run.await.unwrap().is_some());
        assert!(!*execution_active.borrow());
    }
}
