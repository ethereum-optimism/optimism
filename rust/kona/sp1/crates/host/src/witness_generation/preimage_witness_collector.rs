//! This module provides a [`PreimageOracleClient`] implementation.

use std::sync::{Arc, Mutex};

use async_trait::async_trait;
use kona_preimage::{
    CommsClient, HintWriterClient, PreimageKey, PreimageOracleClient, errors::PreimageOracleResult,
};
use kona_proof::FlushableCache;
use kona_sp1_client_utils::witness::preimage_store::PreimageStore;

/// A [`PreimageOracleClient`] that collects preimages into a witness store as they are requested.
#[derive(Clone, Debug)]
pub struct PreimageWitnessCollector<P: CommsClient + FlushableCache + Send + Sync + Clone> {
    /// The underlying preimage oracle.
    pub preimage_oracle: Arc<P>,
    /// The preimage witness store.
    pub preimage_witness_store: Arc<Mutex<PreimageStore>>,
}

#[async_trait]
impl<P> PreimageOracleClient for PreimageWitnessCollector<P>
where
    P: CommsClient + FlushableCache + Send + Sync + Clone,
{
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        let value = self.preimage_oracle.get(key).await?;
        self.save(key, &value);
        Ok(value)
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        self.preimage_oracle.get_exact(key, buf).await?;
        self.save(key, buf);
        Ok(())
    }
}

#[async_trait]
impl<P> HintWriterClient for PreimageWitnessCollector<P>
where
    P: CommsClient + FlushableCache + Send + Sync + Clone,
{
    async fn write(&self, hint: &str) -> PreimageOracleResult<()> {
        self.preimage_oracle.write(hint).await
    }
}

impl<P> FlushableCache for PreimageWitnessCollector<P>
where
    P: CommsClient + FlushableCache + Send + Sync + Clone,
{
    fn flush(&self) {
        self.preimage_oracle.flush();
    }
}

impl<P> PreimageWitnessCollector<P>
where
    P: CommsClient + FlushableCache + Send + Sync + Clone,
{
    /// Saves a preimage to the witness store.
    pub fn save(&self, key: PreimageKey, value: &[u8]) {
        self.preimage_witness_store.lock().unwrap().save_preimage(key, value.to_vec());
    }
}
