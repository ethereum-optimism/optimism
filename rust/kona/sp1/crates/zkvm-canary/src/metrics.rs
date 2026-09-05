//! Bounded Prometheus mapping for canary lifecycle and SP1 execution reports.

use std::time::SystemTime;

use metrics::{Counter, Gauge, counter, describe_counter, describe_gauge, gauge};

use crate::{
    execution::{ReportSummary, StageOutcome, StageResult},
    runner::{AttemptResult, RunOutcome, RunnerEvent},
};

const UP: &str = "kona_zkvm_canary_up";
const SCHEDULER_HEARTBEAT: &str = "kona_zkvm_canary_scheduler_heartbeat_timestamp_seconds";
const RUN_ACTIVE: &str = "kona_zkvm_canary_run_active";
const LAST_ATTEMPT: &str = "kona_zkvm_canary_last_attempt_timestamp_seconds";
const LAST_SUCCESS: &str = "kona_zkvm_canary_last_success_timestamp_seconds";
const LAST_ATTEMPTED_TARGET: &str = "kona_zkvm_canary_last_attempted_target_timestamp";
const LAST_SUCCESSFUL_TARGET: &str = "kona_zkvm_canary_last_successful_target_timestamp";
const CONSECUTIVE_FAILURES: &str = "kona_zkvm_canary_consecutive_failures";
const RUNS: &str = "kona_zkvm_canary_runs_total";
const LAST_RUN_DURATION: &str = "kona_zkvm_canary_last_run_duration_seconds";
const LAST_INPUT_SELECTION_DURATION: &str =
    "kona_zkvm_canary_last_input_selection_duration_seconds";
const LAST_STAGE_WITNESS_DURATION: &str = "kona_zkvm_canary_last_stage_witness_duration_seconds";
const LAST_STAGE_EXECUTE_DURATION: &str = "kona_zkvm_canary_last_stage_execute_duration_seconds";
const SELECTED_SPAN_LENGTH: &str = "kona_zkvm_canary_selected_span_length";
const SELECTED_CHAIN_COUNT: &str = "kona_zkvm_canary_selected_chain_count";
const TARGET_LAG: &str = "kona_zkvm_canary_finalized_target_lag_seconds";
const REPORT_TARGET: &str = "kona_zkvm_canary_report_target_timestamp";
const REPORT_PGU: &str = "kona_zkvm_canary_report_pgu";
const REPORT_INSTRUCTIONS: &str = "kona_zkvm_canary_report_instructions";
const REPORT_SYSCALLS: &str = "kona_zkvm_canary_report_syscalls";
const REPORT_RECORD_BYTES: &str = "kona_zkvm_canary_report_record_bytes";
const REPORT_TOUCHED_ADDRESSES: &str = "kona_zkvm_canary_report_touched_addresses";
const REPORT_EXIT_CODE: &str = "kona_zkvm_canary_report_exit_code";

const MODE_RANGE: &str = "range";
const MODE_CONSOLIDATION: &str = "consolidation";

const OUTCOMES: [RunOutcome; 6] = [
    RunOutcome::Valid,
    RunOutcome::GuestRejected,
    RunOutcome::OutputMismatch,
    RunOutcome::InputError,
    RunOutcome::CycleLimitExceeded,
    RunOutcome::Timeout,
];

/// Registered canary metric handles.
pub struct CanaryMetrics {
    up: Gauge,
    scheduler_heartbeat: Gauge,
    run_active: Gauge,
    last_attempt: Gauge,
    last_success: Gauge,
    last_attempted_target: Gauge,
    last_successful_target: Gauge,
    consecutive_failures: Gauge,
    runs: [Counter; OUTCOMES.len()],
    last_run_duration: Gauge,
    last_input_selection_duration: Gauge,
    selected_span_length: Gauge,
    selected_chain_count: Gauge,
    target_lag: Gauge,
    range: ModeMetrics,
    consolidation: ModeMetrics,
    consecutive_failure_count: u64,
}

impl std::fmt::Debug for CanaryMetrics {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("CanaryMetrics")
            .field("consecutive_failure_count", &self.consecutive_failure_count)
            .finish_non_exhaustive()
    }
}

