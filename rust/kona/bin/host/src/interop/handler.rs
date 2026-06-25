//! [`HintHandler`] for the [`InteropHost`].

use super::InteropHost;
use crate::{
    HintHandler, OnlineHostBackend, OnlineHostBackendCfg, PreimageServer, SharedKeyValueStore,
    backend::util::{
        store_execution_witness, store_history_storage_witness_for_block, store_ordered_trie,
        store_raw_header,
    },
};
use alloy_consensus::{Header, Sealed};
use alloy_eips::{
    eip2718::Encodable2718, eip2935::HISTORY_STORAGE_ADDRESS, eip4844::FIELD_ELEMENTS_PER_BLOB,
};
use alloy_op_evm::OpEvmFactory;
use alloy_primitives::{Address, B256, Bytes, U256, keccak256};
use alloy_provider::Provider;
use alloy_rlp::{Decodable, Encodable};
use alloy_rpc_types::Block;
use anyhow::{Result, anyhow, ensure};
use ark_ff::{BigInteger, PrimeField};
use async_trait::async_trait;
use kona_derive::EthereumDataSource;
use kona_driver::Driver;
use kona_executor::TrieDBProvider;
use kona_genesis::RollupConfig;
use kona_preimage::{
    BidirectionalChannel, HintReader, HintWriter, OracleReader, OracleServer, PreimageKey,
    PreimageKeyType, VerifyingPreimageFetcher,
};
use kona_proof::{
    CachingOracle, Hint,
    executor::KonaExecutor,
    l1::{OracleBlobProvider, OracleL1ChainProvider, OraclePipeline, ROOTS_OF_UNITY},
    l2::OracleL2ChainProvider,
    sync::new_oracle_pipeline_cursor,
};
use kona_proof_interop::{HintType, PreState};
use kona_protocol::{BlockInfo, OutputRoot};
use kona_providers_alloy::BlobWithCommitmentAndProof;
use kona_registry::{L1_CONFIGS, ROLLUP_CONFIGS};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use std::sync::Arc;
use tokio::task;
use tracing::{Instrument, debug, info, info_span};

/// Parses the binary framing of a [`HintType::L2PayloadWitness`] hint.
///
/// Returns `(parent_block_hash, payload_attributes_bytes, chain_id)`.
///
/// ## Format
/// `[parent_hash: 32][payload_attributes_json: variable][chain_id: 8]`
fn parse_l2_payload_witness_hint(data: &[u8]) -> Result<(B256, &[u8], u64)> {
    ensure!(
        data.len() >= 40,
        "Invalid hint data length: expected at least 40 bytes (32 for hash + 8 for chain_id), got {}",
        data.len()
    );
    let parent_block_hash = B256::from_slice(&data[..32]);
    let chain_id = u64::from_be_bytes(data[data.len() - 8..].try_into()?);
    let payload_attributes_bytes = &data[32..data.len() - 8];
    Ok((parent_block_hash, payload_attributes_bytes, chain_id))
}

enum L2AccountStorageHintBlock {
    Number(u64),
    Hash(B256),
}

struct ParsedL2AccountStorageHint {
    block: L2AccountStorageHintBlock,
    address: Address,
    slot: U256,
    chain_id: u64,
}

