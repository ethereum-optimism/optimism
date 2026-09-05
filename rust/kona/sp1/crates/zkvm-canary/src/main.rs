//! Binary entrypoint for the Kona zkVM canary.

use std::{
    future::Future,
    process::ExitCode,
    time::{SystemTime, UNIX_EPOCH},
};

use anyhow::{Context, Result, anyhow, bail};
use clap::Parser;
use kona_sp1_host_utils::{logger::setup_logger, metrics::init_metrics};
use kona_sp1_super_range_executor::{EXIT_INFRA, EXIT_INVALID, EXIT_VALID};
use kona_zkvm_canary::{
    artifact::{ArtifactConfig, ValidatedRangeArtifact, load_range_artifact},
    config::{CanaryConfig, ServiceMode},
    execution::ReportSummary,
    metrics::CanaryMetrics,
    runner::{AttemptResult, RunOutcome, Runner, RunnerEvent},
};
use tokio::sync::watch;

const LOG_PREFIX: &str = "KONA_ZKVM_CANARY";
const MAX_LOG_ERROR_BYTES: usize = 4096;

/// Runs the authenticated super-range ELF continuously or for one diagnostic attempt.
#[derive(Debug, Parser)]
#[command(name = "kona-zkvm-canary", version, about = env!("CARGO_PKG_DESCRIPTION"))]
struct Cli {
    /// Run one normal selection, execution, and revalidation attempt, then exit.
    #[arg(long)]
    once: bool,
}

#[tokio::main]
async fn main() -> ExitCode {
    let cli = Cli::parse();
    setup_logger(LOG_PREFIX);

    match run_service(cli).await {
        Ok(code) => ExitCode::from(code),
        Err(error) => {
            let error = redacted_error(&error);
            tracing::error!(error = %error, "kona-zkvm-canary exited with an infrastructure error");
            ExitCode::from(EXIT_INFRA)
        }
    }
}

async fn run_service(cli: Cli) -> Result<u8> {
    let mode = if cli.once { ServiceMode::Once } else { ServiceMode::Loop };
    let config = CanaryConfig::from_env(mode).context("invalid canary configuration")?;
    let metrics_listen = config.metrics_listen;
    let (artifact, runner, metrics_addr) =
        authenticate_before_metrics(&config.artifact, |artifact| async {
            let runner =
                Runner::new(&config, &artifact).context("failed to initialize canary runner")?;
            let metrics_addr =
                init_metrics(metrics_listen).await.context("failed to initialize metrics")?;
            Ok((artifact, runner, metrics_addr))
        })
        .await?;
    let identity = artifact.identity();
    let mut metrics = CanaryMetrics::register(identity);
    let mut runner = runner;

    tracing::info!(
        service_version = env!("CARGO_PKG_VERSION"),
        prestate = %identity.prestate,
        range_vkey = %identity.range_vkey,
        elf_sha256 = %identity.elf_sha256,
        sp1_version = sp1_sdk::SP1_CIRCUIT_VERSION.trim(),
        metrics_addr = ?metrics_addr,
        once = cli.once,
        "kona-zkvm-canary started"
    );

    if cli.once {
        run_once(&mut runner, &mut metrics).await
    } else {
        run_loop(&mut runner, &mut metrics).await
    }
}

async fn authenticate_before_metrics<T, B, F>(
    config: &ArtifactConfig,
    after_authentication: B,
) -> Result<T>
where
    B: FnOnce(ValidatedRangeArtifact) -> F,
    F: Future<Output = Result<T>>,
{
    let artifact =
        load_range_artifact(config).await.context("failed to authenticate range artifact")?;
    after_authentication(artifact).await
}

async fn run_once(runner: &mut Runner, metrics: &mut CanaryMetrics) -> Result<u8> {
    let (_shutdown_tx, mut shutdown) = watch::channel(false);
    observe_and_log(metrics, &RunnerEvent::SchedulerHeartbeat { unix_time: unix_now() });
    observe_and_log(metrics, &RunnerEvent::RunActive { active: true });
    let result = tokio::select! {
        result = runner.run_one(&mut shutdown) => result,
        signal = termination_signal() => exit_on_signal(signal?, signal_exit_code(true)),
    };
    observe_and_log(metrics, &RunnerEvent::RunActive { active: false });
    let event =
        result?.ok_or_else(|| anyhow!("one-shot run stopped without a terminal outcome"))?;
    metrics.observe(&event);
    log_runner_event(&event);
    metrics.mark_down();
    once_exit_code(&event)
}

