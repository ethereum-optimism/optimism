//! Sync start algorithm for the OP Stack rollup node.

use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;

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
        } else {
            let block = engine_client
                .get_l2_block(safe_cursor.block_info.parent_hash.into())
                .full()
                .await?
                .ok_or(SyncStartError::BlockNotFound(safe_cursor.block_info.parent_hash.into()))?;
            safe_cursor =
                L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &cfg.genesis)?;
        }
    }

    // Leave the finalized block as-is, and return the current forkchoice.
    Ok(current_fc)
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
