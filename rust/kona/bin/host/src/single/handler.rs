//! [`HintHandler`] for the [`SingleChainHost`].

use crate::{
    HintHandler, OnlineHostBackendCfg,
    backend::util::{
        store_execution_witness, store_history_storage_witness_for_block, store_ordered_trie,
        store_raw_header,
    },
    kv::SharedKeyValueStore,
    single::cfg::SingleChainHost,
};
use alloy_consensus::Header;
use alloy_eips::{
    eip2718::Encodable2718, eip2935::HISTORY_STORAGE_ADDRESS, eip4844::FIELD_ELEMENTS_PER_BLOB,
};
use alloy_primitives::{Address, B256, Bytes, U256, keccak256};
use alloy_provider::Provider;
use alloy_rlp::Decodable;
use alloy_rpc_types::Block;
use anyhow::{Result, anyhow, ensure};
use ark_ff::{BigInteger, PrimeField};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_preimage::{PreimageKey, PreimageKeyType};
use kona_proof::{Hint, HintType, l1::ROOTS_OF_UNITY};
use kona_protocol::{BlockInfo, OutputRoot};
use kona_providers_alloy::BlobWithCommitmentAndProof;
use kona_registry::ROLLUP_CONFIGS;
use op_alloy_rpc_types_engine::OpPayloadAttributes;

/// Parses a blob hint, supporting both legacy (48-byte) and new (40-byte) formats.
///
/// Returns the blob hash and timestamp.
///
/// ## Formats
/// - Legacy: hash (32 bytes) + index (8 bytes) + timestamp (8 bytes) = 48 bytes
/// - New: hash (32 bytes) + timestamp (8 bytes) = 40 bytes
///
/// The legacy index field is parsed but ignored.
pub fn parse_blob_hint(hint_data: &[u8]) -> Result<(B256, u64)> {
    match hint_data.len() {
        48 => {
            // Legacy format: hash (32) + index (8) + timestamp (8)
            let hash_data_bytes: [u8; 32] = hint_data[0..32].try_into()?;
            let _index_data_bytes: [u8; 8] = hint_data[32..40].try_into()?; // index no longer used
            let timestamp_data_bytes: [u8; 8] = hint_data[40..48].try_into()?;

            let hash: B256 = hash_data_bytes.into();
            let timestamp = u64::from_be_bytes(timestamp_data_bytes);
            Ok((hash, timestamp))
        }
        40 => {
            // New format: hash (32) + timestamp (8)
            let hash_data_bytes: [u8; 32] = hint_data[0..32].try_into()?;
            let timestamp_data_bytes: [u8; 8] = hint_data[32..40].try_into()?;

            let hash: B256 = hash_data_bytes.into();
            let timestamp = u64::from_be_bytes(timestamp_data_bytes);
            Ok((hash, timestamp))
        }
        _ => {
            anyhow::bail!(
                "Invalid blob hint length: expected 40 or 48 bytes, got {}",
                hint_data.len()
            );
        }
    }
}

fn rollup_config(cfg: &SingleChainHost) -> Result<RollupConfig> {
    if cfg.rollup_config_path.is_some() {
        return Ok(cfg.read_rollup_config()?);
    }

    cfg.l2_chain_id
        .and_then(|id| ROLLUP_CONFIGS.get(&id).cloned())
        .ok_or_else(|| anyhow!("No rollup config found"))
}

enum L2AccountStorageHintBlock {
    Number(u64),
    Hash(B256),
}

struct ParsedL2AccountStorageHint {
    block: L2AccountStorageHintBlock,
    address: Address,
    slot: U256,
}

fn parse_l2_account_storage_hint(data: &[u8]) -> Result<ParsedL2AccountStorageHint> {
    const BLOCK_NUMBER_HINT_LEN: usize = 8 + 20 + 32;
    const BLOCK_HASH_HINT_LEN: usize = 32 + 20 + 32;

    match data.len() {
        BLOCK_NUMBER_HINT_LEN => {
            let block_number = u64::from_be_bytes(data[..8].try_into()?);
            let address = Address::from_slice(&data[8..28]);
            Ok(ParsedL2AccountStorageHint {
                block: L2AccountStorageHintBlock::Number(block_number),
                address,
                slot: U256::from_be_slice(&data[28..60]),
            })
        }
        BLOCK_HASH_HINT_LEN => {
            let block_hash = B256::from_slice(&data[..32]);
            let address = Address::from_slice(&data[32..52]);
            Ok(ParsedL2AccountStorageHint {
                block: L2AccountStorageHintBlock::Hash(block_hash),
                address,
                slot: U256::from_be_slice(&data[52..84]),
            })
        }
        other => anyhow::bail!(
            "Invalid L2AccountStorageProof hint length: expected {BLOCK_NUMBER_HINT_LEN} or {BLOCK_HASH_HINT_LEN}, got {other}"
        ),
    }
}

