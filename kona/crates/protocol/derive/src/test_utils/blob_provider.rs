//! An implementation of the [BlobProvider] trait for tests.

use crate::{BlobProvider, errors::BlobProviderError};
use alloc::{boxed::Box, vec::Vec};
use alloy_eips::eip4844::{Blob, IndexedBlobHash};
use alloy_primitives::{B256, map::HashMap};
use async_trait::async_trait;
use kona_protocol::BlockInfo;

/// A mock blob provider for testing.
#[derive(Debug, Clone, Default)]
pub struct TestBlobProvider {
    /// Maps block hashes to blob data.
    pub blobs: HashMap<B256, Blob>,
    /// whether the blob provider should return an error.
    pub should_error: bool,
}

impl TestBlobProvider {
    /// Insert a blob into the mock blob provider.
    pub fn insert_blob(&mut self, hash: B256, blob: Blob) {
        self.blobs.insert(hash, blob);
    }

    /// Clears blobs from the mock blob provider.
    pub fn clear(&mut self) {
        self.blobs.clear();
    }
}

#[async_trait]
impl BlobProvider for TestBlobProvider {
    type Error = BlobProviderError;

    async fn get_and_validate_blobs(
        &mut self,
        _block_ref: &BlockInfo,
        blob_hashes: &[IndexedBlobHash],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        if self.should_error {
            return Err(BlobProviderError::SlotDerivation);
        }
        let mut blobs = Vec::new();
        for blob_hash in blob_hashes {
            if let Some(data) = self.blobs.get(&blob_hash.hash) {
                blobs.push(Box::new(*data));
            }
        }
        Ok(blobs)
    }
}
