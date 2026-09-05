//! Witness collection and SP1 CPU execution for one validated snapshot.

use std::{collections::BTreeMap, num::NonZeroU64};

use anyhow::{Result, ensure};
use kona_sp1_client_utils::super_root::{
    SuperConsolidationInputs, SuperConsolidationOutputs, SuperRangeInputs, SuperRangeOutputs,
    hash_super_root_proof,
};
use kona_sp1_super_range_executor::{
    HostInputs, SynthesizedExecution, build_interop_host, build_super_consolidation_stdin,
    build_super_range_stdin, collect_consolidation_witness, collect_range_witness,
    decode_super_consolidation_public_values, decode_super_range_public_values,
};
use sp1_core_executor::{ExecutionError, SP1CoreOpts};
use sp1_sdk::{Elf, ExecutionReport, Prover, ProverClient};
use tokio::{sync::watch, time::Instant};

use crate::artifact::ValidatedRangeArtifact;

const MAX_REPORT_DETAILS: usize = 128;
const MAX_CYCLE_PHASES: usize = 64;
const MAX_DETAIL_NAME_BYTES: usize = 64;
const MAX_ERROR_BYTES: usize = 4096;
const OTHER_PHASE: &str = "other";

#[derive(Clone, Debug)]
pub(crate) struct ExecutionActivity {
    active: watch::Sender<bool>,
}

impl ExecutionActivity {
    pub(crate) fn new() -> Self {
        let (active, _) = watch::channel(false);
        Self { active }
    }

    pub(crate) fn subscribe(&self) -> watch::Receiver<bool> {
        self.active.subscribe()
    }

    pub(crate) fn enter(&self) -> ExecutionActivityGuard {
        self.active.send_replace(true);
        ExecutionActivityGuard { active: self.active.clone() }
    }
}

pub(crate) struct ExecutionActivityGuard {
    active: watch::Sender<bool>,
}

impl Drop for ExecutionActivityGuard {
    fn drop(&mut self) {
        self.active.send_replace(false);
    }
}

/// Guest mode represented by a completed stage.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum ExecutionMode {
    /// Optimistic per-chain range execution.
    Range,
    /// Canonical super-root consolidation.
    Consolidation,
}

impl ExecutionMode {
    /// Returns the bounded metric label.
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Range => "range",
            Self::Consolidation => "consolidation",
        }
    }
}

/// Bounded count from an SP1 report map.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ReportDetail {
    /// Debug name supplied by the pinned SP1 SDK.
    pub name: String,
    /// Number of occurrences.
    pub count: u64,
}

/// Bounded cycle-tracker totals for one guest-defined phase.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CyclePhaseSummary {
    /// Guest-defined cycle phase, length-limited at collection.
    pub phase: String,
    /// Total cycles attributed to the phase.
    pub cycles: u64,
    /// Number of tracker invocations.
    pub invocations: u64,
}

/// Immutable telemetry extracted from an SP1 execution report.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ReportSummary {
    /// Executed guest mode.
    pub mode: ExecutionMode,
    /// Normalized proving gas units, when SP1 calculated them.
    pub pgu: Option<u64>,
    /// Total RISC-V instructions.
    pub instructions: u64,
    /// Total syscalls.
    pub syscalls: u64,
    /// Estimated execution-record bytes.
    pub record_bytes: u64,
    /// Number of distinct touched guest addresses.
    pub touched_addresses: u64,
    /// Guest exit code.
    pub exit_code: u64,
    /// Non-zero opcode counts, bounded for logs.
    pub opcode_details: Vec<ReportDetail>,
    /// Non-zero syscall counts, bounded for logs.
    pub syscall_details: Vec<ReportDetail>,
    /// Guest cycle trackers, bounded for logs and metric cardinality.
    pub cycle_phases: Vec<CyclePhaseSummary>,
}

