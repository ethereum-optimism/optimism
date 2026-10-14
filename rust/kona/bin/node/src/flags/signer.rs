//! Block signer CLI Flags

use alloy_primitives::{Address, B256};
use alloy_signer::k256::ecdsa;
use alloy_signer_local::PrivateKeySigner;
use anyhow::Context;
use clap::Args;
use kona_sources::{BlockSigner, ClientCert, RemoteSigner};
use reqwest::header::{HeaderName, HeaderValue};
use std::{path::PathBuf, str::FromStr};
use url::Url;

/// The flags that choose the sequencer's block signer: a local key read from a file, or a remote
/// op-signer. They apply only with `--sequencer.enabled`, and are checked only then, so that a
/// follower ignores them whatever they are.
#[derive(Debug, Clone, Args, PartialEq, Eq)]
pub struct SignerArgs {
    /// Path to a file holding the hex-encoded private key that signs the sequencer's blocks. The
    /// file must exist.
    #[arg(long = "sequencer.signer.key-path", env = "KONA_NODE_SEQUENCER_SIGNER_KEY_PATH")]
    pub key_path: Option<PathBuf>,
    /// A remote op-signer that signs the sequencer's blocks.
    #[command(flatten)]
    pub remote: RemoteSignerArgs,
}

/// The flags that configure a remote op-signer.
///
/// They are optional, since a node need not have a remote signer, but each requires the
/// URL, which requires the address: so a sequencer either gives the URL and the address, or none
/// of these flags.
#[derive(Debug, Clone, Args, PartialEq, Eq)]
pub struct RemoteSignerArgs {
    /// The URL of the remote op-signer.
    #[arg(long = "sequencer.signer.url", env = "KONA_NODE_SEQUENCER_SIGNER_URL")]
    pub url: Option<Url>,
    /// The address of the remote op-signer's key.
    #[arg(long = "sequencer.signer.address", env = "KONA_NODE_SEQUENCER_SIGNER_ADDRESS")]
    pub address: Option<Address>,
    /// A header to send to the remote op-signer, as `name=value`. Pass the flag once per header;
    /// the environment variable sets a single header, whose value may contain commas.
    #[arg(long = "sequencer.signer.header", env = "KONA_NODE_SEQUENCER_SIGNER_HEADER")]
    pub headers: Vec<SignerHeader>,
    /// Path to the CA certificates that verify the remote op-signer.
    #[arg(long = "sequencer.signer.tls.ca-path", env = "KONA_NODE_SEQUENCER_SIGNER_TLS_CA_PATH")]
    pub ca_cert: Option<PathBuf>,
    /// Path to the client certificate presented to the remote op-signer.
    #[arg(
        long = "sequencer.signer.tls.cert-path",
        env = "KONA_NODE_SEQUENCER_SIGNER_TLS_CERT_PATH"
    )]
    pub client_cert: Option<PathBuf>,
    /// Path to the private key of the client certificate.
    #[arg(long = "sequencer.signer.tls.key-path", env = "KONA_NODE_SEQUENCER_SIGNER_TLS_KEY_PATH")]
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
    /// Both a local key and a remote signer were given.
    #[error("give --sequencer.signer.key-path or --sequencer.signer.url, not both")]
    TwoSigners,
    /// A remote signer was given without its URL, or without its address.
    #[error("--sequencer.signer.url and --sequencer.signer.address must be given together")]
    IncompleteRemoteSigner,
    /// A remote signer option was given without a remote signer.
    #[error("{0} requires --sequencer.signer.url")]
    NoRemoteSigner(&'static str),
    /// A client certificate was given without its key, or a key without its certificate.
    #[error(
        "--sequencer.signer.tls.cert-path and --sequencer.signer.tls.key-path must be given \
         together"
    )]
    IncompleteClientCert,
    /// The key file could not be read.
    #[error("failed to read the sequencer signer key file {0}")]
    Read(PathBuf, #[source] std::io::Error),
    /// The key file does not hold a hex-encoded 32-byte key.
    #[error("the sequencer signer key file {0} does not hold a hex-encoded 32-byte key")]
    Decode(PathBuf, #[source] alloy_primitives::hex::FromHexError),
    /// The key is not a valid secp256k1 private key.
    #[error("the sequencer signer key in {0} is not a valid secp256k1 private key")]
    Invalid(PathBuf, #[source] ecdsa::Error),
}

impl SignerArgs {
    /// Returns the block signer the flags choose, or `None` if they choose none.
    pub fn config(self) -> Result<Option<BlockSigner>, SignerError> {
        // Check the remote signer's flags first, so that none is ignored beside a local key.
        let remote = self.remote.config()?;
        if let Some(path) = self.key_path {
            if remote.is_some() {
                return Err(SignerError::TwoSigners);
            }
            let contents =
                std::fs::read_to_string(&path).map_err(|e| SignerError::Read(path.clone(), e))?;
            let key = B256::from_str(contents.trim())
                .map_err(|e| SignerError::Decode(path.clone(), e))?;
            let signer =
                PrivateKeySigner::from_bytes(&key).map_err(|e| SignerError::Invalid(path, e))?;
            return Ok(Some(signer.into()));
        }
        Ok(remote.map(Into::into))
    }
}

impl RemoteSignerArgs {
    /// Returns the remote signer the flags configure, or `None` if they configure none.
    fn config(self) -> Result<Option<RemoteSigner>, SignerError> {
        let (endpoint, address) = match (self.url, self.address) {
            (Some(url), Some(address)) => (url, address),
            (None, None) => {
                let options = [
                    ("--sequencer.signer.header", !self.headers.is_empty()),
                    ("--sequencer.signer.tls.ca-path", self.ca_cert.is_some()),
                    ("--sequencer.signer.tls.cert-path", self.client_cert.is_some()),
                    ("--sequencer.signer.tls.key-path", self.client_key.is_some()),
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

    /// A file holding `contents`, for `--sequencer.signer.key-path`. It is deleted on drop.
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
    use crate::flags::SequencerArgs;
    use clap::{Parser, error::ErrorKind};

    /// Parses the signer flags `args` of a node run with `--sequencer.enabled`.
    fn parse(args: &[&str]) -> Result<SignerArgs, clap::Error> {
        SequencerArgs::try_parse_from(
            ["node", "--sequencer.enabled"].into_iter().chain(args.iter().copied()),
        )
        .map(|sequencer| sequencer.signer)
    }

    const ADDRESS: &str = "0x0000000000000000000000000000000000000001";

    #[test]
    fn no_flags_choose_no_signer() {
        let args = SequencerArgs::try_parse_from(["node"]).unwrap();
        assert!(args.signer.config().unwrap().is_none());
    }

    #[test]
    fn reads_the_local_key_from_its_file() {
        let file = key_file(&format!("{KEY}\n"));
        let args = parse(&["--sequencer.signer.key-path", file.path().to_str().unwrap()]).unwrap();

        let Some(BlockSigner::Local(signer)) = args.config().unwrap() else {
            panic!("expected a local signer")
        };
        assert_eq!(signer.to_bytes(), B256::from_str(KEY).unwrap());
    }

    #[test]
    fn rejects_a_missing_or_malformed_key_file() {
        let args = parse(&["--sequencer.signer.key-path", "/nonexistent/sequencer.key"]).unwrap();
        assert!(matches!(args.config(), Err(SignerError::Read(..))));

        let file = key_file("not hex");
        let args = parse(&["--sequencer.signer.key-path", file.path().to_str().unwrap()]).unwrap();
        assert!(matches!(args.config(), Err(SignerError::Decode(..))));

        let file = key_file(&B256::ZERO.to_string());
        let args = parse(&["--sequencer.signer.key-path", file.path().to_str().unwrap()]).unwrap();
        assert!(matches!(args.config(), Err(SignerError::Invalid(..))));
    }

    #[test]
    fn configures_a_remote_signer() {
        let args = parse(&[
            "--sequencer.signer.url",
            "http://127.0.0.1:8080",
            "--sequencer.signer.address",
            ADDRESS,
            "--sequencer.signer.header",
            "x-api-key=secret,with,commas",
            "--sequencer.signer.tls.cert-path",
            "/cert.pem",
            "--sequencer.signer.tls.key-path",
            "/key.pem",
        ])
        .unwrap();

        let Some(BlockSigner::Remote(signer)) = args.config().unwrap() else {
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
        let remote = ["--sequencer.signer.url", "http://a", "--sequencer.signer.address", ADDRESS];
        let with_remote = |extra: &[&'static str]| [&remote[..], extra].concat();
        let cases: [(Vec<&str>, SignerError); 9] = [
            // two signers
            (with_remote(&["--sequencer.signer.key-path", "/key"]), SignerError::TwoSigners),
            // remote signer options beside a local key
            (
                vec!["--sequencer.signer.key-path", "/key", "--sequencer.signer.address", ADDRESS],
                SignerError::IncompleteRemoteSigner,
            ),
            (
                vec!["--sequencer.signer.key-path", "/key", "--sequencer.signer.header", "a=b"],
                SignerError::NoRemoteSigner("--sequencer.signer.header"),
            ),
            // a remote signer without its address, or its URL
            (vec!["--sequencer.signer.url", "http://a"], SignerError::IncompleteRemoteSigner),
            (vec!["--sequencer.signer.address", ADDRESS], SignerError::IncompleteRemoteSigner),
            // remote signer options without a remote signer
            (
                vec!["--sequencer.signer.header", "a=b"],
                SignerError::NoRemoteSigner("--sequencer.signer.header"),
            ),
            (
                vec!["--sequencer.signer.tls.ca-path", "/ca.pem"],
                SignerError::NoRemoteSigner("--sequencer.signer.tls.ca-path"),
            ),
            // a client certificate without its key, or a key without its certificate
            (
                with_remote(&["--sequencer.signer.tls.cert-path", "/c"]),
                SignerError::IncompleteClientCert,
            ),
            (
                with_remote(&["--sequencer.signer.tls.key-path", "/k"]),
                SignerError::IncompleteClientCert,
            ),
        ];
        for (args, expected) in cases {
            let error = parse(&args).unwrap().config().unwrap_err();
            assert_eq!(error.to_string(), expected.to_string(), "{args:?}");
        }
    }

    #[test]
    fn rejects_a_malformed_header() {
        let error = parse(&["--sequencer.signer.header", "no-value"]).unwrap_err();
        assert_eq!(error.kind(), ErrorKind::ValueValidation);
    }
}
