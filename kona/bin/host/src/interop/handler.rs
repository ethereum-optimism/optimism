//! [HintHandler] for the [InteropHost].

use super::InteropHost;
use crate::{
    HintHandler, OnlineHostBackend, OnlineHostBackendCfg, PreimageServer, SharedKeyValueStore,
    backend::util::store_ordered_trie,
};
use alloy_consensus::{Header, Sealed};
use alloy_eips::{
    eip2718::Encodable2718,
    eip4844::{BlobTransactionSidecarItem, FIELD_ELEMENTS_PER_BLOB, IndexedBlobHash},
};
use alloy_op_evm::OpEvmFactory;
use alloy_primitives::{Address, B256, Bytes, keccak256};
use alloy_provider::Provider;
use alloy_rlp::{Decodable, Encodable};
use alloy_rpc_types::Block;
use anyhow::{Result, anyhow, ensure};
use ark_ff::{BigInteger, PrimeField};
use async_trait::async_trait;
use kona_derive::EthereumDataSource;
use kona_driver::Driver;
use kona_executor::TrieDBProvider;
use kona_preimage::{
    BidirectionalChannel, HintReader, HintWriter, OracleReader, OracleServer, PreimageKey,
    PreimageKeyType,
};
use kona_proof::{
    CachingOracle, Hint,
    executor::KonaExecutor,
    l1::{OracleBlobProvider, OracleL1ChainProvider, OraclePipeline, ROOTS_OF_UNITY},
    l2::OracleL2ChainProvider,
    sync::new_oracle_pipeline_cursor,
};
use kona_proof_interop::{HintType, PreState};
use kona_protocol::{BlockInfo, OutputRoot, Predeploys};
use kona_registry::{L1_CONFIGS, ROLLUP_CONFIGS};
use std::sync::Arc;
use tokio::task;
use tracing::{Instrument, debug, info, info_span, warn};

/// The [HintHandler] for the [InteropHost].
#[derive(Debug, Clone, Copy)]
pub struct InteropHintHandler;

#[async_trait]
impl HintHandler for InteropHintHandler {
    type Cfg = InteropHost;

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

                // Fetch the blob sidecar from the blob provider.
                let mut sidecars = providers
                    .blobs
                    .fetch_filtered_blob_sidecars(&partial_block_ref, &[indexed_hash])
                    .await
                    .map_err(|e| anyhow!("Failed to fetch blob sidecars: {e}"))?;

                if sidecars.len() != 1 {
                    anyhow::bail!("Expected 1 sidecar, got {}", sidecars.len());
                }

                let BlobTransactionSidecarItem {
                    blob,
                    kzg_proof: proof,
                    kzg_commitment: commitment,
                    ..
                } = sidecars.pop().expect("Expected 1 sidecar");

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
                blob_key[72..].copy_from_slice((FIELD_ELEMENTS_PER_BLOB).to_be_bytes().as_ref());
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
            HintType::AgreedPreState => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;

                if hash != keccak256(cfg.agreed_l2_pre_state.as_ref()) {
                    anyhow::bail!("Agreed pre-state hash does not match.");
                }