impl ReportSummary {
    /// Extracts an immutable bounded summary from a mutable SP1 report.
    pub fn from_report(mode: ExecutionMode, report: &ExecutionReport) -> Self {
        let opcode_details = report
            .opcode_counts
            .iter()
            .filter(|(_, count)| **count != 0)
            .take(MAX_REPORT_DETAILS)
            .map(|(opcode, count)| ReportDetail {
                name: bounded_name(&format!("{opcode:?}")),
                count: *count,
            })
            .collect();
        let syscall_details = report
            .syscall_counts
            .iter()
            .filter(|(_, count)| **count != 0)
            .take(MAX_REPORT_DETAILS)
            .map(|(syscall, count)| ReportDetail {
                name: bounded_name(&format!("{syscall:?}")),
                count: *count,
            })
            .collect();
        let mut phase_counts = BTreeMap::<String, (u64, u64)>::new();
        for (phase, cycles) in &report.cycle_tracker {
            let value = phase_counts.entry(phase.clone()).or_default();
            value.0 = value.0.saturating_add(*cycles);
        }
        for (phase, invocations) in &report.invocation_tracker {
            let value = phase_counts.entry(phase.clone()).or_default();
            value.1 = value.1.saturating_add(*invocations);
        }
        let invalid_phase = |phase: &str| {
            phase.is_empty() || phase == OTHER_PHASE || phase.len() > MAX_DETAIL_NAME_BYTES
        };
        let needs_other = phase_counts.len() > MAX_CYCLE_PHASES ||
            phase_counts.keys().any(|phase| invalid_phase(phase));
        let named_limit = MAX_CYCLE_PHASES.saturating_sub(usize::from(needs_other));
        let mut phases = Vec::with_capacity(MAX_CYCLE_PHASES);
        let mut other = (0u64, 0u64);
        for (phase, (cycles, invocations)) in phase_counts {
            if invalid_phase(&phase) || phases.len() >= named_limit {
                other.0 = other.0.saturating_add(cycles);
                other.1 = other.1.saturating_add(invocations);
            } else {
                phases.push(CyclePhaseSummary { phase, cycles, invocations });
            }
        }
        if needs_other {
            phases.push(CyclePhaseSummary {
                phase: OTHER_PHASE.to_string(),
                cycles: other.0,
                invocations: other.1,
            });
        }
        Self {
            mode,
            pgu: report.gas(),
            instructions: report.total_instruction_count(),
            syscalls: report.total_syscall_count(),
            record_bytes: report.total_record_size(),
            touched_addresses: report.touched_memory_addresses,
            exit_code: report.exit_code,
            opcode_details,
            syscall_details,
            cycle_phases: phases,
        }
    }
}

/// Outcome of one execution stage.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum StageOutcome {
    /// Guest output matched both the native output and selected input.
    Valid,
    /// The guest rejected otherwise validated witness inputs.
    GuestRejected,
    /// Guest output differed from the native or selected canonical claim.
    OutputMismatch,
    /// Guest execution reached its configured cycle ceiling.
    CycleLimitExceeded,
    /// A cancellable witness stage reached the attempt deadline.
    TimedOut,
    /// The stage could not start because a prior infrastructure step failed.
    NotRun,
    /// Witness or host infrastructure failed.
    InfrastructureFailure,
}

/// Result and telemetry for one guest mode.
#[derive(Clone, Debug, PartialEq)]
pub struct StageResult {
    /// Stage mode.
    pub mode: ExecutionMode,
    /// Typed stage outcome.
    pub outcome: StageOutcome,
    /// SP1 report for a completed guest invocation.
    pub report: Option<ReportSummary>,
    /// Witness collection duration.
    pub witness_seconds: f64,
    /// Guest execution duration, when SP1 execution started.
    pub execute_seconds: Option<f64>,
    /// Bounded diagnostic detail; never used as a metric label.
    pub error: Option<String>,
}

impl StageResult {
    const fn not_run(mode: ExecutionMode) -> Self {
        Self {
            mode,
            outcome: StageOutcome::NotRun,
            report: None,
            witness_seconds: 0.0,
            execute_seconds: None,
            error: None,
        }
    }
}