async fn run_loop(runner: &mut Runner, metrics: &mut CanaryMetrics) -> Result<u8> {
    let (_shutdown_tx, shutdown) = watch::channel(false);
    tokio::select! {
        result = runner.run(shutdown, |event| {
            metrics.observe(event);
            log_runner_event(event);
        }) => {
            result?;
            Ok(EXIT_VALID)
        }
        signal = termination_signal() => exit_on_signal(signal?, signal_exit_code(false)),
    }
}

#[cfg(unix)]
async fn termination_signal() -> Result<&'static str> {
    use tokio::signal::unix::{SignalKind, signal};

    let mut interrupt = signal(SignalKind::interrupt()).context("failed to listen for SIGINT")?;
    let mut terminate = signal(SignalKind::terminate()).context("failed to listen for SIGTERM")?;
    tokio::select! {
        signal = interrupt.recv() => signal.map(|()| "SIGINT").ok_or_else(|| anyhow!("SIGINT listener closed")),
        signal = terminate.recv() => signal.map(|()| "SIGTERM").ok_or_else(|| anyhow!("SIGTERM listener closed")),
    }
}

#[cfg(not(unix))]
async fn termination_signal() -> Result<&'static str> {
    tokio::signal::ctrl_c().await.context("failed to listen for Ctrl-C")?;
    Ok("Ctrl-C")
}

fn exit_on_signal(signal: &str, exit_code: u8) -> ! {
    tracing::info!(
        signal,
        "termination signal received; exiting and abandoning any in-flight SP1 execution"
    );
    std::process::exit(i32::from(exit_code));
}

const fn signal_exit_code(once: bool) -> u8 {
    if once { EXIT_INFRA } else { EXIT_VALID }
}

fn unix_now() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).unwrap_or_default().as_secs()
}

fn observe_and_log(metrics: &mut CanaryMetrics, event: &RunnerEvent) {
    metrics.observe(event);
    log_runner_event(event);
}

fn log_runner_event(event: &RunnerEvent) {
    match event {
        RunnerEvent::SchedulerHeartbeat { unix_time } => {
            tracing::info!(unix_time, "kona-zkvm-canary scheduler heartbeat");
        }
        RunnerEvent::RunActive { active } => {
            tracing::info!(active, "kona-zkvm-canary run state changed");
        }
        RunnerEvent::Attempt(result) => log_attempt(result),
        RunnerEvent::SkippedSuccessful { fingerprint, target_timestamp } => {
            tracing::info!(
                %fingerprint,
                target_timestamp,
                "kona-zkvm-canary skipped an already successful fingerprint"
            );
        }
        RunnerEvent::ConfirmedFailure { fingerprint, target_timestamp, outcome } => {
            tracing::error!(
                %fingerprint,
                target_timestamp,
                outcome = outcome_label(*outcome),
                "kona-zkvm-canary correctness failure remains confirmed"
            );
        }
    }
}

fn log_attempt(result: &AttemptResult) {
    let (range_report, consolidation_report) = report_summaries(result);
    let range_outcome = result.execution.as_ref().map(|execution| execution.range.outcome);
    let consolidation_outcome =
        result.execution.as_ref().map(|execution| execution.consolidation.outcome);
    let detail = result.detail.as_deref().map(redacted_log_detail);
    tracing::info!(
        outcome = outcome_label(result.outcome),
        fingerprint = ?result.fingerprint,
        target_timestamp = ?result.target_timestamp,
        span_length = ?result.span_length,
        chain_count = ?result.chain_count,
        confirmation = result.confirmation,
        input_selection_seconds = result.input_selection_seconds,
        total_seconds = result.total_seconds,
        range_outcome = ?range_outcome,
        range_report = ?range_report,
        consolidation_outcome = ?consolidation_outcome,
        consolidation_report = ?consolidation_report,
        detail = ?detail,
        "kona-zkvm-canary attempt completed"
    );
}

