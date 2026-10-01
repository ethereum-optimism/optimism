//! A JSON-RPC client over HTTP(S) that reloads its TLS material when it changes on disk.
//!
//! The reload logic is a port of op-service's `tls/certman`: it watches the directories that
//! contain the configured files, reacts to events for those file names or for the Kubernetes
//! `..data` symlink, and rebuilds the client once events have settled.

use std::{
    collections::BTreeSet,
    ffi::OsString,
    path::{Path, PathBuf},
    sync::{
        Arc, PoisonError, RwLock,
        mpsc::{self, Receiver, RecvTimeoutError},
    },
    time::Duration,
};

use alloy_rpc_client::{ClientBuilder, RpcClient};
use alloy_transport_http::{
    Http,
    reqwest::{self, header::HeaderMap},
};
use notify::{
    Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher,
    event::{AccessKind, AccessMode},
};
use rustls::{
    ClientConfig, RootCertStore,
    pki_types::{CertificateDer, PrivateKeyDer, pem::PemObject},
};
use thiserror::Error;
use url::Url;

/// How long to wait after the last relevant file event before reloading.
///
/// Matches op-service's certman. It also covers in-place writes where the certificate and key are
/// not updated atomically.
const RELOAD_DEBOUNCE: Duration = Duration::from_secs(2);

/// Name of the symlink that Kubernetes atomically swaps when it updates a Secret or `ConfigMap`
/// volume.
const KUBERNETES_DATA_DIR: &str = "..data";

/// Client certificate and key pair for mTLS authentication (PEM format)
#[derive(Debug, Clone)]
pub struct ClientCert {
    /// Path to the client certificate in PEM format
    pub cert: PathBuf,
    /// Path to the client private key in PEM format
    pub key: PathBuf,
}

/// PEM parsing error type alias
type PemError = rustls::pki_types::pem::Error;

/// Errors that can occur when handling certificates
#[derive(Debug, Error)]
pub enum CertificateError {
    /// Invalid CA certificate path
    #[error("Invalid CA certificate path: {0}")]
    InvalidCACertificatePath(PemError),
    /// Invalid certificate error
    #[error("Invalid CA certificate: {0}")]
    InvalidCACertificate(PemError),
    /// The CA certificate file contains no certificates
    #[error("No CA certificates found in {}", .0.display())]
    EmptyCACertificate(PathBuf),
    /// Failed to add CA certificate
    #[error("Failed to add CA certificate: {0}")]
    AddCACertificate(rustls::Error),
    /// Failed to configure client auth
    #[error("Failed to configure client auth: {0}")]
    ConfigureClientAuth(rustls::Error),
    /// Invalid client certificate path
    #[error("Invalid client certificate path: {0}")]
    InvalidClientCertificatePath(PemError),
    /// Invalid client certificate
    #[error("Invalid client certificate: {0}")]
    InvalidClientCertificate(PemError),
    /// The client certificate file contains no certificates
    #[error("No client certificates found in {}", .0.display())]
    EmptyClientCertificate(PathBuf),
    /// Invalid private key
    #[error("Invalid private key: {0}")]
    InvalidPrivateKey(PemError),
    /// Failed to configure the TLS protocol versions
    #[error("Failed to configure TLS protocol versions: {0}")]
    ProtocolVersions(rustls::Error),
    /// Failed to load the platform certificate verifier
    #[error("Failed to load platform certificate verifier: {0}")]
    PlatformVerifier(rustls::Error),
}

/// PEM file paths for a TLS client. Files are re-read on every reload.
#[derive(Debug, Clone, Default)]
pub struct TlsPaths {
    /// CA certificate(s) trusted for server verification. When `None`, platform roots are used.
    pub ca_cert: Option<PathBuf>,
    /// Client certificate and key for mutual TLS.
    pub client_cert: Option<ClientCert>,
}

impl TlsPaths {
    /// Whether any TLS material is configured.
    const fn is_configured(&self) -> bool {
        self.ca_cert.is_some() || self.client_cert.is_some()
    }

