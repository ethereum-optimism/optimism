//! [`HintHandler`] for the [`InteropHost`].

use super::InteropHost;
use crate::{
    HintHandler, OnlineHostBackend, OnlineHostBackendCfg, PreimageServer, SharedKeyValueStore,
    backend::util::store_ordered_trie,
};
use alloy_consensus::{Header, Sealed};
use alloy_eips::{eip2718::Encodable2718, eip4844::FIELD_ELEMENTS_PER_BLOB};
use alloy_op_evm::OpEvmFactory;
use alloy_primitives::{Address, B256, Bytes, keccak256};
use alloy_provider::Provider;
use alloy_rlp::{Decodable, Encodable};
use alloy_rpc_types::{Block, debug::ExecutionWitness};
use alloy_transport::{RpcError, TransportErrorKind};
use anyhow::{Result, anyhow, ensure};
use ark_ff::{BigInteger, PrimeField};
use async_trait::async_trait;
use kona_derive::EthereumDataSource;
use kona_driver::Driver;
use kona_executor::{BlockBuildingOutcome, TrieDBProvider};
use kona_interop::DependencySet;
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
use kona_providers_alloy::BlobWithCommitmentAndProof;
use kona_registry::{L1_CONFIGS, ROLLUP_CONFIGS};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use std::sync::Arc;
use tokio::task;
use tracing::{Instrument, debug, info, info_span, warn};

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

/// Returns `true` if the RPC error indicates the node does not support the requested method
/// (JSON-RPC error code -32601: Method not found).
const fn is_rpc_method_not_found(e: &RpcError<TransportErrorKind>) -> bool {
    matches!(e, RpcError::ErrorResp(p) if p.code == -32601)
}

/// Resolves a [`RollupConfig`] for the given chain ID, preferring the on-disk configs and
/// falling back to the embedded global registry.
fn resolve_rollup_config(
    cfg: &InteropHost,
    chain_id: u64,
) -> Result<Arc<kona_genesis::RollupConfig>> {
    cfg.read_rollup_configs()
        .transpose()?
        .and_then(|configs| configs.get(&chain_id).cloned())
        .or_else(|| ROLLUP_CONFIGS.get(&chain_id).cloned())
        .map(Arc::new)
        .ok_or_else(|| anyhow!("No rollup config found for chain ID: {chain_id}"))
}

/// Handles an [`HintType::AgreedPreState`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: lock scope is intentionally tight; the helper exits immediately after the write.
async fn handle_agreed_pre_state(
    hint: &Hint<HintType>,
    cfg: &InteropHost,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    let hash: B256 = hint.data.as_ref().try_into()?;

    if hash != keccak256(cfg.agreed_l2_pre_state.as_ref()) {
        anyhow::bail!("Agreed pre-state hash does not match.");
    }

    let mut kv_write_lock = kv.write().await;
    kv_write_lock
        .set(PreimageKey::new_keccak256(*hash).into(), cfg.agreed_l2_pre_state.clone().into())?;
    Ok(())
}

/// Handles an [`HintType::L2BlockHeader`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: lock scope is intentionally tight; the helper exits immediately after the write.
async fn handle_l2_block_header(
    hint: &Hint<HintType>,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    let hash: B256 = hint.data.as_ref()[..32].try_into()?;
    let chain_id = u64::from_be_bytes(hint.data[32..40].try_into()?);

    let raw_header: Bytes =
        providers.l2(&chain_id)?.client().request("debug_getRawHeader", [hash]).await?;

    let mut kv_lock = kv.write().await;
    kv_lock.set(PreimageKey::new_keccak256(*hash).into(), raw_header.into())?;
    Ok(())
}

/// Handles an [`HintType::L2StateNode`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: lock scope is intentionally tight; the helper exits immediately after the write.
async fn handle_l2_state_node(
    hint: &Hint<HintType>,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    let hash: B256 = hint.data.as_ref().try_into()?;
    let chain_id = u64::from_be_bytes(hint.data[32..40].try_into()?);

    let preimage: Bytes = providers.l2(&chain_id)?.client().request("debug_dbGet", &[hash]).await?;

    let mut kv_write_lock = kv.write().await;
    kv_write_lock.set(PreimageKey::new_keccak256(*hash).into(), preimage.into())?;
    Ok(())
}

