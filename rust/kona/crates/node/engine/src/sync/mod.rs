//! Sync start algorithm for the OP Stack rollup node.

use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};

mod forkchoice;
pub use forkchoice::L2ForkchoiceState;

mod error;
pub use error::SyncStartError;

use tracing::info;

use crate::EngineClient;

/// Searches for the latest [`L2ForkchoiceState`] that we can use to start the sync process with.
///
///   - The *unsafe L2 block*: This is the highest L2 block whose L1 origin is a *plausible*
///     extension of the canonical L1 chain (as known to the rollup node).
///   - The *safe L2 block*: This is the highest L2 block whose epoch's sequencing window is
///     complete within the canonical L1 chain (as known to the rollup node).
///   - The *finalized L2 block*: This is the L2 block which is known to be fully derived from
///     finalized L1 block data.
///
/// Plausible: meaning that the blockhash of the L2 block's L1 origin
/// (as reported in the L1 Attributes deposit within the L2 block) is not canonical at another
/// height in the L1 chain, and the same holds for all its ancestors.
pub async fn find_starting_forkchoice<EngineClient_: EngineClient>(
    cfg: &RollupConfig,
    engine_client: &EngineClient_,
) -> Result<L2ForkchoiceState, SyncStartError> {
    find_starting_forkchoice_with_checkpoints(cfg, engine_client, &[]).await
}

/// A completed safe block and the L1 input that was accepted to derive it.
#[derive(Debug, Clone, Copy)]
pub(crate) struct SafeCheckpoint {
    pub safe: L2BlockInfo,
    pub source: BlockInfo,
}

pub(crate) async fn find_starting_forkchoice_with_checkpoints<C: EngineClient>(
    cfg: &RollupConfig,
    engine_client: &C,
    checkpoints: &[SafeCheckpoint],
) -> Result<L2ForkchoiceState, SyncStartError> {
    let mut current_fc = L2ForkchoiceState::current(cfg, engine_client).await?;
    info!(
        target: "sync_start",
        unsafe = %current_fc.un_safe.block_info.number,
        safe = %current_fc.safe.block_info.number,
        finalized = %current_fc.finalized.block_info.number,
        "Loaded current L2 EL forkchoice state"
    );

    // Search for the highest `unsafe` block, relative to the initial `unsafe` block's L1 origin,
    loop {
        let l1_origin =
            engine_client.get_l1_block(current_fc.un_safe.l1_origin.hash.into()).await?;
        info!(
            target: "sync_start",
            l1_origin = %current_fc.un_safe.l1_origin.number,
            l2_unsafe = %current_fc.un_safe.block_info.number,
            "Searching for L2 unsafe block with canonical L1 origin"
        );

        match l1_origin {
            Some(_) => {
                // Unsafe block has existing L1 origin. Continue with this head.
                info!(
                    target: "sync_start",
                    l2_unsafe = %current_fc.un_safe.block_info.number,
                    "Found L2 unsafe block with canonical L1 origin"
                );
                break;
            }
            None => {
                let l2_parent_hash = current_fc.un_safe.block_info.parent_hash.into();
                let l2_parent = engine_client
                    .get_l2_block(l2_parent_hash)
                    .full()
                    .await?
                    .ok_or(SyncStartError::BlockNotFound(l2_parent_hash))?;

                current_fc.un_safe =
                    L2BlockInfo::from_block_and_genesis(&l2_parent.into_consensus(), &cfg.genesis)?;
            }
        }
    }

    if let Some(safe) = retained_safe_checkpoint(engine_client, &current_fc, checkpoints).await? {
        info!(target: "sync_start", safe = %safe, "Retaining completed safe checkpoint with canonical L1 input");
        current_fc.safe = safe;
        return Ok(current_fc);
    }

    // Search for the highest `safe` block that's L1 origin is at least older than the sequencing
    // window, relative to the L1 origin of the `unsafe` block.
    let mut safe_cursor = current_fc.un_safe;
    loop {
        info!(
            target: "sync_start",
            l1_origin = %safe_cursor.l1_origin.number,
            l2_safe = %safe_cursor.block_info.number,
            "Searching for L2 safe block beyond sequencing window"
        );

        let is_behind_sequence_window =
            current_fc.un_safe.l1_origin.number.saturating_sub(cfg.seq_window_size) >
                safe_cursor.l1_origin.number;
        let is_finalized = safe_cursor.block_info.hash == current_fc.finalized.block_info.hash;
        let is_genesis = safe_cursor.block_info.hash == cfg.genesis.l2.hash;
        if is_behind_sequence_window || is_finalized || is_genesis {
            info!(
                target: "sync_start",
                l2_safe = %safe_cursor.block_info.number,
                is_behind_sequence_window,
                is_finalized,
                is_genesis,
                "Found suitable L2 safe block"
            );
            current_fc.safe = safe_cursor;
            break;
        }
        let block = engine_client
            .get_l2_block(safe_cursor.block_info.parent_hash.into())
            .full()
            .await?
            .ok_or(SyncStartError::BlockNotFound(safe_cursor.block_info.parent_hash.into()))?;
        safe_cursor = L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &cfg.genesis)?;
    }

    // Leave the finalized block as-is, and return the current forkchoice.
    Ok(current_fc)
}