impl CanaryMetrics {
    /// Describes and registers the complete bounded metric family.
    ///
    /// Call this only after the shared metrics recorder is installed. The returned observer is
    /// intended to be passed directly to [`crate::runner::Runner::run`].
    pub fn register() -> Self {
        describe_all();
        let metrics = Self {
            up: gauge!(UP),
            scheduler_heartbeat: gauge!(SCHEDULER_HEARTBEAT),
            run_active: gauge!(RUN_ACTIVE),
            last_attempt: gauge!(LAST_ATTEMPT),
            last_success: gauge!(LAST_SUCCESS),
            last_attempted_target: gauge!(LAST_ATTEMPTED_TARGET),
            last_successful_target: gauge!(LAST_SUCCESSFUL_TARGET),
            consecutive_failures: gauge!(CONSECUTIVE_FAILURES),
            runs: std::array::from_fn(
                |index| counter!(RUNS, "outcome" => outcome_label(OUTCOMES[index])),
            ),
            last_run_duration: gauge!(LAST_RUN_DURATION),
            last_input_selection_duration: gauge!(LAST_INPUT_SELECTION_DURATION),
            selected_span_length: gauge!(SELECTED_SPAN_LENGTH),
            selected_chain_count: gauge!(SELECTED_CHAIN_COUNT),
            target_lag: gauge!(TARGET_LAG),
            range: ModeMetrics::register(MODE_RANGE),
            consolidation: ModeMetrics::register(MODE_CONSOLIDATION),
            consecutive_failure_count: 0,
        };
        metrics.initialize();
        metrics
    }

    /// Applies one lifecycle or terminal runner event to the registered metric handles.
    pub fn observe(&mut self, event: &RunnerEvent) {
        self.observe_at(event, unix_now());
    }

    /// Marks the service as no longer up before a clean process exit.
    pub fn mark_down(&self) {
        self.run_active.set(0.0);
        self.up.set(0.0);
    }

    fn initialize(&self) {
        self.up.set(1.0);
        self.scheduler_heartbeat.set(0.0);
        self.run_active.set(0.0);
        self.last_attempt.set(0.0);
        self.last_success.set(0.0);
        self.last_attempted_target.set(0.0);
        self.last_successful_target.set(0.0);
        self.consecutive_failures.set(0.0);
        self.last_run_duration.set(0.0);
        self.last_input_selection_duration.set(0.0);
        for counter in &self.runs {
            counter.absolute(0);
        }
        self.selected_span_length.set(0.0);
        self.selected_chain_count.set(0.0);
        self.target_lag.set(0.0);
        self.range.initialize();
        self.consolidation.initialize();
    }

    fn observe_at(&mut self, event: &RunnerEvent, now: u64) {
        match event {
            RunnerEvent::SchedulerHeartbeat { unix_time } => {
                self.scheduler_heartbeat.set(*unix_time as f64);
            }
            RunnerEvent::RunActive { active } => {
                self.run_active.set(u8::from(*active) as f64);
            }
            RunnerEvent::Attempt(result) => self.observe_attempt(result, now),
            RunnerEvent::SkippedSuccessful { .. } | RunnerEvent::ConfirmedFailure { .. } => {}
        }
    }

    fn observe_attempt(&mut self, result: &AttemptResult, now: u64) {
        self.last_attempt.set(now as f64);
        self.runs[outcome_index(result.outcome)].increment(1);
        self.last_run_duration.set(result.total_seconds);
        self.last_input_selection_duration.set(result.input_selection_seconds);
        if let Some(target) = result.target_timestamp {
            self.last_attempted_target.set(target as f64);
            self.target_lag.set(now.saturating_sub(target) as f64);
        }
        if let Some(span_length) = result.span_length {
            self.selected_span_length.set(span_length as f64);
        }
        if let Some(chain_count) = result.chain_count {
            self.selected_chain_count.set(chain_count as f64);
        }

        if result.outcome == RunOutcome::Valid {
            self.consecutive_failure_count = 0;
            self.consecutive_failures.set(0.0);
            self.last_success.set(now as f64);
            if let Some(target) = result.target_timestamp {
                self.last_successful_target.set(target as f64);
            }
        } else {
            self.consecutive_failure_count = self.consecutive_failure_count.saturating_add(1);
            self.consecutive_failures.set(self.consecutive_failure_count as f64);
        }

        let Some(execution) = &result.execution else { return };
        self.range.observe_stage_durations(&execution.range);
        self.consolidation.observe_stage_durations(&execution.consolidation);
        let target = result.target_timestamp.unwrap_or_default();
        self.range.observe_stage_report(&execution.range, target);
        self.consolidation.observe_stage_report(&execution.consolidation, target);
    }
}