/// Handles an [`HintType::L1Precompile`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: the write guard must be held across both `set` calls to keep them atomic.
async fn handle_l1_precompile(hint: &Hint<HintType>, kv: &SharedKeyValueStore) -> Result<()> {
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
    kv_lock.set(PreimageKey::new_keccak256(*input_hash).into(), hint.data.clone().into())?;
    kv_lock.set(PreimageKey::new(*input_hash, PreimageKeyType::Precompile).into(), result)?;
    Ok(())
}

/// Handles an [`HintType::L2Code`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: the write guard intentionally extends to the end of the helper.
async fn handle_l2_code(
    hint: &Hint<HintType>,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    // geth hashdb scheme code hash key prefix
    const CODE_PREFIX: u8 = b'c';

    let hash: B256 = B256::from_slice(&hint.data[0..32]);
    let chain_id = u64::from_be_bytes(hint.data[32..40].try_into()?);
    let l2_provider = providers.l2(&chain_id)?;

    let code_key = [&[CODE_PREFIX], hash.as_slice()].concat();
    let code =
        l2_provider.client().request::<&[Bytes; 1], Bytes>("debug_dbGet", &[code_key.into()]).await;

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
    Ok(())
}

/// Handles an [`HintType::L1Blob`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: the write guard must be held across the field-element loop to keep insertions atomic.
async fn handle_l1_blob(
    hint: &Hint<HintType>,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    let (hash, timestamp) = crate::single::parse_blob_hint(&hint.data)?;

    let partial_block_ref = BlockInfo { timestamp, ..Default::default() };

    let mut blobs = providers
        .blobs
        .fetch_blobs_with_proofs(&partial_block_ref, &[hash])
        .await
        .map_err(|e| anyhow!("Failed to fetch blobs with proofs: {e}"))?;

    if blobs.len() != 1 {
        anyhow::bail!("Expected 1 blob, got {}", blobs.len());
    }

    let BlobWithCommitmentAndProof { blob, kzg_proof: proof, kzg_commitment: commitment } =
        blobs.pop().expect("Expected 1 blob");

    let mut kv_lock = kv.write().await;

    kv_lock.set(PreimageKey::new(*hash, PreimageKeyType::Sha256).into(), commitment.to_vec())?;

    let mut blob_key = [0u8; 80];
    blob_key[..48].copy_from_slice(commitment.as_ref());
    #[allow(clippy::cast_possible_truncation)]
    // SAFETY: `i` is bounded by `FIELD_ELEMENTS_PER_BLOB` (4096), well within `usize::MAX`.
    for i in 0..FIELD_ELEMENTS_PER_BLOB {
        blob_key[48..]
            .copy_from_slice(ROOTS_OF_UNITY[i as usize].into_bigint().to_bytes_be().as_ref());
        let blob_key_hash = keccak256(blob_key.as_ref());

        kv_lock.set(PreimageKey::new_keccak256(*blob_key_hash).into(), blob_key.into())?;
        kv_lock.set(
            PreimageKey::new(*blob_key_hash, PreimageKeyType::Blob).into(),
            blob[(i as usize) << 5..(i as usize + 1) << 5].to_vec(),
        )?;
    }

    blob_key[72..].copy_from_slice(FIELD_ELEMENTS_PER_BLOB.to_be_bytes().as_ref());
    let blob_key_hash = keccak256(blob_key.as_ref());

    kv_lock.set(PreimageKey::new_keccak256(*blob_key_hash).into(), blob_key.into())?;
    kv_lock.set(PreimageKey::new(*blob_key_hash, PreimageKeyType::Blob).into(), proof.to_vec())?;
    Ok(())
}

