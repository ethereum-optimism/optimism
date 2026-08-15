//! Long-running semantic engine service.

use crate::engine::{EngineClient, EngineDriver, EngineServiceError, api::EngineRequest};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

/// Default number of semantic engine operations that may wait for execution.
pub const DEFAULT_ENGINE_REQUEST_CAPACITY: usize = 64;

/// Serializes semantic rollup operations over one engine driver.
#[derive(Debug)]
pub struct EngineService<Driver> {
    driver: Driver,
    request_rx: mpsc::Receiver<EngineRequest>,
}

impl<Driver> EngineService<Driver>
where
    Driver: EngineDriver,
{
    /// Creates an engine service and its cloneable semantic client.
    pub fn new(driver: Driver) -> (Self, EngineClient) {
        Self::with_capacity(driver, DEFAULT_ENGINE_REQUEST_CAPACITY)
    }

    /// Creates an engine service with a bounded request capacity.
    pub fn with_capacity(driver: Driver, capacity: usize) -> (Self, EngineClient) {
        let (request_tx, request_rx) = mpsc::channel(capacity);
        (Self { driver, request_rx }, EngineClient::new(request_tx))
    }

    /// Runs until shutdown or until every engine client is dropped.
    ///
    /// Shutdown is observed only between semantic operations. An operation already handed to the
    /// driver is allowed to complete so its Engine API side effects are not cancelled ambiguously.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), EngineServiceError> {
        loop {
            let request = tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(()),
                request = self.request_rx.recv() => {
                    request.ok_or(EngineServiceError::RequestChannelClosed)?
                }
            };

            self.handle_request(request).await;
        }
    }

    async fn handle_request(&mut self, request: EngineRequest) {
        match request {
            EngineRequest::BuildUnsafe { attributes, response } => {
                let _ = response.send(self.driver.build_unsafe(*attributes).await);
            }
            EngineRequest::ImportUnsafe { payload, response } => {
                let _ = response.send(self.driver.import_unsafe(*payload).await);
            }
            EngineRequest::UpdateSafe { update, response } => {
                let _ = response.send(self.driver.update_safe(update).await);
            }
            EngineRequest::UpdateFinalized { block, response } => {
                let _ = response.send(self.driver.update_finalized(block).await);
            }
            EngineRequest::State { response } => {
                let _ = response.send(Ok(self.driver.state()));
            }
        }
    }
}
