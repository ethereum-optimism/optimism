//! Contains the [CachingOracle], which is a wrapper around an [OracleReader] and [HintWriter] that
//! stores a configurable number of responses in an [LruCache] for quick retrieval.
//!
//! [OracleReader]: kona_preimage::OracleReader
//! [HintWriter]: kona_preimage::HintWriter

use alloc::{boxed::Box, sync::Arc, vec::Vec};
use async_trait::async_trait;
use core::num::NonZeroUsize;
use kona_preimage::{
    HintWriterClient, PreimageKey, PreimageOracleClient, errors::PreimageOracleResult,
};
use lru::LruCache;
use spin::Mutex;

/// A wrapper around an [OracleReader] and [HintWriter] that stores a configurable number of
/// responses in an [LruCache] for quick retrieval.
///
/// [OracleReader]: kona_preimage::OracleReader
/// [HintWriter]: kona_preimage::HintWriter
#[allow(unreachable_pub)]
#[derive(Debug, Clone)]
pub struct CachingOracle<OR, HW>
where
    OR: PreimageOracleClient,
    HW: HintWriterClient,
{
    /// The spin-locked cache that stores the responses from the oracle.
    cache: Arc<Mutex<LruCache<PreimageKey, Vec<u8>>>>,
    /// Oracle reader type.
    oracle_reader: OR,
    /// Hint writer type.
    hint_writer: HW,
}

impl<OR, HW> CachingOracle<OR, HW>
where
    OR: PreimageOracleClient,
    HW: HintWriterClient,
{
    /// Creates a new [CachingOracle] that wraps the given [OracleReader] and stores up to `N`
    /// responses in the cache.
    ///
    /// [OracleReader]: kona_preimage::OracleReader
    pub fn new(cache_size: usize, oracle_reader: OR, hint_writer: HW) -> Self {
        Self {
            cache: Arc::new(Mutex::new(LruCache::new(
                NonZeroUsize::new(cache_size).expect("N must be greater than 0"),
            ))),
            oracle_reader,
            hint_writer,
        }
    }
}

/// A trait that provides a method to flush a cache.
pub trait FlushableCache {
    /// Flushes the cache, removing all entries.
    fn flush(&self);
}

impl<OR, HW> FlushableCache for CachingOracle<OR, HW>
where
    OR: PreimageOracleClient,
    HW: HintWriterClient,
{
    /// Flushes the cache, removing all entries.
    fn flush(&self) {
        self.cache.lock().clear();
    }
}

#[async_trait]
impl<OR, HW> PreimageOracleClient for CachingOracle<OR, HW>
where
    OR: PreimageOracleClient + Sync,
    HW: HintWriterClient + Sync,
{
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        if let Some(value) = self.cache.lock().get(&key) {
            Ok(value.clone())
        } else {
            let value = self.oracle_reader.get(key).await?;
            self.cache.lock().put(key, value.clone());
            Ok(value)
        }
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        if let Some(value) = self.cache.lock().get(&key) {
            // SAFETY: The value never enters the cache unless the preimage length matches the
            // buffer length, due to the checks in the OracleReader.
            buf.copy_from_slice(value.as_slice());
            Ok(())
        } else {
            self.oracle_reader.get_exact(key, buf).await?;
            self.cache.lock().put(key, buf.to_vec());
            Ok(())
        }
    }
}

#[async_trait]
impl<OR, HW> HintWriterClient for CachingOracle<OR, HW>
where
    OR: PreimageOracleClient + Sync,
    HW: HintWriterClient + Sync,
{
    async fn write(&self, hint: &str) -> PreimageOracleResult<()> {
        self.hint_writer.write(hint).await
    }
}
