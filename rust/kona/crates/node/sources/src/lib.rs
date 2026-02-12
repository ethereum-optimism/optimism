#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod signer;
pub use signer::{
    BlockSigner, BlockSignerError, BlockSignerHandler, BlockSignerStartError, CertificateError,
    ClientCert, RemoteSigner, RemoteSignerError, RemoteSignerHandler, RemoteSignerStartError,
};
