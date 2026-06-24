//! Contains the forkchoice state for the L2.

use crate::{EngineClient, SyncStartError};
use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_provider::Network;
use alloy_transport::TransportResult;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_network::Optimism;
use std::{fmt::Display, time::Duration};

/// An unsafe, safe, and finalized [`L2BlockInfo`] returned by the
/// [`crate::find_starting_forkchoice`] function.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct L2ForkchoiceState {
    /// The unsafe L2 block.
    pub un_safe: L2BlockInfo,
    /// The safe L2 block.
    pub safe: L2BlockInfo,
    /// The finalized L2 block.
    pub finalized: L2BlockInfo,
}

impl Display for L2ForkchoiceState {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "FINALIZED: {} (#{}) | SAFE: {} (#{}) | UNSAFE: {} (#{})",
            self.finalized.block_info.hash,
            self.finalized.block_info.number,
            self.safe.block_info.hash,
            self.safe.block_info.number,
            self.un_safe.block_info.hash,
            self.un_safe.block_info.number,
        )
    }
}

impl L2ForkchoiceState {
    /// Fetches the current forkchoice state of the L2 execution layer.
    ///
    /// - The finalized block may not always be available. If it is not, we fall back to genesis.
    /// - The safe block may not always be available. If it is not, we fall back to the finalized
    ///   block.
    /// - The unsafe block is always assumed to be available.
    pub async fn current<EngineClient_: EngineClient>(
        cfg: &RollupConfig,
        engine_client: &EngineClient_,
    ) -> Result<Self, SyncStartError> {
        let finalized = {
            let rpc_block =
                match get_block_compat(engine_client, BlockNumberOrTag::Finalized.into()).await {
                    Ok(Some(block)) => block,
                    Ok(None) => engine_client
                        .get_l2_block(cfg.genesis.l2.number.into())
                        .full()
                        .await?
                        .ok_or(SyncStartError::BlockNotFound(cfg.genesis.l2.number.into()))?,
                    Err(e) => return Err(e.into()),
                }
                .into_consensus();

            L2BlockInfo::from_block_and_genesis(&rpc_block, &cfg.genesis)?
        };
        let safe = match get_block_compat(engine_client, BlockNumberOrTag::Safe.into()).await {
            Ok(Some(block)) => {
                L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &cfg.genesis)?
            }
            Ok(None) => finalized,
            Err(e) => return Err(e.into()),
        };
        let un_safe = {
            let rpc_block = get_block_compat(engine_client, BlockNumberOrTag::Latest.into())
                .await?
                .ok_or(SyncStartError::BlockNotFound(BlockNumberOrTag::Latest.into()))?;
            L2BlockInfo::from_block_and_genesis(&rpc_block.into_consensus(), &cfg.genesis)?
        };

        Ok(Self { un_safe, safe, finalized })
    }

    /// Fetches the forkchoice state to install after **execution-layer (EL) sync** completes.
    ///
    /// This mirrors op-node's behavior for `--syncmode.offset-el-safe`: the unsafe head stays at
    /// the synced tip, while the safe and finalized heads are retracted by
    /// `ceil(offset / block_time)` blocks (see [`offset_block_num`]). Retracting forces derivation
    /// to re-derive the most recent `offset` worth of blocks before they are considered safe,
    /// rather than trusting the optimistically-synced tip.
    ///
    /// To avoid re-deriving the offset window on every restart (kona has no persisted safe head),
    /// the safe and finalized heads are never retracted below the EL's currently-reported safe and
    /// finalized heads (read via [`Self::current`]).
    ///
    /// This relies on a property of kona's EL-sync forkchoice updates: the safe and finalized heads
    /// are only ever advanced by *derivation*, never during EL sync (P2P-driven `InsertTask`s send
    /// forkchoice updates with the safe/finalized heads left at their default). So while the chain
    /// is being optimistically synced the EL is told `safe = finalized = 0x0` and reports no
    /// safe/finalized tag; [`Self::current`] then falls back to genesis. Hence on the **initial**
    /// post-EL-sync reset the EL heads are genesis and the offset applies in full, whereas on a
    /// **later restart** the EL has the safe/finalized heads that derivation previously committed
    /// (already at or above the offset head), so nothing is re-derived.
    pub async fn for_el_sync<EngineClient_: EngineClient>(
        cfg: &RollupConfig,
        engine_client: &EngineClient_,
        offset: Duration,
    ) -> Result<Self, SyncStartError> {
        let current = Self::current(cfg, engine_client).await?;
        let tip = current.un_safe;

        let target =
            offset_block_num(offset, cfg.block_time, tip.block_info.number, cfg.genesis.l2.number);
        let offset_ref = if target < tip.block_info.number {
            let block_id = BlockId::Number(target.into());
            let block = engine_client
                .get_l2_block(block_id)
                .full()
                .await?
                .ok_or(SyncStartError::BlockNotFound(block_id))?;
            L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &cfg.genesis)?
        } else {
            tip
        };

        Ok(current.with_el_sync_offset(offset_ref))
    }

    /// Applies the EL-sync offset head to the current forkchoice, never retracting the safe or
    /// finalized heads below where they already are. Pure helper split out of [`Self::for_el_sync`]
    /// to keep the head-selection logic unit-testable without an engine client.
    const fn with_el_sync_offset(self, offset_ref: L2BlockInfo) -> Self {
        let safe = if self.safe.block_info.number >= offset_ref.block_info.number {
            self.safe
        } else {
            offset_ref
        };
        let finalized = if self.finalized.block_info.number >= offset_ref.block_info.number {
            self.finalized
        } else {
            offset_ref
        };
        Self { un_safe: self.un_safe, safe, finalized }
    }
}