    /// Reads the configured files and builds a rustls client configuration.
    ///
    /// With a CA configured, only that CA is trusted. Otherwise the platform roots are trusted.
    fn client_config(&self) -> Result<ClientConfig, CertificateError> {
        let provider = Arc::new(rustls::crypto::aws_lc_rs::default_provider());
        let builder = ClientConfig::builder_with_provider(Arc::clone(&provider))
            .with_safe_default_protocol_versions()
            .map_err(CertificateError::ProtocolVersions)?;

        let builder = match &self.ca_cert {
            Some(path) => {
                let ca_certs: Vec<CertificateDer<'static>> = CertificateDer::pem_file_iter(path)
                    .map_err(CertificateError::InvalidCACertificatePath)?
                    .collect::<Result<Vec<_>, _>>()
                    .map_err(CertificateError::InvalidCACertificate)?;
                if ca_certs.is_empty() {
                    return Err(CertificateError::EmptyCACertificate(path.clone()));
                }

                let mut roots = RootCertStore::empty();
                for cert in ca_certs {
                    roots.add(cert).map_err(CertificateError::AddCACertificate)?;
                }
                builder.with_root_certificates(roots)
            }
            None => {
                let verifier = rustls_platform_verifier::Verifier::new(provider)
                    .map_err(CertificateError::PlatformVerifier)?;
                builder.dangerous().with_custom_certificate_verifier(Arc::new(verifier))
            }
        };

        match &self.client_cert {
            None => Ok(builder.with_no_client_auth()),
            Some(ClientCert { cert, key }) => {
                let certs: Vec<CertificateDer<'static>> = CertificateDer::pem_file_iter(cert)
                    .map_err(CertificateError::InvalidClientCertificatePath)?
                    .collect::<Result<Vec<_>, _>>()
                    .map_err(CertificateError::InvalidClientCertificate)?;
                if certs.is_empty() {
                    return Err(CertificateError::EmptyClientCertificate(cert.clone()));
                }

                let private_key = PrivateKeyDer::from_pem_file(key)
                    .map_err(CertificateError::InvalidPrivateKey)?;

                builder
                    .with_client_auth_cert(certs, private_key)
                    .map_err(CertificateError::ConfigureClientAuth)
            }
        }
    }

    /// Paths of every configured file.
    fn files(&self) -> impl Iterator<Item = &Path> {
        self.ca_cert
            .iter()
            .map(PathBuf::as_path)
            .chain(self.client_cert.iter().flat_map(|c| [c.cert.as_path(), c.key.as_path()]))
    }

    /// Directories containing the configured files.
    fn watch_dirs(&self) -> BTreeSet<PathBuf> {
        self.files()
            .map(|file| match file.parent() {
                Some(dir) if !dir.as_os_str().is_empty() => dir.to_path_buf(),
                _ => PathBuf::from("."),
            })
            .collect()
    }

    /// File names whose events trigger a reload: the configured files and the Kubernetes `..data`
    /// symlink.
    fn watched_names(&self) -> BTreeSet<OsString> {
        self.files()
            .filter_map(Path::file_name)
            .map(OsString::from)
            .chain([OsString::from(KUBERNETES_DATA_DIR)])
            .collect()
    }
}