const fn report_summaries(
    result: &AttemptResult,
) -> (Option<&ReportSummary>, Option<&ReportSummary>) {
    let Some(execution) = &result.execution else { return (None, None) };
    (execution.range.report.as_ref(), execution.consolidation.report.as_ref())
}

fn once_exit_code(event: &RunnerEvent) -> Result<u8> {
    let RunnerEvent::Attempt(result) = event else {
        bail!("one-shot runner returned a non-terminal event")
    };
    Ok(exit_code_for_outcome(result.outcome))
}

const fn exit_code_for_outcome(outcome: RunOutcome) -> u8 {
    match outcome {
        RunOutcome::Valid => EXIT_VALID,
        RunOutcome::GuestRejected | RunOutcome::OutputMismatch => EXIT_INVALID,
        RunOutcome::StaleSnapshot |
        RunOutcome::InputError |
        RunOutcome::CycleLimitExceeded |
        RunOutcome::Timeout => EXIT_INFRA,
    }
}

const fn outcome_label(outcome: RunOutcome) -> &'static str {
    match outcome {
        RunOutcome::Valid => "valid",
        RunOutcome::GuestRejected => "guest_rejected",
        RunOutcome::OutputMismatch => "output_mismatch",
        RunOutcome::StaleSnapshot => "stale_snapshot",
        RunOutcome::InputError => "input_error",
        RunOutcome::CycleLimitExceeded => "cycle_limit_exceeded",
        RunOutcome::Timeout => "timeout",
    }
}

fn redacted_log_detail(detail: &str) -> &str {
    if detail.contains("://") { "[redacted URL-bearing detail]" } else { detail }
}

fn redacted_error(error: &anyhow::Error) -> String {
    let rendered = format!("{error:#}");
    let rendered = redacted_log_detail(&rendered);
    if rendered.len() <= MAX_LOG_ERROR_BYTES {
        return rendered.to_string();
    }
    let mut boundary = MAX_LOG_ERROR_BYTES;
    while !rendered.is_char_boundary(boundary) {
        boundary -= 1;
    }
    rendered[..boundary].to_string()
}

#[cfg(test)]
mod tests {
    use std::{
        cell::Cell,
        io::Write,
        sync::{Arc, Mutex},
        time::Duration,
    };

    use alloy_primitives::B256;
    use clap::error::ErrorKind;
    use flate2::{Compression, write::GzEncoder};
    use kona_zkvm_canary::{
        artifact::ArtifactIdentity,
        execution::{
            ExecutionMode, ExecutionOutcome, ExecutionResult, ReportSummary, StageOutcome,
            StageResult,
        },
    };
    use tempfile::tempdir;
    use tracing_subscriber::fmt::MakeWriter;
    use url::Url;

    use super::*;

    #[derive(Clone, Default)]
    struct LogBuffer(Arc<Mutex<Vec<u8>>>);

    struct LogWriter(Arc<Mutex<Vec<u8>>>);

    impl Write for LogWriter {
        fn write(&mut self, buffer: &[u8]) -> std::io::Result<usize> {
            self.0.lock().unwrap().extend_from_slice(buffer);
            Ok(buffer.len())
        }

        fn flush(&mut self) -> std::io::Result<()> {
            Ok(())
        }
    }

    impl<'a> MakeWriter<'a> for LogBuffer {
        type Writer = LogWriter;

        fn make_writer(&'a self) -> Self::Writer {
            LogWriter(self.0.clone())
        }
    }

    #[tokio::test]
    async fn artifact_mismatch_exits_before_metrics_bind() {
        let directory = tempdir().unwrap();
        let prestate = B256::repeat_byte(0x11);
        let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
        encoder.write_all(b"not the configured ELF").unwrap();
        std::fs::write(
            directory.path().join(format!("{prestate}.range.bin.gz")),
            encoder.finish().unwrap(),
        )
        .unwrap();
        let config = ArtifactConfig {
            base_url: Url::from_directory_path(directory.path()).unwrap(),
            identity: ArtifactIdentity {
                prestate,
                range_vkey: B256::repeat_byte(0x22),
                elf_sha256: B256::repeat_byte(0x33),
            },
            max_compressed_bytes: 1024,
            max_decompressed_bytes: 1024,
            fetch_timeout: Duration::from_secs(1),
            allow_file: true,
        };
        let metrics_bound = Cell::new(false);

        let error = authenticate_before_metrics(&config, |_| async {
            metrics_bound.set(true);
            Ok(())
        })
        .await
        .unwrap_err();

        assert!(format!("{error:#}").contains("SHA-256 mismatch"));
        assert!(!metrics_bound.get());
    }