struct ModeMetrics {
    last_witness_duration: Gauge,
    last_execute_duration: Gauge,
    report_target: Gauge,
    pgu: Gauge,
    instructions: Gauge,
    syscalls: Gauge,
    record_bytes: Gauge,
    touched_addresses: Gauge,
    exit_code: Gauge,
}

impl ModeMetrics {
    fn register(mode: &'static str) -> Self {
        Self {
            last_witness_duration: gauge!(LAST_STAGE_WITNESS_DURATION, "mode" => mode),
            last_execute_duration: gauge!(LAST_STAGE_EXECUTE_DURATION, "mode" => mode),
            report_target: gauge!(REPORT_TARGET, "mode" => mode),
            pgu: gauge!(REPORT_PGU, "mode" => mode),
            instructions: gauge!(REPORT_INSTRUCTIONS, "mode" => mode),
            syscalls: gauge!(REPORT_SYSCALLS, "mode" => mode),
            record_bytes: gauge!(REPORT_RECORD_BYTES, "mode" => mode),
            touched_addresses: gauge!(REPORT_TOUCHED_ADDRESSES, "mode" => mode),
            exit_code: gauge!(REPORT_EXIT_CODE, "mode" => mode),
        }
    }

    fn initialize(&self) {
        self.last_witness_duration.set(0.0);
        self.last_execute_duration.set(0.0);
        self.report_target.set(0.0);
        self.pgu.set(0.0);
        self.instructions.set(0.0);
        self.syscalls.set(0.0);
        self.record_bytes.set(0.0);
        self.touched_addresses.set(0.0);
        self.exit_code.set(0.0);
    }

    fn observe_stage_durations(&self, stage: &StageResult) {
        if stage.outcome != StageOutcome::NotRun {
            self.last_witness_duration.set(stage.witness_seconds);
            if let Some(execute_seconds) = stage.execute_seconds {
                self.last_execute_duration.set(execute_seconds);
            }
        }
    }

    fn observe_stage_report(&self, stage: &StageResult, target: u64) {
        let Some(report) = &stage.report else { return };
        self.observe_report(report, target);
    }

    fn observe_report(&self, report: &ReportSummary, target: u64) {
        self.report_target.set(target as f64);
        if let Some(pgu) = report.pgu {
            self.pgu.set(pgu as f64);
        }
        self.instructions.set(report.instructions as f64);
        self.syscalls.set(report.syscalls as f64);
        self.record_bytes.set(report.record_bytes as f64);
        self.touched_addresses.set(report.touched_addresses as f64);
        self.exit_code.set(report.exit_code as f64);
    }
}

const fn outcome_label(outcome: RunOutcome) -> &'static str {
    match outcome {
        RunOutcome::Valid => "valid",
        RunOutcome::GuestRejected => "guest_rejected",
        RunOutcome::OutputMismatch => "output_mismatch",
        RunOutcome::InputError => "input_error",
        RunOutcome::CycleLimitExceeded => "cycle_limit_exceeded",
        RunOutcome::Timeout => "timeout",
    }
}

const fn outcome_index(outcome: RunOutcome) -> usize {
    match outcome {
        RunOutcome::Valid => 0,
        RunOutcome::GuestRejected => 1,
        RunOutcome::OutputMismatch => 2,
        RunOutcome::InputError => 3,
        RunOutcome::CycleLimitExceeded => 4,
        RunOutcome::Timeout => 5,
    }
}

fn unix_now() -> u64 {
    SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .map_or(0, |duration| duration.as_secs())
}

