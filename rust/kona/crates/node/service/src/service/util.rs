//! Utilities for the rollup node service, internal to the crate.

use std::time::Duration;
use tokio::task::{JoinError, JoinSet};
use tokio_util::sync::CancellationToken;

/// How long to wait for the actors to finish handling their current input once shutdown begins.
///
/// Kept below the 10 second grace period that `docker stop` gives before sending SIGKILL.
const SHUTDOWN_TIMEOUT: Duration = Duration::from_secs(5);

/// Spawns a set of parallel actors in a [`JoinSet`], and cancels all actors if any of them fail.
/// The type of the error in the [`NodeActor`](crate::NodeActor)s is erased to avoid having to
/// specify a common error type between actors.
///
/// Actors are passed in as `Option<actor>`. Each actor is stepped in a loop, with external
/// cancellation via the provided [`CancellationToken`]. Only waiting for the next input, with
/// [`NodeActor::next`](crate::NodeActor::next), is raced against cancellation; once an input is
/// received, [`NodeActor::handle`](crate::NodeActor::handle) runs to completion.
///
/// This macro also handles OS shutdown signals (SIGTERM, SIGINT); see [`wait_for_actors`].
///
/// Evaluates to the `Result<(), String>` of [`wait_for_actors`].
macro_rules! spawn_and_wait {
    ($cancellation:expr, actors = [$($actor:expr),* $(,)?]) => {{
        let mut task_handles = tokio::task::JoinSet::new();

        $(
            if let Some(mut actor) = $actor {
                let cancellation = $cancellation.clone();
                task_handles.spawn(async move {
                    // This guard ensures that the cancellation token is cancelled when the actor
                    // task exits for any reason. This ensures peer actors observe shutdown on
                    // their next macro-level `select!`.
                    // Note the underscore prefix: this is to signal that we don't use the guard
                    // anywhere, but *the compiler shouldn't optimize it away*. Note that using a
                    // simple `_` would not work here because it gets optimized away in release
                    // mode.
                    let _guard = cancellation.clone().drop_guard();
                    loop {
                        let input = tokio::select! {
                            biased;
                            _ = cancellation.cancelled() => return Ok(()),
                            input = $crate::NodeActor::next(&mut actor) => {
                                input.map_err(|e| format!("{e:?}"))?
                            }
                        };
                        $crate::NodeActor::handle(&mut actor, input)
                            .await
                            .map_err(|e| format!("{e:?}"))?;
                    }
                });
            }
        )*

        $crate::service::wait_for_actors($cancellation, task_handles).await
    }};
}

// Export the `spawn_and_wait` macro for use in other modules.
pub(crate) use spawn_and_wait;

/// Waits for a shutdown signal or for the first actor to fail, then cancels the actors and waits
/// for them to finish handling their current input.
///
/// The wait for the actors to stop is bounded by [`SHUTDOWN_TIMEOUT`], and cut short by a second
/// shutdown signal; actors still running after that are aborted.
///
/// Returns the error of the actor that failed first, if any.
pub(crate) async fn wait_for_actors(
    cancellation: CancellationToken,
    mut task_handles: JoinSet<Result<(), String>>,
) -> Result<(), String> {
    let shutdown = shutdown_signal();
    tokio::pin!(shutdown);

    // Wait for the first error or until all actors are finished, or the first shutdown signal.
    let result = loop {
        tokio::select! {
            _ = &mut shutdown => {
                info!(target: "rollup_node", "Received shutdown signal, initiating graceful shutdown...");
                break Ok(());
            }
            exit = task_handles.join_next() => {
                let Some(exit) = exit else { break Ok(()) }; // All tasks completed
                // Actors only exit successfully once cancelled by a peer's exit,
                // whose error may not have been joined yet.
                if let Some(e) = exit_error(exit) {
                    error!(target: "rollup_node", "Critical error in sub-routine: {e}");
                    break Err(e);
                }
            }
        }
    };

    // Signal to any actors that are still running to stop.
    cancellation.cancel();

    // Wait for all remaining actors.
    let join_all = async {
        while let Some(exit) = task_handles.join_next().await {
            // Peers are torn down concurrently, so errors here, such as closed channels, are
            // usually a consequence of the shutdown rather than its cause.
            if let Some(e) = exit_error(exit) {
                warn!(target: "rollup_node", "Actor exited with an error during shutdown: {e}");
            }
        }
    };

    // Give a grace period for actors to stop.
    tokio::select! {
        _ = join_all => info!(target: "rollup_node", "All actors stopped"),
        _ = shutdown_signal() => {
            warn!(target: "rollup_node", "Received second shutdown signal, aborting remaining actors");
        }
        _ = tokio::time::sleep(SHUTDOWN_TIMEOUT) => {
            warn!(target: "rollup_node", timeout = ?SHUTDOWN_TIMEOUT, "Timed out waiting for actors to stop, aborting remaining actors");
        }
    }

    // Dropping `task_handles` aborts any actors that are still running.
    result
}