    #[test]
    fn once_reports_outcome_and_exit_code() {
        let execution = ExecutionResult {
            outcome: ExecutionOutcome::GuestRejected,
            range: stage(ExecutionMode::Range),
            consolidation: stage(ExecutionMode::Consolidation),
        };
        let result = attempt(RunOutcome::GuestRejected, Some(execution));
        let event = RunnerEvent::Attempt(result);

        let logs = LogBuffer::default();
        let subscriber = tracing_subscriber::fmt().json().with_writer(logs.clone()).finish();
        tracing::subscriber::with_default(subscriber, || log_runner_event(&event));
        let rendered = String::from_utf8(logs.0.lock().unwrap().clone()).unwrap();
        let record: serde_json::Value = serde_json::from_str(rendered.trim()).unwrap();
        assert_eq!(record["fields"]["outcome"], "guest_rejected");
        assert!(record["fields"]["range_report"].as_str().unwrap().contains("Range"));
        assert!(
            record["fields"]["consolidation_report"].as_str().unwrap().contains("Consolidation")
        );
        let RunnerEvent::Attempt(result) = &event else { unreachable!() };
        let (range, consolidation) = report_summaries(result);
        assert_eq!(range.unwrap().mode, ExecutionMode::Range);
        assert_eq!(consolidation.unwrap().mode, ExecutionMode::Consolidation);
        assert_eq!(once_exit_code(&event).unwrap(), EXIT_INVALID);

        for (outcome, expected) in [
            (RunOutcome::Valid, EXIT_VALID),
            (RunOutcome::GuestRejected, EXIT_INVALID),
            (RunOutcome::OutputMismatch, EXIT_INVALID),
            (RunOutcome::StaleSnapshot, EXIT_INFRA),
            (RunOutcome::InputError, EXIT_INFRA),
            (RunOutcome::CycleLimitExceeded, EXIT_INFRA),
            (RunOutcome::Timeout, EXIT_INFRA),
        ] {
            assert_eq!(
                once_exit_code(&RunnerEvent::Attempt(attempt(outcome, None))).unwrap(),
                expected
            );
        }
    }

    #[test]
    fn version_flag_reports_build_version() {
        let error = Cli::try_parse_from(["kona-zkvm-canary", "--version"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::DisplayVersion);
        assert!(error.to_string().contains(env!("CARGO_PKG_VERSION")));
    }

    #[test]
    fn signal_during_once_is_an_infrastructure_exit() {
        assert_eq!(signal_exit_code(true), EXIT_INFRA);
        assert_eq!(signal_exit_code(false), EXIT_VALID);
    }

    fn attempt(outcome: RunOutcome, execution: Option<ExecutionResult>) -> AttemptResult {
        AttemptResult {
            fingerprint: Some(B256::repeat_byte(0x44)),
            target_timestamp: Some(100),
            span_length: Some(1),
            chain_count: Some(2),
            confirmation: false,
            outcome,
            execution,
            input_selection_seconds: 0.25,
            total_seconds: 1.5,
            detail: Some("bounded diagnostic".to_string()),
        }
    }

    fn stage(mode: ExecutionMode) -> StageResult {
        StageResult {
            mode,
            outcome: StageOutcome::Valid,
            report: Some(ReportSummary {
                mode,
                pgu: Some(10),
                instructions: 20,
                syscalls: 2,
                record_bytes: 40,
                touched_addresses: 3,
                exit_code: 0,
                opcode_details: Vec::new(),
                syscall_details: Vec::new(),
                cycle_phases: Vec::new(),
            }),
            witness_seconds: 0.5,
            execute_seconds: Some(1.0),
            error: None,
        }
    }
}