fn describe_all() {
    describe_gauge!(UP, "Whether the validated canary service is running.");
    describe_gauge!(SCHEDULER_HEARTBEAT, "Unix time of the latest scheduler cycle.");
    describe_gauge!(RUN_ACTIVE, "Whether one sequential canary attempt is active.");
    describe_gauge!(LAST_ATTEMPT, "Unix completion time of the latest attempted run.");
    describe_gauge!(LAST_SUCCESS, "Unix completion time of the latest valid run.");
    describe_gauge!(LAST_ATTEMPTED_TARGET, "Finalized target of the latest attempted run.");
    describe_gauge!(LAST_SUCCESSFUL_TARGET, "Finalized target of the latest valid run.");
    describe_gauge!(CONSECUTIVE_FAILURES, "Consecutive non-valid run outcomes.");
    describe_counter!(RUNS, "Completed canary runs by bounded terminal outcome.");
    describe_gauge!(
        LAST_RUN_DURATION,
        "Total monotonic duration in seconds of the latest canary attempt."
    );
    describe_gauge!(
        LAST_INPUT_SELECTION_DURATION,
        "Canonical snapshot selection duration in seconds of the latest canary attempt."
    );
    describe_gauge!(
        LAST_STAGE_WITNESS_DURATION,
        "Latest SP1 witness collection duration in seconds by guest mode."
    );
    describe_gauge!(
        LAST_STAGE_EXECUTE_DURATION,
        "Latest SP1 CPU execution duration in seconds by guest mode."
    );
    describe_gauge!(SELECTED_SPAN_LENGTH, "Timestamp count in the latest selected span.");
    describe_gauge!(SELECTED_CHAIN_COUNT, "Chain count in the latest selected snapshot.");
    describe_gauge!(TARGET_LAG, "Latest attempted finalized-target lag in seconds.");
    describe_gauge!(REPORT_TARGET, "Target timestamp associated with the latest mode report.");
    describe_gauge!(REPORT_PGU, "Latest normalized SP1 proving gas units by mode.");
    describe_gauge!(REPORT_INSTRUCTIONS, "Latest SP1 instruction count by mode.");
    describe_gauge!(REPORT_SYSCALLS, "Latest SP1 syscall count by mode.");
    describe_gauge!(REPORT_RECORD_BYTES, "Latest SP1 execution-record bytes by mode.");
    describe_gauge!(REPORT_TOUCHED_ADDRESSES, "Latest distinct touched guest addresses by mode.");
    describe_gauge!(REPORT_EXIT_CODE, "Latest SP1 guest exit code by mode.");
}

#[cfg(test)]
mod tests {
    use std::collections::BTreeMap;

    use alloy_primitives::B256;
    use metrics_util::debugging::{DebugValue, DebuggingRecorder, Snapshot};

    use super::*;
    use crate::execution::{
        CyclePhaseSummary, ExecutionMode, ExecutionOutcome, ExecutionResult, ReportDetail,
    };

    #[derive(Debug, PartialEq)]
    enum Sample {
        Counter(u64),
        Gauge(f64),
        Histogram(Vec<f64>),
    }

    type SampleKey = (String, Vec<(String, String)>);

    fn attempt(
        outcome: RunOutcome,
        target_timestamp: u64,
        execution: Option<ExecutionResult>,
    ) -> AttemptResult {
        AttemptResult {
            fingerprint: Some(B256::repeat_byte(0x55)),
            target_timestamp: Some(target_timestamp),
            span_length: Some(2),
            chain_count: Some(3),
            confirmation: false,
            outcome,
            execution,
            input_selection_seconds: 0.25,
            total_seconds: 1.5,
            detail: None,
        }
    }

    fn report(
        mode: ExecutionMode,
        pgu: Option<u64>,
        instructions: u64,
        touched_addresses: u64,
        phases: &[(&str, u64, u64)],
    ) -> ReportSummary {
        ReportSummary {
            mode,
            pgu,
            instructions,
            syscalls: instructions / 10,
            record_bytes: instructions * 2,
            touched_addresses,
            exit_code: 0,
            opcode_details: Vec::<ReportDetail>::new(),
            syscall_details: Vec::<ReportDetail>::new(),
            cycle_phases: phases
                .iter()
                .map(|(phase, cycles, invocations)| CyclePhaseSummary {
                    phase: (*phase).to_string(),
                    cycles: *cycles,
                    invocations: *invocations,
                })
                .collect(),
        }
    }

