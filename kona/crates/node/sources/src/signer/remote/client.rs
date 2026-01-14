use alloy_primitives::Address;
use alloy_rpc_client::ClientBuilder;
use alloy_transport_http::Http;
use reqwest::header::HeaderMap;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::RwLock;
use url::Url;

use crate::{
    RemoteSignerHandler,
    signer::remote::cert::{CertificateError, ClientCert},
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
/// By default, the process will watch for changes in the client certificate files and reload the
/// client automatically.
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
    /// HTTP client build error
    #[error("HTTP client build error: {0}")]
    HTTPClientBuild(#[from] reqwest::Error),
    /// Invalid certificate error
    #[error("Invalid certificate: {0}")]
    Certificate(#[from] CertificateError),
    /// Certificate watcher error
    #[error("Certificate watcher error: {0}")]
    CertificateWatcher(#[from] notify::Error),
}

impl RemoteSigner {
    /// Creates a new remote signer with the given configuration
    ///
    /// If client certificates are configured, this will automatically start a certificate watcher
    /// that monitors the certificate files for changes. When certificates are updated (e.g., by
    /// cert-manager in Kubernetes), the TLS client will be automatically reloaded with the new
    /// certificates without requiring a restart.
    ///
    /// # Certificate Watching
    ///
    /// The certificate watcher monitors:
    /// - Client certificate file (if mTLS is configured)
    /// - Client private key file (if mTLS is configured)
    /// - CA certificate file (if custom CA is configured)
    ///
    /// When any of these files are modified, the watcher will:
    /// 1. Log the certificate change event
    /// 2. Reload the certificate files from disk
    /// 3. Rebuild the HTTP client with the new TLS configuration
    /// 4. Replace the existing client atomically
    ///
    /// This enables zero-downtime certificate rotation in production environments.
    pub async fn start(self) -> Result<RemoteSignerHandler, RemoteSignerStartError> {
        let http_client = self.build_http_client()?;
        let transport = Http::with_client(http_client, self.endpoint.clone());
        let client = ClientBuilder::default().transport(transport, true);

        // Try to ping the signer to check if it's reachable
        let version: String =
            client.request("health_status", ()).await.map_err(RemoteSignerStartError::Ping)?;

        tracing::info!(target: "signer", version, "Connected to op-signer server");

        let client = Arc::new(RwLock::new(client));

        // Start certificate watcher if client certificates are configured
        let watcher_handle = self.start_certificate_watcher(client.clone()).await?;

        Ok(RemoteSignerHandler { client, watcher_handle, address: self.address })
    }

    /// Builds an HTTP client with certificate handling for the remote signer
    pub(super) fn build_http_client(&self) -> Result<reqwest::Client, RemoteSignerStartError> {
        let mut client_builder = reqwest::Client::builder();

        // Configure TLS if certificates are provided
        if self.client_cert.is_some() || self.ca_cert.is_some() {
            let tls_config = self.build_tls_config()?;
            client_builder = client_builder.use_preconfigured_tls(tls_config);
        }

        // Set headers
        client_builder = client_builder.default_headers(self.headers.clone());

        client_builder.build().map_err(RemoteSignerStartError::HTTPClientBuild)
    }
}