/// Overall result from attempting both guest modes.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ExecutionOutcome {
    /// Both modes accepted the canonical inputs.
    Valid,
    /// At least one mode rejected its validated witness.
    GuestRejected,
    /// At least one decoded output diverged from native or selected inputs.
    OutputMismatch,
    /// At least one guest reached its configured cycle ceiling.
    CycleLimitExceeded,
    /// A cancellable witness stage reached the attempt deadline.
    TimedOut,
    /// Host, witness, or execution setup prevented a complete correctness result.
    InfrastructureFailure,
}

/// Complete in-process execution result.
#[derive(Clone, Debug, PartialEq)]
pub struct ExecutionResult {
    /// Overall typed result.
    pub outcome: ExecutionOutcome,
    /// Range-mode result.
    pub range: StageResult,
    /// Consolidation-mode result.
    pub consolidation: StageResult,
}

/// Executes range and consolidation against a synthesized canonical snapshot.
pub(crate) async fn execute_snapshot(
    host_inputs: &HostInputs,
    synthesized: &SynthesizedExecution,
    artifact: &ValidatedRangeArtifact,
    cycle_limit: NonZeroU64,
    memory_limit: NonZeroU64,
    deadline: Instant,
    execution_activity: &ExecutionActivity,
) -> ExecutionResult {
    let range_witness_started = Instant::now();
    let range_host = match build_interop_host(
        host_inputs,
        synthesized.range_inputs.l1_head,
        &synthesized.previous_super_root_proof_bytes,
        synthesized.current_super_root,
        synthesized.range_inputs.span.end,
    ) {
        Ok(host) => host,
        Err(error) => {
            return infrastructure_result(
                ExecutionMode::Range,
                range_witness_started.elapsed().as_secs_f64(),
                error,
            );
        }
    };
    let range_witness = tokio::time::timeout_at(
        deadline,
        collect_range_witness(
            range_host,
            &synthesized.range_inputs,
            &synthesized.preloaded_preimages,
        ),
    )
    .await;
    let (range_witness, native_range) = match range_witness {
        Ok(Ok(result)) => result,
        Ok(Err(error)) => {
            return infrastructure_result(
                ExecutionMode::Range,
                range_witness_started.elapsed().as_secs_f64(),
                error,
            );
        }
        Err(_) => {
            return timeout_result(
                ExecutionMode::Range,
                range_witness_started.elapsed().as_secs_f64(),
            );
        }
    };
    let range_witness_seconds = range_witness_started.elapsed().as_secs_f64();

    let core_opts = SP1CoreOpts { memory_limit: memory_limit.get(), ..Default::default() };
    let client = ProverClient::builder().cpu().with_opts(core_opts).build().await;
    let range_execute_started = Instant::now();
    let range = match build_super_range_stdin(&synthesized.range_inputs, range_witness) {
        Ok(stdin) => {
            let execution = {
                let _active = execution_activity.enter();
                client
                    .execute(Elf::Dynamic(artifact.bytes()), stdin)
                    .cycle_limit(cycle_limit.get())
                    .await
            };
            match execution {
                Ok((mut public_values, report)) => {
                    let summary = ReportSummary::from_report(ExecutionMode::Range, &report);
                    let classified =
                        decode_super_range_public_values(&mut public_values).and_then(|decoded| {
                            classify_range_outputs(
                                &native_range,
                                &decoded,
                                &synthesized.range_inputs,
                            )
                        });
                    stage_from_classification(
                        ExecutionMode::Range,
                        classified,
                        summary,
                        range_witness_seconds,
                        range_execute_started.elapsed().as_secs_f64(),
                    )
                }
                Err(error) => stage_from_execution_error(
                    ExecutionMode::Range,
                    &error,
                    cycle_limit,
                    range_witness_seconds,
                    range_execute_started.elapsed().as_secs_f64(),
                ),
            }
        }
        Err(error) => StageResult {
            mode: ExecutionMode::Range,
            outcome: StageOutcome::InfrastructureFailure,
            report: None,
            witness_seconds: range_witness_seconds,
            execute_seconds: None,
            error: Some(bounded_error(&error)),
        },
    };

    let consolidation_witness_started = Instant::now();
    let consolidation_host = match build_interop_host(
        host_inputs,
        synthesized.range_inputs.l1_head,
        &synthesized.previous_super_root_proof_bytes,
        synthesized.current_super_root,
        synthesized.range_inputs.span.end,
    ) {
        Ok(host) => host,
        Err(error) => {
            return finish_with_consolidation_failure(
                range,
                consolidation_witness_started.elapsed().as_secs_f64(),
                error,
            );
        }
    };
    let consolidation_witness = tokio::time::timeout_at(
        deadline,
        collect_consolidation_witness(
            consolidation_host,
            &synthesized.consolidation_inputs,
            &synthesized.preloaded_preimages,
        ),
    )
    .await;
    let (consolidation_witness, native_consolidation) = match consolidation_witness {
        Ok(Ok(result)) => result,
        Ok(Err(error)) => {
            return finish_with_consolidation_failure(
                range,
                consolidation_witness_started.elapsed().as_secs_f64(),
                error,
            );
        }
        Err(_) => {
            return finish_with_consolidation_timeout(
                range,
                consolidation_witness_started.elapsed().as_secs_f64(),
            );
        }
    };
    let consolidation_witness_seconds = consolidation_witness_started.elapsed().as_secs_f64();
    let consolidation_execute_started = Instant::now();
    let consolidation = match build_super_consolidation_stdin(
        &synthesized.consolidation_inputs,
        consolidation_witness,
    ) {
        Ok(stdin) => {
            let execution = {
                let _active = execution_activity.enter();
                client
                    .execute(Elf::Dynamic(artifact.bytes()), stdin)
                    .cycle_limit(cycle_limit.get())
                    .await
            };
            match execution {
                Ok((mut public_values, report)) => {
                    let summary = ReportSummary::from_report(ExecutionMode::Consolidation, &report);
                    let classified = decode_super_consolidation_public_values(&mut public_values)
                        .and_then(|decoded| {
                            classify_consolidation_outputs(
                                &native_consolidation,
                                &decoded,
                                &synthesized.consolidation_inputs,
                            )
                        });
                    stage_from_classification(
                        ExecutionMode::Consolidation,
                        classified,
                        summary,
                        consolidation_witness_seconds,
                        consolidation_execute_started.elapsed().as_secs_f64(),
                    )
                }
                Err(error) => stage_from_execution_error(
                    ExecutionMode::Consolidation,
                    &error,
                    cycle_limit,
                    consolidation_witness_seconds,
                    consolidation_execute_started.elapsed().as_secs_f64(),
                ),
            }
        }
        Err(error) => StageResult {
            mode: ExecutionMode::Consolidation,
            outcome: StageOutcome::InfrastructureFailure,
            report: None,
            witness_seconds: consolidation_witness_seconds,
            execute_seconds: None,
            error: Some(bounded_error(&error)),
        },
    };

    finish(range, consolidation)
}

