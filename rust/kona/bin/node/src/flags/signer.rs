//! Block signer CLI Flags

use alloy_primitives::{Address, B256};
use alloy_signer::k256::ecdsa;
use alloy_signer_local::PrivateKeySigner;
use anyhow::Context;
use clap::{Args, error::ErrorKind};
use kona_sources::{BlockSigner, ClientCert, RemoteSigner};
use reqwest::header::{HeaderName, HeaderValue};
use std::{path::PathBuf, str::FromStr};
use url::Url;

/// The flags that choose the block signer, which signs a sequencer's blocks for gossip: a local
/// key read from a file, or a remote op-signer.
///
/// A node signs blocks only when it can sequence and gossips them: with `--sequencer.enabled` and
/// `--p2p.enabled`. It then needs a signer, and these flags must choose exactly one. Otherwise they
/// are ignored: clap still parses each value, but how they combine is not checked, and the key
/// file is not read.
#[derive(Debug, Clone, Args, PartialEq, Eq)]
#[command(next_help_heading = "Signer")]
pub struct SignerArgs {
    /// Path to a file holding the hex-encoded private key that signs the sequencer's blocks. The
    /// file must exist. Give this or `--signer.url`. Used and required only when
    /// `--sequencer.enabled` and `--p2p.enabled`.
    #[arg(long = "signer.key-path", value_name = "PATH", env = "KONA_NODE_SIGNER_KEY_PATH")]
    pub key_path: Option<PathBuf>,
    /// A remote op-signer that signs the sequencer's blocks.
    #[command(flatten)]
    pub remote: RemoteSignerArgs,
}

/// The flags that configure a remote op-signer.
///
/// Each requires the URL, which requires the address: so a remote signer has the URL and the
/// address, and a node without one gives none of these flags.
#[derive(Debug, Clone, Args, PartialEq, Eq)]
pub struct RemoteSignerArgs {
    /// The URL of a remote op-signer that signs the sequencer's blocks. Give this or
    /// `--signer.key-path`. Used and required only when `--sequencer.enabled` and `--p2p.enabled`.
    #[arg(long = "signer.url", env = "KONA_NODE_SIGNER_URL")]
    pub url: Option<Url>,
    /// The address of the remote op-signer's key. Requires `--signer.url`.
    #[arg(long = "signer.address", env = "KONA_NODE_SIGNER_ADDRESS")]
    pub address: Option<Address>,
    /// A header to send to the remote op-signer, as `name=value`. Pass the flag once per header;
    /// the environment variable sets a single header, whose value may contain commas. Requires
    /// `--signer.url`.
    #[arg(long = "signer.header", value_name = "NAME=VALUE", env = "KONA_NODE_SIGNER_HEADER")]
    pub headers: Vec<SignerHeader>,
    /// Path to the CA certificates that verify the remote op-signer. Requires `--signer.url`.
    #[arg(long = "signer.tls.ca-path", value_name = "PATH", env = "KONA_NODE_SIGNER_TLS_CA_PATH")]
    pub ca_cert: Option<PathBuf>,
    /// Path to the client certificate presented to the remote op-signer. Requires `--signer.url`
    /// and `--signer.tls.key-path`.
    #[arg(
        long = "signer.tls.cert-path",
        value_name = "PATH",
        env = "KONA_NODE_SIGNER_TLS_CERT_PATH"
    )]
    pub client_cert: Option<PathBuf>,
    /// Path to the private key of the client certificate. Requires `--signer.url` and
    /// `--signer.tls.cert-path`.
    #[arg(
        long = "signer.tls.key-path",
        value_name = "PATH",
        env = "KONA_NODE_SIGNER_TLS_KEY_PATH"
    )]
    pub client_key: Option<PathBuf>,
}

/// A header sent to the remote op-signer, parsed from `name=value`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SignerHeader {
    /// The header's name.
    pub name: HeaderName,
    /// The header's value.
    pub value: HeaderValue,
}

impl FromStr for SignerHeader {
    type Err = anyhow::Error;

    fn from_str(header: &str) -> Result<Self, Self::Err> {
        let (name, value) = header.split_once('=').context("expected name=value")?;
        Ok(Self { name: name.parse()?, value: value.parse()? })
    }
}

