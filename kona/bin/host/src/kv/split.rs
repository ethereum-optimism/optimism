//! Contains a concrete implementation of the [KeyValueStore] trait that splits between two separate
//! [KeyValueStore]s depending on [PreimageKeyType].

use super::KeyValueStore;
use alloy_primitives::B256;
use anyhow::Result;
use kona_preimage::PreimageKeyType;

/// A split implementation of the [KeyValueStore] trait that splits between two separate
/// [KeyValueStore]s.
#[derive(Clone, Debug)]
pub struct SplitKeyValueStore<L, R>
where
    L: KeyValueStore,
    R: KeyValueStore,
{
    local_store: L,
    remote_store: R,
}

impl<L, R> SplitKeyValueStore<L, R>
where
    L: KeyValueStore,
    R: KeyValueStore,
{
    /// Create a new [SplitKeyValueStore] with the given left and right [KeyValueStore]s.
    pub const fn new(local_store: L, remote_store: R) -> Self {
        Self { local_store, remote_store }
    }
}

impl<L, R> KeyValueStore for SplitKeyValueStore<L, R>
where
    L: KeyValueStore,
    R: KeyValueStore,
{
    fn get(&self, key: B256) -> Option<Vec<u8>> {
        match PreimageKeyType::try_from(key[0]).ok()? {
            PreimageKeyType::Local => self.local_store.get(key),
            _ => self.remote_store.get(key),
        }
    }

    fn set(&mut self, key: B256, value: Vec<u8>) -> Result<()> {
        self.remote_store.set(key, value)
    }
}
