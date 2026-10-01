use alloy_primitives::Address;
use alloy_transport_http::reqwest::header::HeaderMap;
use thiserror::Error;
use url::Url;

use crate::{
    ClientCert, ReloadingRpcClient, ReloadingRpcClientError, RemoteSignerHandler, TlsPaths,
};

/// Configuration for the remote signer client
///
/// This configuration supports various TLS/certificate scenarios:
///
/// 1. **Basic HTTPS**: Only `endpoint` and `address` are required.
/// 2. **Custom CA**: Provide `ca_cert` to verify servers with custom/self-signed certificates.
/// 3. **Mutual TLS (mTLS)**: Provide both `client_cert` and `client_key` for client authentication.
/// 4. **Full mTLS with custom CA**: Combine all certificate options for maximum security.
///
/// Certificate formats supported:
/// - PEM format for all certificates and keys
/// - Certificates should be provided as file paths.
///
/// When a client certificate or CA is configured, the process watches the directories that contain
/// them and reloads the client automatically when they change, including Kubernetes Secret volume
/// updates that swap the `..data` symlink.
#[derive(Debug, Clone)]
pub struct RemoteSigner {
    /// The URL of the remote signer endpoint
    pub endpoint: Url,
    /// The address of the signer.
    pub address: Address,
    /// Optional client certificate for mTLS (PEM format)
    pub client_cert: Option<ClientCert>,
    /// Optional CA certificate for server verification (PEM format)
    pub ca_cert: Option<std::path::PathBuf>,
    /// Headers to pass to the remote signer.
    pub headers: HeaderMap,
}

/// Errors that can occur when starting a remote signer.
#[derive(Debug, Error)]
pub enum RemoteSignerStartError {
    /// Failed to ping signer
    #[error("Failed to ping signer: {0}")]
    Ping(alloy_transport::TransportError),
    /// Failed to build the signer client
    #[error("Failed to build signer client: {0}")]
    Client(#[from] ReloadingRpcClientError),
}

impl RemoteSigner {
    /// Creates a new remote signer with the given configuration
    ///
    /// If a client certificate or CA is configured, this will automatically start a certificate
    /// watcher. When certificates are updated (e.g., by cert-manager in Kubernetes), the TLS client
    /// will be automatically reloaded with the new certificates without requiring a restart.
    ///
    /// # Certificate Watching
    ///
    /// The watcher monitors the directories containing:
    /// - the client certificate file (if mTLS is configured)
    /// - the client private key file (if mTLS is configured)
    /// - the CA certificate file (if a custom CA is configured)
    ///
    /// Changes to any of these files, or to the Kubernetes `..data` symlink in those directories,
    /// cause the watcher to:
    /// 1. Log the certificate change event
    /// 2. Wait two seconds for further changes to settle
    /// 3. Reload the certificate files from disk and rebuild the HTTP client
    /// 4. Replace the existing client atomically, keeping it if the reload fails
    ///
    /// This enables zero-downtime certificate rotation in production environments.
    pub async fn start(self) -> Result<RemoteSignerHandler, RemoteSignerStartError> {
        let client = ReloadingRpcClient::new(
            self.endpoint,
            TlsPaths { ca_cert: self.ca_cert, client_cert: self.client_cert },
            self.headers,
        )?;

        // Try to ping the signer to check if it's reachable
        let version: String = client
            .client()
            .request("health_status", ())
            .await
            .map_err(RemoteSignerStartError::Ping)?;

        tracing::info!(target: "signer", version, "Connected to op-signer server");

        Ok(RemoteSignerHandler { client, address: self.address })
    }
}
