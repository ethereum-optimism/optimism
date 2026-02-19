//! Contains an online implementation of the `BlobProvider` trait.

use crate::BeaconClient;
#[cfg(feature = "metrics")]
use crate::Metrics;
use alloy_eips::eip4844::{Blob, Bytes48, env_settings::EnvKzgSettings};
use alloy_primitives::{B256, FixedBytes};
use async_trait::async_trait;
use kona_derive::{BlobProvider, BlobProviderError};
use kona_protocol::BlockInfo;
use std::{boxed::Box, string::ToString, vec::Vec};

/// A boxed blob.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BoxedBlob {
    /// The blob data.
    pub blob: Box<Blob>,
}

/// A blob with its KZG commitment and proof.
///
/// This is used by the host to store blob preimages. Unlike `BlobTransactionSidecarItem`,
/// this does not include an index field since blobs are fetched by hash, not by index.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BlobWithCommitmentAndProof {
    /// The blob data.
    pub blob: Box<Blob>,
    /// The KZG commitment for the blob.
    pub kzg_commitment: Bytes48,
    /// The KZG proof for the blob.
    pub kzg_proof: Bytes48,
}

/// An online implementation of the [`BlobProvider`] trait.
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
    /// Creates a new instance of the [`OnlineBlobProvider`].
    ///
    /// The `genesis_time` and `slot_interval` arguments are _optional_ and the
    /// [`OnlineBlobProvider`] will attempt to load them dynamically at runtime if they are not
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
        blob_hashes: &[B256],
    ) -> Result<Vec<BoxedBlob>, BlobProviderError> {
        kona_macros::inc!(gauge, Metrics::BLOB_FETCHES);

        let result = self
            .beacon_client
            .filtered_beacon_blobs(slot, blob_hashes)
            .await
            .map_err(|e| BlobProviderError::Backend(e.to_string()));

        #[cfg(feature = "metrics")]
        if result.is_err() {
            kona_macros::inc!(gauge, Metrics::BLOB_FETCH_ERRORS);
        }

        result
    }

    /// Converts a vector of boxed blobs to a vector of blobs with their KZG commitments and proofs.
    ///
    /// Note: for performance reasons, we need to transmute the blobs to the `c_kzg::Blob` type to
    /// avoid the overhead of moving the blobs around or reallocating the memory.
    fn blobs_with_proofs(
        blobs: Vec<BoxedBlob>,
    ) -> Result<Vec<BlobWithCommitmentAndProof>, c_kzg::Error> {
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

                Ok(BlobWithCommitmentAndProof {
                    blob: alloy_blob,
                    kzg_commitment: FixedBytes::from(*commitment),
                    kzg_proof: FixedBytes::from(*proof),
                })
            })
            .collect()
    }

    /// Fetches and validates blobs for the given block reference and blob hashes.
    /// Recomputes the kzg proofs associated with the blobs and returns as a vector of
    /// [`BlobWithCommitmentAndProof`]s.
    ///
    /// Use [`Self::beacon_client`] to fetch the blobs without recomputing the kzg
    /// proofs/commitments.
    pub async fn fetch_blobs_with_proofs(
        &self,
        block_ref: &BlockInfo,
        blob_hashes: &[B256],
    ) -> Result<Vec<BlobWithCommitmentAndProof>, BlobProviderError> {
        if blob_hashes.is_empty() {
            return Ok(Default::default());
        }

        // Calculate the slot for the given timestamp.
        let slot = Self::slot(self.genesis_time, self.slot_interval, block_ref.timestamp)?;

        // Fetch blobs for the slot using.
        let blobs = self.fetch_filtered_blobs(slot, blob_hashes).await?;

        Self::blobs_with_proofs(blobs)
            .map_err(|e| BlobProviderError::Backend(format!("KZG commitment error: {e}")))
    }
}

#[async_trait]
impl<B> BlobProvider for OnlineBlobProvider<B>
where
    B: BeaconClient + Send + Sync,
{
    type Error = BlobProviderError;

    /// Fetches blobs that were confirmed in the specified L1 block with the given versioned
    /// hashes. The blobs are already validated by the beacon client by recomputing their
    /// commitments and checking against the expected hashes.
    async fn get_and_validate_blobs(
        &mut self,
        block_ref: &BlockInfo,
        blob_hashes: &[B256],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        if blob_hashes.is_empty() {
            return Ok(Default::default());
        }

        // Calculate the slot for the given timestamp.
        let slot = Self::slot(self.genesis_time, self.slot_interval, block_ref.timestamp)?;

        // Fetch and validate blobs from the beacon client.
        // The beacon client already validates each blob by recomputing its commitment
        // and checking it against the expected hash.
        let blobs = self.fetch_filtered_blobs(slot, blob_hashes).await?;

        // Extract the blob data from BoxedBlob wrappers
        Ok(blobs.into_iter().map(|boxed_blob| boxed_blob.blob).collect())
    }
}
