use std::path::PathBuf;

use alloy_primitives::{Address, B256};
use alloy_signer::{Signer, k256::ecdsa};
use alloy_signer_local::PrivateKeySigner;
use clap::{Parser, arg};
use kona_cli::SecretKeyLoader;
use kona_sources::{BlockSigner, ClientCert, RemoteSigner};
use reqwest::header::{HeaderMap, HeaderName, HeaderValue};
use std::str::FromStr;
use url::Url;

use crate::flags::GlobalArgs;

/// Signer CLI Flags
#[derive(Debug, Clone, Parser, Default, PartialEq, Eq)]
pub struct SignerArgs {
    /// An optional flag to specify a local private key for the sequencer to sign unsafe blocks.
    #[arg(
        long = "p2p.sequencer.key",
        env = "KONA_NODE_P2P_SEQUENCER_KEY",
        conflicts_with = "endpoint"
    )]
    pub sequencer_key: Option<B256>,
    /// An optional path to a file containing the sequencer private key.
    /// This is mutually exclusive with `p2p.sequencer.key`.
    #[arg(
        long = "p2p.sequencer.key.path",
        env = "KONA_NODE_P2P_SEQUENCER_KEY_PATH",
        conflicts_with = "sequencer_key"
    )]
    pub sequencer_key_path: Option<PathBuf>,
    /// The URL of the remote signer endpoint. If not provided, remote signer will be disabled.
    /// This is mutually exclusive with `p2p.sequencer.key`.
    /// This is required if any of the other signer flags are provided.
    #[arg(
        long = "p2p.signer.endpoint",
        env = "KONA_NODE_P2P_SIGNER_ENDPOINT",
        requires = "address"
    )]
    pub endpoint: Option<Url>,
    /// The address to sign transactions for. Required if `signer.endpoint` is provided.
    #[arg(
        long = "p2p.signer.address",
        env = "KONA_NODE_P2P_SIGNER_ADDRESS",
        requires = "endpoint"
    )]
    pub address: Option<Address>,
    /// Headers to pass to the remote signer. Format `key=value`. Value can contain any character
    /// allowed in a HTTP header. When using env vars, split with commas. When using flags one
    /// key value pair per flag.
    #[arg(long = "p2p.signer.header", env = "KONA_NODE_P2P_SIGNER_HEADER", requires = "endpoint")]
    pub header: Vec<String>,
    /// An optional path to CA certificates to be used for the remote signer.
    #[arg(long = "p2p.signer.tls.ca", env = "KONA_NODE_P2P_SIGNER_TLS_CA", requires = "endpoint")]
    pub ca_cert: Option<PathBuf>,
    /// An optional path to the client certificate for the remote signer. If specified,
    /// `signer.tls.key` must also be specified.
    #[arg(
        long = "p2p.signer.tls.cert",
        env = "KONA_NODE_P2P_SIGNER_TLS_CERT",
        requires = "key",
        requires = "endpoint"
    )]
    pub cert: Option<PathBuf>,
    /// An optional path to the client key for the remote signer. If specified,
    /// `signer.tls.cert` must also be specified.
    #[arg(
        long = "p2p.signer.tls.key",
        env = "KONA_NODE_P2P_SIGNER_TLS_KEY",
        requires = "cert",
        requires = "endpoint"
    )]
    pub key: Option<PathBuf>,
}

