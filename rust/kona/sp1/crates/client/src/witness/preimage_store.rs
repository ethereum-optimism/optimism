//! Preimage store implementation for the zkVM.

use alloy_primitives::keccak256;
use async_trait::async_trait;
use kona_preimage::{
    HintWriterClient, PreimageKey, PreimageKeyType, PreimageOracleClient,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use kona_proof::FlushableCache;
use serde::{Deserialize as SerdeDeserialize, Serialize as SerdeSerialize};
use sha2::Digest;
use std::collections::HashMap;

/// In-memory preimage store for use in the zkVM.
#[derive(
    Clone,
    Debug,
    Default,
    SerdeSerialize,
    SerdeDeserialize,
    rkyv::Serialize,
    rkyv::Archive,
    rkyv::Deserialize,
)]
pub struct PreimageStore {
    preimage_map: HashMap<PreimageKey, Vec<u8>>,
}

impl PreimageStore {
    /// Check that all preimages are valid.
    pub fn check_preimages(&self) -> PreimageOracleResult<()> {
        for (key, value) in &self.preimage_map {
            check_preimage(key, value)?;
        }
        Ok(())
    }

    /// Adds a preimage to storage
    pub fn save_preimage(&mut self, key: PreimageKey, value: Vec<u8>) {
        check_preimage(&key, &value).expect("Invalid preimage");
        if let Some(old) = self.preimage_map.insert(key, value.clone()) {
            assert_eq!(old, value, "Cannot overwrite key");
        }
    }
}

/// Check that the preimage matches the expected hash.
pub fn check_preimage(key: &PreimageKey, value: &[u8]) -> PreimageOracleResult<()> {
    if let Some(expected_hash) = match key.key_type() {
        PreimageKeyType::Keccak256 => Some(keccak256(value).0),
        PreimageKeyType::Sha256 => Some(sha2::Sha256::digest(value).into()),
        PreimageKeyType::Local | PreimageKeyType::GlobalGeneric => None,
        PreimageKeyType::Precompile => unimplemented!("Precompile not supported in zkVM"),
        PreimageKeyType::Blob => unreachable!("Blob keys validated in blob witness"),
    } && key != &PreimageKey::new(expected_hash, key.key_type())
    {
        return Err(PreimageOracleError::InvalidPreimageKey);
    }
    Ok(())
}

#[async_trait]
impl HintWriterClient for PreimageStore {
    async fn write(&self, _hint: &str) -> PreimageOracleResult<()> {
        Ok(())
    }
}

#[async_trait]
impl PreimageOracleClient for PreimageStore {
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        let Some(value) = self.preimage_map.get(&key) else {
            return Err(PreimageOracleError::InvalidPreimageKey);
        };
        Ok(value.clone())
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        buf.copy_from_slice(&self.get(key).await?);
        Ok(())
    }
}

impl FlushableCache for PreimageStore {
    fn flush(&self) {}
}
