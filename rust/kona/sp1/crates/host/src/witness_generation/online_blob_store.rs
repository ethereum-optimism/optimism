//! Provides an implementation of a [`BlobProvider`] that fetches blobs online and stores them in a
//! shared store.

use alloy_consensus::Blob;
use alloy_eips::eip4844::env_settings::EnvKzgSettings;
use alloy_primitives::B256;
use anyhow::Result;
use async_trait::async_trait;
use kona_derive::BlobProvider;
use kona_protocol::BlockInfo;
use kona_sp1_client_utils::witness::BlobData;
use kzg_rs::{Blob as KzgRsBlob, Bytes48};
use std::sync::{Arc, Mutex};

/// A `BlobProvider` that fetches blobs online and stores their data in a shared `BlobData` store.
#[derive(Clone, Debug)]
pub struct OnlineBlobStore<T: BlobProvider> {
    /// The underlying blob provider used to fetch blobs.
    pub provider: T,
    /// Shared blob storage
    pub store: Arc<Mutex<BlobData>>,
}

/// This is only invoked when fetching the blobs in online mode. In zkVM mode,
/// the blobs are given upfront.
#[async_trait]
impl<T: BlobProvider + Send> BlobProvider for OnlineBlobStore<T> {
    type Error = T::Error;

    async fn get_and_validate_blobs(
        &mut self,
        block_ref: &BlockInfo,
        blob_hashes: &[B256],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        let blobs = self.provider.get_and_validate_blobs(block_ref, blob_hashes).await?;
        let settings = EnvKzgSettings::default();

        let mut store = self.store.lock().unwrap();
        for blob in &blobs {
            let (c_kzg_blob, commitment, proof) = get_blob_data(blob, &settings);

            store.blobs.push(c_kzg_blob);
            store.commitments.push(commitment);
            store.proofs.push(proof);
        }
        Ok(blobs)
    }
}

/// Get the blob data for the given blob with the given settings.
#[allow(clippy::large_stack_frames)]
fn get_blob_data(blob: &Blob, settings: &EnvKzgSettings) -> (KzgRsBlob, Bytes48, Bytes48) {
    let c_kzg_blob = c_kzg::Blob::from_bytes(blob.as_slice()).unwrap();
    let commitment = settings.get().blob_to_kzg_commitment(&c_kzg_blob).unwrap();
    let proof = settings.get().compute_blob_kzg_proof(&c_kzg_blob, &commitment.to_bytes()).unwrap();
    (
        KzgRsBlob::from_slice(&*c_kzg_blob).unwrap(),
        kzg_rs::Bytes48(*commitment.to_bytes()),
        kzg_rs::Bytes48(*proof.to_bytes()),
    )
}
