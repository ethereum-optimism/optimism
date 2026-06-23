use crate::witness::BlobData;
use alloc::{boxed::Box, vec::Vec};
use alloy_consensus::Blob;
use alloy_eips::eip4844::kzg_to_versioned_hash;
use alloy_primitives::B256;
use async_trait::async_trait;
use kona_derive::{BlobProvider, BlobProviderError};
use kona_protocol::BlockInfo;
use kzg_rs::get_kzg_settings;

/// In-memory blob store containing verified blobs.
#[derive(Clone, Debug, Default)]
pub struct BlobStore {
    versioned_blobs: Vec<(B256, Blob)>,
}

impl From<BlobData> for BlobStore {
    fn from(value: BlobData) -> Self {
        let blobs: Vec<_> =
            value.blobs.iter().map(|b| kzg_rs::Blob::from_slice(&b.0).unwrap()).collect();
        let versioned_blobs = value
            .commitments
            .iter()
            .map(|c| kzg_to_versioned_hash(c.as_slice()))
            .zip(blobs.iter().map(|b| Blob::from(b.0)))
            .rev()
            .collect();

        match kzg_rs::KzgProof::verify_blob_kzg_proof_batch(
            blobs,
            value.commitments,
            value.proofs,
            &get_kzg_settings(),
        ) {
            Ok(true) => {} // Verification passed
            Ok(false) => panic!("KZG proof verification failed: invalid proofs"),
            Err(e) => panic!("KZG proof verification error: {e}"),
        }

        Self { versioned_blobs }
    }
}

#[async_trait]
impl BlobProvider for BlobStore {
    type Error = BlobProviderError;

    async fn get_and_validate_blobs(
        &mut self,
        _: &BlockInfo,
        blob_hashes: &[B256],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        Ok(blob_hashes
            .iter()
            .filter_map(|hash| {
                let (blob_hash, blob) = self.versioned_blobs.pop().unwrap();
                (*hash == blob_hash).then(|| Box::new(blob))
            })
            .collect())
    }
}
