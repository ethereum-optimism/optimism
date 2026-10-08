//! Supervision of actor lifetime futures.

use crate::NodeActor;
use std::{collections::HashMap, fmt::Debug, future::Future};
use tokio::task::{Id, JoinSet};
use tokio_util::sync::CancellationToken;

/// Spawns actor lifetime futures and shuts down the node when any actor exits.
///
/// Actors handle cancellation themselves. On an error, panic, or shutdown request, remaining tasks
/// are aborted when the supervisor is dropped, matching the existing node shutdown policy.
#[derive(Debug)]
pub(super) struct Supervisor {
    cancellation: CancellationToken,
    tasks: JoinSet<Result<(), String>>,
    actor_names: HashMap<Id, &'static str>,
}

impl Supervisor {
    pub(super) fn new(cancellation: CancellationToken) -> Self {
        Self { cancellation, tasks: JoinSet::new(), actor_names: HashMap::new() }
    }

    /// Starts a lifetime future, retaining the actor's name for errors and panics.
    pub(super) fn spawn<F, E>(&mut self, name: &'static str, task: F)
    where
        F: Future<Output = Result<(), E>> + Send + 'static,
        E: Debug,
    {
        // Construct the guard before spawning so even aborting an unpolled task cancels peers.
        let guard = self.cancellation.clone().drop_guard();
        let handle = self.tasks.spawn(async move {
            let _guard = guard;
            task.await.map_err(|error| format!("{name} actor failed: {error:?}"))
        });
        self.actor_names.insert(handle.id(), name);
    }

    /// Waits for actors and a caller-provided shutdown future.
    pub(super) async fn wait(mut self, shutdown: impl Future<Output = ()>) -> Result<(), String> {
        tokio::pin!(shutdown);

        loop {
            tokio::select! {
                _ = &mut shutdown => {
                    info!(target: "rollup_node", "Received shutdown signal, initiating graceful shutdown...");
                    self.cancellation.cancel();
                    return Ok(());
                }
                result = self.tasks.join_next_with_id() => {
                    match result {
                        Some(Ok((id, result))) => {
                            self.actor_names.remove(&id);
                            if let Err(error) = result {
                                error!(target: "rollup_node", "Critical error in sub-routine: {error}");
                                self.cancellation.cancel();
                                return Err(error);
                            }
                        }
                        Some(Err(error)) => {
                            let name = self.actor_names.remove(&error.id())
                                .expect("supervised task has a registered name");
                            let error = format!("{name} actor task join error: {error}");
                            error!(target: "rollup_node", "{error}");
                            self.cancellation.cancel();
                            return Err(error);
                        }
                        None => return Ok(()),
                    }
                }
            }
        }
    }
}

/// Adapts an existing step-based actor into a cancellable lifetime future.
pub(super) async fn run_node_actor<A: NodeActor>(
    mut actor: A,
    cancellation: CancellationToken,
) -> Result<(), A::Error> {
    loop {
        tokio::select! {
            biased;
            _ = cancellation.cancelled() => return Ok(()),
            result = actor.step() => result?,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        future::pending,
        sync::{
            Arc,
            atomic::{AtomicUsize, Ordering},
        },
    };
    use tokio::sync::oneshot;

    struct StepActor {
        calls: Arc<AtomicUsize>,
        fail_after: Option<usize>,
        started: Option<oneshot::Sender<()>>,
    }

    #[async_trait::async_trait]
    impl NodeActor for StepActor {
        type Error = &'static str;

        async fn step(&mut self) -> Result<(), Self::Error> {
            let calls = self.calls.fetch_add(1, Ordering::Relaxed) + 1;
            if let Some(started) = self.started.take() {
                let _ = started.send(());
            }
            match self.fail_after {
                Some(limit) if calls >= limit => Err("step failed"),
                Some(_) => Ok(()),
                None => pending().await,
            }
        }
    }

    #[tokio::test]
    async fn adapter_repeats_steps_until_error() {
        let calls = Arc::new(AtomicUsize::new(0));
        let actor = StepActor { calls: calls.clone(), fail_after: Some(3), started: None };

        let result = run_node_actor(actor, CancellationToken::new()).await;

        assert_eq!(result, Err("step failed"));
        assert_eq!(calls.load(Ordering::Relaxed), 3);
    }

    #[tokio::test]
    async fn adapter_prioritizes_cancellation_before_stepping() {
        let cancellation = CancellationToken::new();
        cancellation.cancel();
        let calls = Arc::new(AtomicUsize::new(0));
        let actor = StepActor { calls: calls.clone(), fail_after: Some(1), started: None };

        assert_eq!(run_node_actor(actor, cancellation).await, Ok(()));
        assert_eq!(calls.load(Ordering::Relaxed), 0);
    }

    #[tokio::test]
    async fn successful_lifetime_cancels_pending_step_actor() {
        let cancellation = CancellationToken::new();
        let mut supervisor = Supervisor::new(cancellation.clone());
        let (started_tx, started_rx) = oneshot::channel();
        let actor = StepActor {
            calls: Arc::new(AtomicUsize::new(0)),
            fail_after: None,
            started: Some(started_tx),
        };
        supervisor.spawn("stepping", run_node_actor(actor, cancellation.clone()));
        supervisor.spawn("lifetime", async move {
            started_rx.await.unwrap();
            Ok::<(), std::io::Error>(())
        });

        assert_eq!(supervisor.wait(pending()).await, Ok(()));
        assert!(cancellation.is_cancelled());
    }

    #[tokio::test]
    async fn actor_error_identifies_actor_and_cancels_peers() {
        let cancellation = CancellationToken::new();
        let mut supervisor = Supervisor::new(cancellation.clone());
        let (started_tx, started_rx) = oneshot::channel();
        let (cancelled_tx, cancelled_rx) = oneshot::channel();
        let peer_cancellation = cancellation.clone();
        supervisor.spawn("peer", async move {
            started_tx.send(()).unwrap();
            peer_cancellation.cancelled().await;
            cancelled_tx.send(()).unwrap();
            Ok::<(), &'static str>(())
        });
        supervisor.spawn("engine", async move {
            started_rx.await.unwrap();
            Err::<(), _>("engine failed")
        });

        let error = supervisor.wait(pending()).await.unwrap_err();

        assert_eq!(error, "engine actor failed: \"engine failed\"");
        assert!(cancellation.is_cancelled());
        // An error aborts peers; they may observe cancellation before being aborted.
        let _ = cancelled_rx.await;
    }

    #[tokio::test]
    async fn actor_panic_identifies_actor_and_cancels_peers() {
        let cancellation = CancellationToken::new();
        let mut supervisor = Supervisor::new(cancellation.clone());
        supervisor.spawn("derivation", async {
            panic!("derivation panicked");
            #[allow(unreachable_code)]
            Ok::<(), &'static str>(())
        });

        let error = supervisor.wait(pending()).await.unwrap_err();

        assert!(error.starts_with("derivation actor task join error:"));
        assert!(error.contains("derivation panicked"));
        assert!(cancellation.is_cancelled());
    }

    #[tokio::test]
    async fn shutdown_signal_cancels_and_aborts_pending_lifetime() {
        let cancellation = CancellationToken::new();
        let mut supervisor = Supervisor::new(cancellation.clone());
        let (started_tx, started_rx) = oneshot::channel();
        let (dropped_tx, dropped_rx) = oneshot::channel::<()>();
        supervisor.spawn("pending", async move {
            let _dropped = dropped_tx;
            started_tx.send(()).unwrap();
            pending::<Result<(), &'static str>>().await
        });

        let result = supervisor.wait(async { started_rx.await.unwrap() }).await;

        assert_eq!(result, Ok(()));
        assert!(cancellation.is_cancelled());
        assert!(dropped_rx.await.is_err());
    }

    #[tokio::test]
    async fn dropping_supervisor_cancels_unpolled_tasks() {
        let cancellation = CancellationToken::new();
        let mut supervisor = Supervisor::new(cancellation.clone());
        let (dropped_tx, dropped_rx) = oneshot::channel::<()>();
        supervisor.spawn("unpolled", async move {
            let _dropped = dropped_tx;
            pending::<Result<(), &'static str>>().await
        });

        drop(supervisor);

        assert!(dropped_rx.await.is_err());
        assert!(cancellation.is_cancelled());
    }
}
