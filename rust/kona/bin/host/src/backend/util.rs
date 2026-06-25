//! Utilities for the preimage server backend.

use crate::{KeyValueStore, Result};
use alloy_consensus::{EMPTY_ROOT_HASH, Header};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, keccak256};
use alloy_provider::{Provider, RootProvider};
use alloy_rlp::{Decodable, EMPTY_STRING_CODE};
use alloy_rpc_types::debug::ExecutionWitness;
use anyhow::ensure;
use kona_genesis::RollupConfig;
use kona_preimage::{PreimageKey, PreimageKeyType};
use kona_proof::payload_attributes_from_block_header;
use op_alloy_consensus::TxDeposit;
use op_alloy_network::Optimism;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use tokio::sync::RwLock;

const EIP2935_HISTORY_SERVE_WINDOW: u64 = (1 << 13) - 1;

/// Fetches a block's execution witness via `debug_executePayload`.
///
/// Any failure — including "method not found" — is returned as an error rather than swallowed. The
/// backend surfaces eager payload-witness errors as soon as the hint is routed, so hosts fail fast
/// when the configured L2 RPC does not support `debug_executePayload`.
pub(crate) async fn fetch_execution_witness(
    provider: &RootProvider<Optimism>,
    parent_block_hash: B256,
    payload_attributes: OpPayloadAttributes,
) -> anyhow::Result<ExecutionWitness> {
    provider
        .client()
        .request::<(B256, OpPayloadAttributes), ExecutionWitness>(
            "debug_executePayload",
            (parent_block_hash, payload_attributes),
        )
        .await
        .map_err(|e| anyhow::anyhow!("debug_executePayload failed: {e}"))
}

/// Replays synthetic payloads that touch an EIP-2935 history storage slot, then stores the
/// resulting witnesses.
pub(crate) async fn store_history_storage_witness_for_block<KV: KeyValueStore + ?Sized>(
    provider: &RootProvider<Optimism>,
    block_hash: B256,
    slot: U256,
    kv: &RwLock<KV>,
    rollup_config: &RollupConfig,
) -> anyhow::Result<()> {
    let raw_header: Bytes = provider.client().request("debug_getRawHeader", [block_hash]).await?;
    let header_hash = store_raw_header(kv, raw_header.clone()).await?;
    ensure!(
        header_hash == block_hash,
        "debug_getRawHeader returned a header with an unexpected hash"
    );
    let header = Header::decode(&mut raw_header.as_ref())?;

    let target_block_number = eip2935_block_number_for_slot(&header, slot)?;
    let transaction = eip2935_history_call_transaction(block_hash, slot, target_block_number);

    // Build a child on top of the hinted block so op-reth witnesses reads against the hinted
    // block's actual state root. This supplies the state/account trie nodes the guest needs when
    // opening the EIP-2935 account from `header.state_root`.
    let child_header = Header {
        timestamp: header.timestamp,
        mix_hash: header.mix_hash,
        beneficiary: header.beneficiary,
        gas_limit: header.gas_limit,
        parent_beacon_block_root: header.parent_beacon_block_root,
        slot_number: header.slot_number.map(|slot| slot.saturating_add(1)),
        extra_data: header.extra_data.clone(),
        ..Default::default()
    };
    let child_payload_attributes = payload_attributes_from_block_header(
        &child_header,
        vec![transaction.clone()],
        rollup_config,
    )?;
    let child_witness =
        fetch_execution_witness(provider, block_hash, child_payload_attributes).await?;
    store_execution_witness(kv, child_witness).await?;

    // Also build at the hinted block's height. The child payload above runs EIP-2935
    // pre-execution for N+1 and overwrites slot `N % 8191`, exactly the slot used for the oldest
    // in-window lookup from block N. The same-height replay records the requested storage slot
    // after block N's EIP-2935 pre-execution, before the N+1 overwrite can occur.
    let payload_attributes =
        payload_attributes_from_block_header(&header, vec![transaction], rollup_config)?;

    let witness = fetch_execution_witness(provider, header.parent_hash, payload_attributes).await?;
    store_execution_witness(kv, witness).await?;

    Ok(())
}

