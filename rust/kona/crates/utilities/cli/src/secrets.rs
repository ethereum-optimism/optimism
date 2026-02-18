//! Utility functions for working with secret keys.
//!
//! This module is adapted from <https://github.com/paradigmxyz/reth/blob/aef442740c51fc00884d34931ebc3b547e41b9f4/crates/cli/util/src/load_secret_key.rs#L20>

use alloy_primitives::B256;
use libp2p::identity::{Keypair, secp256k1::SecretKey};
use std::{
    path::{Path, PathBuf},
    str::FromStr,
};
use thiserror::Error;

/// A loader type for loading secret keys.
#[derive(Debug, Clone)]
pub struct SecretKeyLoader;

impl SecretKeyLoader {
    /// Attempts to load a [`Keypair`] from a specified path.
    ///
    /// If no file exists there, then it generates a secret key
    /// and stores it in the provided path. I/O errors might occur
    /// during write operations in the form of a [`KeypairError`]
    pub fn load(secret_key_path: &Path) -> Result<Keypair, KeypairError> {
        let exists = secret_key_path.try_exists();

        match exists {
            Ok(true) => {
                let contents = std::fs::read_to_string(secret_key_path)?;
                let contents_trimmed = contents.trim();
                let mut decoded = B256::from_str(contents_trimmed).inspect_err(|e| {
                    tracing::error!(
                        target: "p2p::secrets",
                        path = %secret_key_path.display(),
                        error = %e,
                        contents_len = contents.len(),
                        "Failed to decode secret key from file"
                    );
                })?;
                Ok(Self::parse(&mut decoded.0)?).inspect(|keypair| {
                    tracing::info!(
                        target: "p2p::secrets",
                        path = %secret_key_path.display(),
                        peer_id = %keypair.public().to_peer_id(),
                        "Successfully loaded P2P keypair from file"
                    );
                })
            }
            Ok(false) => {
                if let Some(dir) = secret_key_path.parent() {
                    std::fs::create_dir_all(dir)?;
                }

                let secret = SecretKey::generate();
                let hex = alloy_primitives::hex::encode(secret.to_bytes());
                std::fs::write(secret_key_path, &hex)?;
                let kp = libp2p::identity::secp256k1::Keypair::from(secret);
                Ok(Keypair::from(kp)).inspect(|keypair| {
                    tracing::info!(
                        target: "p2p::secrets",
                        path = %secret_key_path.display(),
                        peer_id = %keypair.public().to_peer_id(),
                        "Generated new P2P keypair and saved to file"
                    );
                })
            }
            Err(error) => {
                tracing::error!(
                    target: "p2p::secrets",
                    path = %secret_key_path.display(),
                    error = %error,
                    "Failed to access secret key file"
                );
                Err(KeypairError::FailedToAccessKeyFile {
                    error,
                    secret_file: secret_key_path.to_path_buf(),
                })
            }
        }
    }

    /// Parses raw bytes into a [`Keypair`].
    pub fn parse(input: &mut [u8]) -> Result<Keypair, ParseKeyError> {
        let sk =
            SecretKey::try_from_bytes(input).map_err(|_| ParseKeyError::FailedToParseSecretKey)?;
        let kp = libp2p::identity::secp256k1::Keypair::from(sk);
        Ok(Keypair::from(kp))
    }
}

/// An error parsing raw secret key bytes into a [`Keypair`].
#[derive(Error, Clone, Copy, Debug, PartialEq, Eq)]
pub enum ParseKeyError {
    /// Failed to parse the bytes into an `secp256k1` secret key.
    #[error("Failed to parse bytes into an `secp256k1` secret key")]
    FailedToParseSecretKey,
}