    fn stage(report: ReportSummary, witness_seconds: f64, execute_seconds: f64) -> StageResult {
        StageResult {
            mode: report.mode,
            outcome: StageOutcome::Valid,
            report: Some(report),
            witness_seconds,
            execute_seconds: Some(execute_seconds),
            error: None,
        }
    }

    fn execution(range: StageResult, consolidation: StageResult) -> ExecutionResult {
        ExecutionResult { outcome: ExecutionOutcome::Valid, range, consolidation }
    }

    fn samples(snapshot: Snapshot) -> BTreeMap<SampleKey, Sample> {
        snapshot
            .into_vec()
            .into_iter()
            .map(|(composite, _, _, value)| {
                let key = composite.key();
                let mut labels = key
                    .labels()
                    .map(|label| (label.key().to_string(), label.value().to_string()))
                    .collect::<Vec<_>>();
                labels.sort();
                let value = match value {
                    DebugValue::Counter(value) => Sample::Counter(value),
                    DebugValue::Gauge(value) => Sample::Gauge(value.into_inner()),
                    DebugValue::Histogram(values) => Sample::Histogram(
                        values.into_iter().map(|value| value.into_inner()).collect(),
                    ),
                };
                ((key.name().to_string(), labels), value)
            })
            .collect()
    }