/// Handles an [`HintType::L2OutputRoot`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: lock scope is intentionally tight; the helper exits immediately after the write.
async fn handle_l2_output_root(
    hint: &Hint<HintType>,
    cfg: &InteropHost,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    let hash = B256::from_slice(&hint.data.as_ref()[0..32]);
    let chain_id = u64::from_be_bytes(hint.data.as_ref()[32..40].try_into()?);
    let l2_provider = providers.l2(&chain_id)?;

    // Decode the pre-state to determine the timestamp of the block.
    let pre = PreState::decode(&mut cfg.agreed_l2_pre_state.as_ref())?;
    let timestamp = match pre {
        PreState::SuperRoot(super_root) => super_root.timestamp,
        PreState::TransitionState(transition_state) => transition_state.pre_state.timestamp,
    };

    // Convert the timestamp to an L2 block number, using the rollup config for the chain ID
    // embedded within the hint.
    let rollup_config = resolve_rollup_config(cfg, chain_id)?;
    let block_number = rollup_config.block_number_from_timestamp(timestamp);

    let raw_header: Bytes = l2_provider
        .client()
        .request("debug_getRawHeader", &[format!("0x{block_number:x}")])
        .await
        .map_err(|e| anyhow!("Failed to fetch header RLP: {e}"))?;
    let header = Header::decode(&mut raw_header.as_ref())?;

    let l2_to_l1_message_passer = l2_provider
        .get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Vec::default())
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
    kv_lock
        .set(PreimageKey::new_keccak256(*output_root_hash).into(), output_root.encode().into())?;
    Ok(())
}

/// Handles an [`HintType::L2AccountProof`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: the write guard must be held across the proof-node loop to keep insertions atomic.
async fn handle_l2_account_proof(
    hint: &Hint<HintType>,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    const BLOCK_NUMBER_HINT_LEN: usize = 8 + 20 + 8;
    const BLOCK_HASH_HINT_LEN: usize = 32 + 20 + 8;
    let (block_id, address, chain_id) = match hint.data.len() {
        BLOCK_NUMBER_HINT_LEN => {
            let block_number = u64::from_be_bytes(hint.data.as_ref()[..8].try_into()?);
            let address = Address::from_slice(&hint.data.as_ref()[8..28]);
            let chain_id = u64::from_be_bytes(hint.data.as_ref()[28..36].try_into()?);
            (block_number.into(), address, chain_id)
        }
        BLOCK_HASH_HINT_LEN => {
            let block_hash = B256::from_slice(&hint.data.as_ref()[..32]);
            let address = Address::from_slice(&hint.data.as_ref()[32..52]);
            let chain_id = u64::from_be_bytes(hint.data.as_ref()[52..60].try_into()?);
            (block_hash.into(), address, chain_id)
        }
        other => anyhow::bail!(
            "Invalid L2AccountProof hint length: expected {BLOCK_NUMBER_HINT_LEN} or {BLOCK_HASH_HINT_LEN}, got {other}"
        ),
    };

    let proof_response =
        providers.l2(&chain_id)?.get_proof(address, Vec::default()).block_id(block_id).await?;

    let mut kv_lock = kv.write().await;
    proof_response.account_proof.into_iter().try_for_each(|node| {
        let node_hash = keccak256(node.as_ref());
        let key = PreimageKey::new_keccak256(*node_hash);
        kv_lock.set(key.into(), node.into())?;
        Ok::<(), anyhow::Error>(())
    })?;
    Ok(())
}