/// Errors returned by loading a [`Keypair`], including IO errors.
#[derive(Error, Debug)]
pub enum KeypairError {
    /// Error encountered during decoding of the secret key.
    #[error(transparent)]
    SecretKeyDecodeError(#[from] ParseKeyError),

    /// An error encountered converting a hex string into [`B256`] bytes.
    #[error(transparent)]
    HexError(#[from] alloy_primitives::hex::FromHexError),

    /// Error related to file system path operations.
    #[error(transparent)]
    SecretKeyFsPathError(#[from] std::io::Error),

    /// Represents an error when failed to access the key file.
    #[error("failed to access key file {secret_file:?}: {error}")]
    FailedToAccessKeyFile {
        /// The encountered IO error.
        error: std::io::Error,
        /// Path to the secret key file.
        secret_file: PathBuf,
    },
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::b256;
    use std::path::PathBuf;

    #[test]
    fn test_parse() {
        let secret_key = SecretKey::generate();
        let mut key = B256::from(secret_key.to_bytes());
        assert!(SecretKeyLoader::parse(&mut key.0).is_ok());
    }

    #[test]
    fn test_load_new_file() {
        let dir = std::env::temp_dir();
        assert!(std::env::set_current_dir(dir).is_ok());

        let path = PathBuf::from("./root/does_not_exist34.txt");
        assert!(SecretKeyLoader::load(&path).is_ok());
    }

    // Github actions panics on this test.
    #[test]
    #[ignore]
    fn test_load_invalid_path() {
        let dir = std::env::temp_dir();
        assert!(std::env::set_current_dir(dir).is_ok());

        let path = PathBuf::from("/root/does_not_exist34.txt");
        assert!(!path.try_exists().unwrap());
        let err = SecretKeyLoader::load(&path).unwrap_err();
        let KeypairError::SecretKeyFsPathError(_) = err else {
            panic!("Incorrect error thrown");
        };
    }

    #[test]
    fn test_load_file() {
        // Create a temporary directory.
        let dir = std::env::temp_dir();
        let mut key_path = dir.clone();
        assert!(std::env::set_current_dir(dir).is_ok());

        // Write a private key to a file.
        let key = b256!("1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be");
        let hex = alloy_primitives::hex::encode(key.0);
        key_path.push("test.txt");
        std::fs::write(&key_path, &hex).unwrap();

        // Validate the keypair and file contents (to make sure it wasn't overwritten).
        assert!(SecretKeyLoader::load(&key_path).is_ok());
        let contents = std::fs::read_to_string(&key_path).unwrap();
        assert_eq!(contents, hex);
    }

    #[test]
    fn test_load_file_with_trailing_newline() {
        // Create a temporary directory.
        let dir = std::env::temp_dir();
        let mut key_path = dir.clone();
        assert!(std::env::set_current_dir(dir).is_ok());

        // Write a private key with trailing newline (common with text editors).
        let key = b256!("1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be");
        let hex = alloy_primitives::hex::encode(key.0);
        let hex_with_newline = format!("{hex}\n");
        key_path.push("test_newline.txt");
        std::fs::write(&key_path, &hex_with_newline).unwrap();

        // Should successfully load despite trailing newline.
        let keypair = SecretKeyLoader::load(&key_path);
        assert!(keypair.is_ok(), "Failed to load key with trailing newline");
    }

    #[test]
    fn test_load_file_with_trailing_crlf() {
        // Create a temporary directory.
        let dir = std::env::temp_dir();
        let mut key_path = dir.clone();
        assert!(std::env::set_current_dir(dir).is_ok());

        // Write a private key with trailing CRLF (common with Windows/ConfigMaps).
        let key = b256!("1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be");
        let hex = alloy_primitives::hex::encode(key.0);
        let hex_with_crlf = format!("{hex}\r\n");
        key_path.push("test_crlf.txt");
        std::fs::write(&key_path, &hex_with_crlf).unwrap();

        // Should successfully load despite trailing CRLF.
        let keypair = SecretKeyLoader::load(&key_path);
        assert!(keypair.is_ok(), "Failed to load key with trailing CRLF");
    }

    #[test]
    fn test_load_file_with_whitespace_produces_same_peer_id() {
        // Create a temporary directory.
        let dir = std::env::temp_dir();
        assert!(std::env::set_current_dir(dir.clone()).is_ok());

        let key = b256!("1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be");
        let hex = alloy_primitives::hex::encode(key.0);

        // Write clean key.
        let clean_path = dir.join("test_clean.txt");
        std::fs::write(&clean_path, &hex).unwrap();

        // Write key with various whitespace.
        let whitespace_path = dir.join("test_whitespace.txt");
        std::fs::write(&whitespace_path, format!("  {hex}\r\n  ")).unwrap();

        // Both should produce the same peer ID.
        let clean_keypair = SecretKeyLoader::load(&clean_path).unwrap();
        let whitespace_keypair = SecretKeyLoader::load(&whitespace_path).unwrap();

        assert_eq!(
            clean_keypair.public().to_peer_id(),
            whitespace_keypair.public().to_peer_id(),
            "Peer IDs should match regardless of whitespace"
        );
    }
}
