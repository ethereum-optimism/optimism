//! Utilities for the rollup node service, internal to the crate.

use std::convert::Infallible;
use tokio::task::JoinSet;

/// Spawns a set of parallel actors in a [`JoinSet`], and stops all of them as soon as any of them
/// fails. The type of the error in the [`NodeActor`](crate::NodeActor)s is erased to avoid having
/// to specify a common error type between actors.
///
/// Actors are passed in as `Option<actor>`, and each one is driven by
/// [`run_actor`](crate::run_actor).
///
/// Shutdown is crash-only: once an actor fails, this returns without waiting for the others, and
/// dropping the [`JoinSet`] aborts them at their next `.await`, even while they handle an input.
/// The node has to recover from being killed at any point anyway, so this is its only shutdown
/// path. Stopping actors only while they wait in [`NodeActor::next`](crate::NodeActor::next), and
/// waiting for them, would let them finish handling their current input instead, without changing
/// any actor.
///
/// OS signals are handled by the caller.
///
/// Evaluates to the `Result<(), String>` of [`wait_for_actors`].
macro_rules! spawn_and_wait {
    (actors = [$($actor:expr),* $(,)?]) => {{
        let mut task_handles = tokio::task::JoinSet::new();

        $(
            if let Some(actor) = $actor {
                task_handles.spawn(async move {
                    $crate::run_actor(actor).await.map_err(|e| format!("{e:?}"))
                });
            }
        )*

        $crate::service::wait_for_actors(task_handles).await
    }};
}

// Export the `spawn_and_wait` macro for use in other modules.
pub(crate) use spawn_and_wait;

/// Waits for the first actor to fail, then returns without waiting for the others: dropping
/// `task_handles` aborts them.
///
/// Returns the error of the actor that failed, or `Ok` if there were no actors.
pub(crate) async fn wait_for_actors(
    mut task_handles: JoinSet<Result<Infallible, String>>,
) -> Result<(), String> {
    // Actors only stop by failing, so the first one to stop is the failure.
    let Some(exit) = task_handles.join_next().await else { return Ok(()) };
    let error = match exit {
        Ok(Err(e)) => e,
        Err(e) => format!("Task join error: {e}"),
    };
    error!(target: "rollup_node", "Critical error in sub-routine: {error}");
    Err(error)
}

#[cfg(test)]
mod tests {
    use crate::NodeActor;
    use async_trait::async_trait;
    use tokio::{
        sync::oneshot,
        time::{self, Duration},
    };

    /// Receives a single input, then blocks in `handle` until released.
    struct Worker {
        started: Option<oneshot::Sender<()>>,
        release: Option<oneshot::Receiver<()>>,
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

    async fn run(worker: Worker, failing: Failing) -> Result<(), String> {
        spawn_and_wait!(actors = [Some(worker), Some(failing)])
    }

    /// When a peer fails, an actor in the middle of handling an input does not get to finish it:
    /// the node returns straight away and the actor is stopped. This fails if the node waits for
    /// the actors to stop, which with stopping actors only in `next` would make shutdown
    /// graceful.
    #[tokio::test(start_paused = true)]
    async fn in_flight_input_is_not_waited_for_on_failure() {
        let (started_tx, started) = oneshot::channel();
        let (mut release, release_rx) = oneshot::channel();
        let (trigger, trigger_rx) = oneshot::channel();
        let worker = Worker { started: Some(started_tx), release: Some(release_rx) };
        let run = tokio::spawn(run(worker, Failing { trigger: trigger_rx }));

        started.await.unwrap();
        trigger.send(()).unwrap();

        // The worker is never released, so waiting for it would hang until the timeout.
        let result = time::timeout(Duration::from_secs(1), run)
            .await
            .expect("the node waited for the in-flight input");
        assert_eq!(result.unwrap(), Err(format!("{:?}", "boom")));
        // Stopping the worker drops its end of the channel, rather than leaving it running.
        time::timeout(Duration::from_secs(1), release.closed())
            .await
            .expect("the worker was not stopped");
    }
}