/// Handles an [`HintType::L2AccountStorageProof`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: the write guard must be held across both proof-node loops to keep insertions atomic.
async fn handle_l2_account_storage_proof(
    hint: &Hint<HintType>,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    const BLOCK_NUMBER_HINT_LEN: usize = 8 + 20 + 32 + 8;
    const BLOCK_HASH_HINT_LEN: usize = 32 + 20 + 32 + 8;
    let (block_id, address, slot, chain_id) = match hint.data.len() {
        BLOCK_NUMBER_HINT_LEN => {
            let block_number = u64::from_be_bytes(hint.data.as_ref()[..8].try_into()?);
            let address = Address::from_slice(&hint.data.as_ref()[8..28]);
            let slot = B256::from_slice(&hint.data.as_ref()[28..60]);
            let chain_id = u64::from_be_bytes(hint.data.as_ref()[60..68].try_into()?);
            (block_number.into(), address, slot, chain_id)
        }
        BLOCK_HASH_HINT_LEN => {
            let block_hash = B256::from_slice(&hint.data.as_ref()[..32]);
            let address = Address::from_slice(&hint.data.as_ref()[32..52]);
            let slot = B256::from_slice(&hint.data.as_ref()[52..84]);
            let chain_id = u64::from_be_bytes(hint.data.as_ref()[84..92].try_into()?);
            (block_hash.into(), address, slot, chain_id)
        }
        other => anyhow::bail!(
            "Invalid L2AccountStorageProof hint length: expected {BLOCK_NUMBER_HINT_LEN} or {BLOCK_HASH_HINT_LEN}, got {other}"
        ),
    };

    let mut proof_response =
        providers.l2(&chain_id)?.get_proof(address, vec![slot]).block_id(block_id).await?;

    let mut kv_lock = kv.write().await;

    proof_response.account_proof.into_iter().try_for_each(|node| {
        let node_hash = keccak256(node.as_ref());
        let key = PreimageKey::new_keccak256(*node_hash);
        kv_lock.set(key.into(), node.into())?;
        Ok::<(), anyhow::Error>(())
    })?;

    let storage_proof = proof_response.storage_proof.remove(0);
    storage_proof.proof.into_iter().try_for_each(|node| {
        let node_hash = keccak256(node.as_ref());
        let key = PreimageKey::new_keccak256(*node_hash);
        kv_lock.set(key.into(), node.into())?;
        Ok::<(), anyhow::Error>(())
    })?;
    Ok(())
}

/// Drives the in-process fault-proof client to re-execute a disputed block and returns the
/// build outcome along with the raw transactions of the produced block.
#[allow(clippy::too_many_arguments)]
// SAFETY: the re-execution helper threads multiple oracle, config, and dependency parameters
// through to a generic pipeline; bundling them into a struct adds indirection without removing
// the underlying complexity.
async fn drive_optimistic_reexecution(
    preimage_client: kona_preimage::NativeChannel,
    hint_client: kona_preimage::NativeChannel,
    l1_head: B256,
    agreed_block_hash: B256,
    rollup_config: Arc<kona_genesis::RollupConfig>,
    l1_config: Arc<kona_genesis::L1ChainConfig>,
    chain_id: u64,
    dependency_set: Option<Arc<DependencySet>>,
) -> Result<(BlockBuildingOutcome, Vec<alloy_primitives::Bytes>)> {
    let oracle = Arc::new(CachingOracle::new(
        1024,
        OracleReader::new(preimage_client),
        HintWriter::new(hint_client),
    ));

    let mut l1_provider = OracleL1ChainProvider::new(l1_head, oracle.clone());
    let mut l2_provider =
        OracleL2ChainProvider::new(agreed_block_hash, rollup_config.clone(), oracle.clone());
    let beacon = OracleBlobProvider::new(oracle.clone());

    l2_provider.set_chain_id(Some(chain_id));

    let safe_head = l2_provider
        .header_by_hash(agreed_block_hash)
        .map(|header| Sealed::new_unchecked(header, agreed_block_hash))?;
    let target_block = safe_head.number + 1;

    let cursor = new_oracle_pipeline_cursor(
        rollup_config.as_ref(),
        safe_head,
        B256::ZERO,
        &mut l1_provider,
        &mut l2_provider,
    )
    .await?;
    l2_provider.set_cursor(cursor.clone());

    let da_provider =
        EthereumDataSource::new_from_parts(l1_provider.clone(), beacon, &rollup_config);
    let pipeline = OraclePipeline::new(
        rollup_config.clone(),
        l1_config,
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
        None,
    );
    let mut driver = Driver::new(cursor, executor, pipeline);

    driver.advance_to_target(rollup_config.as_ref(), Some(target_block)).await?;

    driver.safe_head_artifacts.ok_or_else(|| anyhow!("No artifacts found for the safe head"))
}

