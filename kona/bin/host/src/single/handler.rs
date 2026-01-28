//! [HintHandler] for the [SingleChainHost].

use crate::{
    HintHandler, OnlineHostBackendCfg, backend::util::store_ordered_trie, kv::SharedKeyValueStore,
    single::cfg::SingleChainHost,
};
use alloy_consensus::Header;
use alloy_eips::{
    eip2718::Encodable2718,
    eip4844::{BlobTransactionSidecarItem, FIELD_ELEMENTS_PER_BLOB, IndexedBlobHash},
};
use alloy_primitives::{Address, B256, Bytes, keccak256};
use alloy_provider::Provider;
use alloy_rlp::Decodable;
use alloy_rpc_types::{Block, debug::ExecutionWitness};
use anyhow::{Result, anyhow, ensure};
use ark_ff::{BigInteger, PrimeField};
use async_trait::async_trait;
use kona_preimage::{PreimageKey, PreimageKeyType};
use kona_proof::{Hint, HintType, l1::ROOTS_OF_UNITY};
use kona_protocol::{BlockInfo, OutputRoot, Predeploys};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use tracing::warn;

/// The [HintHandler] for the [SingleChainHost].
#[derive(Debug, Clone, Copy)]
pub struct SingleChainHintHandler;

#[async_trait]
impl HintHandler for SingleChainHintHandler {
    type Cfg = SingleChainHost;

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

