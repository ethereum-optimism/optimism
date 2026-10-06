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
use std::collections::{HashMap, hash_map::Entry};

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

    /// Adds a preimage to storage.
    pub fn save_preimage(&mut self, key: PreimageKey, value: Vec<u8>) -> PreimageOracleResult<()> {
        check_preimage(&key, &value)?;

        match self.preimage_map.entry(key) {
            Entry::Vacant(e) => {
                e.insert(value);
            }
            Entry::Occupied(e) => {
                if e.get() != &value {
                    return Err(PreimageOracleError::Other("cannot overwrite key".to_string()));
                }
            }
        }

        Ok(())
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

/// A [`PreimageStore`] read by the proof program: a key missing from the witness panics.
///
/// Shared kona code treats some oracle errors as protocol outcomes rather than failures: the
/// interop message graph marks a message invalid when its initiating block cannot be read, and
/// span batch validation skips a batch whose parent or overlapped blocks cannot be read. The
/// prover chooses which preimages the witness contains, so an error would let it choose those
/// outcomes. Panicking makes a witness with a missing preimage unprovable instead. This also holds
/// when the proof program's code runs natively, outside the zkVM.
///
/// [`PreimageStore`] itself keeps returning an error for a missing key, because witness collection
/// relies on it to fall back to the host.
#[derive(Clone, Debug)]
pub struct WitnessOracle(PreimageStore);

impl WitnessOracle {
    /// Wraps the witness preimages for reading by the proof program.
    pub const fn new(store: PreimageStore) -> Self {
        Self(store)
    }
}

#[async_trait]
impl HintWriterClient for WitnessOracle {
    async fn write(&self, _hint: &str) -> PreimageOracleResult<()> {
        Ok(())
    }
}

#[async_trait]
impl PreimageOracleClient for WitnessOracle {
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        let Some(value) = self.0.preimage_map.get(&key) else {
            panic!("requested preimage key not present in witness: {key}");
        };
        Ok(value.clone())
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        buf.copy_from_slice(&self.get(key).await?);
        Ok(())
    }
}

impl FlushableCache for WitnessOracle {
    fn flush(&self) {}
}
