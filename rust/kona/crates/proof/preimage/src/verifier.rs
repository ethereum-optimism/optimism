//! A [`PreimageFetcher`] wrapper that re-hashes returned preimages against the requested key.
//!
//! The wrapper catches corrupt RPC responses, hint-handler bugs, and tampered
//! KV stores at fetch time rather than letting bad data propagate.

use crate::{
    HintRouter, PreimageFetcher, PreimageKey, PreimageKeyType,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use alloc::{boxed::Box, string::String, vec::Vec};
use alloy_primitives::keccak256;
use async_trait::async_trait;
use sha2::{Digest, Sha256};

/// Wraps an inner [`PreimageFetcher`] and verifies returned preimages by re-hashing
/// them against the requested key.
#[derive(Debug, Clone, Copy)]
pub struct VerifyingPreimageFetcher<F> {
    inner: F,
}

impl<F> VerifyingPreimageFetcher<F> {
    /// Create a new [`VerifyingPreimageFetcher`] from an inner fetcher.
    pub const fn new(inner: F) -> Self {
        Self { inner }
    }

    /// Returns a reference to the inner fetcher.
    pub const fn inner(&self) -> &F {
        &self.inner
    }
}

/// Verify that `data` is a valid preimage for `key`, when the key type carries a
/// self-describing hash (Keccak256, Sha256). Other key types pass through.
///
/// `GlobalGeneric` is reserved and unused; observing it is treated as a programmer
/// error and returns [`PreimageOracleError::UnsupportedKeyType`].
pub fn verify_preimage(key: PreimageKey, data: &[u8]) -> PreimageOracleResult<()> {
    let key_bytes: [u8; 32] = key.into();
    match key.key_type() {
        PreimageKeyType::Keccak256 => {
            let digest = keccak256(data);
            if digest[1..] != key_bytes[1..] {
                return Err(PreimageOracleError::IncorrectData(key));
            }
        }
        PreimageKeyType::Sha256 => {
            let digest = Sha256::digest(data);
            if digest[1..] != key_bytes[1..] {
                return Err(PreimageOracleError::IncorrectData(key));
            }
        }
        // Pass through key types that aren't self-verifying from `(key, data)` alone:
        //   - Local: opaque local identifier, no hash relation.
        //   - Blob: individual field element; verification needs a KZG proof.
        //   - Precompile: result bytes; verification needs the precompile input.
        PreimageKeyType::Local | PreimageKeyType::Blob | PreimageKeyType::Precompile => {}
        // Reserved, no defined semantics, and no host path produces it today. Reject so a
        // future accidental use surfaces loudly rather than silently passing through.
        PreimageKeyType::GlobalGeneric => {
            return Err(PreimageOracleError::UnsupportedKeyType(PreimageKeyType::GlobalGeneric));
        }
    }
    Ok(())
}

#[async_trait]
impl<F> PreimageFetcher for VerifyingPreimageFetcher<F>
where
    F: PreimageFetcher + Send + Sync,
{
    async fn get_preimage(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        let data = self.inner.get_preimage(key).await?;
        verify_preimage(key, &data)?;
        Ok(data)
    }
}

#[async_trait]
impl<F> HintRouter for VerifyingPreimageFetcher<F>
where
    F: HintRouter + Send + Sync,
{
    async fn route_hint(&self, hint: String) -> PreimageOracleResult<()> {
        self.inner.route_hint(hint).await
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::PreimageKeyType;
    use alloc::sync::Arc;
    use alloy_primitives::keccak256;
    use async_trait::async_trait;
    use std::collections::HashMap;
    use tokio::sync::Mutex;

    /// A test fetcher that returns whatever bytes are stored under each key,
    /// regardless of whether the key actually hashes to those bytes. This is
    /// the "corrupt source" the verifier must reject.
    struct MockFetcher {
        responses: Arc<Mutex<HashMap<PreimageKey, Vec<u8>>>>,
    }

    #[async_trait]
    impl PreimageFetcher for MockFetcher {
        async fn get_preimage(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            let guard = self.responses.lock().await;
            guard.get(&key).cloned().ok_or(PreimageOracleError::KeyNotFound)
        }
    }

    fn make_fetcher(responses: HashMap<PreimageKey, Vec<u8>>) -> MockFetcher {
        MockFetcher { responses: Arc::new(Mutex::new(responses)) }
    }

    #[tokio::test]
    async fn keccak256_valid_data_passes() {
        let data = b"hello world".to_vec();
        let key = PreimageKey::new(*keccak256(&data), PreimageKeyType::Keccak256);
        let mut map = HashMap::new();
        map.insert(key, data.clone());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let got = verifier.get_preimage(key).await.unwrap();
        assert_eq!(got, data);
    }

    #[tokio::test]
    async fn keccak256_corrupt_data_rejected() {
        let data = b"hello world".to_vec();
        let key = PreimageKey::new(*keccak256(&data), PreimageKeyType::Keccak256);
        // Map the key to *wrong* bytes, simulating a corrupt RPC/KV/hint handler.
        let mut map = HashMap::new();
        map.insert(key, b"goodbye".to_vec());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let err = verifier.get_preimage(key).await.unwrap_err();
        assert!(matches!(err, PreimageOracleError::IncorrectData(_)));
    }

    #[tokio::test]
    async fn keccak256_empty_data_rejected() {
        let data = b"hello world".to_vec();
        let key = PreimageKey::new(*keccak256(&data), PreimageKeyType::Keccak256);
        let mut map = HashMap::new();
        map.insert(key, Vec::new());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let err = verifier.get_preimage(key).await.unwrap_err();
        assert!(matches!(err, PreimageOracleError::IncorrectData(_)));
    }

    #[tokio::test]
    async fn sha256_valid_data_passes() {
        let data = b"hello world".to_vec();
        let digest: [u8; 32] = Sha256::digest(&data).into();
        let key = PreimageKey::new(digest, PreimageKeyType::Sha256);
        let mut map = HashMap::new();
        map.insert(key, data.clone());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let got = verifier.get_preimage(key).await.unwrap();
        assert_eq!(got, data);
    }

    #[tokio::test]
    async fn sha256_corrupt_data_rejected() {
        let data = b"hello world".to_vec();
        let digest: [u8; 32] = Sha256::digest(&data).into();
        let key = PreimageKey::new(digest, PreimageKeyType::Sha256);
        let mut map = HashMap::new();
        map.insert(key, b"tampered".to_vec());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let err = verifier.get_preimage(key).await.unwrap_err();
        assert!(matches!(err, PreimageOracleError::IncorrectData(_)));
    }

    #[tokio::test]
    async fn local_key_pass_through() {
        let key = PreimageKey::new_local(7);
        let data = b"anything goes".to_vec();
        let mut map = HashMap::new();
        map.insert(key, data.clone());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let got = verifier.get_preimage(key).await.unwrap();
        assert_eq!(got, data);
    }

    #[tokio::test]
    async fn blob_key_pass_through() {
        let key = PreimageKey::new([0xAA; 32], PreimageKeyType::Blob);
        let data = b"opaque blob field element".to_vec();
        let mut map = HashMap::new();
        map.insert(key, data.clone());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let got = verifier.get_preimage(key).await.unwrap();
        assert_eq!(got, data);
    }

    #[tokio::test]
    async fn precompile_key_pass_through() {
        let key = PreimageKey::new([0xBB; 32], PreimageKeyType::Precompile);
        let data = b"precompile result".to_vec();
        let mut map = HashMap::new();
        map.insert(key, data.clone());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let got = verifier.get_preimage(key).await.unwrap();
        assert_eq!(got, data);
    }

    #[tokio::test]
    async fn global_generic_key_rejected() {
        let key = PreimageKey::new([0xCC; 32], PreimageKeyType::GlobalGeneric);
        let mut map = HashMap::new();
        map.insert(key, b"whatever".to_vec());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let err = verifier.get_preimage(key).await.unwrap_err();
        assert!(matches!(
            err,
            PreimageOracleError::UnsupportedKeyType(PreimageKeyType::GlobalGeneric)
        ));
    }

    #[tokio::test]
    async fn incorrect_data_error_carries_key() {
        let data = b"hello world".to_vec();
        let key = PreimageKey::new(*keccak256(&data), PreimageKeyType::Keccak256);
        let mut map = HashMap::new();
        map.insert(key, b"wrong".to_vec());

        let verifier = VerifyingPreimageFetcher::new(make_fetcher(map));
        let err = verifier.get_preimage(key).await.unwrap_err();
        match err {
            PreimageOracleError::IncorrectData(reported) => assert_eq!(reported, key),
            other => panic!("expected IncorrectData, got {other:?}"),
        }
    }

    #[tokio::test]
    async fn inner_error_propagates() {
        let key = PreimageKey::new([0u8; 32], PreimageKeyType::Keccak256);
        // No mapping for the key, inner returns KeyNotFound.
        let verifier = VerifyingPreimageFetcher::new(make_fetcher(HashMap::new()));
        let err = verifier.get_preimage(key).await.unwrap_err();
        assert!(matches!(err, PreimageOracleError::KeyNotFound));
    }
}
