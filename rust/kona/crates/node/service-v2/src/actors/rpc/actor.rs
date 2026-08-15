//! RPC Server Actor

use crate::{
    NodeActor, RpcActorError,
    actors::rpc::launcher::{RpcServerHandle, RpcServerLauncher},
};
use async_trait::async_trait;
use jsonrpsee::RpcModule;

/// An actor that runs the JSON-RPC server for the rollup node.
///
/// The first launch happens upstream of this actor; restarts (up to `restart_count` provided
/// at construction) are handled inside [`Self::step`].
#[derive(Debug)]
pub struct RpcActor<Launcher: RpcServerLauncher> {
    /// Launcher used to relaunch the server if it stops.
    launcher: Launcher,
    /// Module set used to relaunch the server if it stops.
    modules: RpcModule<()>,
    /// The currently-running server handle. Replaced on each successful relaunch.
    handle: Option<Launcher::Handle>,
    /// Remaining relaunches allowed before [`Self::step`] returns
    /// [`RpcActorError::ServerStopped`].
    restarts_remaining: u32,
}

impl<Launcher: RpcServerLauncher> RpcActor<Launcher> {
    /// Constructs a new [`RpcActor`].
    ///
    /// `handle` is the live server returned by the caller's initial launch; this actor takes
    /// ownership of the running server and handles up to `restarts_remaining` subsequent
    /// relaunches via `launcher`.
    pub const fn new(
        launcher: Launcher,
        modules: RpcModule<()>,
        handle: Launcher::Handle,
        restarts_remaining: u32,
    ) -> Self {
        Self { launcher, modules, handle: Some(handle), restarts_remaining }
    }
}

impl<Launcher: RpcServerLauncher> Drop for RpcActor<Launcher> {
    fn drop(&mut self) {
        // jsonrpsee's ServerHandle is Arc<watch::Sender<()>>; dropping is enough to close the
        // watch and stop the server, but calling `stop()` explicitly is clearer about intent.
        // Errors here mean the server is already stopped.
        if let Some(handle) = self.handle.take() {
            handle.stop();
        }
    }
}

#[async_trait]
impl<Launcher: RpcServerLauncher> NodeActor for RpcActor<Launcher> {
    type Error = RpcActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let handle = self.handle.as_ref().ok_or(RpcActorError::ServerStopped)?;
        handle.stopped().await;

        if self.restarts_remaining == 0 {
            return Err(RpcActorError::ServerStopped);
        }
        self.restarts_remaining = self.restarts_remaining.saturating_sub(1);

