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
pub async fn find_starting_forkchoice(
    cfg: &RollupConfig,
    engine_client: &EngineClient,
) -> Result<L2ForkchoiceState, SyncStartError> {
    let mut current_fc = L2ForkchoiceState::current(cfg, engine_client).await?;
    info!(
        target: "sync_start",
        unsafe = %current_fc.un_safe.block_info.number,
        safe = %current_fc.safe.block_info.number,
        finalized = %current_fc.finalized.block_info.number,
        "Loaded current L2 EL forkchoice state"
    );

    // Walk back to the highest `unsafe` block whose L1 origin is plausible, as op-node's
    // `FindL2Heads` does. A lookup by hash cannot decide that, because L1 clients still serve
    // reorged-out blocks by hash, so compare each origin with the canonical block at its height.
    // An origin above the L1 head cannot be checked yet, so the highest such block is kept only
    // if the walk then reaches a canonical origin, or the finalized head, without passing a
    // reorged-out one. Like op-node, the walk never goes below the finalized head.
    let mut ahead_of_l1 = None;
    loop {
        let l1_origin = current_fc.un_safe.l1_origin;
        let canonical = engine_client.l1_block_hash_by_number(l1_origin.number).await?;
        info!(
            target: "sync_start",
            l1_origin = %l1_origin.number,
            l2_unsafe = %current_fc.un_safe.block_info.number,
            "Searching for L2 unsafe block with plausible L1 origin"
        );

        match canonical {
            Some(hash) if hash == l1_origin.hash => break,
            None => {
                ahead_of_l1.get_or_insert(current_fc.un_safe);
            }
            Some(_) => ahead_of_l1 = None,
        }
        if current_fc.un_safe.block_info.number <= current_fc.finalized.block_info.number {
            break;
        }

        let l2_parent_hash = current_fc.un_safe.block_info.parent_hash.into();
        let l2_parent = engine_client
            .get_l2_block(l2_parent_hash)
            .full()
            .await?
            .ok_or(SyncStartError::BlockNotFound(l2_parent_hash))?;

        current_fc.un_safe =
            L2BlockInfo::from_block_and_genesis(&l2_parent.into_consensus(), &cfg.genesis)?;
    }
    let canonical_origin_head = current_fc.un_safe;
    if let Some(head) = ahead_of_l1 {
        current_fc.un_safe = head;
    }
    info!(
        target: "sync_start",
        l2_unsafe = %current_fc.un_safe.block_info.number,
        "Found L2 unsafe block with plausible L1 origin"
    );

    // Search for the highest `safe` block that's L1 origin is at least older than the sequencing
    // window, relative to the highest canonical L1 origin found above.
    let mut safe_cursor = canonical_origin_head;
    loop {
        info!(
            target: "sync_start",
            l1_origin = %safe_cursor.l1_origin.number,
            l2_safe = %safe_cursor.block_info.number,
            "Searching for L2 safe block beyond sequencing window"
        );

        let is_behind_sequence_window =
            canonical_origin_head.l1_origin.number.saturating_sub(cfg.seq_window_size) >
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

#[cfg(test)]
mod test {
    use super::find_starting_forkchoice;
    use crate::test_utils::test_engine_client;
    use alloy_consensus::transaction::Recovered;
    use alloy_primitives::{B256, Sealed};
    use alloy_provider::Network;
    use alloy_rpc_types_eth::{Block, BlockTransactions};
    use kona_genesis::RollupConfig;
    use kona_protocol::{L1BlockInfoBedrock, L2BlockInfo};
    use kona_registry::ROLLUP_CONFIGS;
    use op_alloy_consensus::{OpTxEnvelope, TxDeposit};
    use op_alloy_network::Optimism;
    use serde_json::json;
    use std::sync::Arc;

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

    /// What the L1 RPC serves at an L2 block's L1 origin height.
    #[derive(Clone, Copy, Debug)]
    enum L1Origin {
        Canonical,
        /// Another block is canonical at that height, though L1 may still serve the origin by
        /// hash.
        ReorgedOut,
        /// L1 has not reached that height yet.
        Ahead,
    }

    fn l1_block(number: u64, fork: u8) -> Block {
        let mut block = Block::<alloy_rpc_types_eth::Transaction>::default();
        block.header.inner.number = number;
        block.header.inner.extra_data = vec![fork].into();
        block.header.hash = block.header.inner.hash_slow();
        block
    }

    fn l2_block(parent_hash: B256, origin: &Block) -> Block<op_alloy_rpc_types::Transaction> {
        let number = origin.header.number;
        let info = L1BlockInfoBedrock::new(
            number,
            0,
            0,
            origin.header.hash,
            0,
            Default::default(),
            Default::default(),
            Default::default(),
        );
        let deposit = OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
            input: info.encode_calldata(),
            ..Default::default()
        }));
        let mut block = Block::<op_alloy_rpc_types::Transaction>::default();
        block.header.inner.number = number;
        block.header.inner.parent_hash = parent_hash;
        block.header.hash = block.header.inner.hash_slow();
        block.transactions = BlockTransactions::Full(vec![op_alloy_rpc_types::Transaction {
            inner: alloy_rpc_types_eth::Transaction {
                block_hash: None,
                block_number: Some(number),
                block_timestamp: None,
                effective_gas_price: Some(1),
                inner: Recovered::new_unchecked(deposit, Default::default()),
                transaction_index: Some(0),
            },
            deposit_nonce: None,
            deposit_receipt_version: None,
        }]);
        block
    }

    /// L2 blocks `1..=n` have L1 origins `1..=n`, and `origins` says what L1 serves at each
    /// height. L2 block 1 is finalized, so neither walk goes below it.
    #[rstest::rstest]
    #[case::canonical(&[L1Origin::Canonical, L1Origin::Canonical], 1000, 2, 1)]
    #[case::reorged_out(&[L1Origin::Canonical, L1Origin::ReorgedOut], 1000, 1, 1)]
    #[case::ahead(&[L1Origin::Canonical, L1Origin::Ahead], 1000, 2, 1)]
    #[case::ahead_of_a_reorged_out_origin(
        &[L1Origin::Canonical, L1Origin::ReorgedOut, L1Origin::Ahead],
        1000,
        1,
        1
    )]
    #[case::reorged_out_of_an_ahead_origin(
        &[L1Origin::Canonical, L1Origin::Ahead, L1Origin::ReorgedOut],
        1000,
        2,
        1
    )]
    #[case::ahead_down_to_finalized(&[L1Origin::Ahead, L1Origin::Ahead], 1000, 2, 1)]
    #[case::reorged_out_down_to_finalized(&[L1Origin::ReorgedOut, L1Origin::Ahead], 1000, 1, 1)]
    // The sequence window is measured from block 4, the highest with a canonical origin, not
    // from the unsafe head.
    #[case::safe_below_ahead_origins(
        &[
            L1Origin::Canonical,
            L1Origin::Canonical,
            L1Origin::Canonical,
            L1Origin::Canonical,
            L1Origin::Ahead,
            L1Origin::Ahead,
        ],
        1,
        6,
        2
    )]
    #[tokio::test]
    async fn unsafe_head_has_a_plausible_l1_origin(
        #[case] origins: &[L1Origin],
        #[case] seq_window_size: u64,
        #[case] expected_unsafe: u64,
        #[case] expected_safe: u64,
    ) {
        let cfg = Arc::new(RollupConfig { seq_window_size, ..Default::default() });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let l1_blocks: Vec<_> = (1..=origins.len() as u64).map(|n| l1_block(n, 0)).collect();
        let mut l2_blocks = Vec::new();
        for origin in &l1_blocks {
            let parent_hash = l2_blocks.last().map_or(B256::ZERO, |b: &Block<_>| b.header.hash);
            l2_blocks.push(l2_block(parent_hash, origin));
        }

        l2.expect_params("eth_getBlockByNumber", json!(["finalized", true]), &l2_blocks[0]);
        l2.expect_params("eth_getBlockByNumber", json!(["safe", true]), &l2_blocks[0]);
        l2.expect_params("eth_getBlockByNumber", json!(["latest", true]), l2_blocks.last());
        let mut index = origins.len() - 1;
        loop {
            let number = index as u64 + 1;
            let served = match origins[index] {
                L1Origin::Canonical => Some(l1_blocks[index].clone()),
                L1Origin::ReorgedOut => Some(l1_block(number, 1)),
                L1Origin::Ahead => None,
            };
            l1.expect_params(
                "eth_getBlockByNumber",
                json!([format!("{number:#x}"), false]),
                served,
            );
            if matches!(origins[index], L1Origin::Canonical) || index == 0 {
                break;
            }
            l2.expect_params(
                "eth_getBlockByHash",
                json!([l2_blocks[index].header.parent_hash, true]),
                &l2_blocks[index - 1],
            );
            index -= 1;
        }
        // The safe walk starts from the highest block with a canonical origin.
        for index in (expected_safe as usize..=index).rev() {
            l2.expect_params(
                "eth_getBlockByHash",
                json!([l2_blocks[index].header.parent_hash, true]),
                &l2_blocks[index - 1],
            );
        }

        let forkchoice = find_starting_forkchoice(&cfg, &client).await.unwrap();
        assert_eq!(forkchoice.un_safe.block_info.number, expected_unsafe);
        assert_eq!(forkchoice.safe.block_info.number, expected_safe);
        l1.assert_finished();
        l2.assert_finished();
    }
}