                let mut kv_write_lock = kv.write().await;
                kv_write_lock.set(
                    PreimageKey::new_keccak256(*hash).into(),
                    cfg.agreed_l2_pre_state.clone().into(),
                )?;
            }
            HintType::L2OutputRoot => {
                ensure!(hint.data.len() >= 32 && hint.data.len() <= 40, "Invalid hint data length");

                let hash = B256::from_slice(&hint.data.as_ref()[0..32]);
                let chain_id = u64::from_be_bytes(hint.data.as_ref()[32..40].try_into()?);
                let l2_provider = providers.l2(&chain_id)?;

                // Decode the pre-state to determine the timestamp of the block.
                let pre = PreState::decode(&mut cfg.agreed_l2_pre_state.as_ref())?;
                let timestamp = match pre {
                    PreState::SuperRoot(super_root) => super_root.timestamp,
                    PreState::TransitionState(transition_state) => {
                        transition_state.pre_state.timestamp
                    }
                };

                // Convert the timestamp to an L2 block number, using the rollup config for the
                // chain ID embedded within the hint.
                let rollup_config = cfg
                    .read_rollup_configs()
                    // If an error occurred while reading the rollup configs, return the error.
                    .transpose()?
                    // Try to find the appropriate rollup config for the chain ID.
                    .and_then(|configs| configs.get(&chain_id).cloned())
                    // If we can't find the rollup config, try to find it in the global rollup
                    // configs.
                    .or_else(|| ROLLUP_CONFIGS.get(&chain_id).cloned())
                    .map(Arc::new)
                    .ok_or(anyhow!("No rollup config found for chain ID: {chain_id}"))?;
                let block_number = rollup_config.block_number_from_timestamp(timestamp);

                // Fetch the header for the L2 head block.
                let raw_header: Bytes = l2_provider
                    .client()
                    .request("debug_getRawHeader", &[format!("0x{block_number:x}")])
                    .await
                    .map_err(|e| anyhow!("Failed to fetch header RLP: {e}"))?;
                let header = Header::decode(&mut raw_header.as_ref())?;

                // Fetch the storage root for the L2 head block.
                let l2_to_l1_message_passer = l2_provider
                    .get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Default::default())
                    .block_id(block_number.into())
                    .await?;

                let output_root = OutputRoot::from_parts(
                    header.state_root,
                    l2_to_l1_message_passer.storage_hash,
                    header.hash_slow(),
                );
                let output_root_hash = output_root.hash();

                ensure!(
                    output_root_hash == hash,
                    "Output root does not match L2 head. Expected: {hash}, got: {output_root_hash}"
                );

                let mut kv_lock = kv.write().await;
                kv_lock.set(
                    PreimageKey::new_keccak256(*output_root_hash).into(),
                    output_root.encode().into(),
                )?;
            }
            HintType::L2BlockHeader => {
                ensure!(hint.data.len() == 40, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref()[..32].try_into()?;
                let chain_id = u64::from_be_bytes(hint.data[32..40].try_into()?);

                let raw_header: Bytes =
                    providers.l2(&chain_id)?.client().request("debug_getRawHeader", [hash]).await?;

                let mut kv_lock = kv.write().await;
                kv_lock.set(PreimageKey::new_keccak256(*hash).into(), raw_header.into())?;
            }
            HintType::L2Transactions => {
                ensure!(hint.data.len() == 40, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref()[..32].try_into()?;
                let chain_id = u64::from_be_bytes(hint.data[32..40].try_into()?);

                let Block { transactions, .. } = providers
                    .l2(&chain_id)?
                    .get_block_by_hash(hash)
                    .full()
                    .await?
                    .ok_or(anyhow!("Block not found"))?;
                let encoded_transactions = transactions
                    .into_transactions()
                    .map(|tx| tx.inner.inner.encoded_2718())
                    .collect::<Vec<_>>();

                store_ordered_trie(kv.as_ref(), encoded_transactions.as_slice()).await?;
            }
            HintType::L2Receipts => {
                ensure!(hint.data.len() == 40, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref()[..32].try_into()?;
                let chain_id = u64::from_be_bytes(hint.data[32..40].try_into()?);

                let raw_receipts: Vec<Bytes> = providers
                    .l2(&chain_id)?
                    .client()
                    .request("debug_getRawReceipts", [hash])
                    .await?;

                store_ordered_trie(kv.as_ref(), raw_receipts.as_slice()).await?;
            }
            HintType::L2Code => {
                // geth hashdb scheme code hash key prefix
                const CODE_PREFIX: u8 = b'c';

                ensure!(hint.data.len() == 40, "Invalid hint data length");

                let hash: B256 = B256::from_slice(&hint.data[0..32]);
                let chain_id = u64::from_be_bytes(hint.data[32..40].try_into()?);
                let l2_provider = providers.l2(&chain_id)?;

                // Attempt to fetch the code from the L2 chain provider.
                let code_key = [&[CODE_PREFIX], hash.as_slice()].concat();
                let code = l2_provider
                    .client()
                    .request::<&[Bytes; 1], Bytes>("debug_dbGet", &[code_key.into()])
                    .await;

                // Check if the first attempt to fetch the code failed. If it did, try fetching the
                // code hash preimage without the geth hashdb scheme prefix.
                let code = match code {
                    Ok(code) => code,
                    Err(_) => l2_provider
                        .client()
                        .request::<&[B256; 1], Bytes>("debug_dbGet", &[hash])
                        .await
                        .map_err(|e| anyhow!("Error fetching code hash preimage: {e}"))?,
                };

                let mut kv_lock = kv.write().await;
                kv_lock.set(PreimageKey::new_keccak256(*hash).into(), code.into())?;
            }
            HintType::L2StateNode => {
                ensure!(hint.data.len() == 40, "Invalid hint data length");

                let hash: B256 = hint.data.as_ref().try_into()?;
                let chain_id = u64::from_be_bytes(hint.data[32..40].try_into()?);

                // Fetch the preimage from the L2 chain provider.
                let preimage: Bytes =
                    providers.l2(&chain_id)?.client().request("debug_dbGet", &[hash]).await?;

                let mut kv_write_lock = kv.write().await;
                kv_write_lock.set(PreimageKey::new_keccak256(*hash).into(), preimage.into())?;
            }
            HintType::L2AccountProof => {
                ensure!(hint.data.len() == 8 + 20 + 8, "Invalid hint data length");

                let block_number = u64::from_be_bytes(hint.data.as_ref()[..8].try_into()?);
                let address = Address::from_slice(&hint.data.as_ref()[8..28]);
                let chain_id = u64::from_be_bytes(hint.data[28..].try_into()?);

                let proof_response = providers
                    .l2(&chain_id)?
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
                ensure!(hint.data.len() == 8 + 20 + 32 + 8, "Invalid hint data length");

                let block_number = u64::from_be_bytes(hint.data.as_ref()[..8].try_into()?);
                let address = Address::from_slice(&hint.data.as_ref()[8..28]);
                let slot = B256::from_slice(&hint.data.as_ref()[28..60]);
                let chain_id = u64::from_be_bytes(hint.data[60..].try_into()?);

                let mut proof_response = providers
                    .l2(&chain_id)?
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
            HintType::L2BlockData => {
                ensure!(hint.data.len() == 72, "Invalid hint data length");

                let agreed_block_hash = B256::from_slice(&hint.data.as_ref()[..32]);
                let disputed_block_hash = B256::from_slice(&hint.data.as_ref()[32..64]);
                let chain_id = u64::from_be_bytes(hint.data.as_ref()[64..72].try_into()?);

                // Return early if the agreed and disputed block are the same. This can occur when
                // the chain has not progressed past its prestate, but the super root timestamp has
                // progressed.
                if agreed_block_hash == disputed_block_hash {
                    debug!(
                        target: "interop_hint_handler",
                        chain_id,
                        "Chain has not progressed. Skipping block data hint."
                    );
                    return Ok(());
                }

                let l2_provider = providers.l2(&chain_id)?;
                let rollup_config = cfg
                    .read_rollup_configs()
                    // If an error occurred while reading the rollup configs, return the error.
                    .transpose()?
                    // Try to find the appropriate rollup config for the chain ID.
                    .and_then(|configs| configs.get(&chain_id).cloned())
                    // If we can't find the rollup config, try to find it in the global rollup
                    // configs.
                    .or_else(|| ROLLUP_CONFIGS.get(&chain_id).cloned())
                    .map(Arc::new)
                    .ok_or(anyhow!("No rollup config found for chain ID: {chain_id}"))?;

                let l1_config = cfg
                    .read_l1_configs()
                    // If an error occurred while reading the l1 configs, return the error.
                    .transpose()?
                    // Try to find the appropriate l1 config for the chain ID.
                    .and_then(|configs| configs.get(&rollup_config.l1_chain_id).cloned())
                    // If we can't find the l1 config, try to find it in the global l1 configs.
                    .or_else(|| L1_CONFIGS.get(&rollup_config.l1_chain_id).cloned())
                    .map(Arc::new)
                    .ok_or(anyhow!(
                        "No l1 config found for chain ID: {}",
                        rollup_config.l1_chain_id
                    ))?;

                // Check if the block is canonical before continuing.
                let parent_block = l2_provider
                    .get_block_by_hash(agreed_block_hash)
                    .await?
                    .ok_or(anyhow!("Block not found."))?;
                let disputed_block = l2_provider
                    .get_block_by_number((parent_block.header.number + 1).into())
                    .await?
                    .ok_or(anyhow!("Block not found."))?;

                // Return early if the disputed block is canonical - preimages can be fetched
                // through the normal flow.
                if disputed_block.header.hash == disputed_block_hash {
                    debug!(
                        target: "interop_hint_handler",
                        number = disputed_block.header.number,
                        hash = ?disputed_block.header.hash,
                        "Block is already canonical. Skipping re-derivation + execution."
                    );
                    return Ok(());
                }

                info!(
                    target: "interop_hint_handler",
                    optimistic_hash = ?disputed_block_hash,
                    "Re-executing optimistic block for witness collection"
                );

                // Reproduce the preimages for the optimistic block's derivation + execution and
                // store them in the key-value store.
                let hint = BidirectionalChannel::new()?;
                let preimage = BidirectionalChannel::new()?;
                let backend =
                    OnlineHostBackend::new(cfg.clone(), kv.clone(), providers.clone(), Self);
                let server_task = task::spawn(
                    PreimageServer::new(
                        OracleServer::new(preimage.host),
                        HintReader::new(hint.host),
                        Arc::new(backend),
                    )
                    .start(),
                );
                let client_task = task::spawn({
                    let l1_head = cfg.l1_head;

                    async move {
                        let oracle = Arc::new(CachingOracle::new(
                            1024,
                            OracleReader::new(preimage.client),
                            HintWriter::new(hint.client),
                        ));

                        let mut l1_provider = OracleL1ChainProvider::new(l1_head, oracle.clone());
                        let mut l2_provider = OracleL2ChainProvider::new(
                            agreed_block_hash,
                            rollup_config.clone(),
                            oracle.clone(),
                        );
                        let beacon = OracleBlobProvider::new(oracle.clone());

                        l2_provider.set_chain_id(Some(chain_id));

                        let safe_head = l2_provider
                            .header_by_hash(agreed_block_hash)
                            .map(|header| Sealed::new_unchecked(header, agreed_block_hash))?;
                        let target_block = safe_head.number + 1;

                        let cursor = new_oracle_pipeline_cursor(
                            rollup_config.as_ref(),
                            safe_head,
                            &mut l1_provider,
                            &mut l2_provider,
                        )
                        .await?;
                        l2_provider.set_cursor(cursor.clone());

                        let da_provider = EthereumDataSource::new_from_parts(
                            l1_provider.clone(),
                            beacon,
                            &rollup_config,
                        );
                        let pipeline = OraclePipeline::new(
                            rollup_config.clone(),
                            l1_config.clone(),
                            cursor.clone(),
                            oracle,
                            da_provider,
                            l1_provider,
                            l2_provider.clone(),
                        )
                        .await?;
                        let executor = KonaExecutor::new(
                            rollup_config.as_ref(),
                            l2_provider.clone(),
                            l2_provider,
                            OpEvmFactory::default(),
                            None,
                        );
                        let mut driver = Driver::new(cursor, executor, pipeline);

                        driver
                            .advance_to_target(rollup_config.as_ref(), Some(target_block))
                            .await?;

                        driver
                            .safe_head_artifacts
                            .ok_or_else(|| anyhow!("No artifacts found for the safe head"))
                    }
                    .instrument(info_span!(
                        "OptimisticBlockReexecution",
                        block_number = disputed_block.header.number
                    ))
                });

                // Wait on both the server and client tasks to complete.
                let (_, client_result) = tokio::try_join!(server_task, client_task)?;
                let (build_outcome, raw_transactions) = client_result?;

                // Store optimistic block hash preimage.
                let mut kv_lock = kv.write().await;
                let mut rlp_buf = Vec::with_capacity(build_outcome.header.length());
                build_outcome.header.encode(&mut rlp_buf);
                kv_lock.set(
                    PreimageKey::new(*build_outcome.header.hash(), PreimageKeyType::Keccak256)
                        .into(),
                    rlp_buf,
                )?;

                // Drop the lock on the key-value store to avoid deadlocks.
                drop(kv_lock);

                // Store receipts root preimages.
                let raw_receipts = build_outcome
                    .execution_result
                    .receipts
                    .into_iter()
                    .map(|receipt| Ok::<_, anyhow::Error>(receipt.encoded_2718()))
                    .collect::<Result<Vec<_>>>()?;
                store_ordered_trie(kv.as_ref(), raw_receipts.as_slice()).await?;

                // Store tx root preimages.
                store_ordered_trie(kv.as_ref(), raw_transactions.as_slice()).await?;

                info!(
                    target: "interop_hint_handler",
                    number = build_outcome.header.number,
                    hash = ?build_outcome.header.hash(),
                    "Re-executed optimistic block and collected witness"
                );
            }
            HintType::L2PayloadWitness => {
                warn!(
                    target: "interop_hint_handler",
                    "L2PayloadWitness hint not implemented for interop hint handler, ignoring hint"
                );
            }
        }

        Ok(())
    }
}