                let mut kv_lock = kv.write().await;
                kv_lock.set(PreimageKey::new_keccak256(*hash).into(), raw_header.into())?;
            }
            HintType::L1Transactions => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;
                let Block { transactions, .. } = providers
                    .l1
                    .get_block_by_hash(hash)
                    .full()
                    .await?
                    .ok_or(anyhow!("Block not found"))?;
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
                ensure!(hint.data.len() == 48, "Invalid hint data length");

                let hash_data_bytes: [u8; 32] = hint.data[0..32].try_into()?;
                let index_data_bytes: [u8; 8] = hint.data[32..40].try_into()?;
                let timestamp_data_bytes: [u8; 8] = hint.data[40..48].try_into()?;

                let hash: B256 = hash_data_bytes.into();
                let index = u64::from_be_bytes(index_data_bytes);
                let timestamp = u64::from_be_bytes(timestamp_data_bytes);

                let partial_block_ref = BlockInfo { timestamp, ..Default::default() };
                let indexed_hash = IndexedBlobHash { index, hash };

                // Fetch the blobs from the blob provider.
                let mut blobs = providers
                    .blobs
                    .fetch_filtered_blob_sidecars(&partial_block_ref, &[indexed_hash])
                    .await
                    .map_err(|e| anyhow!("Failed to fetch blob sidecars: {e}"))?;
                if blobs.len() != 1 {
                    anyhow::bail!("Expected 1 blob, got {}", blobs.len());
                }
                let BlobTransactionSidecarItem {
                    blob,
                    kzg_proof: proof,
                    kzg_commitment: commitment,
                    ..
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

                // Acquire a lock on the key-value store and set the preimage.
                let mut kv_lock = kv.write().await;
                kv_lock.set(PreimageKey::new_keccak256(*hash).into(), raw_header.into())?;
            }
            HintType::L2Transactions => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;
                let Block { transactions, .. } = providers
                    .l2
                    .get_block_by_hash(hash)
                    .full()
                    .await?
                    .ok_or(anyhow!("Block not found."))?;

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

                // Fetch the storage root for the L2 head block.
                let l2_to_l1_message_passer = providers
                    .l2
                    .get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Default::default())
                    .block_id(cfg.agreed_l2_head_hash.into())
                    .await?;

                let output_root = OutputRoot::from_parts(
                    header.state_root,
                    l2_to_l1_message_passer.storage_hash,
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
                // geth hashdb scheme code hash key prefix
                const CODE_PREFIX: u8 = b'c';

                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;

                // Attempt to fetch the code from the L2 chain provider.
                let code_key = [&[CODE_PREFIX], hash.as_slice()].concat();
                let code = providers
                    .l2
                    .client()
                    .request::<&[Bytes; 1], Bytes>("debug_dbGet", &[code_key.into()])
                    .await;

                // Check if the first attempt to fetch the code failed. If it did, try fetching the
                // code hash preimage without the geth hashdb scheme prefix.
                let code = match code {
                    Ok(code) => code,
                    Err(_) => providers
                        .l2
                        .client()
                        .request::<&[B256; 1], Bytes>("debug_dbGet", &[hash])
                        .await
                        .map_err(|e| anyhow!("Error fetching code hash preimage: {e}"))?,
                };

                let mut kv_lock = kv.write().await;
                kv_lock.set(PreimageKey::new_keccak256(*hash).into(), code.into())?;
            }
            HintType::L2StateNode => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;

                warn!(target: "single_hint_handler", "L2StateNode hint was sent for node hash: {}", hash);
                warn!(
                    target: "single_hint_handler",
                    "`debug_executePayload` failed to return a complete witness."
                );

                // Fetch the preimage from the L2 chain provider.
                let preimage: Bytes = providers.l2.client().request("debug_dbGet", &[hash]).await?;

                let mut kv_write_lock = kv.write().await;
                kv_write_lock.set(PreimageKey::new_keccak256(*hash).into(), preimage.into())?;
            }
            HintType::L2AccountProof => {
                ensure!(hint.data.len() == 8 + 20, "Invalid hint data length");

                let block_number = u64::from_be_bytes(hint.data.as_ref()[..8].try_into()?);
                let address = Address::from_slice(&hint.data.as_ref()[8..28]);

                let proof_response = providers
                    .l2
                    .get_proof(address, Default::default())
                    .block_id(block_number.into())
                    .await?;

                // Write the account proof nodes to the key-value store.
                let mut kv_lock = kv.write().await;
                proof_response.account_proof.into_iter().try_for_each(|node| {
                    let node_hash = keccak256(node.as_ref());
                    let key = PreimageKey::new_keccak256(*node_hash);
                    kv_lock.set(key.into(), node.into())?;
                    Ok::<(), anyhow::Error>(())
                })?;
            }
            HintType::L2AccountStorageProof => {
                ensure!(hint.data.len() == 8 + 20 + 32, "Invalid hint data length");

                let block_number = u64::from_be_bytes(hint.data.as_ref()[..8].try_into()?);
                let address = Address::from_slice(&hint.data.as_ref()[8..28]);
                let slot = B256::from_slice(&hint.data.as_ref()[28..]);

                let mut proof_response = providers
                    .l2
                    .get_proof(address, vec![slot])
                    .block_id(block_number.into())
                    .await?;

                let mut kv_lock = kv.write().await;

                // Write the account proof nodes to the key-value store.
                proof_response.account_proof.into_iter().try_for_each(|node| {
                    let node_hash = keccak256(node.as_ref());
                    let key = PreimageKey::new_keccak256(*node_hash);
                    kv_lock.set(key.into(), node.into())?;
                    Ok::<(), anyhow::Error>(())
                })?;

                // Write the storage proof nodes to the key-value store.
                let storage_proof = proof_response.storage_proof.remove(0);
                storage_proof.proof.into_iter().try_for_each(|node| {
                    let node_hash = keccak256(node.as_ref());
                    let key = PreimageKey::new_keccak256(*node_hash);
                    kv_lock.set(key.into(), node.into())?;
                    Ok::<(), anyhow::Error>(())
                })?;
            }
            HintType::L2PayloadWitness => {
                if !cfg.enable_experimental_witness_endpoint {
                    warn!(
                        target: "single_hint_handler",
                        "L2PayloadWitness hint was sent, but payload witness is disabled. Skipping hint."
                    );
                    return Ok(());
                }

                ensure!(hint.data.len() >= 32, "Invalid hint data length");

                let parent_block_hash = B256::from_slice(&hint.data.as_ref()[..32]);
                let payload_attributes: OpPayloadAttributes =
                    serde_json::from_slice(&hint.data[32..])?;

                let Ok(execute_payload_response) = providers
                    .l2
                    .client()
                    .request::<(B256, OpPayloadAttributes), ExecutionWitness>(
                        "debug_executePayload",
                        (parent_block_hash, payload_attributes),
                    )
                    .await
                else {
                    // Allow this hint to fail silently, as not all execution clients support
                    // the `debug_executePayload` method.
                    return Ok(());
                };

                let preimages = execute_payload_response
                    .state
                    .into_iter()
                    .chain(execute_payload_response.codes)
                    .chain(execute_payload_response.keys);

                let mut kv_lock = kv.write().await;
                for preimage in preimages {
                    let computed_hash = keccak256(preimage.as_ref());

                    let key = PreimageKey::new_keccak256(*computed_hash);
                    kv_lock.set(key.into(), preimage.into())?;
                }
            }
        }

        Ok(())
    }
}