    fn sample<'a>(
        samples: &'a BTreeMap<SampleKey, Sample>,
        name: &str,
        labels: &[(&str, &str)],
    ) -> &'a Sample {
        let mut labels = labels
            .iter()
            .map(|(key, value)| ((*key).to_string(), (*value).to_string()))
            .collect::<Vec<_>>();
        labels.sort();
        samples.get(&(name.to_string(), labels)).expect("metric sample exists")
    }

    #[test]
    fn metrics_classify_correctness_and_infrastructure_failures() {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            let mut metrics = CanaryMetrics::register();
            for outcome in OUTCOMES {
                metrics.observe_at(&RunnerEvent::Attempt(attempt(outcome, 100, None)), 200);
            }
        });

        let samples = samples(snapshotter.snapshot());
        for outcome in OUTCOMES {
            assert_eq!(
                sample(&samples, RUNS, &[("outcome", outcome_label(outcome))]),
                &Sample::Counter(1),
            );
        }
        assert_eq!(
            sample(&samples, RUNS, &[("outcome", "cycle_limit_exceeded")]),
            &Sample::Counter(1),
        );
        assert_ne!(
            outcome_index(RunOutcome::CycleLimitExceeded),
            outcome_index(RunOutcome::GuestRejected)
        );
        assert_eq!(sample(&samples, CONSECUTIVE_FAILURES, &[]), &Sample::Gauge(5.0));

        let allowed_labels = ["mode", "outcome"];
        for (_, labels) in samples.keys() {
            assert!(labels.iter().all(|(key, _)| allowed_labels.contains(&key.as_str())));
        }
    }

    #[test]
    fn report_metrics_preserve_aggregate_counts_and_modes() {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            let mut metrics = CanaryMetrics::register();
            let first = execution(
                stage(
                    report(
                        ExecutionMode::Range,
                        Some(101),
                        1_000,
                        7,
                        &[("range", 10, 1), ("old", 3, 1)],
                    ),
                    1.0,
                    2.0,
                ),
                stage(
                    report(
                        ExecutionMode::Consolidation,
                        Some(201),
                        2_000,
                        8,
                        &[("consolidation", 30, 2)],
                    ),
                    3.0,
                    4.0,
                ),
            );
            metrics.observe_at(
                &RunnerEvent::Attempt(attempt(RunOutcome::Valid, 100, Some(first))),
                110,
            );

            let second = execution(
                stage(report(ExecutionMode::Range, None, 3_000, 9, &[("range", 20, 4)]), 5.0, 6.0),
                stage(
                    report(
                        ExecutionMode::Consolidation,
                        Some(202),
                        4_000,
                        10,
                        &[("consolidation", 40, 5)],
                    ),
                    7.0,
                    8.0,
                ),
            );
            let mut second_attempt = attempt(RunOutcome::Valid, 200, Some(second));
            second_attempt.input_selection_seconds = 0.75;
            second_attempt.total_seconds = 12.0;
            metrics.observe_at(&RunnerEvent::Attempt(second_attempt), 210);
        });

        let samples = samples(snapshotter.snapshot());
        assert_eq!(sample(&samples, LAST_RUN_DURATION, &[]), &Sample::Gauge(12.0));
        assert_eq!(sample(&samples, LAST_INPUT_SELECTION_DURATION, &[]), &Sample::Gauge(0.75));
        assert_eq!(sample(&samples, REPORT_PGU, &[("mode", MODE_RANGE)]), &Sample::Gauge(101.0));
        assert_eq!(
            sample(&samples, REPORT_PGU, &[("mode", MODE_CONSOLIDATION)]),
            &Sample::Gauge(202.0),
        );
        assert_eq!(
            sample(&samples, REPORT_INSTRUCTIONS, &[("mode", MODE_RANGE)]),
            &Sample::Gauge(3_000.0),
        );
        assert_eq!(
            sample(&samples, REPORT_TOUCHED_ADDRESSES, &[("mode", MODE_RANGE)]),
            &Sample::Gauge(9.0),
        );
        assert_eq!(
            sample(&samples, REPORT_TARGET, &[("mode", MODE_CONSOLIDATION)]),
            &Sample::Gauge(200.0),
        );
        assert_eq!(
            sample(&samples, LAST_STAGE_WITNESS_DURATION, &[("mode", MODE_RANGE)]),
            &Sample::Gauge(5.0),
        );
        assert_eq!(
            sample(&samples, LAST_STAGE_EXECUTE_DURATION, &[("mode", MODE_CONSOLIDATION)]),
            &Sample::Gauge(8.0),
        );
        assert!(
            samples.keys().all(|(name, _)| !name.starts_with("kona_zkvm_canary_cycle_tracker_"))
        );
    }

    #[test]
    fn timeout_attempt_records_completed_report_and_durations() {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || {
            let mut metrics = CanaryMetrics::register();
            let current = execution(
                stage(report(ExecutionMode::Range, Some(10), 100, 3, &[]), 1.0, 2.0),
                stage(report(ExecutionMode::Consolidation, Some(20), 200, 4, &[]), 3.0, 4.0),
            );
            metrics.observe_at(
                &RunnerEvent::Attempt(attempt(RunOutcome::Valid, 100, Some(current))),
                110,
            );

            let timed_out = ExecutionResult {
                outcome: ExecutionOutcome::TimedOut,
                range: stage(report(ExecutionMode::Range, Some(30), 300, 5, &[]), 9.0, 10.0),
                consolidation: StageResult {
                    mode: ExecutionMode::Consolidation,
                    outcome: StageOutcome::TimedOut,
                    report: None,
                    witness_seconds: 11.0,
                    execute_seconds: None,
                    error: None,
                },
            };
            metrics.observe_at(
                &RunnerEvent::Attempt(attempt(RunOutcome::Timeout, 300, Some(timed_out))),
                310,
            );
        });

        let samples = samples(snapshotter.snapshot());
        assert_eq!(
            sample(&samples, LAST_STAGE_WITNESS_DURATION, &[("mode", MODE_RANGE)]),
            &Sample::Gauge(9.0),
        );
        assert_eq!(
            sample(&samples, REPORT_INSTRUCTIONS, &[("mode", MODE_RANGE)]),
            &Sample::Gauge(300.0),
        );
        assert_eq!(
            sample(&samples, REPORT_TARGET, &[("mode", MODE_CONSOLIDATION)]),
            &Sample::Gauge(100.0),
        );
        assert_eq!(
            sample(&samples, LAST_STAGE_EXECUTE_DURATION, &[("mode", MODE_RANGE)]),
            &Sample::Gauge(10.0),
        );
        assert_eq!(
            sample(&samples, LAST_STAGE_WITNESS_DURATION, &[("mode", MODE_CONSOLIDATION)]),
            &Sample::Gauge(11.0),
        );
        assert_eq!(
            sample(&samples, LAST_STAGE_EXECUTE_DURATION, &[("mode", MODE_CONSOLIDATION)]),
            &Sample::Gauge(4.0),
        );
    }
}