fn classify_range_outputs(
    native: &SuperRangeOutputs,
    guest: &SuperRangeOutputs,
    inputs: &SuperRangeInputs,
) -> Result<()> {
    ensure!(guest == native, "range guest output differs from native witness output");
    let expected_previous = inputs
        .previous_super_root_proofs
        .iter()
        .map(hash_super_root_proof)
        .collect::<std::result::Result<Vec<_>, _>>()?;
    ensure!(guest.span == inputs.span, "range output span does not match selected inputs");
    ensure!(guest.l1_head == inputs.l1_head, "range output L1 head does not match selected inputs");
    ensure!(
        guest.previous_super_roots == expected_previous,
        "range previous roots do not match selected inputs",
    );
    ensure!(
        guest.transitions == inputs.claimed_transitions,
        "range transitions do not match selected inputs",
    );
    Ok(())
}

fn classify_consolidation_outputs(
    native: &SuperConsolidationOutputs,
    guest: &SuperConsolidationOutputs,
    inputs: &SuperConsolidationInputs,
) -> Result<()> {
    ensure!(guest == native, "consolidation guest output differs from native witness output");
    ensure!(guest.span == inputs.span, "consolidation span does not match selected inputs");
    ensure!(
        guest.previous_super_root == inputs.previous_super_root,
        "consolidation previous root does not match selected inputs",
    );
    ensure!(
        guest.transitions.len() == inputs.transitions.len(),
        "consolidation transition count does not match selected inputs",
    );
    for (output, input) in guest.transitions.iter().zip(&inputs.transitions) {
        let expected_root = hash_super_root_proof(&input.claimed_super_root_proof)?;
        ensure!(
            output.timestamp == input.claimed_super_root_proof.super_root.timestamp,
            "consolidation timestamp does not match selected inputs",
        );
        ensure!(
            output.optimistic_blocks == input.optimistic_blocks,
            "consolidation optimistic blocks do not match selected inputs",
        );
        ensure!(
            output.super_root == expected_root,
            "consolidation root does not match selected inputs",
        );
    }
    Ok(())
}

