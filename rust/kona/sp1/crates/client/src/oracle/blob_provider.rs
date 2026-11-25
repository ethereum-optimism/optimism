use crate::witness::BlobData;
use alloc::{boxed::Box, collections::BTreeMap, sync::Arc, vec::Vec};
use alloy_consensus::Blob;
use alloy_eips::eip4844::kzg_to_versioned_hash;
use alloy_primitives::B256;
use async_trait::async_trait;
use kona_derive::{BlobProvider, BlobProviderError};
use kona_protocol::BlockInfo;
use kzg_rs::get_kzg_settings;

/// In-memory blob store containing verified blobs.
///
/// Clones share the blob storage, and a stored blob is returned on every request for its hash.
#[derive(Clone, Debug, Default)]
pub struct BlobStore {
    /// Verified blobs keyed by versioned hash. Boxed so map nodes do not inline 128 KiB slots.
    blobs_by_hash: Arc<BTreeMap<B256, Box<Blob>>>,
}

impl From<BlobData> for BlobStore {
    fn from(value: BlobData) -> Self {
        assert_eq!(value.blobs.len(), value.commitments.len());
        assert_eq!(value.commitments.len(), value.proofs.len());

        let mut blobs_by_hash = BTreeMap::new();
        for (commitment, blob) in value.commitments.iter().zip(&value.blobs) {
            blobs_by_hash
                .insert(kzg_to_versioned_hash(commitment.as_slice()), Box::new(Blob::from(blob.0)));
        }

        // Skip decoding the KZG trusted setup if there are no blobs to verify.
        if !value.blobs.is_empty() {
            match kzg_rs::KzgProof::verify_blob_kzg_proof_batch(
                value.blobs,
                value.commitments,
                value.proofs,
                &get_kzg_settings(),
            ) {
                Ok(true) => {} // Verification passed
                Ok(false) => panic!("KZG proof verification failed: invalid proofs"),
                Err(e) => panic!("KZG proof verification error: {e}"),
            }
        }

        Self { blobs_by_hash: Arc::new(blobs_by_hash) }
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
            .map(|hash| {
                let blob = self.blobs_by_hash.get(hash).unwrap_or_else(|| {
                    panic!("requested blob hash not present in BlobStore: {hash}")
                });
                blob.clone()
            })
            .collect())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::witness::BlobData;
    use kona_proof::block_on;
    use kona_protocol::BlockInfo;

    fn store_with<const N: usize>(entries: [(B256, Blob); N]) -> BlobStore {
        let blobs_by_hash =
            entries.into_iter().map(|(hash, blob)| (hash, Box::new(blob))).collect();
        BlobStore { blobs_by_hash: Arc::new(blobs_by_hash) }
    }

    // Looking up blobs must be by versioned hash, not by position in the witness — derivation
    // does not guarantee that requested hashes arrive in the same order they were packed in the
    // witness.
    #[test]
    fn out_of_order_request_returns_all_blobs() {
        let hash_a = B256::repeat_byte(0xAA);
        let hash_b = B256::repeat_byte(0xBB);
        let mut blob_a = Blob::default();
        blob_a[0] = 0xAA;
        let mut blob_b = Blob::default();
        blob_b[0] = 0xBB;

        let mut store = store_with([(hash_b, blob_b), (hash_a, blob_a)]);

        let result =
            block_on(store.get_and_validate_blobs(&BlockInfo::default(), &[hash_b, hash_a]))
                .expect("get_and_validate_blobs failed");

        assert_eq!(result.len(), 2);
        assert_eq!(*result[0], blob_b);
        assert_eq!(*result[1], blob_a);
    }

    #[test]
    #[should_panic(expected = "requested blob hash not present in BlobStore")]
    fn missing_hash_request_panics() {
        let hash_a = B256::repeat_byte(0xAA);
        let hash_missing = B256::repeat_byte(0xCC);
        let mut store = store_with([(hash_a, Blob::default())]);
        let _ = block_on(store.get_and_validate_blobs(&BlockInfo::default(), &[hash_missing]));
    }

    /// Pins that a stored blob is served on every request for its hash.
    #[test]
    fn repeated_request_returns_blob_each_time() {
        let hash = B256::repeat_byte(0xAA);
        let mut blob = Blob::default();
        blob[0] = 0xAA;
        let mut store = store_with([(hash, blob)]);
        let blobs =
            block_on(store.get_and_validate_blobs(&BlockInfo::default(), &[hash, hash])).unwrap();
        assert_eq!(blobs.len(), 2);
        assert!(blobs.iter().all(|b| **b == blob));
    }

    // The local length asserts reject a mismatch before any blob is decoded or verified.
    #[test]
    #[should_panic(expected = "assertion `left == right` failed")]
    fn from_blob_data_length_mismatch_panics() {
        let bd = BlobData {
            blobs: vec![],
            commitments: vec![kzg_rs::Bytes48([0u8; 48])],
            proofs: vec![],
        };
        let _ = BlobStore::from(bd);
    }
}
