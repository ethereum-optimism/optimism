//! Shared helpers for integration tests under `tests/`.

#![allow(dead_code)]

use async_trait::async_trait;
use kona_preimage::{
    HintWriterClient, PreimageKey, PreimageOracleClient,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use std::{collections::HashMap, sync::Arc};
use tokio::sync::Mutex;

#[derive(Clone, Debug, Default)]
pub struct MockOracle {
    preimages: Arc<Mutex<HashMap<PreimageKey, Vec<u8>>>>,
}

impl MockOracle {
    pub fn from_preimages(preimages: HashMap<PreimageKey, Vec<u8>>) -> Self {
        Self { preimages: Arc::new(Mutex::new(preimages)) }
    }

    pub fn single(key: PreimageKey, value: Vec<u8>) -> Self {
        let mut preimages = HashMap::new();
        preimages.insert(key, value);
        Self::from_preimages(preimages)
    }
}

#[async_trait]
impl PreimageOracleClient for MockOracle {
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        self.preimages.lock().await.get(&key).cloned().ok_or(PreimageOracleError::KeyNotFound)
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        let data = self.get(key).await?;
        if data.len() != buf.len() {
            return Err(PreimageOracleError::BufferLengthMismatch(buf.len(), data.len()));
        }
        buf.copy_from_slice(&data);
        Ok(())
    }
}

#[async_trait]
impl HintWriterClient for MockOracle {
    async fn write(&self, _hint: &str) -> PreimageOracleResult<()> {
        Ok(())
    }
}