fn is_history_storage_hint(data: &[u8]) -> Result<bool> {
    Ok(parse_l2_account_storage_hint(data)?.address == HISTORY_STORAGE_ADDRESS)
}

/// The [`HintHandler`] for the [`SingleChainHost`].
#[derive(Debug, Clone, Copy)]
pub struct SingleChainHintHandler;

#[async_trait]
impl HintHandler for SingleChainHintHandler {
    type Cfg = SingleChainHost;

    async fn fetch_hint_eager(
        hint: &Hint<<Self::Cfg as OnlineHostBackendCfg>::HintType>,
        cfg: &Self::Cfg,
        providers: &<Self::Cfg as OnlineHostBackendCfg>::Providers,
        kv: SharedKeyValueStore,
    ) -> Result<bool> {
        match hint.ty {
            HintType::L1BlockHeader |
            HintType::L1Transactions |
            HintType::L1Receipts |
            HintType::L1Blob |
            HintType::L1Precompile |
            HintType::L2BlockHeader |
            HintType::L2Transactions |
            HintType::StartingL2Output |
            HintType::L2PayloadWitness => {
                Self::fetch_hint(hint.clone(), cfg, providers, kv).await?;
                Ok(true)
            }
            HintType::L2AccountStorageProof if is_history_storage_hint(&hint.data)? => {
                Self::fetch_hint(hint.clone(), cfg, providers, kv).await?;
                Ok(true)
            }
            _ => Ok(false),
        }
    }

    async fn fetch_hint(
        hint: Hint<<Self::Cfg as OnlineHostBackendCfg>::HintType>,
        cfg: &Self::Cfg,
        providers: &<Self::Cfg as OnlineHostBackendCfg>::Providers,
        kv: SharedKeyValueStore,
    ) -> Result<()> {
        match hint.ty {
            HintType::L1BlockHeader => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;
                let raw_header: Bytes =
                    providers.l1.client().request("debug_getRawHeader", [hash]).await?;

                store_raw_header(kv.as_ref(), raw_header).await?;
            }
            HintType::L1Transactions => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;
                let Block { transactions, .. } = providers
                    .l1
                    .get_block_by_hash(hash)
                    .full()
                    .await?
                    .ok_or_else(|| anyhow!("Block not found"))?;
                let encoded_transactions = transactions
                    .into_transactions()
                    .map(|tx| tx.inner.encoded_2718())
                    .collect::<Vec<_>>();

                store_ordered_trie(kv.as_ref(), encoded_transactions.as_slice()).await?;
            }
            HintType::L1Receipts => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;
                let raw_receipts: Vec<Bytes> =
                    providers.l1.client().request("debug_getRawReceipts", [hash]).await?;