/// Stores the optimistic block, receipts root, and transactions root preimages produced by
/// `drive_optimistic_reexecution`.
async fn store_reexecution_artifacts(
    kv: &SharedKeyValueStore,
    build_outcome: BlockBuildingOutcome,
    raw_transactions: Vec<alloy_primitives::Bytes>,
) -> Result<()> {
    {
        let mut kv_lock = kv.write().await;
        let mut rlp_buf = Vec::with_capacity(build_outcome.header.length());
        build_outcome.header.encode(&mut rlp_buf);
        kv_lock.set(
            PreimageKey::new(*build_outcome.header.hash(), PreimageKeyType::Keccak256).into(),
            rlp_buf,
        )?;
    }

    let raw_receipts = build_outcome
        .execution_result
        .receipts
        .into_iter()
        .map(|receipt| Ok::<_, anyhow::Error>(receipt.encoded_2718()))
        .collect::<Result<Vec<_>>>()?;
    store_ordered_trie(kv.as_ref(), raw_receipts.as_slice()).await?;

    store_ordered_trie(kv.as_ref(), raw_transactions.as_slice()).await?;

    info!(
        target: "interop_hint_handler",
        number = build_outcome.header.number,
        hash = ?build_outcome.header.hash(),
        "Re-executed optimistic block and collected witness"
    );
    Ok(())
}

