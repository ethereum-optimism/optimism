//! Client-specific utilities to support L2 block derivation.

use alloy_primitives::B256;
use anyhow::Result;
use kona_preimage::{CommsClient, PreimageKey};
use kona_proof::{HintType, errors::OracleProviderError};

/// Fetches the safe head hash of the L2 chain based on the agreed upon L2 output root in the
/// [`kona_proof::BootInfo`].
pub(crate) async fn fetch_safe_head_hash<O>(
    caching_oracle: &O,
    agreed_l2_output_root: B256,
) -> Result<B256, OracleProviderError>
where
    O: CommsClient,
{
    let mut output_preimage = [0u8; 128];
    HintType::StartingL2Output
        .with_data(&[agreed_l2_output_root.as_ref()])
        .send(caching_oracle)
        .await?;
    caching_oracle
        .get_exact(PreimageKey::new_keccak256(*agreed_l2_output_root), output_preimage.as_mut())
        .await?;

    if output_preimage[..32] != [0u8; 32] {
        return Err(OracleProviderError::UnknownOutputVersion(B256::from_slice(
            &output_preimage[..32],
        )));
    }

    output_preimage[96..128].try_into().map_err(OracleProviderError::SliceConversion)
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{B256, keccak256};
    use kona_preimage::PreimageKey;
    use kona_proof::{block_on, errors::OracleProviderError};

    use super::fetch_safe_head_hash;
    use crate::witness::preimage_store::PreimageStore;

    #[test]
    fn fetch_safe_head_hash_rejects_unknown_output_version() {
        let mut output_preimage = [0u8; 128];
        output_preimage[0] = 0x01;

        let agreed_root = B256::from(keccak256(output_preimage));
        let mut oracle = PreimageStore::default();
        oracle
            .save_preimage(PreimageKey::new_keccak256(*agreed_root), output_preimage.to_vec())
            .unwrap();

        let err = block_on(fetch_safe_head_hash(&oracle, agreed_root)).unwrap_err();
        match err {
            OracleProviderError::UnknownOutputVersion(version) => {
                assert_eq!(version[0], 0x01);
            }
            other => panic!("unexpected error: {other:?}"),
        }
    }
}