/// A canonical origin alone does not prove safety: the batch may have been submitted later.
/// Retain only completed blocks whose actual L1 input and L2 ancestry are still canonical.
async fn retained_safe_checkpoint<C: EngineClient>(
    client: &C,
    heads: &L2ForkchoiceState,
    checkpoints: &[SafeCheckpoint],
) -> Result<Option<L2BlockInfo>, SyncStartError> {
    use alloy_eips::BlockId;
    if checkpoints.is_empty() {
        return Ok(None);
    }
    let anchor =
        client.get_l1_block(BlockId::latest()).await?.ok_or(SyncStartError::ChainViewChanged)?;
    for checkpoint in checkpoints.iter().rev() {
        if checkpoint.safe.block_info.number < heads.finalized.block_info.number ||
            checkpoint.safe.block_info.number > heads.un_safe.block_info.number ||
            checkpoint.source.number > anchor.header.number
        {
            continue;
        }
        let source = client.get_l1_block(checkpoint.source.number.into()).await?;
        if source.is_none_or(|block| block.header.hash != checkpoint.source.hash) {
            continue;
        }
        // Number lookups are on one canonical L2 chain. Recheck its unsafe anchor after the
        // checkpoint lookup so an EL reorg cannot combine two different chain views.
        let safe = client.get_l2_block(checkpoint.safe.block_info.number.into()).await?;
        if safe.is_none_or(|block| block.header.hash != checkpoint.safe.hash()) {
            continue;
        }
        let unsafe_anchor = client.get_l2_block(heads.un_safe.block_info.number.into()).await?;
        if unsafe_anchor.is_none_or(|block| block.header.hash != heads.un_safe.hash()) {
            return Err(SyncStartError::ChainViewChanged);
        }
        // A successful consolidation can have received SYNCING. Numeric canonical lookups
        // establish ancestry, but only a fresh VALID response establishes execution validity,
        // including after an EL restart. Do not trust the permanent el_sync_finished latch.
        let response = client
            .fork_choice_updated_v3(
                alloy_rpc_types_engine::ForkchoiceState {
                    head_block_hash: heads.un_safe.hash(),
                    safe_block_hash: checkpoint.safe.hash(),
                    finalized_block_hash: heads.finalized.hash(),
                },
                None,
            )
            .await?;
        if !matches!(
            response.payload_status.status,
            alloy_rpc_types_engine::PayloadStatusEnum::Valid
        ) {
            return Err(SyncStartError::CheckpointNotValidated);
        }
        let current_anchor = client.get_l1_block(anchor.header.number.into()).await?;
        if current_anchor.is_none_or(|block| block.header.hash != anchor.header.hash) {
            return Err(SyncStartError::ChainViewChanged);
        }
        return Ok(Some(checkpoint.safe));
    }
    Ok(None)
}