/// An error in the block signer's flags, or reading the signing key they name.
#[derive(Debug, thiserror::Error)]
pub enum SignerError {
    /// No signer was given.
    #[error(
        "--sequencer.enabled with --p2p.enabled requires a block signer: --signer.key-path, or \
         --signer.url with --signer.address"
    )]
    NoSigner,
    /// Both a local key and a remote signer were given.
    #[error("give --signer.key-path or --signer.url, not both")]
    TwoSigners,
    /// A remote signer was given without its URL, or without its address.
    #[error("--signer.url and --signer.address must be given together")]
    IncompleteRemoteSigner,
    /// A remote signer option was given without a remote signer.
    #[error("{0} requires --signer.url")]
    NoRemoteSigner(&'static str),
    /// A client certificate was given without its key, or a key without its certificate.
    #[error("--signer.tls.cert-path and --signer.tls.key-path must be given together")]
    IncompleteClientCert,
    /// The key file could not be read.
    #[error("failed to read the signer key file {0}")]
    Read(PathBuf, #[source] std::io::Error),
    /// The key file does not hold a hex-encoded 32-byte key.
    #[error("the signer key file {0} does not hold a hex-encoded 32-byte key")]
    Decode(PathBuf, #[source] alloy_primitives::hex::FromHexError),
    /// The key is not a valid secp256k1 private key.
    #[error("the signer key in {0} is not a valid secp256k1 private key")]
    Invalid(PathBuf, #[source] ecdsa::Error),
}

impl SignerError {
    /// The kind of clap error an error in how the flags combine is, or `None` for an error in the
    /// key file they name.
    pub const fn kind(&self) -> Option<ErrorKind> {
        match self {
            Self::TwoSigners => Some(ErrorKind::ArgumentConflict),
            Self::NoSigner |
            Self::IncompleteRemoteSigner |
            Self::NoRemoteSigner(_) |
            Self::IncompleteClientCert => Some(ErrorKind::MissingRequiredArgument),
            Self::Read(..) | Self::Decode(..) | Self::Invalid(..) => None,
        }
    }
}

/// The signer the flags choose, before its key file is read.
enum Choice {
    Local(PathBuf),
    Remote(Box<RemoteSigner>),
}

impl SignerArgs {
    /// Checks that the flags choose exactly one complete signer, without reading its key file.
    pub fn check(&self) -> Result<(), SignerError> {
        self.clone().choose().map(drop)
    }

    /// Returns the block signer the flags choose, reading its key file.
    pub fn config(self) -> Result<BlockSigner, SignerError> {
        match self.choose()? {
            Choice::Local(path) => {
                let contents = std::fs::read_to_string(&path)
                    .map_err(|e| SignerError::Read(path.clone(), e))?;
                let key = B256::from_str(contents.trim())
                    .map_err(|e| SignerError::Decode(path.clone(), e))?;
                let signer = PrivateKeySigner::from_bytes(&key)
                    .map_err(|e| SignerError::Invalid(path, e))?;
                Ok(signer.into())
            }
            Choice::Remote(remote) => Ok((*remote).into()),
        }
    }

    fn choose(self) -> Result<Choice, SignerError> {
        // Check the remote signer's flags first, so that none is ignored beside a local key.
        match (self.key_path, self.remote.config()?) {
            (Some(_), Some(_)) => Err(SignerError::TwoSigners),
            (Some(path), None) => Ok(Choice::Local(path)),
            (None, Some(remote)) => Ok(Choice::Remote(Box::new(remote))),
            (None, None) => Err(SignerError::NoSigner),
        }
    }
}

impl RemoteSignerArgs {
    /// Returns the remote signer the flags configure, or `None` if they configure none.
    fn config(self) -> Result<Option<RemoteSigner>, SignerError> {
        let (endpoint, address) = match (self.url, self.address) {
            (Some(url), Some(address)) => (url, address),
            (None, None) => {
                let options = [
                    ("--signer.header", !self.headers.is_empty()),
                    ("--signer.tls.ca-path", self.ca_cert.is_some()),
                    ("--signer.tls.cert-path", self.client_cert.is_some()),
                    ("--signer.tls.key-path", self.client_key.is_some()),
                ];
                return match options.into_iter().find(|(_, given)| *given) {
                    Some((flag, _)) => Err(SignerError::NoRemoteSigner(flag)),
                    None => Ok(None),
                };
            }
            _ => return Err(SignerError::IncompleteRemoteSigner),
        };
        let client_cert = match (self.client_cert, self.client_key) {
            (Some(cert), Some(key)) => Some(ClientCert { cert, key }),
            (None, None) => None,
            _ => return Err(SignerError::IncompleteClientCert),
        };
        Ok(Some(RemoteSigner {
            address,
            endpoint,
            ca_cert: self.ca_cert,
            client_cert,
            headers: self.headers.into_iter().map(|header| (header.name, header.value)).collect(),
        }))
    }
}

/// Signing keys for tests.
#[cfg(test)]
pub(crate) mod test_utils {
    use std::io::Write;

    /// A valid hex-encoded signing key.
    pub(crate) const KEY: &str =
        "0x1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be";

    /// A file holding `contents`, for `--signer.key-path`. It is deleted on drop.
    pub(crate) fn key_file(contents: &str) -> tempfile::NamedTempFile {
        let mut file = tempfile::NamedTempFile::new().unwrap();
        write!(file, "{contents}").unwrap();
        file
    }
}

#[cfg(test)]
mod tests {
    use super::{
        test_utils::{KEY, key_file},
        *,
    };
    use clap::{Parser, error::ErrorKind};

    #[derive(Parser, Debug)]
    struct TestCli {
        #[command(flatten)]
        signer: SignerArgs,
    }

    /// Parses the signer flags `args`.
    fn parse(args: &[&str]) -> Result<SignerArgs, clap::Error> {
        TestCli::try_parse_from(std::iter::once("node").chain(args.iter().copied()))
            .map(|cli| cli.signer)
    }

    const ADDRESS: &str = "0x0000000000000000000000000000000000000001";

    #[test]
    fn reads_the_local_key_from_its_file() {
        let file = key_file(&format!("{KEY}\n"));
        let args = parse(&["--signer.key-path", file.path().to_str().unwrap()]).unwrap();

        let BlockSigner::Local(signer) = args.config().unwrap() else {
            panic!("expected a local signer")
        };
        assert_eq!(signer.to_bytes(), B256::from_str(KEY).unwrap());
    }

    /// The key file is read when the signer is configured, not when the flags are checked.
    #[test]
    fn rejects_a_missing_or_malformed_key_file() {
        let args = parse(&["--signer.key-path", "/nonexistent/sequencer.key"]).unwrap();
        args.check().unwrap();
        assert!(matches!(args.config(), Err(SignerError::Read(..))));

        let file = key_file("not hex");
        let args = parse(&["--signer.key-path", file.path().to_str().unwrap()]).unwrap();
        assert!(matches!(args.config(), Err(SignerError::Decode(..))));

        let file = key_file(&B256::ZERO.to_string());
        let args = parse(&["--signer.key-path", file.path().to_str().unwrap()]).unwrap();
        assert!(matches!(args.config(), Err(SignerError::Invalid(..))));
    }

    #[test]
    fn configures_a_remote_signer() {
        let args = parse(&[
            "--signer.url",
            "http://127.0.0.1:8080",
            "--signer.address",
            ADDRESS,
            "--signer.header",
            "x-api-key=secret,with,commas",
            "--signer.tls.cert-path",
            "/cert.pem",
            "--signer.tls.key-path",
            "/key.pem",
        ])
        .unwrap();

        let BlockSigner::Remote(signer) = args.config().unwrap() else {
            panic!("expected a remote signer")
        };
        assert_eq!(signer.endpoint.as_str(), "http://127.0.0.1:8080/");
        assert_eq!(signer.address, Address::with_last_byte(1));
        assert_eq!(signer.headers["x-api-key"], "secret,with,commas");
        let cert = signer.client_cert.unwrap();
        assert_eq!((cert.cert, cert.key), (PathBuf::from("/cert.pem"), PathBuf::from("/key.pem")));
    }

    #[test]
    fn rejects_inconsistent_signer_flags() {
        let remote = ["--signer.url", "http://a", "--signer.address", ADDRESS];
        let with_remote = |extra: &[&'static str]| [&remote[..], extra].concat();
        let cases: [(Vec<&str>, SignerError); 10] = [
            // no signer
            (vec![], SignerError::NoSigner),
            // two signers
            (with_remote(&["--signer.key-path", "/key"]), SignerError::TwoSigners),
            // remote signer options beside a local key
            (
                vec!["--signer.key-path", "/key", "--signer.address", ADDRESS],
                SignerError::IncompleteRemoteSigner,
            ),
            (
                vec!["--signer.key-path", "/key", "--signer.header", "a=b"],
                SignerError::NoRemoteSigner("--signer.header"),
            ),
            // a remote signer without its address, or its URL
            (vec!["--signer.url", "http://a"], SignerError::IncompleteRemoteSigner),
            (vec!["--signer.address", ADDRESS], SignerError::IncompleteRemoteSigner),
            // remote signer options without a remote signer
            (vec!["--signer.header", "a=b"], SignerError::NoRemoteSigner("--signer.header")),
            (
                vec!["--signer.tls.ca-path", "/ca.pem"],
                SignerError::NoRemoteSigner("--signer.tls.ca-path"),
            ),
            // a client certificate without its key, or a key without its certificate
            (with_remote(&["--signer.tls.cert-path", "/c"]), SignerError::IncompleteClientCert),
            (with_remote(&["--signer.tls.key-path", "/k"]), SignerError::IncompleteClientCert),
        ];
        for (raw, expected) in cases {
            let args = parse(&raw).unwrap();
            let error = args.check().unwrap_err();
            assert!(error.kind().is_some(), "{raw:?}");
            assert_eq!(error.to_string(), expected.to_string(), "{raw:?}");
            assert_eq!(args.config().unwrap_err().to_string(), expected.to_string(), "{raw:?}");
        }
    }

    #[test]
    fn rejects_a_malformed_header() {
        let error = parse(&["--signer.header", "no-value"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::ValueValidation);
    }
}
