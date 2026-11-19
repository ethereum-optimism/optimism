//! Sync start algorithm for the OP Stack rollup node.

use alloy_eips::BlockNumHash;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;

mod forkchoice;
pub use forkchoice::L2ForkchoiceState;

mod error;
pub use error::SyncStartError;

use tracing::{error, info};

use crate::EngineClient;

/// A block known only by its number and hash, which is all a forkchoice update or a finality
/// comparison needs of it.
pub(crate) fn l2_block_info_at(id: BlockNumHash) -> L2BlockInfo {
    L2BlockInfo {
        block_info: kona_protocol::BlockInfo {
            hash: id.hash,
            number: id.number,
            ..Default::default()
        },
        ..Default::default()
    }
}

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
                // At or below the finalized height, only the finalized block itself proves the
                // L1 provider is behind: its L1 data is finalized and cannot reorg. Any other
                // block there is on a fork from below finality, and walking further back would
                // move finality back.
                if current_fc.un_safe.block_info.number <= current_fc.finalized.block_info.number {
                    if current_fc.un_safe.block_info.hash != current_fc.finalized.block_info.hash {
                        return Err(finalized_block_not_on_walked_chain(
                            &current_fc.finalized,
                            &current_fc.un_safe,
                        ));
                    }
                    return Err(finalized_l1_origin_missing_by_hash(
                        engine_client,
                        &current_fc.un_safe,
                    )
                    .await);
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
        if !is_finalized && safe_cursor.block_info.number <= current_fc.finalized.block_info.number
        {
            return Err(finalized_block_not_on_walked_chain(&current_fc.finalized, &safe_cursor));
        }
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

    ensure_finalized_block_on_walked_chain(engine_client, &current_fc).await?;

    // Leave the finalized block as-is, and return the current forkchoice.
    Ok(current_fc)
}

/// Checks that the finalized block is on the walked chain when the safe walk stopped above it, as
/// the sequencing window can stop it. The walk follows the execution layer's canonical chain, so
/// its block at the finalized height is the canonical one there.
async fn ensure_finalized_block_on_walked_chain(
    engine_client: &EngineClient,
    current_fc: &L2ForkchoiceState,
) -> Result<(), SyncStartError> {
    let finalized = current_fc.finalized.block_info;
    if current_fc.safe.block_info.number <= finalized.number {
        return Ok(());
    }
    info!(target: "sync_start", finalized = %finalized.number, "Checking that the finalized block is on the walked chain");
    let walked = engine_client
        .get_l2_block(finalized.number.into())
        .await?
        .ok_or(SyncStartError::FinalizedHeightUnavailable(finalized.number))?
        .header;
    if walked.hash == finalized.hash {
        return Ok(());
    }
    let walked = l2_block_info_at(BlockNumHash { hash: walked.hash, number: walked.number });
    Err(finalized_block_not_on_walked_chain(&current_fc.finalized, &walked))
}

/// The error for the finalized block whose L1 origin the L1 provider does not serve by hash. A
/// provider without a block at that height lags and is retried; one with a different block there
/// follows another L1 chain, which retrying cannot fix; one with that very block there serves it
/// only by number, as a broken or lagging block-hash index does.
async fn finalized_l1_origin_missing_by_hash(
    engine_client: &EngineClient,
    finalized: &L2BlockInfo,
) -> SyncStartError {
    let l1_origin = finalized.l1_origin;
    let l1_block = match engine_client.get_l1_block(l1_origin.number.into()).await {
        Ok(l1_block) => l1_block,
        Err(err) => return err.into(),
    };
    match l1_block {
        Some(l1_block) if l1_block.header.hash != l1_origin.hash => {
            error!(
                target: "sync_start",
                l1_origin = %l1_origin.number,
                l1_block_hash = %l1_block.header.hash,
                "The L1 provider follows another chain than the finalized block's L1 origin"
            );
            SyncStartError::FinalizedL1OriginNotCanonical {
                l1_block: BlockNumHash { hash: l1_block.header.hash, number: l1_origin.number },
                l1_origin,
            }
        }
        Some(_) => {
            error!(
                target: "sync_start",
                l1_origin = %l1_origin.number,
                l1_origin_hash = %l1_origin.hash,
                "The L1 provider serves the finalized block's L1 origin by number but not by hash"
            );
            SyncStartError::FinalizedL1OriginNotServedByHash { l1_origin }
        }
        None => {
            error!(
                target: "sync_start",
                l2_block = %finalized.block_info.number,
                l1_origin = %l1_origin.number,
                "The L1 provider does not have the L1 origin of the finalized block"
            );
            SyncStartError::FinalizedL1OriginUnavailable {
                l1_origin,
                l2_block: finalized.block_info.id(),
            }
        }
    }
}

