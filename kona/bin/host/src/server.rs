//! This module contains the [PreimageServer] struct and its implementation.

use kona_preimage::{
    HintReaderServer, PreimageOracleServer, PreimageServerBackend, errors::PreimageOracleError,
};
use std::sync::Arc;
use tokio::spawn;
use tracing::{error, info};

/// The [PreimageServer] is responsible for waiting for incoming preimage requests and
/// serving them to the client.
#[derive(Debug)]
pub struct PreimageServer<P, H, B> {
    /// The oracle server.
    oracle_server: P,
    /// The hint router.
    hint_reader: H,
    /// [PreimageServerBackend] that routes hints and retrieves preimages.
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
    /// Create a new [PreimageServer] with the given [PreimageOracleServer],
    /// [HintReaderServer], and [PreimageServerBackend].
    pub const fn new(oracle_server: P, hint_reader: H, backend: Arc<B>) -> Self {
        Self { oracle_server, hint_reader, backend }
    }

    /// Starts the [PreimageServer] and waits for incoming requests.
    pub async fn start(self) -> Result<(), PreimageServerError> {
        // Create the futures for the oracle server and hint router.
        let server = spawn(Self::start_oracle_server(self.oracle_server, self.backend.clone()));
        let hint_router = spawn(Self::start_hint_router(self.hint_reader, self.backend.clone()));

        // Race the two futures to completion, returning the result of the first one to finish.
        tokio::select! {
            s = server => s?,
            h = hint_router => h?,
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
                Ok(_) => continue,
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
                Ok(_) => continue,
                Err(PreimageOracleError::IOError(_)) => return Ok(()),
                Err(e) => {
                    error!(target: "host_server", "Failed to serve route hint: {e}");
                    return Err(PreimageServerError::RouteHintFailed(e));
                }
            }
        }
    }
}
