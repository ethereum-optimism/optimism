//! A JSON-RPC client over HTTP(S) that reloads its TLS material when it changes on disk.
//!
//! The reload logic is a port of op-service's `tls/certman`: it watches the directories that
//! contain the configured files, reacts to events for those file names or for the Kubernetes
//! `..data` symlink, and rebuilds the client once events have settled.

use std::{
    collections::BTreeSet,
    ffi::OsString,
    panic::{self, AssertUnwindSafe},
    path::{Path, PathBuf},
    sync::{
        Arc, PoisonError, RwLock,
        mpsc::{self, Receiver, RecvTimeoutError},
    },
    time::{Duration, Instant},
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

/// How often a reload that keeps failing is logged again, as a warning.
const RELOAD_FAILURE_LOG_INTERVAL: Duration = Duration::from_secs(60);

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
    ///
    /// With a `request_timeout`, each request fails if it does not complete, including reading
    /// the response, within that time. Clients rebuilt for new TLS files keep it.
    pub fn new(
        endpoint: Url,
        tls: TlsPaths,
        headers: HeaderMap,
        request_timeout: Option<Duration>,
    ) -> Result<Self, ReloadingRpcClientError> {
        if !tls.is_configured() {
            let client = build_rpc_client(&endpoint, &tls, &headers, request_timeout)?;
            return Ok(Self { client: Arc::new(RwLock::new(client)), watcher: None });
        }

        // Like certman, watch before the first load so that a change during the load queues a
        // reload instead of being missed. Events queue in the channel until the thread starts.
        let log_endpoint = redacted_url(&endpoint);
        let dirs = tls.watch_dirs();
        let names = tls.watched_names();
        let (tx, rx) = mpsc::channel::<()>();
        let watcher_endpoint = log_endpoint.clone();
        let mut watcher = notify::recommended_watcher(
            move |res: notify::Result<Event>| match res {
                Ok(event) if is_reload_trigger(&event, &names) => {
                    tracing::debug!(
                        target: "signer:certificate-watcher",
                        endpoint = %watcher_endpoint,
                        event = ?event,
                        "Certificate file changed, queueing TLS reload"
                    );
                    if tx.send(()).is_err() {
                        tracing::error!(target: "signer:certificate-watcher", endpoint = %watcher_endpoint, "TLS reload thread is not running, ignoring certificate change");
                    }
                }
                Ok(event) => {
                    tracing::trace!(target: "signer:certificate-watcher", endpoint = %watcher_endpoint, event = ?event, "Ignoring certificate watcher event");
                }
                Err(e) => {
                    tracing::error!(target: "signer:certificate-watcher", endpoint = %watcher_endpoint, error = %e, "Failed to receive event from watcher channel.");
                }
            },
        )?;
        for dir in &dirs {
            watcher.watch(dir, RecursiveMode::NonRecursive)?;
        }

        let client =
            Arc::new(RwLock::new(build_rpc_client(&endpoint, &tls, &headers, request_timeout)?));
        let reload_client = Arc::clone(&client);
        let thread_endpoint = log_endpoint.clone();
        std::thread::Builder::new()
            .name("tls-reload".into())
            .spawn(move || {
                run_reload_thread(
                    &rx,
                    &reload_client,
                    &endpoint,
                    &tls,
                    &headers,
                    request_timeout,
                    &thread_endpoint,
                )
            })
            .map_err(ReloadingRpcClientError::ReloadThread)?;
        tracing::info!(target: "signer", endpoint = %log_endpoint, dirs = ?dirs, "Starting certificate watcher for automatic TLS reload");

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

/// Renders a URL for logging with any userinfo stripped.
pub fn redacted_url(url: &Url) -> String {
    let mut url = url.clone();
    let _ = url.set_username("");
    let _ = url.set_password(None);
    url.to_string()
}

/// Builds a JSON-RPC client from the current TLS files.
fn build_rpc_client(
    endpoint: &Url,
    tls: &TlsPaths,
    headers: &HeaderMap,
    request_timeout: Option<Duration>,
) -> Result<RpcClient, ReloadingRpcClientError> {
    let mut builder = reqwest::Client::builder().default_headers(headers.clone());
    if let Some(timeout) = request_timeout {
        builder = builder.timeout(timeout);
    }
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

/// Runs [`reload_loop`] until the watcher is dropped.
///
/// A panic in the loop is logged and the loop restarts, waiting for the next change, so that
/// later changes are still reloaded.
fn run_reload_thread(
    rx: &Receiver<()>,
    client: &RwLock<RpcClient>,
    endpoint: &Url,
    tls: &TlsPaths,
    headers: &HeaderMap,
    request_timeout: Option<Duration>,
    log_endpoint: &str,
) {
    while panic::catch_unwind(AssertUnwindSafe(|| {
        reload_loop(rx, client, endpoint, tls, headers, request_timeout, log_endpoint);
    }))
    .is_err()
    {
        tracing::error!(target: "signer:certificate-watcher", endpoint = %log_endpoint, "TLS reload panicked, waiting for the next certificate change");
    }
}

/// Rebuilds the client after each burst of reload requests, until the watcher is dropped.
///
/// A failed rebuild is retried every `RELOAD_DEBOUNCE` until it succeeds, because the event that
/// caused it (for example a partially written file) may be the last one for a while. The first
/// failure is logged as an error; while rebuilds keep failing, a warning is logged every
/// `RELOAD_FAILURE_LOG_INTERVAL`.
fn reload_loop(
    rx: &Receiver<()>,
    client: &RwLock<RpcClient>,
    endpoint: &Url,
    tls: &TlsPaths,
    headers: &HeaderMap,
    request_timeout: Option<Duration>,
    log_endpoint: &str,
) {
    // When a failure was last logged; set while rebuilds are failing.
    let mut failure_logged_at: Option<Instant> = None;
    loop {
        if failure_logged_at.is_none() && rx.recv().is_err() {
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

        tracing::debug!(target: "signer:certificate-watcher", endpoint = %log_endpoint, "Reloading TLS configuration");
        match build_rpc_client(endpoint, tls, headers, request_timeout) {
            Ok(new) => {
                *client.write().unwrap_or_else(PoisonError::into_inner) = new;
                failure_logged_at = None;
                tracing::info!(target: "signer:certificate-watcher", endpoint = %log_endpoint, "TLS configuration reloaded successfully");
            }
            Err(e) => match failure_logged_at {
                None => {
                    failure_logged_at = Some(Instant::now());
                    tracing::error!(target: "signer:certificate-watcher", endpoint = %log_endpoint, error = %e, "Failed to reload TLS configuration, retrying");
                }
                Some(at) if at.elapsed() >= RELOAD_FAILURE_LOG_INTERVAL => {
                    failure_logged_at = Some(Instant::now());
                    tracing::warn!(target: "signer:certificate-watcher", endpoint = %log_endpoint, error = %e, "TLS configuration still fails to reload, retrying");
                }
                Some(_) => {
                    tracing::debug!(target: "signer:certificate-watcher", endpoint = %log_endpoint, error = %e, "Failed to reload TLS configuration, retrying");
                }
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Kubernetes-style Secret rotations against a real mTLS server. They rely on Unix symlinks
    /// and inotify event semantics.
    #[cfg(target_os = "linux")]
    mod rotation {
        use std::{
            fs,
            io::{self, Read, Write},
            net::{TcpListener, TcpStream},
            os::unix::fs::symlink,
        };

        use rustls::{ServerConfig, ServerConnection, StreamOwned, server::WebPkiClientVerifier};

        use super::*;

        fn fixture(name: &str) -> PathBuf {
            Path::new(env!("CARGO_MANIFEST_DIR")).join("testdata/mtls").join(name)
        }

        fn certs(name: &str) -> Vec<CertificateDer<'static>> {
            CertificateDer::pem_file_iter(fixture(name)).unwrap().collect::<Result<_, _>>().unwrap()
        }

        /// Starts an mTLS JSON-RPC server that answers every request with the name of the client
        /// certificate that authenticated the connection. Returns its port.
        fn spawn_identity_server() -> u16 {
            let provider = Arc::new(rustls::crypto::aws_lc_rs::default_provider());
            let mut roots = RootCertStore::empty();
            for cert in certs("ca.crt") {
                roots.add(cert).unwrap();
            }
            let verifier =
                WebPkiClientVerifier::builder_with_provider(Arc::new(roots), Arc::clone(&provider))
                    .build()
                    .unwrap();
            let config = Arc::new(
                ServerConfig::builder_with_provider(provider)
                    .with_safe_default_protocol_versions()
                    .unwrap()
                    .with_client_cert_verifier(verifier)
                    .with_single_cert(
                        certs("server.crt"),
                        PrivateKeyDer::from_pem_file(fixture("server.key")).unwrap(),
                    )
                    .unwrap(),
            );
            let identities = [
                ("client-a", certs("client-a.crt").remove(0)),
                ("client-b", certs("client-b.crt").remove(0)),
            ];

            let listener = TcpListener::bind("127.0.0.1:0").unwrap();
            let port = listener.local_addr().unwrap().port();
            std::thread::spawn(move || {
                for tcp in listener.incoming().flatten() {
                    // Failed handshakes and malformed requests are skipped; the test sees them as
                    // request errors.
                    let _ = serve_identity(&config, &identities, tcp);
                }
            });
            port
        }

        fn serve_identity(
            config: &Arc<ServerConfig>,
            identities: &[(&str, CertificateDer<'static>)],
            tcp: TcpStream,
        ) -> io::Result<()> {
            tcp.set_read_timeout(Some(Duration::from_secs(5)))?;
            let conn = ServerConnection::new(Arc::clone(config)).map_err(io::Error::other)?;
            let mut stream = StreamOwned::new(conn, tcp);

            let mut buf = Vec::new();
            let mut chunk = [0u8; 4096];
            let mut read_more = |stream: &mut StreamOwned<ServerConnection, TcpStream>,
                                 buf: &mut Vec<u8>| {
                let n = stream.read(&mut chunk)?;
                if n == 0 {
                    return Err(io::Error::from(io::ErrorKind::UnexpectedEof));
                }
                buf.extend_from_slice(&chunk[..n]);
                Ok(())
            };
            let body_start = loop {
                if let Some(i) = buf.windows(4).position(|w| w == b"\r\n\r\n") {
                    break i + 4;
                }
                read_more(&mut stream, &mut buf)?;
            };
            let headers = String::from_utf8_lossy(&buf[..body_start]).to_ascii_lowercase();
            let content_length: usize = headers
                .lines()
                .find_map(|line| line.strip_prefix("content-length:"))
                .and_then(|value| value.trim().parse().ok())
                .unwrap_or(0);
            while buf.len() < body_start + content_length {
                read_more(&mut stream, &mut buf)?;
            }
            let request: serde_json::Value =
                serde_json::from_slice(&buf[body_start..body_start + content_length])?;

            let peer = stream.conn.peer_certificates().and_then(|certs| certs.first());
            let identity = identities
                .iter()
                .find(|(_, cert)| Some(cert) == peer)
                .map_or("unknown", |(name, _)| *name);
            let body =
                serde_json::json!({ "jsonrpc": "2.0", "id": request["id"], "result": identity })
                    .to_string();
            write!(
                stream,
                "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
                body.len()
            )?;
            stream.conn.send_close_notify();
            stream.flush()
        }

        /// Publishes `client`'s certificate into `mount` the way kubelet updates a Secret volume:
        /// write a new `..<generation>` directory, atomically swap the `..data` symlink to it, then
        /// delete the `previous` generation. `key` replaces the client key's contents when set.
        fn publish(
            mount: &Path,
            generation: &str,
            client: &str,
            key: Option<&str>,
            previous: Option<&str>,
        ) {
            let dir = mount.join(format!("..{generation}"));
            fs::create_dir(&dir).unwrap();
            fs::copy(fixture("ca.crt"), dir.join("ca.crt")).unwrap();
            fs::copy(fixture(&format!("{client}.crt")), dir.join("tls.crt")).unwrap();
            match key {
                Some(key) => fs::write(dir.join("tls.key"), key).unwrap(),
                None => {
                    fs::copy(fixture(&format!("{client}.key")), dir.join("tls.key")).unwrap();
                }
            }

            let tmp = mount.join("..data_tmp");
            symlink(format!("..{generation}"), &tmp).unwrap();
            fs::rename(&tmp, mount.join(KUBERNETES_DATA_DIR)).unwrap();

            for name in ["ca.crt", "tls.crt", "tls.key"] {
                let link = mount.join(name);
                if fs::symlink_metadata(&link).is_err() {
                    symlink(Path::new(KUBERNETES_DATA_DIR).join(name), &link).unwrap();
                }
            }

            if let Some(previous) = previous {
                fs::remove_dir_all(mount.join(format!("..{previous}"))).unwrap();
            }
        }

        /// Returns the identity the server saw, giving up after five seconds so that a stuck
        /// request cannot outlast the caller's deadline.
        async fn identity(client: &ReloadingRpcClient) -> Result<String, String> {
            let request = client.client().request::<_, String>("health_status", ());
            tokio::time::timeout(Duration::from_secs(5), request)
                .await
                .map_err(|_| "request timed out".to_string())?
                .map_err(|e| e.to_string())
        }

        async fn wait_for_identity(client: &ReloadingRpcClient, expected: &str) {
            let deadline = Instant::now() + Duration::from_secs(15);
            loop {
                let last = identity(client).await;
                if last.as_deref() == Ok(expected) {
                    return;
                }
                assert!(
                    Instant::now() < deadline,
                    "timed out waiting for the server to see {expected}; last result: {last:?}"
                );
                tokio::time::sleep(Duration::from_millis(200)).await;
            }
        }

        #[tokio::test(flavor = "multi_thread")]
        async fn reloads_client_certificate_across_repeated_secret_rotations() {
            let port = spawn_identity_server();
            let mount = tempfile::tempdir().unwrap();
            let mount = mount.path();
            publish(mount, "gen1", "client-a", None, None);

            let client = ReloadingRpcClient::new(
                format!("https://127.0.0.1:{port}").parse().unwrap(),
                TlsPaths {
                    ca_cert: Some(mount.join("ca.crt")),
                    client_cert: Some(ClientCert {
                        cert: mount.join("tls.crt"),
                        key: mount.join("tls.key"),
                    }),
                },
                HeaderMap::new(),
                None,
            )
            .unwrap();
            assert!(client.is_watching());
            wait_for_identity(&client, "client-a").await;

            publish(mount, "gen2", "client-b", None, Some("gen1"));
            wait_for_identity(&client, "client-b").await;

            // A watch on the file paths is dropped once kubelet deletes the first generation, so
            // this second rotation is the one such a watcher misses.
            publish(mount, "gen3", "client-a", None, Some("gen2"));
            wait_for_identity(&client, "client-a").await;

            // A rotation that cannot be loaded keeps the current client.
            publish(mount, "gen4", "client-b", Some("not a key"), Some("gen3"));
            tokio::time::sleep(RELOAD_DEBOUNCE * 2).await;
            assert_eq!(identity(&client).await.as_deref(), Ok("client-a"));

            // Repairing the key inside the generation directory produces no event in the watched
            // directory, so only the retry of the failed reload can pick it up.
            fs::copy(fixture("client-b.key"), mount.join("..gen4/tls.key")).unwrap();
            wait_for_identity(&client, "client-b").await;
        }
    }

    #[test]
    fn reload_trigger_matches_configured_names_and_ignores_reads() {
        use notify::event::{CreateKind, Flag, ModifyKind};

        let tls = TlsPaths {
            ca_cert: Some("/mnt/ca.crt".into()),
            client_cert: Some(ClientCert {
                cert: "/mnt/tls.crt".into(),
                key: "/mnt/tls.key".into(),
            }),
        };
        let names = tls.watched_names();
        let event = |kind, path: &str| Event::new(kind).add_path(path.into());

        for (kind, path, expected) in [
            (EventKind::Create(CreateKind::Any), "/mnt/..data", true),
            (EventKind::Modify(ModifyKind::Any), "/mnt/tls.key", true),
            (EventKind::Access(AccessKind::Close(AccessMode::Write)), "/mnt/tls.crt", true),
            (EventKind::Access(AccessKind::Open(AccessMode::Any)), "/mnt/tls.crt", false),
            (EventKind::Access(AccessKind::Close(AccessMode::Read)), "/mnt/ca.crt", false),
            (EventKind::Modify(ModifyKind::Any), "/mnt/other.crt", false),
        ] {
            assert_eq!(is_reload_trigger(&event(kind, path), &names), expected, "{kind:?} {path}");
        }
        assert!(is_reload_trigger(&Event::new(EventKind::Other).set_flag(Flag::Rescan), &names));
    }

    #[test]
    fn watch_dirs_uses_current_dir_for_bare_file_names() {
        let tls = TlsPaths {
            ca_cert: Some("ca.crt".into()),
            client_cert: Some(ClientCert { cert: "/a/tls.crt".into(), key: "/b/tls.key".into() }),
        };
        assert_eq!(
            tls.watch_dirs(),
            BTreeSet::from([PathBuf::from("."), PathBuf::from("/a"), PathBuf::from("/b")])
        );
    }

    #[test]
    fn redacted_url_strips_userinfo() {
        let url: Url = "https://user:secret@rpc.example.com/key".parse().unwrap();
        assert_eq!(redacted_url(&url), "https://rpc.example.com/key");
        let plain: Url = "http://127.0.0.1:8545/".parse().unwrap();
        assert_eq!(redacted_url(&plain), "http://127.0.0.1:8545/");
        // file:// URLs cannot carry userinfo: set_username/set_password return
        // Err, which redacted_url ignores. This pins that choice (panicking on
        // Err would break file:// prestate URLs) and that the URL renders
        // unchanged.
        let file: Url = "file:///data/prestates".parse().unwrap();
        assert_eq!(redacted_url(&file), "file:///data/prestates");
    }
}