/// The error for a walk from the execution layer's unsafe head that reached the height of its
/// finalized block without meeting that block. A forkchoice built from such a walk would either
/// finalize a block that is not an ancestor of its head or move finality back, so the sync start
/// refuses it, like op-node's `ReorgFinalizedErr`.
fn finalized_block_not_on_walked_chain(
    finalized: &L2BlockInfo,
    walked: &L2BlockInfo,
) -> SyncStartError {
    error!(
        target: "sync_start",
        finalized = %finalized.block_info.number,
        finalized_hash = %finalized.block_info.hash,
        walked = %walked.block_info.number,
        walked_hash = %walked.block_info.hash,
        "The chain walked from the execution layer's unsafe head does not contain its finalized block"
    );
    SyncStartError::MismatchedFinalizedBlock {
        finalized: finalized.block_info.id(),
        walked: walked.block_info.id(),
    }
}

#[cfg(test)]
mod finality_tests {
    use super::*;
    use crate::test_utils::{
        expect_sync_start_walk, test_engine_client, test_l2_block_at_l1_origin, test_l2_chain,
        test_rpc_block,
    };
    use alloy_primitives::B256;
    use std::sync::Arc;

    const CANONICAL: u8 = 0;
    const FORK: u8 = 1;

    #[tokio::test]
    async fn walk_that_meets_the_finalized_block_stops_there() {
        let cfg = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let canonical = test_l2_chain(B256::ZERO, 1, 4, CANONICAL);
        expect_sync_start_walk(
            &l1,
            &l2,
            &canonical[1],
            &canonical[3],
            true,
            &[&canonical[2], &canonical[1]],
        );

        let start = find_starting_forkchoice(&cfg, &client).await.unwrap();

        assert_eq!(start.safe.block_info.hash, canonical[1].header.hash_slow());
        assert_eq!(start.finalized, start.safe);
        l1.assert_finished();
        l2.assert_finished();
    }