                store_ordered_trie(kv.as_ref(), raw_receipts.as_slice()).await?;
            }
            HintType::L1Blob => {
                let (hash, timestamp) = parse_blob_hint(&hint.data)?;

                let partial_block_ref = BlockInfo { timestamp, ..Default::default() };

                // Fetch the blobs from the blob provider.
                let mut blobs = providers
                    .blobs
                    .fetch_blobs_with_proofs(&partial_block_ref, &[hash])
                    .await
                    .map_err(|e| anyhow!("Failed to fetch blobs with proofs: {e}"))?;
                if blobs.len() != 1 {
                    anyhow::bail!("Expected 1 blob, got {}", blobs.len());
                }
                let BlobWithCommitmentAndProof {
                    blob,
                    kzg_proof: proof,
                    kzg_commitment: commitment,
                } = blobs.pop().expect("Expected 1 blob");

                // Acquire a lock on the key-value store and set the preimages.
                let mut kv_lock = kv.write().await;

                // Set the preimage for the blob commitment.
                kv_lock.set(
                    PreimageKey::new(*hash, PreimageKeyType::Sha256).into(),
                    commitment.to_vec(),
                )?;

                // Write all the field elements to the key-value store. There should be 4096.
                // The preimage oracle key for each field element is the keccak256 hash of
                // `abi.encodePacked(sidecar.KZGCommitment, bytes32(ROOTS_OF_UNITY[i]))`.
                let mut blob_key = [0u8; 80];
                blob_key[..48].copy_from_slice(commitment.as_ref());
                for i in 0..FIELD_ELEMENTS_PER_BLOB {
                    blob_key[48..].copy_from_slice(
                        ROOTS_OF_UNITY[i as usize].into_bigint().to_bytes_be().as_ref(),
                    );
                    let blob_key_hash = keccak256(blob_key.as_ref());

                    kv_lock
                        .set(PreimageKey::new_keccak256(*blob_key_hash).into(), blob_key.into())?;
                    kv_lock.set(
                        PreimageKey::new(*blob_key_hash, PreimageKeyType::Blob).into(),
                        blob[(i as usize) << 5..(i as usize + 1) << 5].to_vec(),
                    )?;
                }

                // Write the KZG Proof as the 4096th element.
                // Note: This is not associated with a root of unity, as to be backwards compatible
                // with ZK users of kona that use this proof for the overall blob.
                blob_key[72..].copy_from_slice(FIELD_ELEMENTS_PER_BLOB.to_be_bytes().as_ref());
                let blob_key_hash = keccak256(blob_key.as_ref());

                kv_lock.set(PreimageKey::new_keccak256(*blob_key_hash).into(), blob_key.into())?;
                kv_lock.set(
                    PreimageKey::new(*blob_key_hash, PreimageKeyType::Blob).into(),
                    proof.to_vec(),
                )?;
            }
            HintType::L1Precompile => {
                ensure!(hint.data.len() >= 28, "Invalid hint data length");

                let address = Address::from_slice(&hint.data.as_ref()[..20]);
                let gas = u64::from_be_bytes(hint.data.as_ref()[20..28].try_into()?);
                let input = hint.data[28..].to_vec();
                let input_hash = keccak256(hint.data.as_ref());

                let result = crate::eth::execute(address, input, gas).map_or_else(
                    |_| vec![0u8; 1],
                    |raw_res| {
                        let mut res = Vec::with_capacity(1 + raw_res.len());
                        res.push(0x01);
                        res.extend_from_slice(&raw_res);
                        res
                    },
                );

                let mut kv_lock = kv.write().await;
                kv_lock.set(PreimageKey::new_keccak256(*input_hash).into(), hint.data.into())?;
                kv_lock.set(
                    PreimageKey::new(*input_hash, PreimageKeyType::Precompile).into(),
                    result,
                )?;
            }
            HintType::L2BlockHeader => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                // Fetch the raw header from the L2 chain provider.
                let hash: B256 = hint.data.as_ref().try_into()?;
                let raw_header: Bytes =
                    providers.l2.client().request("debug_getRawHeader", [hash]).await?;

                // Store the header under its actual hash. The verifier enforces the key/data
                // relation when the client asks for the preimage.
                store_raw_header(kv.as_ref(), raw_header).await?;
            }
            HintType::L2Transactions => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;
                let Block { transactions, .. } = providers
                    .l2
                    .get_block_by_hash(hash)
                    .full()
                    .await?
                    .ok_or_else(|| anyhow!("Block not found."))?;

                let encoded_transactions = transactions
                    .into_transactions()
                    .map(|tx| tx.inner.inner.encoded_2718())
                    .collect::<Vec<_>>();
                store_ordered_trie(kv.as_ref(), encoded_transactions.as_slice()).await?;
            }
            HintType::StartingL2Output => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                // Fetch the header for the L2 head block.
                let raw_header: Bytes = providers
                    .l2
                    .client()
                    .request("debug_getRawHeader", &[cfg.agreed_l2_head_hash])
                    .await?;
                let header = Header::decode(&mut raw_header.as_ref())?;

                let rollup_config = rollup_config(cfg)?;
                let Some(withdrawals_root) = header.withdrawals_root else {
                    anyhow::bail!("withdrawals_root is required for post-Isthmus L2 output roots");
                };
                ensure!(
                    rollup_config.is_isthmus_active(header.timestamp),
                    "withdrawals_root is required for post-Isthmus L2 output roots"
                );
                let output_root = OutputRoot::from_parts(
                    header.state_root,
                    withdrawals_root,
                    cfg.agreed_l2_head_hash,
                );
                let output_root_hash = output_root.hash();

                ensure!(
                    output_root_hash == cfg.agreed_l2_output_root,
                    "Output root does not match L2 head."
                );

                let mut kv_write_lock = kv.write().await;
                kv_write_lock.set(
                    PreimageKey::new_keccak256(*output_root_hash).into(),
                    output_root.encode().into(),
                )?;
            }
            HintType::L2Code => {
                anyhow::bail!(
                    "L2Code fallback is disabled; code preimages must be supplied by debug_executePayload"
                );
            }
            HintType::L2StateNode => {
                anyhow::bail!(
                    "L2StateNode fallback is disabled; state node preimages must be supplied by debug_executePayload"
                );
            }
            HintType::L2AccountProof => {
                anyhow::bail!(
                    "L2AccountProof fallback is disabled; account proof preimages must be supplied by debug_executePayload"
                );
            }
            HintType::L2AccountStorageProof => {
                let parsed = parse_l2_account_storage_hint(&hint.data)?;
                ensure!(
                    parsed.address == HISTORY_STORAGE_ADDRESS,
                    "L2AccountStorageProof fallback is disabled; storage proof preimages must be supplied by debug_executePayload"
                );
                let block_hash = match parsed.block {
                    L2AccountStorageHintBlock::Hash(hash) => hash,
                    L2AccountStorageHintBlock::Number(number) => {
                        providers
                            .l2
                            .get_block_by_number(number.into())
                            .await?
                            .ok_or_else(|| anyhow!("Block not found: {number}"))?
                            .header
                            .hash
                    }
                };
                let rollup_config = rollup_config(cfg)?;
                store_history_storage_witness_for_block(
                    &providers.l2,
                    block_hash,
                    parsed.slot,
                    kv.as_ref(),
                    &rollup_config,
                )
                .await?;
            }
            HintType::L2PayloadWitness => {
                ensure!(hint.data.len() >= 32, "Invalid hint data length");

                let parent_block_hash = B256::from_slice(&hint.data.as_ref()[..32]);
                let payload_attributes: OpPayloadAttributes =
                    serde_json::from_slice(&hint.data[32..])?;

                let execute_payload_response = crate::backend::util::fetch_execution_witness(
                    &providers.l2,
                    parent_block_hash,
                    payload_attributes,
                )
                .await?;

                store_execution_witness(kv.as_ref(), execute_payload_response).await?;
            }
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const TEST_HASH: B256 = B256::new([0x42u8; 32]);
    const TEST_TIMESTAMP: u64 = 1234567890;

    // Legacy format: hash (32 bytes) + index (8 bytes) + timestamp (8 bytes) = 48 bytes
    const LEGACY_HINT: [u8; 48] = [
        0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42,
        0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42,
        0x42, 0x42, // Hash (32 bytes):
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFA, 0xCA, // Index (8 bytes, ignored)
        0x00, 0x00, 0x00, 0x00, 0x49, 0x96, 0x02, 0xD2, // Timestamp (8 bytes): 1234567890
    ];

    // New format: hash (32 bytes) + timestamp (8 bytes) = 40 bytes
    const NEW_HINT: [u8; 40] = [
        0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42,
        0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42, 0x42,
        0x42, 0x42, // Hash (32 bytes)
        0x00, 0x00, 0x00, 0x00, 0x49, 0x96, 0x02, 0xD2, // Timestamp (8 bytes): 1234567890
    ];

    #[test]
    fn test_parse_blob_hint_formats() {
        let (legacy_hash, legacy_timestamp) = parse_blob_hint(&LEGACY_HINT).unwrap();
        let (new_hash, new_timestamp) = parse_blob_hint(&NEW_HINT).unwrap();

        assert_eq!(legacy_hash, TEST_HASH);
        assert_eq!(legacy_timestamp, TEST_TIMESTAMP);
        assert_eq!(new_hash, TEST_HASH);
        assert_eq!(new_timestamp, TEST_TIMESTAMP);
    }

    #[test]
    fn test_parse_blob_hint_invalid_length() {
        let hint_data = vec![0u8; 35];
        let result = parse_blob_hint(&hint_data);

        assert!(result.is_err());
        let err_msg = result.unwrap_err().to_string();
        assert!(err_msg.contains("Invalid blob hint length"));
        assert!(err_msg.contains("expected 40 or 48 bytes"));
        assert!(err_msg.contains("got 35"));
    }

    #[test]
    fn history_storage_hint_detects_eip2935_address() {
        let mut hint_data = Vec::new();
        hint_data.extend_from_slice(TEST_HASH.as_slice());
        hint_data.extend_from_slice(HISTORY_STORAGE_ADDRESS.as_slice());
        hint_data.extend_from_slice(B256::with_last_byte(0x42).as_slice());

        let parsed = parse_l2_account_storage_hint(&hint_data).unwrap();
        assert!(is_history_storage_hint(&hint_data).unwrap());
        assert_eq!(parsed.slot, U256::from(0x42));
    }

    #[test]
    fn history_storage_hint_ignores_other_addresses() {
        let mut hint_data = Vec::new();
        hint_data.extend_from_slice(TEST_HASH.as_slice());
        hint_data.extend_from_slice(Address::repeat_byte(0x11).as_slice());
        hint_data.extend_from_slice(B256::ZERO.as_slice());

        assert!(!is_history_storage_hint(&hint_data).unwrap());
    }
}
