#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod engine;
pub use engine::{EngineClient, EngineClientError};

mod metrics;
pub use metrics::Metrics;

mod signer;
pub use signer::{
    BlockSigner, BlockSignerError, BlockSignerHandler, BlockSignerStartError, RemoteSigner,
    RemoteSignerError, RemoteSignerHandler, RemoteSignerStartError,
};

mod tls;
pub use tls::{
    CertificateError, ClientCert, ReloadingRpcClient, ReloadingRpcClientError, TlsPaths,
    redacted_url,
};

#[cfg(any(test, feature = "test-utils"))]
pub mod test_utils;
