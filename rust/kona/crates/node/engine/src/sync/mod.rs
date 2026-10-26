//! Sync start algorithm for the OP Stack rollup node.

use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;

mod forkchoice;
pub use forkchoice::L2ForkchoiceState;

mod error;
pub use error::SyncStartError;

mod reorg;
pub use reorg::{DerivedSafeHead, find_reorg_forkchoice};

mod unsafe_head;
use unsafe_head::find_canonical_unsafe_head;

use tracing::info;

use crate::EngineClient;

/// Searches for the latest [`L2ForkchoiceState`] that we can use to start the sync process with.
///
///   - The *unsafe L2 block*: This is the highest L2 block, at or above the finalized one, whose L1
///     origin is canonical: the L1 node has that block at that height.
///   - The *safe L2 block*: This is the highest L2 block whose epoch's sequencing window is
///     complete within the canonical L1 chain (as known to the rollup node).
///   - The *finalized L2 block*: This is the L2 block which is known to be fully derived from
///     finalized L1 block data.
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

    // Search for the highest `unsafe` block whose L1 origin is canonical. The finalized head's
    // is, so the walk stops there at the latest.
    current_fc.un_safe =
        find_canonical_unsafe_head(engine_client, current_fc.un_safe, current_fc.finalized).await?;
    info!(
        target: "sync_start",
        l2_unsafe = %current_fc.un_safe.block_info.number,
        "Found L2 unsafe block with canonical L1 origin"
    );

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
mod cold_start_tests {
    use super::*;
    use crate::{
        EngineClient, EngineResetError,
        test_utils::{L2RpcBlock, RpcMock, TestChain, expect_l1_block, test_engine_client},
    };
    use rstest::rstest;
    use serde_json::json;
    use std::sync::Arc;

    /// The test chain, and an engine client over scripted L1 and L2 nodes. The sequencing window
    /// is long enough that the safe head search always stops at the finalized head.
    fn setup() -> (TestChain, Arc<RollupConfig>, EngineClient, RpcMock, RpcMock) {
        let config = Arc::new(RollupConfig { seq_window_size: 1000, ..Default::default() });
        let (client, l1, l2) = test_engine_client(config.clone());
        (TestChain::new(), config, client, l1, l2)
    }

    /// How the L1 node answers for L1 block 105, the unsafe head's L1 origin.
    #[derive(Debug, Clone, Copy)]
    enum Origin {
        Canonical,
        ReplacedAtTheSameHeight,
        NotSeenYet,
    }

