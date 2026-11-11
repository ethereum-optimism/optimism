//! Sync start algorithm for the OP Stack rollup node.

use alloy_eips::BlockNumberOrTag;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;

mod forkchoice;
pub use forkchoice::L2ForkchoiceState;

mod error;
pub use error::SyncStartError;

use tracing::info;

use crate::EngineClient;

/// How many sequence windows of L1 origins the unsafe head may be rewound across, as in op-node.
/// A deeper rewind is more likely an L1 RPC on the wrong chain than a real L1 reorg. op-node
/// measures this in `SyncLookback`, which exceeds the sequence window only on alt-DA chains, which
/// kona does not derive.
const MAX_REORG_SEQ_WINDOWS: u64 = 5;

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
///
/// Like op-node's `FindL2Heads`, this fails rather than return a forkchoice that rewinds past the
/// finalized head, builds on another L1 genesis, or rewinds the unsafe head across more than
/// [`MAX_REORG_SEQ_WINDOWS`] sequence windows.
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

    // An L1 origin above the L1 head cannot be checked yet, so it is never looked up by number:
    // most L1 clients answer `null` above their head, but some answer with an error.
    let l1_head = engine_client
        .get_l1_block(BlockNumberOrTag::Latest.into())
        .await?
        .ok_or(SyncStartError::BlockNotFound(BlockNumberOrTag::Latest.into()))?
        .header
        .number;

    // Walk back to the highest `unsafe` block whose L1 origin is plausible, as op-node's
    // `FindL2Heads` does. A lookup by hash cannot decide that, because L1 clients still serve
    // reorged-out blocks by hash, so compare each origin with the canonical block at its height.
    // An origin above the L1 head is kept only if the walk then reaches a canonical origin, or
    // the finalized head, without passing a reorged-out one. Like op-node, the walk never goes
    // below the finalized head.
    let previous_unsafe = current_fc.un_safe;
    let max_reorg_depth = MAX_REORG_SEQ_WINDOWS.saturating_mul(cfg.seq_window_size);
    let too_deep = |candidate: &L2BlockInfo| {
        (candidate.l1_origin.number.saturating_add(max_reorg_depth) <
            previous_unsafe.l1_origin.number)
            .then_some(SyncStartError::TooDeepReorg {
                unsafe_head: candidate.block_info.number,
                previous_unsafe_head: previous_unsafe.block_info.number,
            })
    };
    let mut ahead_of_l1 = None;
    loop {
        let block = current_fc.un_safe;
        let l1_origin = block.l1_origin;
        // With no block above held back, this block is the best unsafe head left, so stop as
        // soon as it is too deep, as op-node does, rather than walk on to finalized.
        if ahead_of_l1.is_none() &&
            let Some(err) = too_deep(&block)
        {
            return Err(err);
        }
        info!(
            target: "sync_start",
            l1_origin = %l1_origin.number,
            l2_unsafe = %block.block_info.number,
            "Searching for L2 unsafe block with plausible L1 origin"
        );

        let canonical = if l1_origin.number > l1_head {
            None
        } else {
            let canonical = engine_client
                .get_l1_block(l1_origin.number.into())
                .await?
                // L1 reported a head at or above this height, so it is serving an inconsistent
                // view, as a lagging backend behind a load balancer can.
                .ok_or(SyncStartError::L1BlockMissing { number: l1_origin.number, l1_head })?;
            Some(canonical.header.hash)
        };
        // The L2 genesis's L1 origin is the L1 genesis, so a different canonical block there
        // means the L1 RPC serves another chain.
        if block.block_info.number == cfg.genesis.l2.number &&
            let Some(hash) = canonical &&
            hash != cfg.genesis.l1.hash
        {
            return Err(SyncStartError::InvalidL1GenesisHash(cfg.genesis.l1.hash, hash));
        }
        check_finalized(&current_fc.finalized, &block)?;

        match canonical {
            Some(hash) if hash == l1_origin.hash => break,
            None => {
                ahead_of_l1.get_or_insert(block);
            }
            Some(_) => ahead_of_l1 = None,
        }
        if block.block_info.number <= current_fc.finalized.block_info.number {
            break;
        }

        let l2_parent_hash = block.block_info.parent_hash.into();
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
    // A reorged-out origin at the last block can discard a held block after the check above.
    if let Some(err) = too_deep(&current_fc.un_safe) {
        return Err(err);
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
        check_finalized(&current_fc.finalized, &safe_cursor)?;

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

/// Fails when a walk back from the unsafe head reaches the finalized height on another block,
/// as op-node's `ReorgFinalizedErr` does: the EL's chain no longer contains its finalized head.
fn check_finalized(finalized: &L2BlockInfo, block: &L2BlockInfo) -> Result<(), SyncStartError> {
    if block.block_info.number == finalized.block_info.number &&
        block.block_info.hash != finalized.block_info.hash
    {
        return Err(SyncStartError::MismatchedFinalizedBlock(
            finalized.block_info.hash,
            block.block_info.hash,
        ));
    }
    Ok(())
}

#[cfg(test)]
mod test {
    use super::{SyncStartError, find_starting_forkchoice};
    use crate::test_utils::{RpcMock, rpc_l2_block, test_engine_client};
    use alloy_consensus::Header;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use alloy_provider::Network;
    use alloy_rpc_types_eth::Block;
    use kona_genesis::{ChainGenesis, RollupConfig};
    use kona_protocol::{L1BlockInfoBedrock, L2BlockInfo};
    use kona_registry::ROLLUP_CONFIGS;
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
        /// Above the L1 head, so it is never looked up.
        Ahead,
        /// At or below the L1 head, yet L1 serves no block there.
        Missing,
    }

    type L2Block = Block<op_alloy_rpc_types::Transaction>;

    /// What [`find_starting_forkchoice`] returns: the unsafe and safe head numbers, or the error
    /// message.
    type Expected = Result<(u64, u64), &'static str>;

    fn l1_block(number: u64, fork: u8) -> Block {
        let mut block = Block::<alloy_rpc_types_eth::Transaction>::default();
        block.header.inner.number = number;
        block.header.inner.extra_data = vec![fork].into();
        block.header.hash = block.header.inner.hash_slow();
        block
    }

    fn l2_block(parent_hash: B256, origin: &Block) -> L2Block {
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
        rpc_l2_block(Header { number, parent_hash, ..Default::default() }, &info)
    }

    /// L1 blocks `1..=n`, and L2 blocks `1..=n` built on them, one L2 block per L1 origin.
    fn chain(n: u64) -> (Vec<Block>, Vec<L2Block>) {
        let l1_blocks: Vec<_> = (1..=n).map(|number| l1_block(number, 0)).collect();
        let mut l2_blocks: Vec<L2Block> = Vec::new();
        for origin in &l1_blocks {
            let parent_hash = l2_blocks.last().map_or(B256::ZERO, |block| block.header.hash);
            l2_blocks.push(l2_block(parent_hash, origin));
        }
        (l1_blocks, l2_blocks)
    }

    /// Expects the EL to report `finalized` as both the finalized and safe head, and `unsafe_head`
    /// as the unsafe head, then L1 to report a head at `l1_head`.
    fn expect_heads(
        l1: &RpcMock,
        l2: &RpcMock,
        finalized: &L2Block,
        unsafe_head: &L2Block,
        l1_head: u64,
    ) {
        l2.expect_params("eth_getBlockByNumber", json!(["finalized", true]), finalized);
        l2.expect_params("eth_getBlockByNumber", json!(["safe", true]), finalized);
        l2.expect_params("eth_getBlockByNumber", json!(["latest", true]), unsafe_head);
        l1.expect_params("eth_getBlockByNumber", json!(["latest", false]), l1_block(l1_head, 0));
    }

    fn expect_parent(l2: &RpcMock, child: &L2Block, parent: &L2Block) {
        l2.expect_params("eth_getBlockByHash", json!([child.header.parent_hash, true]), parent);
    }

    /// L2 blocks `1..=n` have L1 origins `1..=n`, and `origins` says what L1 serves at each
    /// height. The L1 head is the highest origin that is not [`L1Origin::Ahead`]. L2 block 1 is
    /// finalized, so neither walk goes below it.
    #[rstest::rstest]
    #[case::canonical(&[L1Origin::Canonical, L1Origin::Canonical], 1000, Ok((2, 1)))]
    #[case::reorged_out(&[L1Origin::Canonical, L1Origin::ReorgedOut], 1000, Ok((1, 1)))]
    #[case::ahead(&[L1Origin::Canonical, L1Origin::Ahead], 1000, Ok((2, 1)))]
    #[case::ahead_of_a_reorged_out_origin(
        &[L1Origin::Canonical, L1Origin::ReorgedOut, L1Origin::Ahead],
        1000,
        Ok((1, 1))
    )]
    // L1 reorgs from head 5 onto a branch that forks after block 2 and only reaches height 3.
    // Origins 4 and 5 are above the new head, but origin 3 is reorged out, so neither is kept.
    #[case::reorg_to_a_shorter_l1_chain(
        &[
            L1Origin::Canonical,
            L1Origin::Canonical,
            L1Origin::ReorgedOut,
            L1Origin::Ahead,
            L1Origin::Ahead,
        ],
        1000,
        Ok((2, 1))
    )]
    #[case::ahead_down_to_finalized(&[L1Origin::Ahead, L1Origin::Ahead], 1000, Ok((2, 1)))]
    #[case::reorged_out_down_to_finalized(
        &[L1Origin::ReorgedOut, L1Origin::Ahead],
        1000,
        Ok((1, 1))
    )]
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
        Ok((6, 2))
    )]
    #[case::missing_below_the_l1_head(
        &[L1Origin::Canonical, L1Origin::Canonical, L1Origin::Missing, L1Origin::Ahead],
        1000,
        Err("L1 block 3 is missing below the L1 head 3")
    )]
    // Rewinding from origin 6 to origin 1 crosses exactly five sequence windows of one block.
    #[case::deepest_allowed_reorg(
        &[
            L1Origin::Canonical,
            L1Origin::ReorgedOut,
            L1Origin::ReorgedOut,
            L1Origin::ReorgedOut,
            L1Origin::ReorgedOut,
            L1Origin::ReorgedOut,
        ],
        1,
        Ok((1, 1))
    )]
    #[tokio::test]
    async fn unsafe_head_has_a_plausible_l1_origin(
        #[case] origins: &[L1Origin],
        #[case] seq_window_size: u64,
        #[case] expected: Expected,
    ) {
        let cfg = Arc::new(RollupConfig { seq_window_size, ..Default::default() });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let (l1_blocks, l2_blocks) = chain(origins.len() as u64);
        let l1_head = origins
            .iter()
            .rposition(|origin| !matches!(origin, L1Origin::Ahead))
            .map_or(0, |index| index as u64 + 1);
        expect_heads(&l1, &l2, &l2_blocks[0], l2_blocks.last().unwrap(), l1_head);

        let mut index = origins.len() - 1;
        loop {
            let number = index as u64 + 1;
            let served = match origins[index] {
                L1Origin::Canonical => Some(l1_blocks[index].clone()),
                L1Origin::ReorgedOut => Some(l1_block(number, 1)),
                L1Origin::Missing | L1Origin::Ahead => None,
            };
            if !matches!(origins[index], L1Origin::Ahead) {
                l1.expect_params(
                    "eth_getBlockByNumber",
                    json!([format!("{number:#x}"), false]),
                    served,
                );
            }
            if matches!(origins[index], L1Origin::Canonical | L1Origin::Missing) || index == 0 {
                break;
            }
            expect_parent(&l2, &l2_blocks[index], &l2_blocks[index - 1]);
            index -= 1;
        }

        if let Ok((_, expected_safe)) = expected {
            // The safe walk starts from the highest block with a canonical origin.
            for index in (expected_safe as usize..=index).rev() {
                expect_parent(&l2, &l2_blocks[index], &l2_blocks[index - 1]);
            }
        }

        let result = find_starting_forkchoice(&cfg, &client).await;
        match expected {
            Ok((expected_unsafe, expected_safe)) => {
                let forkchoice = result.unwrap();
                assert_eq!(forkchoice.un_safe.block_info.number, expected_unsafe);
                assert_eq!(forkchoice.safe.block_info.number, expected_safe);
            }
            Err(message) => assert_eq!(result.unwrap_err().to_string(), message),
        }
        l1.assert_finished();
        l2.assert_finished();
    }

    /// The EL's block at the finalized height is not its finalized head. Op-node fails with
    /// `ReorgFinalizedErr` rather than walk past the finalized height, from either walk.
    #[rstest::rstest]
    // Origin 3 is canonical, so the unsafe walk stops at once and the safe walk reaches height 1.
    #[case::safe_walk(L1Origin::Canonical)]
    // Origins 3 and 2 are reorged out, so the unsafe walk itself reaches height 1.
    #[case::unsafe_walk(L1Origin::ReorgedOut)]
    #[tokio::test]
    async fn finalized_head_must_be_on_the_unsafe_chain(#[case] origin: L1Origin) {
        let cfg = Arc::new(RollupConfig { seq_window_size: 1000, ..Default::default() });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let (l1_blocks, l2_blocks) = chain(3);
        let finalized = l2_block(B256::with_last_byte(1), &l1_blocks[0]);
        expect_heads(&l1, &l2, &finalized, &l2_blocks[2], 3);
        // Either walk fetches blocks 2 and 1 from the EL; only the L1 lookups differ.
        expect_parent(&l2, &l2_blocks[2], &l2_blocks[1]);
        expect_parent(&l2, &l2_blocks[1], &l2_blocks[0]);
        if matches!(origin, L1Origin::Canonical) {
            l1.expect_params("eth_getBlockByNumber", json!(["0x3", false]), &l1_blocks[2]);
        } else {
            l1.expect_params("eth_getBlockByNumber", json!(["0x3", false]), l1_block(3, 1));
            l1.expect_params("eth_getBlockByNumber", json!(["0x2", false]), l1_block(2, 1));
            l1.expect_params("eth_getBlockByNumber", json!(["0x1", false]), &l1_blocks[0]);
        }

        let err = find_starting_forkchoice(&cfg, &client).await.unwrap_err();
        assert!(
            matches!(
                err,
                SyncStartError::MismatchedFinalizedBlock(expected, got)
                    if expected == finalized.header.hash && got == l2_blocks[0].header.hash
            ),
            "unexpected error: {err:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    /// With sequence windows of one block, rewinding from origin 8 below origin 3 crosses more
    /// than five. Like op-node, the walk stops at block 2, the first block that deep, without
    /// looking up its origin or walking on to finalized.
    #[tokio::test]
    async fn too_deep_reorg_stops_the_walk() {
        let cfg = Arc::new(RollupConfig { seq_window_size: 1, ..Default::default() });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let (_, l2_blocks) = chain(8);
        expect_heads(&l1, &l2, &l2_blocks[0], &l2_blocks[7], 8);
        for number in (3..=8u64).rev() {
            l1.expect_params(
                "eth_getBlockByNumber",
                json!([format!("{number:#x}"), false]),
                l1_block(number, 1),
            );
            let index = number as usize - 1;
            expect_parent(&l2, &l2_blocks[index], &l2_blocks[index - 1]);
        }

        let err = find_starting_forkchoice(&cfg, &client).await.unwrap_err();
        assert!(
            matches!(err, SyncStartError::TooDeepReorg { unsafe_head: 2, previous_unsafe_head: 8 }),
            "unexpected error: {err:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    /// Op-node fails with `WrongChainErr` when the walk reaches the L2 genesis and L1 serves
    /// another block at the L1 genesis height: the L1 RPC is on another chain.
    #[tokio::test]
    async fn l1_genesis_must_be_canonical() {
        let l1_genesis = l1_block(0, 0);
        let l2_genesis = l2_block(B256::ZERO, &l1_genesis);
        let l2_head = l2_block(l2_genesis.header.hash, &l1_block(1, 0));
        let cfg = Arc::new(RollupConfig {
            seq_window_size: 1000,
            genesis: ChainGenesis {
                l1: BlockNumHash { number: 0, hash: l1_genesis.header.hash },
                l2: BlockNumHash { number: 0, hash: l2_genesis.header.hash },
                ..Default::default()
            },
            ..Default::default()
        });
        let (client, l1, l2) = test_engine_client(cfg.clone());
        expect_heads(&l1, &l2, &l2_genesis, &l2_head, 1);
        l1.expect_params("eth_getBlockByNumber", json!(["0x1", false]), l1_block(1, 1));
        expect_parent(&l2, &l2_head, &l2_genesis);
        let other_genesis = l1_block(0, 1);
        l1.expect_params("eth_getBlockByNumber", json!(["0x0", false]), &other_genesis);

        let err = find_starting_forkchoice(&cfg, &client).await.unwrap_err();
        assert!(
            matches!(
                err,
                SyncStartError::InvalidL1GenesisHash(expected, got)
                    if expected == l1_genesis.header.hash && got == other_genesis.header.hash
            ),
            "unexpected error: {err:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }
}
