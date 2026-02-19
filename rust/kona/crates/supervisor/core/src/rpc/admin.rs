use crate::syncnode::ClientConfig;
use alloy_rpc_types_engine::JwtSecret;
use async_trait::async_trait;
use derive_more::Constructor;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject, ErrorObjectOwned},
};
use kona_supervisor_rpc::SupervisorAdminApiServer;
use std::time::Duration;
use thiserror::Error;
use tokio::{
    sync::{mpsc::Sender, oneshot},
    time::timeout,
};
use tracing::warn;

/// Error types for Supervisor Admin RPC operations.
#[derive(Debug, Error)]
pub enum AdminError {
    /// Indicates that the JWT secret is invalid.
    #[error("invalid jwt secret: {0}")]
    InvalidJwtSecret(String),

    /// Indicates that the request to the admin channel failed to send.
    #[error("failed to send admin request")]
    SendFailed,

    /// Indicates that the sender dropped before a response was received.
    #[error("admin request sender dropped")]
    SenderDropped,

    /// Indicates that the admin request timed out.
    #[error("admin request timed out")]
    Timeout,

    /// Indicates a service error occurred during processing the request.
    #[error("service error: {0}")]
    ServiceError(String),
}

impl From<AdminError> for ErrorObjectOwned {
    fn from(err: AdminError) -> Self {
        match err {
            // todo: handle these errors more gracefully
            AdminError::InvalidJwtSecret(_) => ErrorObjectOwned::from(ErrorCode::InvalidParams),
            AdminError::SendFailed |
            AdminError::SenderDropped |
            AdminError::Timeout |
            AdminError::ServiceError(_) => ErrorObjectOwned::from(ErrorCode::InternalError),
        }
    }
}

// timeout for admin requests (seconds)
const ADMIN_REQUEST_TIMEOUT_SECS: u64 = 3;

/// Represents Admin Request types
#[derive(Debug)]
pub enum AdminRequest {
    /// Adds a new L2 RPC to the Supervisor.
    AddL2Rpc {
        /// The configuration for the L2 RPC client.
        cfg: ClientConfig,
        /// The response channel to send the result back.
        resp: oneshot::Sender<Result<(), AdminError>>,
    },
}

/// Supervisor Admin RPC interface
#[derive(Debug, Constructor)]
pub struct AdminRpc {
    admin_tx: Sender<AdminRequest>,
}

#[async_trait]
impl SupervisorAdminApiServer for AdminRpc {
    /// Adds L2RPC to the supervisor.
    async fn add_l2_rpc(&self, url: String, secret: String) -> RpcResult<()> {
        let (resp_tx, resp_rx) = oneshot::channel();

        let jwt_secret = JwtSecret::from_hex(secret).map_err(|err| {
            warn!(target: "supervisor::admin_rpc", %url, %err, "Failed to decode JWT secret");
            ErrorObject::from(AdminError::InvalidJwtSecret(err.to_string()))
        })?;

        let request = AdminRequest::AddL2Rpc {
            cfg: ClientConfig { url: url.clone(), jwt_secret },
            resp: resp_tx,
        };

        self.admin_tx.send(request).await.map_err(|err| {
            warn!(target: "supervisor::admin_rpc", %url, %err, "Failed to send AdminRequest");
            ErrorObject::from(AdminError::SendFailed)
        })?;

        // wait for response with a timeout
        timeout(Duration::from_secs(ADMIN_REQUEST_TIMEOUT_SECS), resp_rx)
            .await
            .map_or_else(
                |_| {
                    warn!(target: "supervisor::admin_rpc", %url, "AdminRequest timed out");
                    Err(ErrorObject::from(AdminError::Timeout))
                },
                |res| res
                    .unwrap_or(Err(AdminError::SenderDropped))
                    .map_err(|err| {
                        warn!(target: "supervisor::admin_rpc", %url, %err, "Failed to process AdminRequest");
                        ErrorObject::from(err)
                    }),
            )
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::{
        sync::mpsc,
        time::{self, Duration},
    };

    // valid 32-byte hex (64 hex chars)
    const VALID_SECRET: &str = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";

    #[tokio::test]
    async fn test_add_l2_rpc_success() {
        let (tx, mut rx) = mpsc::channel::<AdminRequest>(1);
        let admin = AdminRpc::new(tx.clone());

        // spawn a task that simulates the service handling the admin request
        let handler = tokio::spawn(async move {
            if let Some(AdminRequest::AddL2Rpc { cfg, resp }) = rx.recv().await {
                assert_eq!(cfg.url, "http://node:8545");
                // reply success
                let _ = resp.send(Ok(()));
            } else {
                panic!("expected AddL2Rpc request");
            }
        });

        let res = admin.add_l2_rpc("http://node:8545".to_string(), VALID_SECRET.to_string()).await;
        assert!(res.is_ok(), "expected successful response");

        handler.await.unwrap();
    }

    #[tokio::test]
    async fn test_add_l2_rpc_invalid_jwt() {
        // admin with working channel (not used because parsing fails early)
        let (tx, _rx) = mpsc::channel::<AdminRequest>(1);
        let admin = AdminRpc::new(tx);

        let res = admin.add_l2_rpc("http://node:8545".to_string(), "zzzz".to_string()).await;
        assert!(res.is_err(), "expected error for invalid jwt secret");
    }

    #[tokio::test]
    async fn test_add_l2_rpc_send_failed() {
        // create channel and drop the receiver to force send() -> Err
        let (tx, rx) = mpsc::channel::<AdminRequest>(1);
        drop(rx);
        let admin = AdminRpc::new(tx);

        let res = admin.add_l2_rpc("http://node:8545".to_string(), VALID_SECRET.to_string()).await;
        assert!(res.is_err(), "expected error when admin channel receiver is closed");
    }

    #[tokio::test]
    async fn test_add_l2_rpc_service_response_dropped() {
        let (tx, mut rx) = mpsc::channel::<AdminRequest>(1);
        let admin = AdminRpc::new(tx.clone());

        // handler drops the response sender to simulate service failure before replying
        let handler = tokio::spawn(async move {
            if let Some(AdminRequest::AddL2Rpc { cfg: _, resp }) = rx.recv().await {
                // drop the sender without sending -> receiver side will get Err
                drop(resp);
            } else {
                panic!("expected AddL2Rpc request");
            }
        });

        let res = admin.add_l2_rpc("http://node:8545".to_string(), VALID_SECRET.to_string()).await;
        assert!(res.is_err(), "expected error when service drops response channel");
        handler.await.unwrap();
    }

    #[tokio::test]
    async fn test_add_l2_rpc_timeout() {
        // use a handler that receives the request but does not reply (keeps sender alive)
        let (tx, mut rx) = mpsc::channel::<AdminRequest>(1);
        let admin = AdminRpc::new(tx.clone());

        let handler = tokio::spawn(async move {
            if let Some(AdminRequest::AddL2Rpc { cfg: _, resp: _ }) = rx.recv().await {
                // hold the sender (do nothing) so the rpc call times out
                // keep the future alive long enough (we use tokio::time::advance in the test)
                time::sleep(Duration::from_secs(ADMIN_REQUEST_TIMEOUT_SECS + 5)).await;
            } else {
                panic!("expected AddL2Rpc request");
            }
        });

        // call the rpc concurrently
        let call = tokio::spawn(async move {
            admin.add_l2_rpc("http://node:8545".to_string(), VALID_SECRET.to_string()).await
        });

        let res = call.await.unwrap();
        assert!(res.is_err(), "expected timeout error for long-running admin handler");

        // let handler finish cleanly
        handler.await.unwrap();
    }
}
