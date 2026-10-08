//! Supervise tokio tasks.

use std::{collections::HashMap, fmt::Debug, future::Future};
use tokio::task::{Id, JoinSet};
use tokio_util::sync::CancellationToken;

/// Spawns tasks and shuts down when one of them exits.
#[derive(Debug)]
pub(super) struct Supervisor {
    cancellation: CancellationToken,
    tasks: JoinSet<Result<(), String>>,
    names: HashMap<Id, &'static str>,
}

impl Supervisor {
    pub(super) fn new(cancellation: CancellationToken) -> Self {
        Self { cancellation, tasks: JoinSet::new(), names: HashMap::new() }
    }

    /// Starts a lifetime future, retaining the task's name for errors and panics.
    pub(super) fn spawn<F, E>(&mut self, name: &'static str, task: F)
    where
        F: Future<Output = Result<(), E>> + Send + 'static,
        E: Debug,
    {
        // Construct the guard before spawning so even aborting an unpolled task cancels peers.
        let guard = self.cancellation.clone().drop_guard();
        let handle = self.tasks.spawn(async move {
            let _guard = guard;
            task.await.map_err(|error| format!("task {name}: {error:?}"))
        });
        self.names.insert(handle.id(), name);
    }

    /// Waits for tasks, relying on their drop guards to cancel peers on exit.
    pub(super) async fn wait(mut self) -> Result<(), String> {
        while let Some(result) = self.tasks.join_next_with_id().await {
            let (id, result) = result.map_err(|error| {
                let name =
                    self.names.get(&error.id()).expect("supervised task has a registered name");
                format!("join task {name}: {error}")
            })?;

            self.names.remove(&id);
            result?;
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::future::pending;
    use tokio::sync::oneshot;

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

        let error = supervisor.wait().await.unwrap_err();

        assert_eq!(error, "task engine: \"engine failed\"");
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

        let error = supervisor.wait().await.unwrap_err();

        assert!(error.starts_with("join task derivation:"));
        assert!(error.contains("derivation panicked"));
        assert!(cancellation.is_cancelled());
    }

    #[tokio::test]
    async fn dropping_wait_cancels_and_aborts_pending_lifetime() {
        let cancellation = CancellationToken::new();
        let mut supervisor = Supervisor::new(cancellation.clone());
        let (started_tx, started_rx) = oneshot::channel();
        let (dropped_tx, dropped_rx) = oneshot::channel::<()>();
        supervisor.spawn("pending", async move {
            let _dropped = dropped_tx;
            started_tx.send(()).unwrap();
            pending::<Result<(), &'static str>>().await
        });

        tokio::select! {
            result = supervisor.wait() => panic!("pending task unexpectedly exited: {result:?}"),
            result = started_rx => result.unwrap(),
        }

        assert!(dropped_rx.await.is_err());
        assert!(cancellation.is_cancelled());
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