/// Errors that can occur when parsing the signer arguments.
#[derive(Debug, thiserror::Error)]
pub enum SignerArgsParseError {
    /// The local sequencer key and remote signer cannot be specified at the same time.
    #[error("A local sequencer key and a remote signer cannot be specified at the same time.")]
    LocalAndRemoteSigner,
    /// Both sequencer key and sequencer key path cannot be specified at the same time.
    #[error(
        "Both sequencer key and sequencer key path cannot be specified at the same time. Use either --p2p.sequencer.key or --p2p.sequencer.key.path."
    )]
    ConflictingSequencerKeyInputs,
    /// The sequencer key is invalid.
    #[error("The sequencer key is invalid.")]
    SequencerKeyInvalid(#[from] ecdsa::Error),
    /// Failed to load sequencer key from file.
    #[error("Failed to load sequencer key from file")]
    SequencerKeyFileError(#[from] kona_cli::KeypairError),
    /// The address is required if `signer.endpoint` is provided.
    #[error("The address is required if `signer.endpoint` is provided.")]
    AddressRequired,
    /// The header is invalid.
    #[error("The header is invalid.")]
    InvalidHeader,
    /// The private key field is required if `signer.tls.cert` is provided.
    #[error("The private key field is required if `signer.tls.cert` is provided.")]
    KeyRequired,
    /// The header name is invalid.
    #[error("The header name is invalid.")]
    InvalidHeaderName(#[from] reqwest::header::InvalidHeaderName),
    /// The header value is invalid.
    #[error("The header value is invalid.")]
    InvalidHeaderValue(#[from] reqwest::header::InvalidHeaderValue),
}

impl SignerArgs {
    /// Creates a [`BlockSigner`] from the [`SignerArgs`].
    pub fn config(self, args: &GlobalArgs) -> Result<Option<BlockSigner>, SignerArgsParseError> {
        // First, resolve the sequencer key from either raw input or file
        let sequencer_key = self.resolve_sequencer_key()?;

        // The sequencer signer obtained from the CLI arguments.
        let gossip_signer: Option<BlockSigner> = match (sequencer_key, self.config_remote()?) {
            (Some(_), Some(_)) => return Err(SignerArgsParseError::LocalAndRemoteSigner),
            (Some(key), None) => {
                let signer: BlockSigner = PrivateKeySigner::from_bytes(&key)?
                    .with_chain_id(Some(args.l2_chain_id.into()))
                    .into();
                Some(signer)
            }
            (None, Some(signer)) => Some(signer.into()),
            (None, None) => None,
        };

        Ok(gossip_signer)
    }

    /// Resolves the sequencer key from either the raw key or the key file.
    fn resolve_sequencer_key(&self) -> Result<Option<B256>, SignerArgsParseError> {
        match (self.sequencer_key, &self.sequencer_key_path) {
            (Some(key), None) => Ok(Some(key)),
            (None, Some(path)) => {
                let keypair = SecretKeyLoader::load(path)?;
                // Extract the private key bytes from the secp256k1 keypair
                keypair.try_into_secp256k1().map_or_else(
                    |_| Err(SignerArgsParseError::SequencerKeyInvalid(ecdsa::Error::new())),
                    |secp256k1_keypair| {
                        let private_key_bytes = secp256k1_keypair.secret().to_bytes();
                        let key = B256::from_slice(&private_key_bytes);
                        Ok(Some(key))
                    },
                )
            }
            (Some(_), Some(_)) => Err(SignerArgsParseError::ConflictingSequencerKeyInputs),
            (None, None) => Ok(None),
        }
    }

    /// Creates a [`RemoteSigner`] from the [`SignerArgs`].
    fn config_remote(self) -> Result<Option<RemoteSigner>, SignerArgsParseError> {
        let Some(endpoint) = self.endpoint else {
            return Ok(None);
        };

        let Some(address) = self.address else {
            return Err(SignerArgsParseError::AddressRequired);
        };

        let headers = self
            .header
            .iter()
            .map(|h| {
                let (key, value) = h.split_once('=').ok_or(SignerArgsParseError::InvalidHeader)?;
                Ok((HeaderName::from_str(key)?, HeaderValue::from_str(value)?))
            })
            .collect::<Result<HeaderMap, SignerArgsParseError>>()?;

        let client_cert = self
            .cert
            .clone()
            .map(|cert| {
                Ok::<_, SignerArgsParseError>(ClientCert {
                    cert,
                    key: self.key.clone().ok_or(SignerArgsParseError::KeyRequired)?,
                })
            })
            .transpose()?;

        Ok(Some(RemoteSigner {
            address,
            endpoint,
            ca_cert: self.ca_cert.clone(),
            client_cert,
            headers,
        }))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::b256;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_resolve_sequencer_key_from_file() {
        // Create a temporary file with a private key
        let mut temp_file = NamedTempFile::new().unwrap();
        let key = b256!("1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be");
        let hex = alloy_primitives::hex::encode(key.0);
        write!(temp_file, "{hex}").unwrap();

        let signer_args = SignerArgs {
            sequencer_key: None,
            sequencer_key_path: Some(temp_file.path().to_path_buf()),
            ..Default::default()
        };

        let resolved = signer_args.resolve_sequencer_key().unwrap();
        assert_eq!(resolved, Some(key));
    }

    #[test]
    fn test_resolve_sequencer_key_conflicting_inputs() {
        let signer_args = SignerArgs {
            sequencer_key: Some(b256!(
                "bcc617ea05150ff60490d3c6058630ba94ae9f12a02a87efd291349ca0e54e0a"
            )),
            sequencer_key_path: Some(PathBuf::from("/path/to/key.txt")),
            ..Default::default()
        };

        let result = signer_args.resolve_sequencer_key();
        assert!(matches!(result, Err(SignerArgsParseError::ConflictingSequencerKeyInputs)));
    }
}
