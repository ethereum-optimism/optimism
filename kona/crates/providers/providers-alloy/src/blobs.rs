//! Contains an online implementation of the `BlobProvider` trait.

use crate::BeaconClient;
#[cfg(feature = "metrics")]
use crate::Metrics;
use alloy_eips::eip4844::{
    Blob, BlobTransactionSidecarItem, IndexedBlobHash, env_settings::EnvKzgSettings,
};
use alloy_primitives::FixedBytes;
use async_trait::async_trait;
use kona_derive::{BlobProvider, BlobProviderError};
use kona_protocol::BlockInfo;
use std::{boxed::Box, string::ToString, vec::Vec};

/// A boxed blob with index.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BoxedBlobWithIndex {
    /// The index of the blob.
    pub index: u64,
    /// The blob data.
    pub blob: Box<Blob>,
}

/// An online implementation of the [BlobProvider] trait.
#[derive(Debug, Clone)]
pub struct OnlineBlobProvider<B: BeaconClient> {
    /// The Beacon API client.
    pub beacon_client: B,
    /// Beacon Genesis time used for the time to slot conversion.
    pub genesis_time: u64,
    /// Slot interval used for the time to slot conversion.
    pub slot_interval: u64,
}

impl<B: BeaconClient> OnlineBlobProvider<B> {
    /// Creates a new instance of the [OnlineBlobProvider].
    ///
    /// The `genesis_time` and `slot_interval` arguments are _optional_ and the
    /// [OnlineBlobProvider] will attempt to load them dynamically at runtime if they are not
    /// provided.
    ///
    /// ## Panics
    /// Panics if the genesis time or slot interval cannot be loaded from the beacon client.
    pub async fn init(beacon_client: B) -> Self {
        let genesis_time = beacon_client
            .genesis_time()
            .await
            .map(|r| r.data.genesis_time)
            .map_err(|e| BlobProviderError::Backend(e.to_string()))
            .expect("Failed to load genesis time from beacon client");
        let slot_interval = beacon_client
            .slot_interval()
            .await
            .map(|r| r.data.seconds_per_slot)
            .map_err(|e| BlobProviderError::Backend(e.to_string()))
            .expect("Failed to load slot interval from beacon client");
        Self { beacon_client, genesis_time, slot_interval }
    }

    /// Computes the slot for the given timestamp.
    pub const fn slot(
        genesis: u64,
        slot_time: u64,
        timestamp: u64,
    ) -> Result<u64, BlobProviderError> {
        if timestamp < genesis {
            return Err(BlobProviderError::SlotDerivation);
        }
        Ok((timestamp - genesis) / slot_time)
    }

    /// Fetches blobs for the given slot.
    async fn fetch_filtered_blobs(
        &self,
        slot: u64,
        blob_hashes: &[IndexedBlobHash],
    ) -> Result<Vec<BoxedBlobWithIndex>, BlobProviderError> {
        kona_macros::inc!(gauge, Metrics::BLOB_SIDECAR_FETCHES);

        let result = self
            .beacon_client
            .filtered_beacon_blobs(slot, blob_hashes)
            .await
            .map_err(|e| BlobProviderError::Backend(e.to_string()));

        #[cfg(feature = "metrics")]
        if result.is_err() {
            kona_macros::inc!(gauge, Metrics::BLOB_SIDECAR_FETCH_ERRORS);
        }

        result
    }

    /// Converts a vector of boxed blobs with index to a vector of blob transaction sidecar items.
    ///
    /// Note: for performance reasons, we need to transmute the blobs to the c_kzg::Blob type to
    /// avoid the overhead of moving the blobs around or reallocating the memory.
    fn sidecar_from_blobs(
        blobs: Vec<BoxedBlobWithIndex>,
    ) -> Result<Vec<BlobTransactionSidecarItem>, c_kzg::Error> {
        blobs
            .into_iter()
            .map(|blob| {
                let kzg_settings = EnvKzgSettings::Default;

                // SAFETY: all types have the same size and alignment
                let kzg_blob =
                    unsafe { Box::from_raw(Box::<Blob>::into_raw(blob.blob) as *mut c_kzg::Blob) };

                let commitment = kzg_settings
                    .get()
                    .blob_to_kzg_commitment(&kzg_blob)
                    .map(|blob| blob.to_bytes())?;
                let proof = kzg_settings
                    .get()
                    .compute_blob_kzg_proof(&kzg_blob, &commitment)
                    .map(|proof| proof.to_bytes())?;

                // SAFETY: all types have the same size and alignment
                let alloy_blob =
                    unsafe { Box::from_raw(Box::<c_kzg::Blob>::into_raw(kzg_blob) as *mut Blob) };

                Ok(BlobTransactionSidecarItem {
                    index: blob.index,
                    blob: alloy_blob,
                    kzg_commitment: FixedBytes::from(*commitment),
                    kzg_proof: FixedBytes::from(*proof),
                })
            })
            .collect()
    }

    /// Fetches blob sidecars for the given block reference and blob hashes.
    /// Does not validate the blobs. Recomputes the kzg proofs associated with the blobs.
    ///
    /// Use [`Self::beacon_client`] to fetch the blobs without recomputing the kzg
    /// proofs/commitments.
    pub async fn fetch_filtered_blob_sidecars(
        &self,
        block_ref: &BlockInfo,
        blob_hashes: &[IndexedBlobHash],
    ) -> Result<Vec<BlobTransactionSidecarItem>, BlobProviderError> {
        if blob_hashes.is_empty() {
            return Ok(Default::default());
        }

        // Calculate the slot for the given timestamp.
        let slot = Self::slot(self.genesis_time, self.slot_interval, block_ref.timestamp)?;

        // Fetch blobs for the slot using.
        let blobs = self.fetch_filtered_blobs(slot, blob_hashes).await?;

        Self::sidecar_from_blobs(blobs)
            .map_err(|e| BlobProviderError::Backend(format!("KZG commitment error: {e}")))
    }
}

#[async_trait]
impl<B> BlobProvider for OnlineBlobProvider<B>
where
    B: BeaconClient + Send + Sync,
{
    type Error = BlobProviderError;

    /// Fetches blobs that were confirmed in the specified L1 block with the given indexed
    /// hashes. The blobs are validated for their index and hashes using the specified
    /// [IndexedBlobHash].
    async fn get_and_validate_blobs(
        &mut self,
        block_ref: &BlockInfo,
        blob_hashes: &[IndexedBlobHash],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        // Fetch the blob sidecars for the given block reference and blob hashes.
        let blobs = self.fetch_filtered_blob_sidecars(block_ref, blob_hashes).await?;

        // Validate the blob sidecars straight away with the num hashes.
        let blobs = blobs
            .into_iter()
            .enumerate()
            .map(|(i, sidecar)| {
                let hash = blob_hashes
                    .get(i)
                    .ok_or(BlobProviderError::Backend("Missing blob hash".to_string()))?
                    .hash
                    .as_slice();

                if sidecar.to_kzg_versioned_hash() != hash {
                    return Err(BlobProviderError::Backend("KZG commitment mismatch".to_string()));
                }

                Ok(sidecar.blob)
            })
            .collect::<Result<Vec<Box<Blob>>, BlobProviderError>>()
            .map_err(|e| BlobProviderError::Backend(e.to_string()))?;
        Ok(blobs)
    }
}
