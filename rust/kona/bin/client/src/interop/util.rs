//! Utilities for the interop proof program

use alloc::string::ToString;
use alloy_primitives::B256;
use kona_preimage::{CommsClient, PreimageKey, errors::PreimageOracleError};
use kona_proof::errors::OracleProviderError;
use kona_proof_interop::{HintType, PreState};

/// Fetches the safe head hash of the L2 chain based on the agreed upon L2 output root in the
/// [PreState].
pub(crate) async fn fetch_l2_safe_head_hash<O>(
    caching_oracle: &O,
    pre: &PreState,
) -> Result<B256, OracleProviderError>
where
    O: CommsClient,
{
    // Fetch the output root of the safe head block for the current L2 chain.
    let rich_output = pre
        .active_l2_output_root()
        .ok_or(PreimageOracleError::Other("Missing active L2 output root".to_string()))?;

    fetch_output_block_hash(caching_oracle, rich_output.output_root, rich_output.chain_id).await
}

/// Fetches the block hash that the passed output root commits to.
pub(crate) async fn fetch_output_block_hash<O>(
    caching_oracle: &O,
    output_root: B256,
    chain_id: u64,
) -> Result<B256, OracleProviderError>
where
    O: CommsClient,
{
    HintType::L2OutputRoot
        .with_data(&[output_root.as_slice(), chain_id.to_be_bytes().as_slice()])
        .send(caching_oracle)
        .await?;
    let output_preimage = caching_oracle
        .get(PreimageKey::new_keccak256(*output_root))
        .await
        .map_err(OracleProviderError::Preimage)?;

    output_preimage[96..128].try_into().map_err(OracleProviderError::SliceConversion)
}
