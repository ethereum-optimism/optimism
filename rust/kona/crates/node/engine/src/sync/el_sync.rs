//! The forkchoice state to adopt when execution-layer sync completes.

use alloy_eips::BlockId;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::time::Duration;
use tracing::info;

use crate::{EngineClient, L2ForkchoiceState, SyncStartError, find_starting_forkchoice};

/// Matches op-node's default `--syncmode.offset-el-safe`: how far behind an EL-synced tip the
/// safe and finalized heads are placed. The tip arrived over P2P and may not be batched to L1
/// yet, so it must not be labelled safe or finalized.
pub const DEFAULT_EL_SYNC_SAFE_OFFSET: Duration = Duration::from_secs(12 * 60 * 60);

/// Returns the forkchoice state to adopt once execution-layer sync has completed.
///
/// Like op-node, EL sync applies only when the execution layer had no finalized block beyond
/// genesis. Then the unsafe head stays at the synced tip, and safe and finalized are placed
/// `el_sync_safe_offset` behind it, as op-node's `headsAfterELSync` does when it has no usable
/// safe head database. The execution layer's own safe label is kept when it is closer to the tip.
/// Otherwise the execution layer already had finality, so this is a restart and the heads come
/// from [`find_starting_forkchoice`].
pub async fn find_el_sync_forkchoice(
    cfg: &RollupConfig,
    engine_client: &EngineClient,
    el_sync_safe_offset: Duration,
) -> Result<L2ForkchoiceState, SyncStartError> {
    let current = L2ForkchoiceState::current(cfg, engine_client).await?;
    if current.finalized.block_info.hash != cfg.genesis.l2.hash {
        return find_starting_forkchoice(cfg, engine_client).await;
    }

    let tip = current.un_safe;
    let target = offset_block_number(
        el_sync_safe_offset,
        cfg.block_time,
        tip.block_info.number,
        cfg.genesis.l2.number,
    );
    let offset_head = if target < tip.block_info.number {
        let block = engine_client.get_l2_block(BlockId::number(target)).full().await?.ok_or(
            SyncStartError::ElSyncOffsetBlockUnavailable {
                number: target,
                offset: el_sync_safe_offset,
            },
        )?;
        L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &cfg.genesis)?
    } else {
        tip
    };
    let safe = if current.safe.block_info.number > offset_head.block_info.number &&
        current.safe.block_info.number <= tip.block_info.number
    {
        current.safe
    } else {
        offset_head
    };
    let forkchoice = L2ForkchoiceState { un_safe: tip, safe, finalized: offset_head };
    info!(target: "sync_start", ?el_sync_safe_offset, %forkchoice, "Adopting EL-synced forkchoice");
    Ok(forkchoice)
}