    /// A cold start keeps the unsafe head only if the L1 node has its L1 origin at that height.
    /// Otherwise it walks back to the highest block with a canonical L1 origin: the replaced
    /// origin's other block is still served by hash, so finding it by hash would keep a head
    /// that can never become safe. Blocks 25 and 26 share L1 origin 105, checked once; block 24's
    /// is 104. The safe head is then searched down to the finalized block 18.
    #[rstest]
    #[case::canonical(Origin::Canonical, 26)]
    #[case::replaced_at_the_same_height(Origin::ReplacedAtTheSameHeight, 24)]
    #[case::not_seen_yet(Origin::NotSeenYet, 24)]
    #[tokio::test]
    async fn cold_start_keeps_only_an_unsafe_head_with_a_canonical_l1_origin(
        #[case] origin: Origin,
        #[case] unsafe_head: u64,
    ) {
        let (chain, config, client, l1, l2) = setup();
        chain.expect_heads(&l2, 18, 20, 26);
        match origin {
            Origin::Canonical => expect_l1_block(&l1, 105, 106),
            Origin::ReplacedAtTheSameHeight => expect_l1_block(&l1, 105, 105),
            Origin::NotSeenYet => l1.expect_params(
                "eth_getBlockByNumber",
                json!(["0x69", false]),
                Option::<alloy_rpc_types_eth::Block>::None,
            ),
        }
        if unsafe_head < 26 {
            chain.expect_by_hash(&l2, 25);
            chain.expect_by_hash(&l2, 24);
            expect_l1_block(&l1, 104, 105);
        }
        for number in (18..unsafe_head).rev() {
            chain.expect_by_hash(&l2, number);
        }

        let forkchoice = find_starting_forkchoice(&config, &client).await.unwrap();

        assert_eq!(forkchoice.un_safe, chain.info(unsafe_head));
        assert_eq!(forkchoice.safe, chain.info(18));
        assert_eq!(forkchoice.finalized, chain.info(18));
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Block 18 with a different header, so a different block at the finalized height.
    fn other_18(chain: &TestChain) -> L2RpcBlock {
        let mut block = chain.rpc(18).clone();
        block.header.inner.timestamp += 1;
        block.header.hash = block.header.inner.hash_slow();
        block
    }

    /// The walk never passes the finalized head. With every L1 origin above it replaced, the
    /// unsafe head becomes the finalized head, which the block above it must link to; an unsafe
    /// chain that does not lead there fails the reset for good.
    #[rstest]
    #[case::leads_to_the_finalized_head(false)]
    #[case::does_not_lead_to_the_finalized_head(true)]
    #[tokio::test]
    async fn cold_start_walks_back_no_further_than_the_finalized_head(
        #[case] other_finalized: bool,
    ) {
        let (chain, config, client, l1, l2) = setup();
        let finalized = if other_finalized { other_18(&chain) } else { chain.rpc(18).clone() };
        l2.expect_params("eth_getBlockByNumber", json!(["finalized", true]), &finalized);
        l2.expect_params("eth_getBlockByNumber", json!(["safe", true]), chain.rpc(20));
        l2.expect_params("eth_getBlockByNumber", json!(["latest", true]), chain.rpc(26));
        for number in [105, 104, 103, 102, 101, 100] {
            expect_l1_block(&l1, number, 100);
        }
        for number in (19..26).rev() {
            chain.expect_by_hash(&l2, number);
        }

        let forkchoice = find_starting_forkchoice(&config, &client).await;

        if other_finalized {
            let err = forkchoice.unwrap_err();
            assert!(
                matches!(err, SyncStartError::UnsafeChainMismatch { found, .. } if found == chain.info(18).block_info.hash),
                "got {err:?}"
            );
            assert!(!EngineResetError::SyncStart(err).is_temporary());
        } else {
            let forkchoice = forkchoice.unwrap();
            assert_eq!(forkchoice.un_safe, chain.info(18));
            assert_eq!(forkchoice.safe, chain.info(18));
        }
        l1.assert_finished();
        l2.assert_finished();
    }

    /// An unsafe head at or below the finalized head needs no L1 lookup: at the finalized head it
    /// is kept, and as another block at that height or below it, the reset fails for good.
    #[rstest]
    #[case::the_finalized_head(18, false)]
    #[case::another_block_at_its_height(18, true)]
    #[case::below_it(20, false)]
    #[tokio::test]
    async fn cold_start_checks_an_unsafe_head_at_or_below_the_finalized_head(
        #[case] finalized: u64,
        #[case] other_unsafe: bool,
    ) {
        let (chain, config, client, l1, l2) = setup();
        let unsafe_head = if other_unsafe { other_18(&chain) } else { chain.rpc(18).clone() };
        l2.expect_params("eth_getBlockByNumber", json!(["finalized", true]), chain.rpc(finalized));
        l2.expect_params("eth_getBlockByNumber", json!(["safe", true]), chain.rpc(finalized));
        l2.expect_params("eth_getBlockByNumber", json!(["latest", true]), &unsafe_head);

        let forkchoice = find_starting_forkchoice(&config, &client).await;

        match (finalized, other_unsafe) {
            (18, false) => assert_eq!(forkchoice.unwrap().un_safe, chain.info(18)),
            (18, true) => assert!(matches!(
                forkchoice.unwrap_err(),
                SyncStartError::UnsafeChainMismatch { expected, .. } if expected == chain.info(18).block_info.hash
            )),
            _ => assert!(matches!(
                forkchoice.unwrap_err(),
                SyncStartError::UnsafeHeadBelowFloor { unsafe_head: 18, floor: 20 }
            )),
        }
        l1.assert_finished();
        l2.assert_finished();
    }
}