fn stage_from_classification(
    mode: ExecutionMode,
    result: Result<()>,
    report: ReportSummary,
    witness_seconds: f64,
    execute_seconds: f64,
) -> StageResult {
    match result {
        Ok(()) => StageResult {
            mode,
            outcome: StageOutcome::Valid,
            report: Some(report),
            witness_seconds,
            execute_seconds: Some(execute_seconds),
            error: None,
        },
        Err(error) => StageResult {
            mode,
            outcome: StageOutcome::OutputMismatch,
            report: Some(report),
            witness_seconds,
            execute_seconds: Some(execute_seconds),
            error: Some(bounded_error(&error)),
        },
    }
}

fn stage_from_execution_error(
    mode: ExecutionMode,
    error: &ExecutionError,
    cycle_limit: NonZeroU64,
    witness_seconds: f64,
    execute_seconds: f64,
) -> StageResult {
    StageResult {
        mode,
        outcome: classify_execution_error(error, cycle_limit),
        report: None,
        witness_seconds,
        execute_seconds: Some(execute_seconds),
        error: Some(bounded_error(error)),
    }
}

fn classify_execution_error(error: &ExecutionError, cycle_limit: NonZeroU64) -> StageOutcome {
    let expected = ExecutionError::ExceededCycleLimit(cycle_limit.get()).to_string();
    match error {
        ExecutionError::ExceededCycleLimit(_) => StageOutcome::CycleLimitExceeded,
        ExecutionError::Other(message)
            if message == &expected || message.strip_prefix("Execution: ") == Some(&expected) =>
        {
            StageOutcome::CycleLimitExceeded
        }
        ExecutionError::KilledByMemoryMonitor(_) | ExecutionError::ChildKilled() => {
            StageOutcome::InfrastructureFailure
        }
        _ => StageOutcome::GuestRejected,
    }
}

fn infrastructure_result(
    mode: ExecutionMode,
    witness_seconds: f64,
    error: anyhow::Error,
) -> ExecutionResult {
    let failed = StageResult {
        mode,
        outcome: StageOutcome::InfrastructureFailure,
        report: None,
        witness_seconds,
        execute_seconds: None,
        error: Some(bounded_error(&error)),
    };
    match mode {
        ExecutionMode::Range => finish(failed, StageResult::not_run(ExecutionMode::Consolidation)),
        ExecutionMode::Consolidation => finish(StageResult::not_run(ExecutionMode::Range), failed),
    }
}