/// Port of op-node's `OffsetBlockNum`: the block number `ceil(offset / block_time)` blocks behind
/// `head`, never below `genesis`. Like op-node, sub-second parts of `offset` are ignored.
pub(crate) fn offset_block_number(
    offset: Duration,
    block_time: u64,
    head: u64,
    genesis: u64,
) -> u64 {
    if block_time == 0 || head <= genesis {
        return head;
    }
    head - offset.as_secs().div_ceil(block_time).min(head - genesis)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{RpcMock, test_engine_client, test_l1_hash, test_l2_block};
    use alloy_primitives::B256;
    use serde_json::json;
    use std::sync::Arc;

    type L1Block = alloy_rpc_types_eth::Block<alloy_rpc_types_eth::Transaction>;
    type L2Block = alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction>;

    /// Two-second blocks, so a 10s offset is 5 blocks.
    const BLOCK_TIME: u64 = 2;
    const OFFSET: Duration = Duration::from_secs(10);

    fn info(block: &L2Block, cfg: &RollupConfig) -> L2BlockInfo {
        L2BlockInfo::from_block_and_genesis(&block.clone().into_consensus(), &cfg.genesis).unwrap()
    }

    fn expect_heads(mock: &RpcMock, finalized: &L2Block, safe: &L2Block, head: &L2Block) {
        for (tag, block) in [("finalized", finalized), ("safe", safe), ("latest", head)] {
            mock.expect_params("eth_getBlockByNumber", json!([tag, true]), block);
        }
    }

    fn config(genesis: &L2Block) -> RollupConfig {
        let mut cfg = RollupConfig { block_time: BLOCK_TIME, ..Default::default() };
        cfg.genesis.l2.hash = genesis.header.hash;
        cfg
    }

    // With finalized at genesis, the execution layer was EL-synced: safe and finalized go the
    // offset behind the tip, clamped at genesis, unless its safe label is closer to the tip.
    #[rstest::rstest]
    #[case::offset_behind_tip(100, 0, 95, 95)]
    #[case::clamped_at_genesis(3, 0, 0, 0)]
    #[case::safe_label_closer_to_tip(100, 98, 98, 95)]
    #[case::safe_label_further_back(100, 50, 95, 95)]
    #[tokio::test]
    async fn el_synced_heads_are_placed_the_offset_behind_the_tip(
        #[case] tip: u64,
        #[case] safe_label: u64,
        #[case] expected_safe: u64,
        #[case] expected_finalized: u64,
    ) {
        let genesis = test_l2_block(0, 0, 0, B256::ZERO);
        let cfg = config(&genesis);
        let block = |number: u64| {
            if number == 0 { genesis.clone() } else { test_l2_block(number, number, 0, B256::ZERO) }
        };
        let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
        expect_heads(&l2, &genesis, &block(safe_label), &block(tip));
        if expected_finalized < tip {
            l2.expect_params(
                "eth_getBlockByNumber",
                json!([format!("{expected_finalized:#x}"), true]),
                block(expected_finalized),
            );
        }

        let result = find_el_sync_forkchoice(&cfg, &client, OFFSET).await.unwrap();

        assert_eq!(result.un_safe, info(&block(tip), &cfg));
        assert_eq!(result.safe, info(&block(expected_safe), &cfg));
        assert_eq!(result.finalized, info(&block(expected_finalized), &cfg));
        l1.assert_finished();
        l2.assert_finished();
    }

    // With a zero offset, safe and finalized are the tip and nothing else is fetched.
    #[tokio::test]
    async fn zero_offset_uses_the_tip() {
        let genesis = test_l2_block(0, 0, 0, B256::ZERO);
        let cfg = config(&genesis);
        let tip = test_l2_block(100, 100, 0, B256::ZERO);
        let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
        expect_heads(&l2, &genesis, &genesis, &tip);

        let result = find_el_sync_forkchoice(&cfg, &client, Duration::ZERO).await.unwrap();

        let tip = info(&tip, &cfg);
        assert_eq!(result, L2ForkchoiceState { un_safe: tip, safe: tip, finalized: tip });
        l1.assert_finished();
        l2.assert_finished();
    }

    // A safe label above the tip is stale, so safe goes to the offset head instead.
    #[tokio::test]
    async fn safe_label_above_the_tip_is_ignored() {
        let genesis = test_l2_block(0, 0, 0, B256::ZERO);
        let cfg = config(&genesis);
        let tip = test_l2_block(100, 100, 0, B256::ZERO);
        let offset_head = test_l2_block(95, 95, 0, B256::ZERO);
        let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
        expect_heads(&l2, &genesis, &test_l2_block(120, 120, 0, B256::ZERO), &tip);
        l2.expect_params("eth_getBlockByNumber", json!(["0x5f", true]), &offset_head);

        let result = find_el_sync_forkchoice(&cfg, &client, OFFSET).await.unwrap();

        let offset_head = info(&offset_head, &cfg);
        assert_eq!(
            result,
            L2ForkchoiceState {
                un_safe: info(&tip, &cfg),
                safe: offset_head,
                finalized: offset_head
            }
        );
        l1.assert_finished();
        l2.assert_finished();
    }

    // An execution layer that pruned the block at the offset height cannot complete EL sync. The
    // error names the setting to change, and retrying would not help.
    #[tokio::test]
    async fn pruned_offset_block_is_reported() {
        let genesis = test_l2_block(0, 0, 0, B256::ZERO);
        let cfg = config(&genesis);
        let tip = test_l2_block(100, 100, 0, B256::ZERO);
        let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
        expect_heads(&l2, &genesis, &genesis, &tip);
        l2.expect_params("eth_getBlockByNumber", json!(["0x5f", true]), Option::<L2Block>::None);

        let err = find_el_sync_forkchoice(&cfg, &client, OFFSET).await.unwrap_err();

        assert!(
            matches!(err, SyncStartError::ElSyncOffsetBlockUnavailable { number: 95, offset }
                if offset == OFFSET),
            "{err:?}"
        );
        assert!(err.to_string().contains("--syncmode.offset-el-safe"));
        assert!(!crate::EngineResetError::SyncStart(err).is_temporary());
        l1.assert_finished();
        l2.assert_finished();
    }

    // An execution layer that already has a finalized block beyond genesis was not EL-synced, so
    // the heads come from the regular sync-start search instead of the offset.
    #[tokio::test]
    async fn established_finality_uses_the_sync_start_search() {
        let genesis = test_l2_block(0, 0, 0, B256::ZERO);
        let cfg = config(&genesis);
        let head = test_l2_block(100, 99, 0, B256::ZERO);
        let (client, l1, l2) = test_engine_client(Arc::new(cfg.clone()));
        for _ in 0..2 {
            expect_heads(&l2, &head, &head, &head);
        }
        let mut origin = L1Block::default();
        origin.header.hash = test_l1_hash(99);
        origin.header.inner.number = 99;
        l1.expect_params("eth_getBlockByHash", json!([test_l1_hash(99), false]), origin);

        let result = find_el_sync_forkchoice(&cfg, &client, OFFSET).await.unwrap();

        let head = info(&head, &cfg);
        assert_eq!(result, L2ForkchoiceState { un_safe: head, safe: head, finalized: head });
        l1.assert_finished();
        l2.assert_finished();
    }

    // Mirrors op-node's OffsetBlockNum: ceiling division by the block time, clamped at genesis.
    #[rstest::rstest]
    #[case::zero_offset(0, 2, 100, 0, 100)]
    #[case::rounds_up(3_000, 2, 100, 0, 98)]
    #[case::whole_blocks(4_000, 2, 100, 0, 98)]
    #[case::ignores_sub_second(1_500, 2, 100, 0, 99)]
    #[case::clamped_at_genesis(1_000_000, 2, 105, 100, 100)]
    #[case::head_at_genesis(10_000, 2, 100, 100, 100)]
    #[case::zero_block_time(10_000, 0, 100, 0, 100)]
    fn offset_block_number_matches_op_node(
        #[case] offset_millis: u64,
        #[case] block_time: u64,
        #[case] head: u64,
        #[case] genesis: u64,
        #[case] expected: u64,
    ) {
        let offset = Duration::from_millis(offset_millis);
        assert_eq!(offset_block_number(offset, block_time, head, genesis), expected);
    }
}