/// Handles an [`HintType::L2BlockData`] hint by re-executing the disputed block locally to
/// collect witness preimages.
async fn handle_l2_block_data(
    hint: &Hint<HintType>,
    cfg: &InteropHost,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    let agreed_block_hash = B256::from_slice(&hint.data.as_ref()[..32]);
    let disputed_block_hash = B256::from_slice(&hint.data.as_ref()[32..64]);
    let chain_id = u64::from_be_bytes(hint.data.as_ref()[64..72].try_into()?);

    if agreed_block_hash == disputed_block_hash {
        debug!(
            target: "interop_hint_handler",
            chain_id,
            "Chain has not progressed. Skipping block data hint."
        );
        return Ok(());
    }

    let l2_provider = providers.l2(&chain_id)?;
    let rollup_config = resolve_rollup_config(cfg, chain_id)?;

    let l1_config = cfg
        .read_l1_config()
        .or_else(|_| {
            L1_CONFIGS.get(&rollup_config.l1_chain_id).cloned().ok_or_else(|| {
                anyhow!("No L1 config found for chain ID: {}", rollup_config.l1_chain_id)
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

    let host_hint = BidirectionalChannel::new()?;
    let preimage = BidirectionalChannel::new()?;
    let backend =
        OnlineHostBackend::new(cfg.clone(), kv.clone(), providers.clone(), InteropHintHandler);
    let server_task = task::spawn(
        PreimageServer::new(
            OracleServer::new(preimage.host),
            HintReader::new(host_hint.host),
            Arc::new(backend),
        )
        .start(),
    );
    let dependency_set = cfg
        .read_dependency_set()
        .transpose()
        .map_err(|e| anyhow!("failed to read interop dep-set: {e}"))?
        .map(Arc::new);
    let client_task = task::spawn(
        drive_optimistic_reexecution(
            preimage.client,
            host_hint.client,
            cfg.l1_head,
            agreed_block_hash,
            rollup_config.clone(),
            l1_config.clone(),
            chain_id,
            dependency_set,
        )
        .instrument(info_span!(
            "OptimisticBlockReexecution",
            block_number = disputed_block.header.number
        )),
    );

    let (_, client_result) = tokio::try_join!(server_task, client_task)?;
    let (build_outcome, raw_transactions) = client_result?;

    store_reexecution_artifacts(kv, build_outcome, raw_transactions).await
}

/// Handles an [`HintType::L2PayloadWitness`] hint.
#[allow(clippy::significant_drop_tightening)]
// SAFETY: the write guard must be held across the preimage-insertion loop to keep it atomic.
async fn handle_l2_payload_witness(
    hint: &Hint<HintType>,
    cfg: &InteropHost,
    providers: &<InteropHost as OnlineHostBackendCfg>::Providers,
    kv: &SharedKeyValueStore,
) -> Result<()> {
    if !cfg.enable_experimental_witness_endpoint {
        warn!(
            target: "interop_hint_handler",
            "L2PayloadWitness hint was sent, but payload witness is disabled. Skipping hint."
        );
        return Ok(());
    }

    let (parent_block_hash, payload_attributes_bytes, chain_id) =
        parse_l2_payload_witness_hint(&hint.data)?;
    let payload_attributes: OpPayloadAttributes = serde_json::from_slice(payload_attributes_bytes)?;

    let l2_provider = providers.l2(&chain_id)?;

    let execute_payload_response = match l2_provider
        .client()
        .request::<(B256, OpPayloadAttributes), ExecutionWitness>(
            "debug_executePayload",
            (parent_block_hash, payload_attributes),
        )
        .await
    {
        Ok(response) => response,
        Err(e) => {
            info!(
                target: "interop_hint_handler",
                err = %e,
                chain_id,
                method_not_found = is_rpc_method_not_found(&e),
                "debug_executePayload unavailable, skipping witness preimage collection"
            );
            return Ok(());
        }
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
    Ok(())
}

/// The [`HintHandler`] for the [`InteropHost`].
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
            HintType::L1Blob => handle_l1_blob(&hint, providers, &kv).await?,
            HintType::L1Precompile => {
                ensure!(hint.data.len() >= 28, "Invalid hint data length");
                handle_l1_precompile(&hint, &kv).await?;
            }
            HintType::AgreedPreState => {
                ensure!(hint.data.len() == 32, "Invalid hint data length");
                handle_agreed_pre_state(&hint, cfg, &kv).await?;
            }
            HintType::L2OutputRoot => {
                ensure!(hint.data.len() >= 32 && hint.data.len() <= 40, "Invalid hint data length");
                handle_l2_output_root(&hint, cfg, providers, &kv).await?;
            }
            HintType::L2BlockHeader => {
                ensure!(hint.data.len() == 40, "Invalid hint data length");
                handle_l2_block_header(&hint, providers, &kv).await?;
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
                ensure!(hint.data.len() == 40, "Invalid hint data length");
                handle_l2_code(&hint, providers, &kv).await?;
            }
            HintType::L2StateNode => {
                ensure!(hint.data.len() == 40, "Invalid hint data length");
                handle_l2_state_node(&hint, providers, &kv).await?;
            }
            HintType::L2AccountProof => handle_l2_account_proof(&hint, providers, &kv).await?,
            HintType::L2AccountStorageProof => {
                handle_l2_account_storage_proof(&hint, providers, &kv).await?;
            }
            HintType::L2BlockData => handle_l2_block_data(&hint, cfg, providers, &kv).await?,
            HintType::L2PayloadWitness => {
                handle_l2_payload_witness(&hint, cfg, providers, &kv).await?;
            }
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_json_rpc::ErrorPayload;
    use alloy_transport::TransportErrorKind;

    #[test]
    fn test_is_rpc_method_not_found_true() {
        let e = RpcError::<TransportErrorKind>::ErrorResp(ErrorPayload {
            code: -32601,
            message: "method not found".into(),
            data: None,
        });
        assert!(is_rpc_method_not_found(&e));
    }

    #[test]
    fn test_is_rpc_method_not_found_false_wrong_code() {
        let e = RpcError::<TransportErrorKind>::ErrorResp(ErrorPayload {
            code: -32600,
            message: "invalid request".into(),
            data: None,
        });
        assert!(!is_rpc_method_not_found(&e));
    }

    #[test]
    fn test_is_rpc_method_not_found_false_null_resp() {
        let e = RpcError::<TransportErrorKind>::NullResp;
        assert!(!is_rpc_method_not_found(&e));
    }

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
}
