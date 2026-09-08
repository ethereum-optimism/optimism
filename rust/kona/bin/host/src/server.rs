//! This module contains the [`PreimageServer`] struct and its implementation.

use kona_preimage::{
    HintReaderServer, PreimageOracleServer, PreimageServerBackend, errors::PreimageOracleError,
};
use std::sync::Arc;
use tokio::{spawn, task::JoinHandle};
use tracing::{error, info};

struct AbortOnDrop<T>(JoinHandle<T>);

impl<T> Drop for AbortOnDrop<T> {
    fn drop(&mut self) {
        self.0.abort();
    }
}

/// The [`PreimageServer`] is responsible for waiting for incoming preimage requests and
/// serving them to the client.
#[derive(Debug)]
pub struct PreimageServer<P, H, B> {
    /// The oracle server.
    oracle_server: P,
    /// The hint router.
    hint_reader: H,
    /// [`PreimageServerBackend`] that routes hints and retrieves preimages.
    backend: Arc<B>,
}

/// An error that can occur when handling preimage requests
#[derive(Debug, thiserror::Error)]
pub enum PreimageServerError {
    /// A preimage request error.
    #[error("Failed to serve preimage request: {0}")]
    PreimageRequestFailed(PreimageOracleError),
    /// An error when failed to serve route hint.
    #[error("Failed to route hint: {0}")]
    RouteHintFailed(PreimageOracleError),
    /// Task failed to execute to completion.
    #[error("Join error: {0}")]
    ExecutionError(#[from] tokio::task::JoinError),
}

impl<P, H, B> PreimageServer<P, H, B>
where
    P: PreimageOracleServer + Send + Sync + 'static,
    H: HintReaderServer + Send + Sync + 'static,
    B: PreimageServerBackend + Send + Sync + 'static,
{
    /// Create a new [`PreimageServer`] with the given [`PreimageOracleServer`],
    /// [`HintReaderServer`], and [`PreimageServerBackend`].
    pub const fn new(oracle_server: P, hint_reader: H, backend: Arc<B>) -> Self {
        Self { oracle_server, hint_reader, backend }
    }

    /// Starts the [`PreimageServer`] and waits for incoming requests.
    pub async fn start(self) -> Result<(), PreimageServerError> {
        let mut server =
            AbortOnDrop(spawn(Self::start_oracle_server(self.oracle_server, self.backend.clone())));
        let mut hint_router =
            AbortOnDrop(spawn(Self::start_hint_router(self.hint_reader, self.backend.clone())));

        // Race the two futures to completion, returning the result of the first one to finish.
        tokio::select! {
            result = &mut server.0 => result?,
            result = &mut hint_router.0 => result?,
        }
    }

    /// Starts the oracle server, which waits for incoming preimage requests and serves them to the
    /// client.
    async fn start_oracle_server(
        oracle_server: P,
        backend: Arc<B>,
    ) -> Result<(), PreimageServerError> {
        info!(target: "host_server", "Starting oracle server");
        loop {
            // Serve the next preimage request. This `await` will yield to the runtime
            // if no progress can be made.
            match oracle_server.next_preimage_request(backend.as_ref()).await {
                Ok(_) => {}
                Err(PreimageOracleError::IOError(_)) => return Ok(()),
                Err(e) => {
                    error!(target: "host_server", "Failed to serve preimage request: {e}");
                    return Err(PreimageServerError::PreimageRequestFailed(e));
                }
            }
        }
    }

    /// Starts the hint router, which waits for incoming hints and routes them to the appropriate
    /// handler.
    async fn start_hint_router(hint_reader: H, backend: Arc<B>) -> Result<(), PreimageServerError> {
        info!(target: "host_server", "Starting hint router");
        loop {
            // Route the next hint. This `await` will yield to the runtime if no progress can be
            // made.
            match hint_reader.next_hint(backend.as_ref()).await {
                Ok(_) => {}
                Err(PreimageOracleError::IOError(_)) => return Ok(()),
                Err(e) => {
                    error!(target: "host_server", "Failed to serve route hint: {e}");
                    return Err(PreimageServerError::RouteHintFailed(e));
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use std::{
        future::pending,
        sync::{
            Arc, Condvar, Mutex,
            atomic::{AtomicUsize, Ordering},
        },
        time::Duration,
    };

    use async_trait::async_trait;
    use kona_preimage::{HintRouter, PreimageFetcher, PreimageKey, errors::PreimageOracleResult};
    use tokio::sync::Barrier;

    use super::*;

    struct ActiveCall(Arc<AtomicUsize>);

    impl ActiveCall {
        fn new(active: Arc<AtomicUsize>) -> Self {
            active.fetch_add(1, Ordering::SeqCst);
            Self(active)
        }
    }

    impl Drop for ActiveCall {
        fn drop(&mut self) {
            self.0.fetch_sub(1, Ordering::SeqCst);
        }
    }

    #[derive(Clone)]
    struct PendingWorker {
        active: Arc<AtomicUsize>,
        started: Arc<Barrier>,
    }

    #[async_trait]
    impl PreimageOracleServer for PendingWorker {
        async fn next_preimage_request<F>(&self, _get_preimage: &F) -> PreimageOracleResult<()>
        where
            F: PreimageFetcher + Send + Sync,
        {
            let _active = ActiveCall::new(self.active.clone());
            self.started.wait().await;
            pending().await
        }
    }

    #[async_trait]
    impl HintReaderServer for PendingWorker {
        async fn next_hint<R>(&self, _route_hint: &R) -> PreimageOracleResult<()>
        where
            R: HintRouter + Send + Sync,
        {
            let _active = ActiveCall::new(self.active.clone());
            self.started.wait().await;
            pending().await
        }
    }

    struct UnusedBackend;

    #[async_trait]
    impl PreimageFetcher for UnusedBackend {
        async fn get_preimage(&self, _key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            unreachable!("pending server never fetches a preimage")
        }
    }

    #[async_trait]
    impl HintRouter for UnusedBackend {
        async fn route_hint(&self, _hint: String) -> PreimageOracleResult<()> {
            unreachable!("pending server never routes a hint")
        }
    }

    #[derive(Default)]
    struct BlockingWorkerState {
        started: usize,
        released: bool,
    }

    #[derive(Clone)]
    struct BlockingWorker(Arc<(Mutex<BlockingWorkerState>, Condvar)>);

    impl BlockingWorker {
        fn block(&self) -> PreimageOracleResult<()> {
            let (state, state_changed) = self.0.as_ref();
            let mut state = state.lock().unwrap();
            state.started += 1;
            state_changed.notify_all();
            while !state.released {
                state = state_changed.wait(state).unwrap();
            }
            Err(PreimageOracleError::Other("worker released".into()))
        }
    }

    #[async_trait]
    impl PreimageOracleServer for BlockingWorker {
        async fn next_preimage_request<F>(&self, _get_preimage: &F) -> PreimageOracleResult<()>
        where
            F: PreimageFetcher + Send + Sync,
        {
            self.block()
        }
    }

    #[async_trait]
    impl HintReaderServer for BlockingWorker {
        async fn next_hint<R>(&self, _route_hint: &R) -> PreimageOracleResult<()>
        where
            R: HintRouter + Send + Sync,
        {
            self.block()
        }
    }

    #[tokio::test(flavor = "multi_thread", worker_threads = 4)]
    async fn blocking_channel_workers_run_concurrently() {
        let state = Arc::new((Mutex::new(BlockingWorkerState::default()), Condvar::new()));
        let worker = BlockingWorker(state.clone());
        let server = PreimageServer::new(worker.clone(), worker, Arc::new(UnusedBackend));
        let server_task = tokio::spawn(server.start());

        let wait_state = state.clone();
        let both_started = tokio::task::spawn_blocking(move || {
            let (state, state_changed) = wait_state.as_ref();
            let state = state.lock().unwrap();
            let (state, _) = state_changed
                .wait_timeout_while(state, Duration::from_secs(2), |state| state.started < 2)
                .unwrap();
            state.started == 2
        })
        .await
        .unwrap();

        let (worker_state, state_changed) = state.as_ref();
        worker_state.lock().unwrap().released = true;
        state_changed.notify_all();
        let _ = server_task.await;

        assert!(both_started, "oracle and hint workers did not run concurrently");
    }

    #[tokio::test]
    async fn aborting_server_stops_nested_workers() {
        let active = Arc::new(AtomicUsize::new(0));
        let started = Arc::new(Barrier::new(3));
        let worker = PendingWorker { active: active.clone(), started: started.clone() };
        let server = PreimageServer::new(worker.clone(), worker, Arc::new(UnusedBackend));
        let task = tokio::spawn(server.start());

        started.wait().await;
        assert_eq!(active.load(Ordering::SeqCst), 2);
        task.abort();
        assert!(task.await.unwrap_err().is_cancelled());

        assert_eq!(active.load(Ordering::SeqCst), 0);
    }
}
