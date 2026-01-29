use std::{future::Future, time::Duration};
use tokio::{select, task::JoinHandle, time::sleep};
use tokio_util::sync::CancellationToken;
use tracing::{error, info, warn};

/// Spawns a background task that retries the given async operation with backoff on failure.
///
/// - `operation`: The async task to retry (must return `Result<(), E>`)
/// - `cancel_token`: Cancels the retry loop
/// - `max_retries`: Max retries before exiting (use `usize::MAX` for infinite)
pub(super) fn spawn_task_with_retry<Fut, E>(
    operation: impl Fn() -> Fut + Send + Sync + 'static,
    cancel_token: CancellationToken,
    max_retries: usize,
) -> JoinHandle<()>
where
    Fut: Future<Output = Result<(), E>> + Send + 'static,
    E: std::fmt::Display + Send + 'static,
{
    tokio::spawn(async move {
        let mut attempt = 0;

        loop {
            if cancel_token.is_cancelled() {
                info!(target: "supervisor::retrier", "Retry loop cancelled before starting");
                break;
            }

            match operation().await {
                Ok(()) => {
                    info!(target: "supervisor::retrier", "Task exited successfully, restarting");
                    attempt = 0; // Reset attempt count on success
                }
                Err(err) => {
                    attempt += 1;

                    if attempt > max_retries {
                        error!(target: "supervisor::retrier", %err, "Retry limit ({max_retries}) exceeded");
                        break;
                    }

                    let delay = backoff_delay(attempt);
                    warn!(
                        target: "supervisor::retrier",
                        %err,
                        ?delay,
                        "Attempt {attempt}/{max_retries} failed, retrying after delay"
                    );

                    select! {
                        _ = sleep(delay) => {}
                        _ = cancel_token.cancelled() => {
                            warn!(target: "supervisor::retrier", "Retry loop cancelled during backoff");
                            break;
                        }
                    }
                }
            }
        }
    })
}

/// Calculates exponential backoff delay with a max cap (30s).
fn backoff_delay(attempt: usize) -> Duration {
    let secs = 2u64.saturating_pow(attempt.min(5) as u32);
    Duration::from_secs(secs.min(30))
}
