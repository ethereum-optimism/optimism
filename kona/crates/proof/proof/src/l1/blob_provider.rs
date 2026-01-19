//! Contains the concrete implementation of the [BlobProvider] trait for the client program.

use crate::{HintType, errors::OracleProviderError};
use alloc::{boxed::Box, sync::Arc, vec::Vec};
use alloy_consensus::Blob;
use alloy_eips::eip4844::{FIELD_ELEMENTS_PER_BLOB, IndexedBlobHash};
use alloy_primitives::keccak256;
use ark_bls12_381::Fr;
use ark_ff::{AdditiveGroup, BigInteger, BigInteger256, Field, PrimeField};
use async_trait::async_trait;
use core::str::FromStr;
use kona_derive::BlobProvider;
use kona_preimage::{CommsClient, PreimageKey, PreimageKeyType};
use kona_protocol::BlockInfo;
use spin::Lazy;

/// An oracle-backed blob provider.
#[derive(Debug, Clone)]
pub struct OracleBlobProvider<T: CommsClient> {
    oracle: Arc<T>,
}

impl<T: CommsClient> OracleBlobProvider<T> {
    /// Constructs a new `OracleBlobProvider`.
    pub const fn new(oracle: Arc<T>) -> Self {
        Self { oracle }
    }

    /// Retrieves a blob from the oracle.
    ///
    /// ## Takes
    /// - `block_ref`: The block reference.
    /// - `blob_hash`: The blob hash.
    ///
    /// ## Returns
    /// - `Ok(blob)`: The blob.
    /// - `Err(e)`: The blob could not be retrieved.
    async fn get_blob(
        &self,
        block_ref: &BlockInfo,
        blob_hash: &IndexedBlobHash,
    ) -> Result<Blob, OracleProviderError> {
        let mut blob_req_meta = [0u8; 48];
        blob_req_meta[0..32].copy_from_slice(blob_hash.hash.as_ref());
        blob_req_meta[32..40].copy_from_slice((blob_hash.index).to_be_bytes().as_ref());
        blob_req_meta[40..48].copy_from_slice(block_ref.timestamp.to_be_bytes().as_ref());

        // Send a hint for the blob commitment and field elements.
        HintType::L1Blob.with_data(&[blob_req_meta.as_ref()]).send(self.oracle.as_ref()).await?;

        // Fetch the blob commitment.
        let mut commitment = [0u8; 48];
        self.oracle
            .get_exact(PreimageKey::new(*blob_hash.hash, PreimageKeyType::Sha256), &mut commitment)
            .await
            .map_err(OracleProviderError::Preimage)?;

        // Reconstruct the blob from the 4096 field elements.
        let mut blob = Blob::default();
        let mut field_element_key = [0u8; 80];
        field_element_key[..48].copy_from_slice(commitment.as_ref());
        for i in 0..FIELD_ELEMENTS_PER_BLOB {
            field_element_key[48..]
                .copy_from_slice(ROOTS_OF_UNITY[i as usize].into_bigint().to_bytes_be().as_ref());

            let mut field_element = [0u8; 32];
            self.oracle
                .get_exact(
                    PreimageKey::new(*keccak256(field_element_key), PreimageKeyType::Blob),
                    &mut field_element,
                )
                .await
                .map_err(OracleProviderError::Preimage)?;
            blob[(i as usize) << 5..(i as usize + 1) << 5].copy_from_slice(field_element.as_ref());
        }

        tracing::info!(
            target: "client_blob_oracle",
            index = blob_hash.index,
            hash = ?blob_hash.hash,
            "Retrieved blob"
        );

        Ok(blob)
    }
}

#[async_trait]
impl<T: CommsClient + Sync + Send> BlobProvider for OracleBlobProvider<T> {
    type Error = OracleProviderError;

    async fn get_and_validate_blobs(
        &mut self,
        block_ref: &BlockInfo,
        blob_hashes: &[IndexedBlobHash],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        let mut blobs = Vec::with_capacity(blob_hashes.len());
        for hash in blob_hashes {
            blobs.push(Box::new(self.get_blob(block_ref, hash).await?));
        }
        Ok(blobs)
    }
}

/// The 4096th bit-reversed roots of unity used in EIP-4844 as predefined evaluation points.
///
/// See `generate_roots_of_unity` for details on how these roots of unity are generated.
pub static ROOTS_OF_UNITY: Lazy<[Fr; FIELD_ELEMENTS_PER_BLOB as usize]> =
    Lazy::new(generate_roots_of_unity);