fn parse_l2_account_storage_hint(data: &[u8]) -> Result<ParsedL2AccountStorageHint> {
    const BLOCK_NUMBER_HINT_LEN: usize = 8 + 20 + 32 + 8;
    const BLOCK_HASH_HINT_LEN: usize = 32 + 20 + 32 + 8;

    match data.len() {
        BLOCK_NUMBER_HINT_LEN => {
            let block_number = u64::from_be_bytes(data[..8].try_into()?);
            let address = Address::from_slice(&data[8..28]);
            let chain_id = u64::from_be_bytes(data[60..68].try_into()?);
            Ok(ParsedL2AccountStorageHint {
                block: L2AccountStorageHintBlock::Number(block_number),
                address,
                slot: U256::from_be_slice(&data[28..60]),
                chain_id,
            })
        }
        BLOCK_HASH_HINT_LEN => {
            let block_hash = B256::from_slice(&data[..32]);
            let address = Address::from_slice(&data[32..52]);
            let chain_id = u64::from_be_bytes(data[84..92].try_into()?);
            Ok(ParsedL2AccountStorageHint {
                block: L2AccountStorageHintBlock::Hash(block_hash),
                address,
                slot: U256::from_be_slice(&data[52..84]),
                chain_id,
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

fn rollup_config_for_chain(cfg: &InteropHost, chain_id: u64) -> Result<RollupConfig> {
    cfg.read_rollup_configs()
        .transpose()?
        .and_then(|configs| configs.get(&chain_id).cloned())
        .or_else(|| ROLLUP_CONFIGS.get(&chain_id).cloned())
        .ok_or_else(|| anyhow!("No rollup config found for chain ID: {chain_id}"))
}

/// The [`HintHandler`] for the [`InteropHost`].
#[derive(Debug, Clone, Copy)]
pub struct InteropHintHandler;

#[async_trait]
impl HintHandler for InteropHintHandler {
    type Cfg = InteropHost;

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
            HintType::L2Receipts |
            HintType::AgreedPreState |
            HintType::L2OutputRoot |
            HintType::L2BlockData |
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
                let (hash, timestamp) = crate::single::parse_blob_hint(&hint.data)?;

                let partial_block_ref = BlockInfo { timestamp, ..Default::default() };

                // Fetch the blob with proof from the blob provider.
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
                let rollup_config = Arc::new(rollup_config_for_chain(cfg, chain_id)?);
                let block_number = rollup_config.block_number_from_timestamp(timestamp);

                // Fetch the header for the L2 head block.
                let raw_header: Bytes = l2_provider
                    .client()
                    .request("debug_getRawHeader", &[format!("0x{block_number:x}")])
                    .await
                    .map_err(|e| anyhow!("Failed to fetch header RLP: {e}"))?;
                let header = Header::decode(&mut raw_header.as_ref())?;

                let Some(withdrawals_root) = header.withdrawals_root else {
                    anyhow::bail!("L2 output-root preimage is required before Isthmus");
                };
                ensure!(
                    rollup_config.is_isthmus_active(header.timestamp),
                    "L2 output-root preimage is required before Isthmus"
                );
                let output_root =
                    OutputRoot::from_parts(header.state_root, withdrawals_root, header.hash_slow());
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

                store_raw_header(kv.as_ref(), raw_header).await?;
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
                    .ok_or_else(|| anyhow!("Block not found"))?;
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
                let l2_provider = providers.l2(&parsed.chain_id)?;
                let block_hash = match parsed.block {
                    L2AccountStorageHintBlock::Hash(hash) => hash,
                    L2AccountStorageHintBlock::Number(number) => {
                        l2_provider
                            .get_block_by_number(number.into())
                            .await?
                            .ok_or_else(|| anyhow!("Block not found: {number}"))?
                            .header
                            .hash
                    }
                };
                let rollup_config = rollup_config_for_chain(cfg, parsed.chain_id)?;
                store_history_storage_witness_for_block(
                    l2_provider,
                    block_hash,
                    parsed.slot,
                    kv.as_ref(),
                    &rollup_config,
                )
                .await?;
            }
            HintType::L2BlockData => {
                ensure!(hint.data.len() == 72, "Invalid hint data length");

                let agreed_block_hash = B256::from_slice(&hint.data.as_ref()[..32]);
                let disputed_block_hash = B256::from_slice(&hint.data.as_ref()[32..64]);
                let chain_id = u64::from_be_bytes(hint.data.as_ref()[64..72].try_into()?);
                let l2_provider = providers.l2(&chain_id)?;

                // Return early if the agreed and disputed block are the same. This can occur when
                // the chain has not progressed past its prestate, but the super root timestamp has
                // progressed.
                if agreed_block_hash == disputed_block_hash {
                    let raw_header: Bytes = l2_provider
                        .client()
                        .request("debug_getRawHeader", [agreed_block_hash])
                        .await?;
                    store_raw_header(kv.as_ref(), raw_header).await?;

                    debug!(
                        target: "interop_hint_handler",
                        chain_id,
                        "Chain has not progressed. Stored agreed block header and skipped block data hint."
                    );
                    return Ok(());
                }

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
                    .ok_or_else(|| anyhow!("No rollup config found for chain ID: {chain_id}"))?;

                let l1_config = cfg
                    .read_l1_config()
                    .or_else(|_| {
                        L1_CONFIGS.get(&rollup_config.l1_chain_id).cloned().ok_or_else(|| {
                            anyhow!(
                                "No L1 config found for chain ID: {}",
                                rollup_config.l1_chain_id
                            )
                        })
                    })
                    .map(Arc::new)?;

                // Check if the block is canonical before continuing.
                let parent_block = l2_provider
                    .get_block_by_hash(agreed_block_hash)
                    .await?
                    .ok_or_else(|| anyhow!("Block not found."))?;
                let disputed_block = l2_provider
                    .get_block_by_number((parent_block.header.number + 1).into())
                    .await?
                    .ok_or_else(|| anyhow!("Block not found."))?;

                // Return early if the disputed block is canonical - preimages can be fetched
                // through the normal flow.
                if disputed_block.header.hash == disputed_block_hash {
                    let raw_header: Bytes = l2_provider
                        .client()
                        .request("debug_getRawHeader", [disputed_block_hash])
                        .await?;
                    store_raw_header(kv.as_ref(), raw_header).await?;

                    debug!(
                        target: "interop_hint_handler",
                        number = disputed_block.header.number,
                        hash = ?disputed_block.header.hash,
                        "Block is already canonical. Stored header and skipped re-derivation + execution."
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
                        Arc::new(VerifyingPreimageFetcher::new(backend)),
                    )
                    .start(),
                );
                // The host re-execution path mirrors the client's
                // `sub_transition` fault-proof pipeline, so we must pass the same
                // dependency set that the client will derive from BootInfo. Parse
                // it from the same path the KV store uses (`--interop-dep-set-path`)
                // here so the owned value can move into the spawned task.
                let dependency_set = cfg
                    .read_dependency_set()
                    .transpose()
                    .map_err(|e| anyhow!("failed to read interop dep-set: {e}"))?
                    .map(Arc::new);
                let client_task = task::spawn({
                    let l1_head = cfg.l1_head;
                    let dependency_set = dependency_set.clone();

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

                        // The output root is unused in the host re-execution context,
                        // which only collects preimages for witness generation.
                        let cursor = new_oracle_pipeline_cursor(
                            rollup_config.as_ref(),
                            safe_head,
                            B256::ZERO,
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
                            dependency_set,
                        )
                        .await?;
                        let executor = KonaExecutor::new(
                            rollup_config.as_ref(),
                            l2_provider.clone(),
                            l2_provider,
                            OpEvmFactory::<alloy_op_evm::OpTx>::default(),
                            alloy_op_evm::block::OpAlloyReceiptBuilder::default(),
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
                // 1. Parse hint data
                let (parent_block_hash, payload_attributes_bytes, chain_id) =
                    parse_l2_payload_witness_hint(&hint.data)?;
                let payload_attributes: OpPayloadAttributes =
                    serde_json::from_slice(payload_attributes_bytes)?;

                // 2. Route to correct L2 provider
                let l2_provider = providers.l2(&chain_id)?;

                // 3. Call debug_executePayload RPC.
                let execute_payload_response = crate::backend::util::fetch_execution_witness(
                    l2_provider,
                    parent_block_hash,
                    payload_attributes,
                )
                .await?;

                // 4. Store preimages in KV store
                store_execution_witness(kv.as_ref(), execute_payload_response).await?;
            }
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_hint(parent_hash: B256, json: &[u8], chain_id: u64) -> Vec<u8> {
        let mut data = Vec::new();
        data.extend_from_slice(parent_hash.as_slice());
        data.extend_from_slice(json);
        data.extend_from_slice(&chain_id.to_be_bytes());
        data
    }

    #[test]
    fn test_parse_l2_payload_witness_hint() {
        let parent_hash = B256::from([0x42u8; 32]);
        let json = b"{\"key\":\"value\"}";
        let chain_id = 10u64;

        let hint_data = make_hint(parent_hash, json, chain_id);

        let (parsed_hash, parsed_json, parsed_chain_id) =
            parse_l2_payload_witness_hint(&hint_data).unwrap();
        assert_eq!(parsed_hash, parent_hash);
        assert_eq!(parsed_json, json);
        assert_eq!(parsed_chain_id, chain_id);
    }

    #[test]
    fn test_parse_l2_payload_witness_hint_too_short() {
        let hint_data = vec![0u8; 39];
        let err = parse_l2_payload_witness_hint(&hint_data).unwrap_err();
        assert!(err.to_string().contains("Invalid hint data length"));
    }

    #[test]
    fn test_parse_l2_payload_witness_hint_various_chain_ids() {
        let parent_hash = B256::from([0xAAu8; 32]);
        for chain_id in [1u64, 10, 8453, u64::MAX] {
            let hint_data = make_hint(parent_hash, b"{}", chain_id);
            let (_, _, parsed_chain_id) = parse_l2_payload_witness_hint(&hint_data).unwrap();
            assert_eq!(parsed_chain_id, chain_id);
        }
    }

    #[test]
    fn history_storage_hint_detects_eip2935_address_and_chain_id() {
        let block_hash = B256::from([0x42u8; 32]);
        let chain_id = 10u64;
        let mut hint_data = Vec::new();
        hint_data.extend_from_slice(block_hash.as_slice());
        hint_data.extend_from_slice(HISTORY_STORAGE_ADDRESS.as_slice());
        hint_data.extend_from_slice(B256::with_last_byte(0x42).as_slice());
        hint_data.extend_from_slice(&chain_id.to_be_bytes());

        let parsed = parse_l2_account_storage_hint(&hint_data).unwrap();

        assert!(is_history_storage_hint(&hint_data).unwrap());
        assert_eq!(parsed.address, HISTORY_STORAGE_ADDRESS);
        assert_eq!(parsed.slot, U256::from(0x42));
        assert_eq!(parsed.chain_id, chain_id);
    }

    #[test]
    fn history_storage_hint_ignores_other_addresses() {
        let block_hash = B256::from([0x42u8; 32]);
        let mut hint_data = Vec::new();
        hint_data.extend_from_slice(block_hash.as_slice());
        hint_data.extend_from_slice(Address::repeat_byte(0x11).as_slice());
        hint_data.extend_from_slice(B256::ZERO.as_slice());
        hint_data.extend_from_slice(&10u64.to_be_bytes());

        assert!(!is_history_storage_hint(&hint_data).unwrap());
    }
}