fn timeout_result(mode: ExecutionMode, witness_seconds: f64) -> ExecutionResult {
    let timed_out = StageResult {
        mode,
        outcome: StageOutcome::TimedOut,
        report: None,
        witness_seconds,
        execute_seconds: None,
        error: Some("attempt deadline elapsed during witness collection".to_string()),
    };
    match mode {
        ExecutionMode::Range => {
            finish(timed_out, StageResult::not_run(ExecutionMode::Consolidation))
        }
        ExecutionMode::Consolidation => {
            finish(StageResult::not_run(ExecutionMode::Range), timed_out)
        }
    }
}

fn finish_with_consolidation_failure(
    range: StageResult,
    witness_seconds: f64,
    error: anyhow::Error,
) -> ExecutionResult {
    let consolidation = StageResult {
        mode: ExecutionMode::Consolidation,
        outcome: StageOutcome::InfrastructureFailure,
        report: None,
        witness_seconds,
        execute_seconds: None,
        error: Some(bounded_error(&error)),
    };
    finish(range, consolidation)
}

fn finish_with_consolidation_timeout(range: StageResult, witness_seconds: f64) -> ExecutionResult {
    let consolidation = StageResult {
        mode: ExecutionMode::Consolidation,
        outcome: StageOutcome::TimedOut,
        report: None,
        witness_seconds,
        execute_seconds: None,
        error: Some("attempt deadline elapsed during witness collection".to_string()),
    };
    finish(range, consolidation)
}

fn finish(range: StageResult, consolidation: StageResult) -> ExecutionResult {
    let outcomes = [range.outcome, consolidation.outcome];
    let outcome = if outcomes.contains(&StageOutcome::OutputMismatch) {
        ExecutionOutcome::OutputMismatch
    } else if outcomes.contains(&StageOutcome::GuestRejected) {
        ExecutionOutcome::GuestRejected
    } else if outcomes.contains(&StageOutcome::CycleLimitExceeded) {
        ExecutionOutcome::CycleLimitExceeded
    } else if outcomes.contains(&StageOutcome::TimedOut) {
        ExecutionOutcome::TimedOut
    } else if outcomes.contains(&StageOutcome::InfrastructureFailure) ||
        outcomes.contains(&StageOutcome::NotRun)
    {
        ExecutionOutcome::InfrastructureFailure
    } else {
        ExecutionOutcome::Valid
    };
    ExecutionResult { outcome, range, consolidation }
}

fn bounded_error(error: &impl std::fmt::Display) -> String {
    bounded_text(&error.to_string(), MAX_ERROR_BYTES)
}

fn bounded_name(name: &str) -> String {
    bounded_text(name, MAX_DETAIL_NAME_BYTES)
}