        match self.launcher.launch(self.modules.clone()).await {
            Ok(new_handle) => {
                self.handle = Some(new_handle);
                Ok(())
            }
            Err(err) => {
                error!(target: "rpc", ?err, "Failed to launch rpc server");
                Err(RpcActorError::ServerStopped)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{
        Arc,
        atomic::{AtomicU32, Ordering},
    };
    use tokio::sync::watch;

    /// Mock handle backed by a [`watch::channel`] of `bool`.
    ///
    /// The handle starts with the value `false` ("running") and `stop()` sets it to `true`
    /// ("stopped"). [`Self::stopped`] resolves when the value becomes `true`, including
    /// retroactively if `stop()` was called before any awaiter started waiting — which matches
    /// the behavior of `jsonrpsee::ServerHandle` (and avoids the lost-wakeup hazard of `Notify`).
    ///
    /// Holds an idle [`watch::Receiver`] so that `Sender::send` from `stop()` always succeeds
    /// (a watch sender errors when every receiver has been dropped).
    #[derive(Debug, Clone)]
    struct MockHandle {
        stopped_tx: watch::Sender<bool>,
        _keepalive: Arc<watch::Receiver<bool>>,
    }

    impl MockHandle {
        fn new() -> Self {
            let (stopped_tx, keepalive) = watch::channel(false);
            Self { stopped_tx, _keepalive: Arc::new(keepalive) }
        }
    }

    #[async_trait]
    impl RpcServerHandle for MockHandle {
        async fn stopped(&self) {
            let mut rx = self.stopped_tx.subscribe();
            // If the value is already `true`, this returns immediately; otherwise it waits for
            // the next change.
            if *rx.borrow() {
                return;
            }
            let _ = rx.changed().await;
        }

        fn stop(&self) {
            let _ = self.stopped_tx.send(true);
        }
    }

    /// Mock launcher that hands out `MockHandle`s and counts launch invocations.
    ///
    /// When `fail_after` is `Some(n)`, the n-th launch (0-indexed) returns an error instead.
    #[derive(Debug)]
    struct MockLauncher {
        launch_count: Arc<AtomicU32>,
        fail_after: Option<u32>,
        handles: std::sync::Mutex<Vec<MockHandle>>,
    }

    impl MockLauncher {
        fn new(fail_after: Option<u32>) -> Self {
            Self {
                launch_count: Arc::new(AtomicU32::new(0)),
                fail_after,
                handles: std::sync::Mutex::new(Vec::new()),
            }
        }

        fn launch_count(&self) -> u32 {
            self.launch_count.load(Ordering::SeqCst)
        }

        /// Returns a clone of the handle from the n-th successful launch (0-indexed).
        fn handle(&self, n: usize) -> MockHandle {
            self.handles.lock().unwrap()[n].clone()
        }
    }

    #[async_trait]
    impl RpcServerLauncher for MockLauncher {
        type Handle = MockHandle;

        async fn launch(&self, _modules: RpcModule<()>) -> Result<Self::Handle, std::io::Error> {
            let call = self.launch_count.fetch_add(1, Ordering::SeqCst);
            if Some(call) == self.fail_after {
                return Err(std::io::Error::other("simulated launch failure"));
            }
            let handle = MockHandle::new();
            self.handles.lock().unwrap().push(handle.clone());
            Ok(handle)
        }
    }

    // Allow `Arc<MockLauncher>` to be used directly as the actor's launcher generic, so tests can
    // both hold a reference for assertions and pass ownership to the actor.
    #[async_trait]
    impl RpcServerLauncher for Arc<MockLauncher> {
        type Handle = MockHandle;

        async fn launch(&self, modules: RpcModule<()>) -> Result<Self::Handle, std::io::Error> {
            (**self).launch(modules).await
        }
    }

    async fn make_actor_with_initial_handle(
        launcher: Arc<MockLauncher>,
        restarts_remaining: u32,
    ) -> RpcActor<Arc<MockLauncher>> {
        let initial_handle = launcher.launch(RpcModule::new(())).await.expect("initial launch");
        RpcActor::new(launcher, RpcModule::new(()), initial_handle, restarts_remaining)
    }

    #[tokio::test]
    async fn step_relaunches_on_first_stop() {
        let launcher = Arc::new(MockLauncher::new(None));
        let mut actor = make_actor_with_initial_handle(launcher.clone(), 2).await;

        // Signal the first handle to stop, then drive one step.
        launcher.handle(0).stop();
        actor.step().await.expect("first relaunch should succeed");

        assert_eq!(launcher.launch_count(), 2, "expected one relaunch after the initial launch");
    }

    #[tokio::test]
    async fn step_exhausts_restarts_then_errors() {
        let launcher = Arc::new(MockLauncher::new(None));
        let mut actor = make_actor_with_initial_handle(launcher.clone(), 1).await;

        // First stop: should relaunch.
        launcher.handle(0).stop();
        actor.step().await.expect("first relaunch should succeed");

        // Second stop: restart budget is exhausted; step should error.
        launcher.handle(1).stop();
        let err = actor.step().await.expect_err("second stop should exhaust restarts");
        assert!(matches!(err, RpcActorError::ServerStopped));
    }

    #[tokio::test]
    async fn step_errors_with_zero_restarts_budget() {
        let launcher = Arc::new(MockLauncher::new(None));
        let mut actor = make_actor_with_initial_handle(launcher.clone(), 0).await;

        launcher.handle(0).stop();
        let err = actor.step().await.expect_err("zero-restart budget should error on first stop");
        assert!(matches!(err, RpcActorError::ServerStopped));
        assert_eq!(launcher.launch_count(), 1, "relaunch should not have been attempted");
    }

    #[tokio::test]
    async fn step_errors_when_relaunch_fails() {
        // `fail_after = 1` means the second launch (the first relaunch) fails.
        let launcher = Arc::new(MockLauncher::new(Some(1)));
        let mut actor = make_actor_with_initial_handle(launcher.clone(), 3).await;

        launcher.handle(0).stop();
        let err = actor.step().await.expect_err("failed relaunch should surface as ServerStopped");
        assert!(matches!(err, RpcActorError::ServerStopped));
    }

    #[tokio::test]
    async fn drop_calls_stop_on_live_handle() {
        let launcher = Arc::new(MockLauncher::new(None));
        let actor = make_actor_with_initial_handle(launcher.clone(), 0).await;

        let handle_copy = launcher.handle(0);
        assert!(!*handle_copy.stopped_tx.borrow(), "handle should start running");

        drop(actor);

        assert!(*handle_copy.stopped_tx.borrow(), "drop should have stopped the handle");
    }
}
