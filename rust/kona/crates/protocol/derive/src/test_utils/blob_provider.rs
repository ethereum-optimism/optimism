//! An implementation of the [`BlobProvider`] trait for tests.

use crate::{BlobProvider, errors::BlobProviderError};
use alloc::{boxed::Box, vec::Vec};
use alloy_eips::eip4844::Blob;
use alloy_primitives::{B256, map::HashMap};
use async_trait::async_trait;
use kona_protocol::BlockInfo;

/// A mock blob provider for testing.
#[derive(Debug, Clone, Default)]
pub struct TestBlobProvider {
    /// Maps block hashes to blob data.
    pub blobs: HashMap<B256, Blob>,
    /// Whether the blob provider should return a generic backend error.
    pub should_error: bool,
    /// When `true`, `get_and_validate_blobs` returns `BlobProviderError::BlobNotFound`,
    /// simulating a missed/orphaned beacon slot (HTTP 404 from the beacon node).
    pub should_return_not_found: bool,
    /// When `true`, `get_and_validate_blobs` appends one extra blob beyond those
    /// requested, simulating a buggy provider that returns too many blobs (over-fill).
    pub should_return_extra_blob: bool,
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
        blob_hashes: &[B256],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        if self.should_error {
            return Err(BlobProviderError::SlotDerivation);
        }
        if self.should_return_not_found {
            return Err(BlobProviderError::BlobNotFound {
                slot: 0,
                reason: "mock: slot not found".into(),
            });
        }
        let mut blobs = Vec::new();
        for blob_hash in blob_hashes {
            if let Some(data) = self.blobs.get(blob_hash) {
                blobs.push(Box::new(*data));
            }
        }
        if self.should_return_extra_blob {
            blobs.push(Box::new(Blob::default()));
        }
        Ok(blobs)
    }
}