/// Returns `ceil(offset / block_time)` as a block count.
///
/// Ceiling division is used so the retraction always covers the full requested duration. A zero
/// offset or zero block time yields `0`. Sub-second remainders are truncated, matching op-node's
/// `DurationToBlocks`.
pub const fn duration_to_blocks(offset: Duration, block_time: u64) -> u64 {
    if offset.is_zero() || block_time == 0 {
        return 0;
    }
    let secs = offset.as_secs();
    secs.div_ceil(block_time)
}

/// Returns the block number that is `offset` behind `head`, clamped so it never goes below
/// `genesis`. Returns `head` unchanged when the offset is zero or `head` is already at or below
/// `genesis`. Mirrors op-node's `OffsetBlockNum`.
pub const fn offset_block_num(offset: Duration, block_time: u64, head: u64, genesis: u64) -> u64 {
    let n = duration_to_blocks(offset, block_time);
    if n == 0 || head <= genesis {
        return head;
    }
    let max_retract = head - genesis;
    let n = if n > max_retract { max_retract } else { n };
    head - n
}

/// Wrapper function around [`EngineClient::get_l2_block`] to handle compatibility issues with geth
/// and erigon. When serving a block-by-number request, these clients will return non-standard
/// errors for the safe and finalized heads when the chain has just started and nothing is marked as
/// safe or finalized yet.
async fn get_block_compat<EngineClient_: EngineClient>(
    engine_client: &EngineClient_,
    block_id: BlockId,
) -> TransportResult<Option<<Optimism as Network>::BlockResponse>> {
    match engine_client.get_l2_block(block_id).full().await {
        Err(e) => {
            let err_str = e.to_string();
            if err_str.contains("block not found") || err_str.contains("Unknown block") {
                Ok(None)
            } else {
                Err(e)
            }
        }
        r => r,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_protocol::BlockInfo;
    use rstest::rstest;

    #[rstest]
    #[case::zero_offset(Duration::ZERO, 2, 0)]
    #[case::zero_block_time(Duration::from_secs(3600), 0, 0)]
    #[case::rounds_up(Duration::from_secs(3), 2, 2)]
    #[case::exact_multiple(Duration::from_secs(4), 2, 2)]
    #[case::rounds_up_larger_block_time(Duration::from_secs(15), 4, 4)]
    #[case::exact_multiple_larger_block_time(Duration::from_secs(16), 4, 4)]
    #[case::sub_second_truncates(Duration::from_millis(500), 1, 0)]
    #[case::twelve_hours(Duration::from_secs(12 * 3600), 2, 21600)]
    fn test_duration_to_blocks(
        #[case] offset: Duration,
        #[case] block_time: u64,
        #[case] expected: u64,
    ) {
        assert_eq!(duration_to_blocks(offset, block_time), expected);
    }

    #[rstest]
    #[case::zero_offset(Duration::ZERO, 2, 100, 0, 100)]
    #[case::head_at_genesis(Duration::from_secs(10), 2, 0, 0, 0)]
    #[case::head_below_genesis(Duration::from_secs(10), 2, 5, 10, 5)]
    #[case::retracts_five_blocks(Duration::from_secs(10), 2, 100, 0, 95)]
    #[case::clamped_at_genesis(Duration::from_secs(1000 * 3600), 2, 10, 0, 0)]
    #[case::retracts_exactly_to_genesis(Duration::from_secs(10), 2, 15, 10, 10)]
    #[case::rounds_up(Duration::from_secs(15), 4, 100, 0, 96)]
    fn test_offset_block_num(
        #[case] offset: Duration,
        #[case] block_time: u64,
        #[case] head: u64,
        #[case] genesis: u64,
        #[case] expected: u64,
    ) {
        assert_eq!(offset_block_num(offset, block_time, head, genesis), expected);
    }

    /// Builds an [`L2BlockInfo`] carrying only the block number, which is all the EL-sync
    /// head-selection logic inspects.
    fn block_at(number: u64) -> L2BlockInfo {
        L2BlockInfo { block_info: BlockInfo { number, ..Default::default() }, ..Default::default() }
    }

    #[test]
    fn with_el_sync_offset_retracts_to_offset_head_on_initial_sync() {
        // On the initial EL sync the EL reports genesis (0) for safe and finalized, so both
        // retract to the offset head while the unsafe head stays at the tip.
        let current = L2ForkchoiceState {
            un_safe: block_at(1000),
            safe: block_at(0),
            finalized: block_at(0),
        };
        let result = current.with_el_sync_offset(block_at(900));
        assert_eq!(result.un_safe.block_info.number, 1000);
        assert_eq!(result.safe.block_info.number, 900);
        assert_eq!(result.finalized.block_info.number, 900);
    }

    #[test]
    fn with_el_sync_offset_never_retracts_below_current_heads() {
        // On a restart the EL already reports safe/finalized at or above the offset head, so they
        // are left untouched and the offset window is not re-derived.
        let current = L2ForkchoiceState {
            un_safe: block_at(1000),
            safe: block_at(950),
            finalized: block_at(920),
        };
        let result = current.with_el_sync_offset(block_at(900));
        assert_eq!(result.un_safe.block_info.number, 1000);
        assert_eq!(result.safe.block_info.number, 950);
        assert_eq!(result.finalized.block_info.number, 920);
    }

    #[test]
    fn with_el_sync_offset_advances_finalized_to_offset_when_behind() {
        // Mixed case: safe is ahead of the offset head (kept), finalized is behind (advanced).
        let current = L2ForkchoiceState {
            un_safe: block_at(1000),
            safe: block_at(950),
            finalized: block_at(800),
        };
        let result = current.with_el_sync_offset(block_at(900));
        assert_eq!(result.safe.block_info.number, 950);
        assert_eq!(result.finalized.block_info.number, 900);
    }
}
