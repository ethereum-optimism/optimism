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
/// Clones share the blob storage but track consumption independently: each stored blob is
/// returned at most once per instance.
#[derive(Clone, Debug, Default)]
pub struct BlobStore {
    /// Verified blobs keyed by versioned hash, in witness order per hash.
    blobs_by_hash: Arc<BTreeMap<B256, Vec<Blob>>>,
    /// Number of blobs per versioned hash already returned by this instance.
    consumed: BTreeMap<B256, usize>,
}

impl From<BlobData> for BlobStore {
    fn from(value: BlobData) -> Self {
        assert_eq!(value.blobs.len(), value.commitments.len());
        assert_eq!(value.commitments.len(), value.proofs.len());

        let mut blobs_by_hash = BTreeMap::<B256, Vec<Blob>>::new();
        for (commitment, blob) in value.commitments.iter().zip(&value.blobs) {
            blobs_by_hash
                .entry(kzg_to_versioned_hash(commitment.as_slice()))
                .or_default()
                .push(Blob::from(blob.0));
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

        Self { blobs_by_hash: Arc::new(blobs_by_hash), consumed: BTreeMap::new() }
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
                let taken = self.consumed.entry(*hash).or_default();
                let blob = self
                    .blobs_by_hash
                    .get(hash)
                    .and_then(|blobs| blobs.get(*taken))
                    .unwrap_or_else(|| {
                        panic!("requested blob hash not present in BlobStore: {hash}")
                    });
                *taken += 1;
                Box::new(*blob)
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

        let mut store = BlobStore {
            blobs_by_hash: Arc::new(BTreeMap::from([
                (hash_b, vec![blob_b]),
                (hash_a, vec![blob_a]),
            ])),
            consumed: BTreeMap::new(),
        };

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
        let mut store = BlobStore {
            blobs_by_hash: Arc::new(BTreeMap::from([(hash_a, vec![Blob::default()])])),
            consumed: BTreeMap::new(),
        };
        let _ = block_on(store.get_and_validate_blobs(&BlockInfo::default(), &[hash_missing]));
    }

    // Clones share blob storage but consume independently; each instance still returns a
    // stored blob at most once.
    #[test]
    #[should_panic(expected = "requested blob hash not present in BlobStore")]
    fn blob_is_consumed_once_per_instance() {
        let hash = B256::repeat_byte(0xAA);
        let mut store = BlobStore {
            blobs_by_hash: Arc::new(BTreeMap::from([(hash, vec![Blob::default()])])),
            consumed: BTreeMap::new(),
        };
        let mut clone = store.clone();

        block_on(clone.get_and_validate_blobs(&BlockInfo::default(), &[hash]))
            .expect("clone returns the blob");
        block_on(store.get_and_validate_blobs(&BlockInfo::default(), &[hash]))
            .expect("original returns the blob independently of the clone");
        let _ = block_on(store.get_and_validate_blobs(&BlockInfo::default(), &[hash]));
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
