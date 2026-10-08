//! Supervise tokio tasks.

use std::{collections::HashMap, fmt::Debug, future::Future};
use tokio::task::{Id, JoinSet};

/// Spawns tasks and shuts down when one of them exits.
#[derive(Debug)]
pub(super) struct Supervisor {
    tasks: JoinSet<Result<(), String>>,
    names: HashMap<Id, &'static str>,
}

impl Supervisor {
    pub(super) fn new() -> Self {
        Self { tasks: JoinSet::new(), names: HashMap::new() }
    }

    /// Starts a lifetime future, retaining the task's name for errors and panics.
    pub(super) fn spawn<F, E>(&mut self, name: &'static str, task: F)
    where
        F: Future<Output = Result<(), E>> + Send + 'static,
        E: Debug,
    {
        let handle = self
            .tasks
            .spawn(async move { task.await.map_err(|error| format!("task {name}: {error:?}")) });
        self.names.insert(handle.id(), name);
    }

    /// Waits for the first task to exit, then aborts peers.
    pub(super) async fn wait(mut self) -> Result<(), String> {
        match self.tasks.join_next_with_id().await {
            Some(Ok((_, result))) => result,
            Some(Err(error)) => {
                let name =
                    self.names.get(&error.id()).expect("supervised task has a registered name");
                Err(format!("join task {name}: {error}"))
            }
            None => Ok(()),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::future::pending;
    use tokio::sync::oneshot;

    fn pending_peer(supervisor: &mut Supervisor) -> (oneshot::Receiver<()>, oneshot::Receiver<()>) {
        let (started_tx, started_rx) = oneshot::channel();
        let (dropped_tx, dropped_rx) = oneshot::channel::<()>();
        supervisor.spawn("peer", async move {
            let _dropped = dropped_tx;
            started_tx.send(()).unwrap();
            pending::<Result<(), &'static str>>().await
        });
        (started_rx, dropped_rx)
    }

    #[tokio::test]
    async fn actor_error_identifies_actor_and_aborts_peers() {
        let mut supervisor = Supervisor::new();
        let (started_rx, dropped_rx) = pending_peer(&mut supervisor);
        supervisor.spawn("engine", async move {
            started_rx.await.unwrap();
            Err::<(), _>("engine failed")
        });

        assert_eq!(supervisor.wait().await.unwrap_err(), "task engine: \"engine failed\"");
        assert!(dropped_rx.await.is_err());
    }

    #[tokio::test]
    async fn actor_panic_identifies_actor_and_aborts_peers() {
        let mut supervisor = Supervisor::new();
        let (started_rx, dropped_rx) = pending_peer(&mut supervisor);
        supervisor.spawn("derivation", async move {
            started_rx.await.unwrap();
            panic!("derivation panicked");
            #[allow(unreachable_code)]
            Ok::<(), &'static str>(())
        });

        let error = supervisor.wait().await.unwrap_err();
        assert!(error.starts_with("join task derivation:"));
        assert!(error.contains("derivation panicked"));
        assert!(dropped_rx.await.is_err());
    }

    #[tokio::test]
    async fn successful_exit_aborts_peers() {
        let mut supervisor = Supervisor::new();
        let (started_rx, dropped_rx) = pending_peer(&mut supervisor);
        supervisor.spawn("lifetime", async move {
            started_rx.await.unwrap();
            Ok::<(), &'static str>(())
        });

        assert_eq!(supervisor.wait().await, Ok(()));
        assert!(dropped_rx.await.is_err());
    }

    #[tokio::test]
    async fn dropping_wait_aborts_pending_lifetime() {
        let mut supervisor = Supervisor::new();
        let (started_rx, dropped_rx) = pending_peer(&mut supervisor);
        tokio::select! {
            result = supervisor.wait() => panic!("pending task unexpectedly exited: {result:?}"),
            result = started_rx => result.unwrap(),
        }
        assert!(dropped_rx.await.is_err());
    }

    #[tokio::test]
    async fn dropping_supervisor_aborts_unpolled_tasks() {
        let mut supervisor = Supervisor::new();
        let (_started_rx, dropped_rx) = pending_peer(&mut supervisor);
        drop(supervisor);
        assert!(dropped_rx.await.is_err());
    }
}