    /// The execution layer's unsafe head is on a fork from below its finalized block, as after a
    /// rejected forkchoice update that it made canonical first, so the safe walk reaches the
    /// finalized height on another block.
    #[tokio::test]
    async fn walk_on_a_fork_below_the_finalized_block_is_refused() {
        let cfg = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let canonical = test_l2_chain(B256::ZERO, 1, 2, CANONICAL);
        let fork = test_l2_chain(canonical[0].header.hash_slow(), 2, 4, FORK);
        expect_sync_start_walk(&l1, &l2, &canonical[1], &fork[2], true, &[&fork[1], &fork[0]]);

        let error = find_starting_forkchoice(&cfg, &client).await.unwrap_err();

        assert!(
            matches!(
                error,
                SyncStartError::MismatchedFinalizedBlock { finalized, walked }
                    if finalized.hash == canonical[1].header.hash_slow()
                        && walked.hash == fork[0].header.hash_slow()
            ),
            "got {error:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A safe walk that the sequencing window stops above the finalized height still checks the
    /// finalized block: on a fork from below it, the canonical block at its height differs.
    #[tokio::test]
    async fn walk_stopped_above_finality_by_the_sequencing_window_checks_the_finalized_block() {
        let cfg = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let canonical = test_l2_chain(B256::ZERO, 1, 2, CANONICAL);
        let fork_at_2 = test_l2_block_at_l1_origin(2, canonical[0].header.hash_slow(), FORK, 0);
        let fork_at_3 = test_l2_block_at_l1_origin(3, fork_at_2.header.hash_slow(), FORK, 1);
        let fork_at_4 = test_l2_block_at_l1_origin(4, fork_at_3.header.hash_slow(), FORK, 2);
        expect_sync_start_walk(&l1, &l2, &canonical[1], &fork_at_4, true, &[&fork_at_3]);
        l2.expect_block_header_by_number(2, Some(test_rpc_block(&fork_at_2)));

        let error = find_starting_forkchoice(&cfg, &client).await.unwrap_err();

        assert!(
            matches!(
                error,
                SyncStartError::MismatchedFinalizedBlock { finalized, walked }
                    if finalized.hash == canonical[1].header.hash_slow()
                        && walked.hash == fork_at_2.header.hash_slow()
            ),
            "got {error:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    /// An execution layer that answers no block at its finalized height, although the walk shows
    /// its canonical chain above it, answered inconsistently; the sync start reports that, rather
    /// than a missing block, so the reset is retried.
    #[tokio::test]
    async fn missing_block_at_the_finalized_height_is_reported_as_such() {
        let cfg = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let canonical = test_l2_chain(B256::ZERO, 1, 2, CANONICAL);
        let at_3 = test_l2_block_at_l1_origin(3, canonical[1].header.hash_slow(), CANONICAL, 1);
        let at_4 = test_l2_block_at_l1_origin(4, at_3.header.hash_slow(), CANONICAL, 2);
        expect_sync_start_walk(&l1, &l2, &canonical[1], &at_4, true, &[&at_3]);
        l2.expect_block_header_by_number(2, None);

        let error = find_starting_forkchoice(&cfg, &client).await.unwrap_err();

        assert!(matches!(error, SyncStartError::FinalizedHeightUnavailable(2)), "got {error:?}");
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A missing L1 origin of the finalized block, with no L1 block at its height, means the L1
    /// provider is behind, since that block's L1 data is finalized, so the reset waits for it
    /// instead of moving finality back.
    #[tokio::test]
    async fn missing_l1_origin_of_the_finalized_block_waits_for_the_l1_provider() {
        let cfg = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let canonical = test_l2_chain(B256::ZERO, 1, 2, CANONICAL);
        expect_sync_start_walk(&l1, &l2, &canonical[1], &canonical[1], false, &[]);
        l1.expect("eth_getBlockByNumber", None::<crate::test_utils::TestRpcBlock>);

        let error = find_starting_forkchoice(&cfg, &client).await.unwrap_err();

        assert!(
            matches!(
                error,
                SyncStartError::FinalizedL1OriginUnavailable { l2_block, .. }
                    if l2_block.hash == canonical[1].header.hash_slow()
            ),
            "got {error:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    /// A missing L1 origin of another block at the finalized height is that fork's L1 origin, so
    /// the walk refuses the fork instead of waiting for the L1 provider.
    #[tokio::test]
    async fn missing_l1_origin_of_a_fork_block_at_the_finalized_height_is_refused() {
        let cfg = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(cfg.clone());
        let canonical = test_l2_chain(B256::ZERO, 1, 2, CANONICAL);
        let fork = test_l2_chain(canonical[0].header.hash_slow(), 2, 2, FORK);
        expect_sync_start_walk(&l1, &l2, &canonical[1], &fork[0], false, &[]);

        let error = find_starting_forkchoice(&cfg, &client).await.unwrap_err();

        assert!(
            matches!(
                error,
                SyncStartError::MismatchedFinalizedBlock { finalized, walked }
                    if finalized.hash == canonical[1].header.hash_slow()
                        && walked.hash == fork[0].header.hash_slow()
            ),
            "got {error:?}"
        );
        l1.assert_finished();
        l2.assert_finished();
    }
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