#[cfg(test)]
mod test {
    use alloy_provider::Network;
    use alloy_rpc_types_eth::Block;
    use kona_protocol::L2BlockInfo;
    use kona_registry::ROLLUP_CONFIGS;
    use op_alloy_network::Optimism;

    const OP_SEPOLIA_CHAIN_ID: u64 = 11155420;
    const OP_SEPOLIA_GENESIS_RPC_RESPONSE: &str = "{\"hash\":\"0x102de6ffb001480cc9b8b548fd05c34cd4f46ae4aa91759393db90ea0409887d\",\"parentHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x4200000000000000000000000000000000000011\",\"stateRoot\":\"0x06787a17a3ed87c339a39dbbeeb311578a0c83ed29daa2db95da62b28efce8a9\",\"transactionsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"receiptsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\"difficulty\":\"0x0\",\"number\":\"0x0\",\"gasLimit\":\"0x1c9c380\",\"gasUsed\":\"0x0\",\"timestamp\":\"0x64d6dbac\",\"extraData\":\"0x424544524f434b\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":\"0x0000000000000000\",\"baseFeePerGas\":\"0x3b9aca00\",\"size\":\"0x209\",\"uncles\":[],\"transactions\":[]}";

    /// Sanity regression test - `alloy_rpc_types`' `Block::into_consensus` failed to saturate the
    /// header of the `alloy_consensus::Header` type on an old version. This test covers the
    /// conversion to ensure an OP genesis block's conversion to the consensus type works for
    /// the sake of `L2BlockInfo::from_block_and_genesis`.
    #[tokio::test]
    async fn test_genesis_block_hash() {
        let rollup_config = ROLLUP_CONFIGS.get(&OP_SEPOLIA_CHAIN_ID).unwrap();
        let genesis_block: Block<<Optimism as Network>::TransactionResponse> =
            serde_json::from_str(OP_SEPOLIA_GENESIS_RPC_RESPONSE).unwrap();

        let rpc_reported_hash = genesis_block.header.hash;
        let consensus_block = genesis_block.into_consensus();

        // Check that the genesis block's RPC-reported hash is equal to the manually computed hash.
        assert_eq!(rpc_reported_hash, consensus_block.hash_slow());

        // Convert to `L2BlockInfo` and check the same.
        let l2_block_info =
            L2BlockInfo::from_block_and_genesis(&consensus_block, &rollup_config.genesis).unwrap();
        assert_eq!(rpc_reported_hash, l2_block_info.block_info.hash);
    }
}

#[cfg(test)]
mod checkpoint_tests {
    use super::*;
    use crate::test_utils::{MockEngineClient, test_block_info};
    use alloy_eips::BlockId;
    use alloy_primitives::B256;

    #[derive(Debug, Clone, Copy)]
    enum Scenario {
        Retain,
        InvalidInput,
        InvalidL2,
        ChangedL1,
        ChangedL2,
        NewerInvalid,
        MissingHead,
        SyncingAfterValid,
    }