fn eip2935_block_number_for_slot(header: &Header, slot: U256) -> anyhow::Result<u64> {
    ensure!(
        slot <= U256::from(EIP2935_HISTORY_SERVE_WINDOW - 1),
        "EIP-2935 history storage slot exceeds the ring-buffer window"
    );
    let slot = {
        let slot_bytes = slot.to_be_bytes::<32>();
        u64::from_be_bytes(slot_bytes[24..].try_into()?)
    };

    let header_slot = header.number % EIP2935_HISTORY_SERVE_WINDOW;
    let distance = if header_slot >= slot {
        header_slot - slot
    } else {
        header_slot + EIP2935_HISTORY_SERVE_WINDOW - slot
    };
    ensure!(
        distance <= header.number,
        "EIP-2935 history storage slot does not map to a valid block number"
    );

    Ok(header.number - distance)
}

fn eip2935_history_call_transaction(
    block_hash: B256,
    slot: U256,
    target_block_number: u64,
) -> Bytes {
    let mut source_preimage = Vec::with_capacity(64);
    source_preimage.extend_from_slice(block_hash.as_slice());
    source_preimage.extend_from_slice(slot.to_be_bytes::<32>().as_slice());

    TxDeposit {
        source_hash: keccak256(source_preimage),
        from: Address::default(),
        to: TxKind::Call(alloy_eips::eip2935::HISTORY_STORAGE_ADDRESS),
        mint: 0,
        value: U256::ZERO,
        gas_limit: 100_000,
        is_system_transaction: false,
        input: Bytes::from(U256::from(target_block_number).to_be_bytes::<32>().to_vec()),
    }
    .encoded_2718()
    .into()
}

/// Stores a raw RLP header under its actual keccak preimage key.
pub(crate) async fn store_raw_header<KV: KeyValueStore + ?Sized>(
    kv: &RwLock<KV>,
    raw_header: Bytes,
) -> Result<B256> {
    let hash = keccak256(raw_header.as_ref());
    kv.write().await.set(PreimageKey::new_keccak256(*hash).into(), raw_header.into())?;
    Ok(hash)
}

/// Stores all preimages returned by a block execution witness.
pub(crate) async fn store_execution_witness<KV: KeyValueStore + ?Sized>(
    kv: &RwLock<KV>,
    execute_payload_response: ExecutionWitness,
) -> Result<()> {
    let preimages = execute_payload_response
        .state
        .into_iter()
        .chain(execute_payload_response.codes)
        .chain(execute_payload_response.keys)
        .chain(execute_payload_response.headers);

    let mut kv_lock = kv.write().await;
    for preimage in preimages {
        let computed_hash = keccak256(preimage.as_ref());

        let key = PreimageKey::new_keccak256(*computed_hash);
        kv_lock.set(key.into(), preimage.into())?;
    }

    Ok(())
}

/// Constructs a merkle patricia trie from the ordered list passed and stores all encoded
/// intermediate nodes of the trie in the [`KeyValueStore`].
pub(crate) async fn store_ordered_trie<KV: KeyValueStore + ?Sized, T: AsRef<[u8]>>(
    kv: &RwLock<KV>,
    values: &[T],
) -> Result<()> {
    let mut kv_write_lock = kv.write().await;

    // If the list of nodes is empty, store the empty root hash and exit early.
    // The `HashBuilder` will not push the preimage of the empty root hash to the
    // `ProofRetainer` in the event that there are no leaves inserted.
    if values.is_empty() {
        let empty_key = PreimageKey::new(*EMPTY_ROOT_HASH, PreimageKeyType::Keccak256);
        return kv_write_lock.set(empty_key.into(), [EMPTY_STRING_CODE].into());
    }

    let mut hb = kona_mpt::ordered_trie_with_encoder(values, |node, buf| {
        buf.put_slice(node.as_ref());
    });
    hb.root();
    let intermediates = hb.take_proof_nodes().into_inner();

    for (_, value) in intermediates {
        let value_hash = keccak256(value.as_ref());
        let key = PreimageKey::new(*value_hash, PreimageKeyType::Keccak256);

        kv_write_lock.set(key.into(), value.into())?;
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn eip2935_slot_maps_to_newest_matching_block_number() {
        let header = Header { number: 20_000, ..Default::default() };
        let slot = U256::from(4_000u64);

        let block_number = eip2935_block_number_for_slot(&header, slot).unwrap();

        assert_eq!(block_number, 12_191);
        assert_eq!(block_number % EIP2935_HISTORY_SERVE_WINDOW, 4_000);
    }

    #[test]
    fn eip2935_parent_slot_maps_to_parent_block_number() {
        let header = Header { number: 20_000, ..Default::default() };
        let slot = U256::from(header.number % EIP2935_HISTORY_SERVE_WINDOW);

        let block_number = eip2935_block_number_for_slot(&header, slot).unwrap();

        assert_eq!(block_number, header.number);
    }
}