fn bounded_text(value: &str, max_bytes: usize) -> String {
    if value.len() <= max_bytes {
        return value.to_string();
    }
    let mut boundary = max_bytes;
    while !value.is_char_boundary(boundary) {
        boundary -= 1;
    }
    value[..boundary].to_string()
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{B256, U256};
    use kona_sp1_client_utils::super_root::{
        SuperConsolidationTransition, SuperConsolidationTransitionInput, SuperOptimisticBlock,
        SuperOutputRoot, SuperRangeTransition, SuperRootProof, TimestampSpan,
    };
    use serde_json::json;

    use super::*;

    fn report_with_stats(include_gas: bool) -> ExecutionReport {
        let mut value = serde_json::to_value(ExecutionReport::default()).unwrap();
        if include_gas {
            value["gas"] = json!(1910);
        }
        let opcode = value["opcode_counts"].as_object_mut().unwrap().values_mut().next().unwrap();
        *opcode = json!(7);
        let syscall = value["syscall_counts"].as_object_mut().unwrap().values_mut().next().unwrap();
        *syscall = json!(3);
        value["cycle_tracker"] = json!({"derive": 99});
        value["invocation_tracker"] = json!({"derive": 2});
        value["touched_memory_addresses"] = json!(42);
        serde_json::from_value(value).unwrap()
    }

    #[test]
    fn report_summary_preserves_range_and_consolidation_stats() {
        let report = report_with_stats(true);
        for mode in [ExecutionMode::Range, ExecutionMode::Consolidation] {
            let summary = ReportSummary::from_report(mode, &report);
            assert_eq!(summary.mode, mode);
            assert_eq!(summary.pgu, Some(100));
            assert_eq!(summary.instructions, 7);
            assert_eq!(summary.syscalls, 3);
            assert_eq!(summary.touched_addresses, 42);
            assert_eq!(summary.cycle_phases[0].cycles, 99);
            assert_eq!(summary.cycle_phases[0].invocations, 2);
        }
    }

    #[test]
    fn absent_pgu_is_not_reported_as_zero() {
        assert_eq!(
            ReportSummary::from_report(ExecutionMode::Range, &report_with_stats(false)).pgu,
            None,
        );
    }

    #[test]
    fn cycle_limit_exhaustion_is_not_a_guest_rejection() {
        let limit = NonZeroU64::new(10).unwrap();
        let flattened = ExecutionError::Other("exceeded cycle limit of 10".to_string());
        let task_error_flattened =
            ExecutionError::Other("Execution: exceeded cycle limit of 10".to_string());
        for error in [ExecutionError::ExceededCycleLimit(10), flattened, task_error_flattened] {
            let outcome = classify_execution_error(&error, limit);
            assert_eq!(outcome, StageOutcome::CycleLimitExceeded);
            assert_ne!(outcome, StageOutcome::GuestRejected);
        }
        assert_eq!(
            classify_execution_error(
                &ExecutionError::Other("exceeded cycle limit of 11".to_string()),
                limit,
            ),
            StageOutcome::GuestRejected,
        );
    }

    #[test]
    fn host_dependent_kills_are_retryable_infrastructure_failures() {
        assert_eq!(
            classify_execution_error(
                &ExecutionError::KilledByMemoryMonitor(24_576),
                NonZeroU64::MIN,
            ),
            StageOutcome::InfrastructureFailure,
        );
        assert_eq!(
            classify_execution_error(&ExecutionError::ChildKilled(), NonZeroU64::MIN),
            StageOutcome::InfrastructureFailure,
        );
    }

    #[test]
    fn excess_and_invalid_cycle_phases_are_aggregated_as_other() {
        let mut value = serde_json::to_value(ExecutionReport::default()).unwrap();
        let mut cycles = serde_json::Map::new();
        let mut invocations = serde_json::Map::new();
        for index in 0..(MAX_CYCLE_PHASES + 5) {
            let phase = format!("phase-{index:02}");
            cycles.insert(phase.clone(), json!(1));
            invocations.insert(phase, json!(1));
        }
        let long = "x".repeat(MAX_DETAIL_NAME_BYTES + 1);
        cycles.insert(long.clone(), json!(100));
        invocations.insert(long, json!(10));
        cycles.insert(OTHER_PHASE.to_string(), json!(200));
        invocations.insert(OTHER_PHASE.to_string(), json!(20));
        value["cycle_tracker"] = cycles.into();
        value["invocation_tracker"] = invocations.into();

        let report = serde_json::from_value(value).unwrap();
        let summary = ReportSummary::from_report(ExecutionMode::Range, &report);

        assert_eq!(summary.cycle_phases.len(), MAX_CYCLE_PHASES);
        assert!(summary.cycle_phases.iter().all(|phase| {
            phase.phase == OTHER_PHASE || phase.phase.len() <= MAX_DETAIL_NAME_BYTES
        }));
        let other = summary.cycle_phases.iter().find(|phase| phase.phase == OTHER_PHASE).unwrap();
        assert_eq!(other.cycles, 306);
        assert_eq!(other.invocations, 36);
    }

    fn stage(mode: ExecutionMode, outcome: StageOutcome) -> StageResult {
        StageResult {
            mode,
            outcome,
            report: None,
            witness_seconds: 0.0,
            execute_seconds: None,
            error: None,
        }
    }

    #[test]
    fn completed_correctness_failure_is_not_masked_by_later_failure() {
        let mismatch = finish(
            stage(ExecutionMode::Range, StageOutcome::OutputMismatch),
            stage(ExecutionMode::Consolidation, StageOutcome::InfrastructureFailure),
        );
        assert_eq!(mismatch.outcome, ExecutionOutcome::OutputMismatch);

        let rejection = finish(
            stage(ExecutionMode::Range, StageOutcome::GuestRejected),
            stage(ExecutionMode::Consolidation, StageOutcome::CycleLimitExceeded),
        );
        assert_eq!(rejection.outcome, ExecutionOutcome::GuestRejected);
    }

    fn range_fixture() -> (SuperRangeInputs, SuperRangeOutputs) {
        let span = TimestampSpan::new(10, 10).unwrap();
        let proof = SuperRootProof::new(9, vec![SuperOutputRoot::new(1, B256::repeat_byte(7))]);
        let transition = SuperRangeTransition {
            timestamp: 10,
            optimistic_block: SuperOptimisticBlock {
                chain_id: U256::from(1),
                block_hash: B256::repeat_byte(1),
                output_root: B256::repeat_byte(2),
            },
        };
        let inputs = SuperRangeInputs {
            span,
            l1_head: B256::repeat_byte(3),
            chain_ids: vec![U256::from(1)],
            previous_super_root_proofs: vec![proof.clone()],
            claimed_transitions: vec![transition],
        };
        let outputs = SuperRangeOutputs {
            span,
            l1_head: inputs.l1_head,
            previous_super_roots: vec![hash_super_root_proof(&proof).unwrap()],
            transitions: vec![transition],
        };
        (inputs, outputs)
    }

    #[test]
    fn range_output_mismatch_is_a_correctness_failure() {
        let (inputs, native) = range_fixture();
        let mut guest = native.clone();
        guest.transitions[0].optimistic_block.output_root = B256::repeat_byte(9);
        assert!(classify_range_outputs(&native, &guest, &inputs).is_err());
    }

    #[test]
    fn decoded_outputs_must_bind_to_synthesized_inputs() {
        let (mut inputs, native) = range_fixture();
        inputs.l1_head = B256::repeat_byte(8);
        assert!(classify_range_outputs(&native, &native, &inputs).is_err());

        let span = TimestampSpan::new(10, 10).unwrap();
        let proof = SuperRootProof::new(10, vec![SuperOutputRoot::new(1, B256::repeat_byte(6))]);
        let consolidation_inputs = SuperConsolidationInputs {
            span,
            previous_super_root: B256::repeat_byte(4),
            transitions: vec![SuperConsolidationTransitionInput {
                optimistic_blocks: Vec::new(),
                claimed_super_root_proof: proof,
            }],
        };
        let native = SuperConsolidationOutputs {
            span,
            previous_super_root: B256::repeat_byte(5),
            transitions: vec![SuperConsolidationTransition {
                timestamp: 10,
                optimistic_blocks: Vec::new(),
                super_root: B256::ZERO,
            }],
        };
        assert!(classify_consolidation_outputs(&native, &native, &consolidation_inputs).is_err());
    }

    #[test]
    fn consolidation_failure_preserves_range_report() {
        let summary = ReportSummary::from_report(ExecutionMode::Range, &report_with_stats(true));
        let range = StageResult {
            mode: ExecutionMode::Range,
            outcome: StageOutcome::Valid,
            report: Some(summary.clone()),
            witness_seconds: 1.0,
            execute_seconds: Some(2.0),
            error: None,
        };
        let result = finish_with_consolidation_failure(range, 3.0, anyhow::anyhow!("failure"));
        assert_eq!(result.range.report, Some(summary));
        assert_eq!(result.consolidation.outcome, StageOutcome::InfrastructureFailure);
    }
}