/// Errors building a [`ReloadingRpcClient`].
#[derive(Debug, Error)]
pub enum ReloadingRpcClientError {
    /// Invalid certificate error
    #[error("Invalid certificate: {0}")]
    Certificate(#[from] CertificateError),
    /// HTTP client build error
    #[error("HTTP client build error: {0}")]
    HttpClientBuild(#[from] reqwest::Error),
    /// Certificate watcher error
    #[error("Certificate watcher error: {0}")]
    CertificateWatcher(#[from] notify::Error),
    /// Failed to spawn the reload thread
    #[error("Failed to spawn TLS reload thread: {0}")]
    ReloadThread(std::io::Error),
}

/// JSON-RPC client over HTTP(S) that rebuilds itself when its TLS files change on disk.
///
/// When a CA or client certificate is configured, the directories containing them are watched.
/// Changes to the configured files, including Kubernetes Secret volume updates (which swap the
/// `..data` symlink), cause the client to be rebuilt from disk two seconds after the last change.
/// If rebuilding fails, the previous client is kept and the rebuild is retried every two seconds.
#[derive(Debug, Clone)]
pub struct ReloadingRpcClient {
    client: Arc<RwLock<RpcClient>>,
    watcher: Option<Arc<RecommendedWatcher>>,
}

impl ReloadingRpcClient {
    /// Builds the client from the current files and, if any TLS material is configured, starts
    /// watching it for changes.
    pub fn new(
        endpoint: Url,
        tls: TlsPaths,
        headers: HeaderMap,
    ) -> Result<Self, ReloadingRpcClientError> {
        let client = Arc::new(RwLock::new(build_rpc_client(&endpoint, &tls, &headers)?));
        if !tls.is_configured() {
            return Ok(Self { client, watcher: None });
        }

        let dirs = tls.watch_dirs();
        let names = tls.watched_names();
        let (tx, rx) = mpsc::channel::<()>();
        let reload_client = Arc::clone(&client);
        std::thread::Builder::new()
            .name("tls-reload".into())
            .spawn(move || reload_loop(rx, reload_client, endpoint, tls, headers))
            .map_err(ReloadingRpcClientError::ReloadThread)?;

        let mut watcher = notify::recommended_watcher(
            move |res: notify::Result<Event>| match res {
                Ok(event) if is_reload_trigger(&event, &names) => {
                    tracing::debug!(
                        target: "signer:certificate-watcher",
                        event = ?event,
                        "Certificate file changed, queueing TLS reload"
                    );
                    let _ = tx.send(());
                }
                Ok(event) => {
                    tracing::trace!(target: "signer:certificate-watcher", event = ?event, "Ignoring certificate watcher event");
                }
                Err(e) => {
                    tracing::error!(target: "signer:certificate-watcher", error = %e, "Failed to receive event from watcher channel.");
                }
            },
        )?;
        for dir in &dirs {
            watcher.watch(dir, RecursiveMode::NonRecursive)?;
        }
        tracing::info!(target: "signer", dirs = ?dirs, "Starting certificate watcher for automatic TLS reload");

        Ok(Self { client, watcher: Some(Arc::new(watcher)) })
    }

    /// Returns the current client.
    ///
    /// Cloning is cheap. Use the returned client for a single request so that reloads take
    /// effect.
    pub fn client(&self) -> RpcClient {
        self.client.read().unwrap_or_else(PoisonError::into_inner).clone()
    }

    /// Whether TLS files are being watched.
    pub const fn is_watching(&self) -> bool {
        self.watcher.is_some()
    }
}

/// Builds a JSON-RPC client from the current TLS files.
fn build_rpc_client(
    endpoint: &Url,
    tls: &TlsPaths,
    headers: &HeaderMap,
) -> Result<RpcClient, ReloadingRpcClientError> {
    let mut builder = reqwest::Client::builder().default_headers(headers.clone());
    if tls.is_configured() {
        builder = builder.tls_backend_preconfigured(tls.client_config()?);
    }
    let http = builder.build()?;
    Ok(ClientBuilder::default().transport(Http::with_client(http, endpoint.clone()), false))
}

/// Whether a watcher event should trigger a reload.
///
/// Access events are ignored, except close-after-write: unlike Go's fsnotify, notify reports
/// opens and reads, and the reload itself reads the files. A rescan notice (inotify queue overflow)
/// triggers a reload because the events it replaces may have included a rotation.
fn is_reload_trigger(event: &Event, names: &BTreeSet<OsString>) -> bool {
    if event.need_rescan() {
        return true;
    }
    !matches!(event.kind, EventKind::Access(access) if access != AccessKind::Close(AccessMode::Write)) &&
        event.paths.iter().any(|p| p.file_name().is_some_and(|n| names.contains(n)))
}

/// Rebuilds the client after each burst of reload requests, until the watcher is dropped.
///
/// A failed rebuild is retried every `RELOAD_DEBOUNCE` until it succeeds, because the event that
/// caused it (for example a partially written file) may be the last one for a while.
fn reload_loop(
    rx: Receiver<()>,
    client: Arc<RwLock<RpcClient>>,
    endpoint: Url,
    tls: TlsPaths,
    headers: HeaderMap,
) {
    let mut retry = false;
    loop {
        if !retry && rx.recv().is_err() {
            return;
        }

        // Wait until no further events arrive for `RELOAD_DEBOUNCE`.
        loop {
            match rx.recv_timeout(RELOAD_DEBOUNCE) {
                Ok(()) => {}
                Err(RecvTimeoutError::Timeout) => break,
                Err(RecvTimeoutError::Disconnected) => return,
            }
        }

        tracing::debug!(target: "signer:certificate-watcher", "Reloading TLS configuration");
        match build_rpc_client(&endpoint, &tls, &headers) {
            Ok(new) => {
                *client.write().unwrap_or_else(PoisonError::into_inner) = new;
                retry = false;
                tracing::info!(target: "signer:certificate-watcher", "TLS configuration reloaded successfully");
            }
            Err(e) => {
                retry = true;
                tracing::error!(target: "signer:certificate-watcher", error = %e, "Failed to reload TLS configuration, retrying");
            }
        }
    }
}