    #[rstest::rstest]
    #[case(Scenario::Retain)]
    #[case(Scenario::InvalidInput)]
    #[case(Scenario::InvalidL2)]
    #[case(Scenario::ChangedL1)]
    #[case(Scenario::ChangedL2)]
    #[case(Scenario::NewerInvalid)]
    #[case(Scenario::MissingHead)]
    #[case(Scenario::SyncingAfterValid)]
    #[tokio::test]
    async fn checkpoint_requires_canonical_batch_input_and_l2_ancestry(#[case] scenario: Scenario) {
        let input_canonical = !matches!(scenario, Scenario::InvalidInput);
        let l2_canonical = !matches!(scenario, Scenario::InvalidL2);
        // The L2 origin is unchanged in every case. Its batch input was submitted later.
        let safe = test_block_info(117029);
        let unsafe_head = test_block_info(117174);
        let source = BlockInfo { number: 10679, hash: B256::repeat_byte(1), ..Default::default() };
        let checkpoint = SafeCheckpoint { safe, source };
        let heads =
            L2ForkchoiceState { safe, un_safe: unsafe_head, finalized: test_block_info(44302) };
        let mut head: <alloy_network::Ethereum as alloy_provider::Network>::BlockResponse =
            Default::default();
        head.header.inner.number = 10690;
        head.header.hash = B256::repeat_byte(2); // New same-height head, source is unchanged.
        let mut input: <alloy_network::Ethereum as alloy_provider::Network>::BlockResponse =
            Default::default();
        input.header.inner.number = source.number;
        input.header.hash = if input_canonical { source.hash } else { B256::repeat_byte(3) };
        let mut safe_block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
            Default::default();
        safe_block.header.inner.number = safe.block_info.number;
        safe_block.header.hash = if l2_canonical { safe.hash() } else { B256::repeat_byte(4) };
        let mut unsafe_block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse = Default::default();
        unsafe_block.header.inner.number = unsafe_head.block_info.number;
        unsafe_block.header.hash = unsafe_head.hash();
        let mut anchor = head.clone();
        if matches!(scenario, Scenario::ChangedL1) {
            anchor.header.hash = B256::repeat_byte(5);
        }
        if matches!(scenario, Scenario::ChangedL2) {
            unsafe_block.header.hash = B256::repeat_byte(6);
        }
        let mut builder = MockEngineClient::builder()
            .with_config(std::sync::Arc::new(RollupConfig::default()))
            .with_fork_choice_updated_v3_response(alloy_rpc_types_engine::ForkchoiceUpdated::new(
                alloy_rpc_types_engine::PayloadStatus::from_status(
                    alloy_rpc_types_engine::PayloadStatusEnum::Valid,
                ),
            ))
            .with_l1_block(head.header.number.into(), anchor)
            .with_l1_block(source.number.into(), input)
            .with_l2_block(safe.block_info.number.into(), safe_block)
            .with_l2_block(unsafe_head.block_info.number.into(), unsafe_block);
        if !matches!(scenario, Scenario::MissingHead) {
            builder = builder.with_l1_block(BlockId::latest(), head);
        }
        let mut checkpoints = vec![checkpoint];
        if matches!(scenario, Scenario::NewerInvalid) {
            let mut newer = checkpoint;
            newer.safe.block_info.number += 1;
            newer.source.number += 1;
            // This newer batch input no longer exists; the earlier canonical checkpoint wins.
            checkpoints.push(newer);
        }
        let client = builder.build();
        if matches!(scenario, Scenario::SyncingAfterValid) {
            assert_eq!(
                retained_safe_checkpoint(&client, &heads, &checkpoints).await.unwrap(),
                Some(safe)
            );
            client
                .set_fork_choice_updated_v3_response(
                    alloy_rpc_types_engine::ForkchoiceUpdated::new(
                        alloy_rpc_types_engine::PayloadStatus::from_status(
                            alloy_rpc_types_engine::PayloadStatusEnum::Syncing,
                        ),
                    ),
                )
                .await;
        }
        let selected = retained_safe_checkpoint(&client, &heads, &checkpoints).await;
        match scenario {
            Scenario::SyncingAfterValid => {
                assert!(matches!(selected, Err(SyncStartError::CheckpointNotValidated)))
            }
            Scenario::ChangedL1 | Scenario::ChangedL2 | Scenario::MissingHead => {
                assert!(matches!(selected, Err(SyncStartError::ChainViewChanged)));
            }
            Scenario::InvalidInput | Scenario::InvalidL2 => assert_eq!(selected.unwrap(), None),
            Scenario::Retain | Scenario::NewerInvalid => assert_eq!(selected.unwrap(), Some(safe)),
        }
    }

    #[tokio::test]
    async fn startup_without_accepted_checkpoints_uses_conservative_walkback() {
        let client = MockEngineClient::builder()
            .with_config(std::sync::Arc::new(RollupConfig::default()))
            .build();
        let block = test_block_info(1);
        let heads = L2ForkchoiceState { safe: block, un_safe: block, finalized: block };
        assert_eq!(retained_safe_checkpoint(&client, &heads, &[]).await.unwrap(), None);
    }
}