/// Returns the error an actor task exited with, if any.
fn exit_error(exit: Result<Result<(), String>, JoinError>) -> Option<String> {
    match exit {
        Ok(Ok(())) => None,
        Ok(Err(e)) => Some(e),
        Err(e) => Some(format!("Task join error: {e}")),
    }
}

/// Listens for OS shutdown signals (SIGTERM, SIGINT)
///
/// Only signals received after this future is first polled are observed.
pub(crate) async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c().await.expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            tracing::info!(target: "rollup_node", "Received SIGINT (Ctrl+C)");
        },
        _ = terminate => {
            tracing::info!(target: "rollup_node", "Received SIGTERM");
        },
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::NodeActor;
    use async_trait::async_trait;
    use std::sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    };
    use tokio::sync::oneshot;

    /// Receives a single input, then blocks in `handle` until released.
    struct Worker {
        started: Option<oneshot::Sender<()>>,
        release: Option<oneshot::Receiver<()>>,
        done: Arc<AtomicBool>,
    }

    #[async_trait]
    impl NodeActor for Worker {
        type Error = String;
        type Input = ();

        async fn next(&mut self) -> Result<(), String> {
            if self.release.is_none() {
                std::future::pending::<()>().await;
            }
            Ok(())
        }

        async fn handle(&mut self, (): ()) -> Result<(), String> {
            let _ = self.started.take().unwrap().send(());
            let _ = self.release.take().unwrap().await;
            self.done.store(true, Ordering::SeqCst);
            Ok(())
        }
    }

    /// Fails once triggered.
    struct Failing {
        trigger: oneshot::Receiver<()>,
    }

    #[async_trait]
    impl NodeActor for Failing {
        type Error = String;
        type Input = ();

        async fn next(&mut self) -> Result<(), String> {
            let _ = (&mut self.trigger).await;
            Ok(())
        }

        async fn handle(&mut self, (): ()) -> Result<(), String> {
            Err("boom".to_string())
        }
    }

    async fn run(
        cancellation: CancellationToken,
        worker: Worker,
        failing: Failing,
    ) -> Result<(), String> {
        spawn_and_wait!(cancellation, actors = [Some(worker), Some(failing)])
    }

    struct Harness {
        cancellation: CancellationToken,
        started: oneshot::Receiver<()>,
        release: oneshot::Sender<()>,
        trigger: oneshot::Sender<()>,
        done: Arc<AtomicBool>,
        run: tokio::task::JoinHandle<Result<(), String>>,
    }

    fn harness() -> Harness {
        let cancellation = CancellationToken::new();
        let (started_tx, started) = oneshot::channel();
        let (release, release_rx) = oneshot::channel();
        let (trigger, trigger_rx) = oneshot::channel();
        let done = Arc::new(AtomicBool::new(false));
        let worker =
            Worker { started: Some(started_tx), release: Some(release_rx), done: done.clone() };
        let run = tokio::spawn(run(cancellation.clone(), worker, Failing { trigger: trigger_rx }));
        Harness { cancellation, started, release, trigger, done, run }
    }

    /// An actor's in-flight input is handled to completion when a peer fails.
    #[tokio::test]
    async fn in_flight_input_is_handled_to_completion_on_shutdown() {
        let h = harness();
        h.started.await.unwrap();
        h.trigger.send(()).unwrap();
        h.cancellation.cancelled().await;
        assert!(!h.run.is_finished(), "shutdown must wait for the in-flight input");

        h.release.send(()).unwrap();
        assert_eq!(h.run.await.unwrap(), Err(format!("{:?}", "boom")));
        assert!(h.done.load(Ordering::SeqCst));
    }

    /// An actor that does not finish handling its input is abandoned after the timeout.
    #[tokio::test(start_paused = true)]
    async fn stuck_actor_is_abandoned_after_timeout() {
        let h = harness();
        h.started.await.unwrap();
        h.trigger.send(()).unwrap();

        assert_eq!(h.run.await.unwrap(), Err(format!("{:?}", "boom")));
        assert!(!h.done.load(Ordering::SeqCst));
        drop(h.release);
    }
}
