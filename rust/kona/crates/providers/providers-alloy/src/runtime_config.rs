//! Reads of the current runtime configuration, pinned to an L1 block hash.

use alloy_primitives::{Address, B256, b256};
use alloy_provider::Provider;
use alloy_transport::TransportResult;

/// Reads the unsafe block signer from `SystemConfig` at `block_hash`.
///
/// Reading state rather than individual update logs also recovers rotations missed between polls
/// and restores the previous signer when a rotation is reorged out.
pub async fn unsafe_block_signer(
    provider: &impl Provider,
    system_config: Address,
    block_hash: B256,
) -> TransportResult<Address> {
    // bytes32(uint256(keccak256("systemconfig.unsafeblocksigner")) - 1)
    const SLOT: B256 = b256!("65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08");
    let value =
        provider.get_storage_at(system_config, SLOT.into()).hash_canonical(block_hash).await?;
    Ok(Address::from_slice(&value.to_be_bytes::<32>()[12..]))
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_provider::RootProvider;
    use httpmock::MockServer;
    use serde_json::json;

    #[tokio::test]
    async fn snapshots_are_pinned_to_the_requested_contract_and_branch() {
        let server = MockServer::start();
        let provider = RootProvider::new_http(server.url("/").parse().unwrap());
        let contract = Address::repeat_byte(1);
        // Distinct branches can have different signers, including a return to the old signer.
        for (request_id, index) in [2, 3, 2].into_iter().enumerate() {
            let hash = B256::repeat_byte(index);
            let signer = Address::repeat_byte(index + 10);
            let mut mock = server.mock(|when, then| {
                when.method("POST").json_body(json!({
                    "jsonrpc": "2.0", "id": request_id,
                    "method": "eth_getStorageAt",
                    "params": [contract, "0x65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08", {"blockHash": hash, "requireCanonical": true}]
                }));
                then.json_body(json!({"jsonrpc":"2.0", "id":request_id, "result": signer.into_word()}));
            });
            let result = unsafe_block_signer(&provider, contract, hash).await;
            mock.assert();
            assert_eq!(result.unwrap(), signer);
            mock.delete();
        }
    }
}