/// Generates the 4096th bit-reversed roots of unity used in EIP-4844 as predefined evaluation
/// points. To compute the field element at index i in a blob, the blob polynomial is evaluated at
/// the i'th root of unity. Based on go-kzg-4844: <https://github.com/crate-crypto/go-kzg-4844/blob/8bcf6163d3987313a3194595cf1f33fd45d7301a/internal/kzg/domain.go#L44-L98>
/// Also, see the consensus specs:
///   - compute_roots_of_unity <https://github.com/ethereum/consensus-specs/blob/bf09edef17e2900258f7e37631e9452941c26e86/specs/deneb/polynomial-commitments.md#compute_roots_of_unity>
///   - bit-reversal permutation: <https://github.com/ethereum/consensus-specs/blob/bf09edef17e2900258f7e37631e9452941c26e86/specs/deneb/polynomial-commitments.md#bit-reversal-permutation>
fn generate_roots_of_unity() -> [Fr; FIELD_ELEMENTS_PER_BLOB as usize] {
    const MAX_ORDER_ROOT: u64 = 32;

    let mut roots_of_unity = [Fr::ZERO; FIELD_ELEMENTS_PER_BLOB as usize];

    // Generator of the largest 2-adic subgroup of order 2^32.
    let root_of_unity = Fr::new(
        BigInteger256::from_str(
            "10238227357739495823651030575849232062558860180284477541189508159991286009131",
        )
        .expect("Failed to initialize root of unity"),
    );

    // Find generator subgroup of order x.
    // This can be constructed by powering a generator of the largest 2-adic subgroup of order 2^32
    // by an exponent of (2^32)/x, provided x is <= 2^32.
    let log_x = FIELD_ELEMENTS_PER_BLOB.trailing_zeros() as u64;
    let expo = 1u64 << (MAX_ORDER_ROOT - log_x);

    // Generator has order x now
    let generator = root_of_unity.pow([expo]);

    // Compute all relevant roots of unity, i.e. the multiplicative subgroup of size x
    let mut current = Fr::ONE;
    (0..FIELD_ELEMENTS_PER_BLOB).for_each(|i| {
        roots_of_unity[i as usize] = current;
        current *= generator;
    });

    let shift_correction = 64 - FIELD_ELEMENTS_PER_BLOB.trailing_zeros();
    (0..FIELD_ELEMENTS_PER_BLOB).for_each(|i| {
        // Find index irev, such that i and irev get swapped
        let irev = i.reverse_bits() >> shift_correction;
        if irev > i {
            roots_of_unity.swap(i as usize, irev as usize);
        }
    });

    roots_of_unity
}

#[cfg(test)]
mod test {
    use super::ROOTS_OF_UNITY;
    use alloy_eips::eip4844::{FIELD_ELEMENTS_PER_BLOB, env_settings::EnvKzgSettings};
    use ark_ff::{BigInteger, PrimeField};
    use c_kzg::{BYTES_PER_BLOB, Blob, Bytes32, Bytes48};
    use rand::Rng;
    use rayon::iter::{IntoParallelIterator, ParallelIterator};

    #[test]
    fn test_roots_of_unity() {
        // Initiate the default Ethereum KZG settings.
        let kzg = EnvKzgSettings::default();

        // Create a blob with random data
        let mut bytes = [0u8; BYTES_PER_BLOB];
        rand::rng().fill(bytes.as_mut_slice());

        // Ensure the blob is valid by keeping each field element within range.
        (0..FIELD_ELEMENTS_PER_BLOB).for_each(|i| {
            bytes[(i as usize) << 5] = 0;
        });

        let blob = Blob::new(bytes);
        let blob_commitment = {
            let raw = kzg.get().blob_to_kzg_commitment(&blob).unwrap();
            Bytes48::new(raw.as_slice().try_into().unwrap())
        };

        // Validate each field element in the blob
        (0..FIELD_ELEMENTS_PER_BLOB).into_par_iter().for_each(|i| {
            let field_element = {
                let mut fe = [0u8; 32];
                fe.copy_from_slice(&blob[(i as usize) << 5..(i as usize + 1) << 5]);
                Bytes32::new(fe)
            };

            let z_bytes = Bytes32::new(ROOTS_OF_UNITY[i as usize].into_bigint().to_bytes_be().try_into().unwrap());
            let (proof, fe) = kzg.get().compute_kzg_proof(&blob, &z_bytes).unwrap();

            // Ensure the field element matches the expected value
            assert_eq!(
                fe.as_slice(),
                field_element.as_slice(),
                "Field element {i} does not match the expected value. Expected: {field_element:?}, Got: {fe:?}"
            );

            // Ensure the proof can be verified
            let proof_bytes = Bytes48::new(proof.as_slice().try_into().unwrap());
            let is_valid = kzg.get().verify_kzg_proof(&blob_commitment, &z_bytes, &field_element, &proof_bytes).unwrap();
            assert!(
                is_valid,
                "KZG proof verification failed for field element {i}. Commitment: {blob_commitment:?}, Z: {z_bytes:?}, Field Element: {field_element:?}, Proof: {proof_bytes:?}"
            );
        });
    }
}
