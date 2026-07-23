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
        assert_eq!(value.blobs.len(), value.commitments.len());
        assert_eq!(value.commitments.len(), value.proofs.len());

        let blobs: Vec<_> =
            value.blobs.iter().map(|b| kzg_rs::Blob::from_slice(&b.0).unwrap()).collect();
        let versioned_blobs = value
            .commitments
            .iter()
            .map(|c| kzg_to_versioned_hash(c.as_slice()))
            .zip(blobs.iter().map(|b| Blob::from(b.0)))
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
            .map(|hash| {
                let pos = self
                    .versioned_blobs
                    .iter()
                    .position(|(stored_hash, _)| stored_hash == hash)
                    .unwrap_or_else(|| {
                        panic!("requested blob hash not present in BlobStore: {hash}")
                    });
                let (_, blob) = self.versioned_blobs.swap_remove(pos);
                Box::new(blob)
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

    // Looking up blobs must be by versioned hash, not by position in the
    // internal `versioned_blobs` vec — derivation does not guarantee that
    // requested hashes arrive in the same order they were packed in the
    // witness.
    #[test]
    fn out_of_order_request_returns_all_blobs() {
        let hash_a = B256::repeat_byte(0xAA);
        let hash_b = B256::repeat_byte(0xBB);
        let mut blob_a = Blob::default();
        blob_a[0] = 0xAA;
        let mut blob_b = Blob::default();
        blob_b[0] = 0xBB;

        let mut store = BlobStore { versioned_blobs: vec![(hash_b, blob_b), (hash_a, blob_a)] };

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
        let mut store = BlobStore { versioned_blobs: vec![(hash_a, Blob::default())] };
        let _ = block_on(store.get_and_validate_blobs(&BlockInfo::default(), &[hash_missing]));
    }

    // `kzg_rs::verify_blob_kzg_proof_batch` short-circuits at zero blobs
    // without checking lengths, so we must catch the mismatch locally.
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
