//! Utilities for the rollup node service, internal to the crate.

use tokio::task::{JoinError, JoinSet};
use tokio_util::sync::CancellationToken;

/// Spawns a set of parallel actors in a [`JoinSet`], and cancels all actors if any of them fail.
/// The type of the error in the [`NodeActor`](crate::NodeActor)s is erased to avoid having to
/// specify a common error type between actors.
///
/// Actors are passed in as `Option<actor>`. Each actor is stepped in a loop, with external
/// cancellation via the provided [`CancellationToken`].
///
/// Shutdown is crash-only: each whole [`NodeActor::step`](crate::NodeActor::step) is raced against
/// cancellation, so an actor stops at its next `.await` even while handling an input, and the node
/// does not wait for actors to stop. The node has to recover from being killed at any point anyway,
/// so this is its only shutdown path. Racing only [`NodeActor::next`](crate::NodeActor::next)
/// against cancellation, and waiting for the actors to stop, would let them finish handling their
/// current input instead, without changing any actor.
///
/// OS signals are left to the caller: the CLI stops the node on SIGINT, and SIGTERM terminates the
/// process.
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
                        tokio::select! {
                            biased;
                            _ = cancellation.cancelled() => return Ok(()),
                            result = $crate::NodeActor::step(&mut actor) => {
                                result.map_err(|e| format!("{e:?}"))?;
                            }
                        }
                    }
                });
            }
        )*

        $crate::service::wait_for_actors($cancellation, task_handles).await
    }};
}

// Export the `spawn_and_wait` macro for use in other modules.
pub(crate) use spawn_and_wait;

/// Waits for the first actor to fail, then cancels the others, without waiting for them to stop.
///
/// Returns the error of the actor that failed first, or `Ok` if all actors finished.
pub(crate) async fn wait_for_actors(
    cancellation: CancellationToken,
    mut task_handles: JoinSet<Result<(), String>>,
) -> Result<(), String> {
    let result = loop {
        let Some(exit) = task_handles.join_next().await else { break Ok(()) }; // All tasks completed
        // Actors only exit successfully once cancelled by a peer's exit,
        // whose error may not have been joined yet.
        if let Some(e) = exit_error(exit) {
            error!(target: "rollup_node", "Critical error in sub-routine: {e}");
            break Err(e);
        }
    };

    cancellation.cancel();

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::NodeActor;
    use async_trait::async_trait;
    use std::sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    };
    use tokio::{
        sync::oneshot,
        time::{self, Duration},
    };

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
        let run = tokio::spawn(run(cancellation, worker, Failing { trigger: trigger_rx }));
        Harness { started, release, trigger, done, run }
    }

    /// When a peer fails, an actor in the middle of handling an input does not get to finish it:
    /// the node returns straight away. This fails if the node both races only `next` against
    /// cancellation and waits for the actors to stop, which would make shutdown graceful.
    #[tokio::test(start_paused = true)]
    async fn in_flight_input_is_not_waited_for_on_failure() {
        let h = harness();
        h.started.await.unwrap();
        h.trigger.send(()).unwrap();

        // The worker never gets released, so waiting for it would hang until the timeout.
        let result = time::timeout(Duration::from_secs(1), h.run)
            .await
            .expect("the node waited for the in-flight input");
        assert_eq!(result.unwrap(), Err(format!("{:?}", "boom")));
        assert!(!h.done.load(Ordering::SeqCst));
        drop(h.release);
    }
}
