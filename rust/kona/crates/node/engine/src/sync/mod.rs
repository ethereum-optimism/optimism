//! Sync start algorithm for the OP Stack rollup node.

use alloy_eips::BlockId;
use alloy_primitives::B256;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};

mod forkchoice;
pub use forkchoice::L2ForkchoiceState;

mod error;
pub use error::SyncStartError;

use std::time::Duration;
use tracing::{info, warn};

use crate::EngineClient;

/// Matches op-node's threshold for recovering EL-sync forkchoice labels left at genesis.
const RECOVER_MIN_SEQ_WINDOWS: u64 = 14;

/// Matches op-node's `MaxReorgSeqWindows`: how many sync lookback windows the unsafe head's L1
/// origin may be rewound before recovery gives up.
const MAX_REORG_SEQ_WINDOWS: u64 = 5;

/// How many L2 blocks the walk traverses between progress logs.
const PROGRESS_LOG_INTERVAL: u64 = 10_000;

/// Matches op-node's default `--syncmode.offset-el-safe`: how far behind an EL-synced tip the
/// recovered safe and finalized heads are placed. The tip arrived over P2P and may not be
/// batched to L1 yet, so it must not be labelled safe or finalized.
pub const DEFAULT_EL_SYNC_SAFE_OFFSET: Duration = Duration::from_secs(12 * 60 * 60);

/// Searches for a sync starting point using op-node's `FindL2Heads` rules, and returns:
///
/// - The *unsafe* head: the highest L2 block whose L1 origin, and every ancestor's, is a plausible
///   extension of the canonical L1 chain. An origin not yet visible on L1 is plausible.
/// - The *safe* head: the highest L2 block whose epoch's sequencing window is complete within the
///   canonical L1 chain. Recovery stops at an epoch boundary below both the previous safe head and
///   a full lookback window, or at finality.
/// - The *finalized* head: unchanged, as it is fully derived from finalized L1 data.
///
/// When the execution layer lost its safe and finalized labels after EL sync, or was rewound
/// below them, the unsafe head stays at the tip and safe and finalized are placed
/// `el_sync_safe_offset` behind it, as op-node's `L2HeadsForELSyncWithOffset` does.
pub async fn find_starting_forkchoice(
    cfg: &RollupConfig,
    engine_client: &EngineClient,
    el_sync_safe_offset: Duration,
) -> Result<L2ForkchoiceState, SyncStartError> {
    let mut current_fc = L2ForkchoiceState::current(cfg, engine_client).await?;
    info!(
        target: "sync_start",
        unsafe = %current_fc.un_safe.block_info.number,
        safe = %current_fc.safe.block_info.number,
        finalized = %current_fc.finalized.block_info.number,
        "Loaded current L2 EL forkchoice state"
    );

    let missing_finality = current_fc.finalized.hash() == cfg.genesis.l2.hash &&
        current_fc.safe.hash() == cfg.genesis.l2.hash &&
        current_fc.un_safe.block_info.number > cfg.genesis.l2.number &&
        current_fc.un_safe.l1_origin.number >
            cfg.genesis
                .l1
                .number
                .saturating_add(RECOVER_MIN_SEQ_WINDOWS.saturating_mul(cfg.seq_window_size));
    let rewound = current_fc.un_safe.block_info.number < current_fc.safe.block_info.number ||
        current_fc.un_safe.block_info.number < current_fc.finalized.block_info.number;
    if missing_finality || rewound {
        // Handle this before the normal walk: stale labels are not a reorg finality boundary.
        warn!(target: "sync_start", ?current_fc, missing_finality, rewound, ?el_sync_safe_offset, "Recovering missing or stale execution-layer forkchoice labels");
        return heads_for_el_sync(cfg, engine_client, el_sync_safe_offset, current_fc.un_safe)
            .await;
    }

    let previous_unsafe = current_fc.un_safe;
    let previous_safe = current_fc.safe;
    let mut cursor = previous_unsafe;
    let mut unsafe_candidate = Some(cursor);
    let mut highest_canonical: Option<L2BlockInfo> = None;
    let mut walker = SyncWalker::new(cfg, engine_client);
    let lookback = cfg.seq_window_size;
    let mut walked = 0u64;

    // Port of op-node's normal FindL2Heads walk. Keep plausible heads ahead of L1, discard
    // replaced origins, and validate every traversed child/parent epoch and sequence relation.
    loop {
        if cursor.block_info.number < current_fc.finalized.block_info.number ||
            (cursor.block_info.number == current_fc.finalized.block_info.number &&
                cursor.hash() != current_fc.finalized.hash())
        {
            return Err(SyncStartError::MismatchedFinalizedBlock(
                current_fc.finalized.hash(),
                cursor.hash(),
            ));
        }
        let origin = walker.origin(cursor).await?;
        if cursor.block_info.number == cfg.genesis.l2.number {
            if cursor.hash() != cfg.genesis.l2.hash {
                return Err(SyncStartError::InvalidL2GenesisHash(
                    cfg.genesis.l2.hash,
                    cursor.hash(),
                ));
            }
            if let Some(l1) = walker.l1 &&
                l1.hash != cfg.genesis.l1.hash
            {
                return Err(SyncStartError::InvalidL1GenesisHash(cfg.genesis.l1.hash, l1.hash));
            }
        }
        let unsafe_head = match unsafe_candidate {
            Some(head) => head,
            None => {
                if cursor
                    .l1_origin
                    .number
                    .saturating_add(MAX_REORG_SEQ_WINDOWS.saturating_mul(lookback)) <
                    previous_unsafe.l1_origin.number
                {
                    return Err(SyncStartError::ReorgTooDeep);
                }
                unsafe_candidate = Some(cursor);
                cursor
            }
        };
        match origin {
            OriginStatus::Ahead => highest_canonical = None,
            OriginStatus::Canonical => {
                highest_canonical.get_or_insert(cursor);
            }
            OriginStatus::Replaced => {
                unsafe_candidate = None;
                highest_canonical = None;
            }
        }
        if cursor.block_info.number == current_fc.finalized.block_info.number {
            // op-node returns an empty unsafe head here. Refuse instead: L1 reorged below the
            // finalized block's origin, which finality rules out.
            current_fc.un_safe =
                unsafe_candidate.ok_or(SyncStartError::FinalizedL1OriginReplaced)?;
            current_fc.safe = cursor;
            break;
        }
        // Use the parent of an epoch boundary below both the previous safe head and a full
        // canonical lookback window. Finality above is the lower bound.
        let ready = cursor.block_info.number <= previous_safe.block_info.number &&
            highest_canonical.is_some_and(|high| {
                cursor.l1_origin.number.saturating_add(lookback) < high.l1_origin.number
            }) &&
            cursor.seq_num == 0;
        cursor = walker.parent(cursor).await?;
        if ready {
            // A ready block's origin is canonical, so this iteration kept its unsafe head.
            current_fc.un_safe = unsafe_head;
            current_fc.safe = cursor;
            break;
        }
        walked += 1;
        if walked.is_multiple_of(PROGRESS_LOG_INTERVAL) {
            info!(
                target: "sync_start",
                walked,
                cursor = cursor.block_info.number,
                finalized = current_fc.finalized.block_info.number,
                "Walking back to find the sync start"
            );
        }
    }
    info!(target: "sync_start", walked, %current_fc, "Found sync start");

    // Leave the finalized block as-is, and return the current forkchoice.
    Ok(current_fc)
}

/// Port of op-node's `L2HeadsForELSyncWithOffset`: the unsafe head stays at the EL-synced `tip`,
/// and safe and finalized retract by `offset`, converted to blocks with ceiling division and
/// clamped at genesis.
async fn heads_for_el_sync(
    cfg: &RollupConfig,
    engine_client: &EngineClient,
    offset: Duration,
    tip: L2BlockInfo,
) -> Result<L2ForkchoiceState, SyncStartError> {
    let target =
        offset_block_number(offset, cfg.block_time, tip.block_info.number, cfg.genesis.l2.number);
    let derived = if target < tip.block_info.number {
        let id = BlockId::number(target);
        let block = engine_client
            .get_l2_block(id)
            .full()
            .await?
            .ok_or(SyncStartError::BlockNotFound(id))?;
        L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &cfg.genesis)?
    } else {
        tip
    };
    Ok(L2ForkchoiceState { un_safe: tip, safe: derived, finalized: derived })
}

/// Port of op-node's `OffsetBlockNum`: the block number `ceil(offset / block_time)` blocks behind
/// `head`, never below `genesis`.
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

#[derive(Debug, PartialEq, Eq)]
enum OriginStatus {
    Canonical,
    Replaced,
    Ahead,
}

/// The same L1-origin cursor and child/parent checks used by op-node `FindL2Heads`.
struct SyncWalker<'a> {
    cfg: &'a RollupConfig,
    client: &'a EngineClient,
    l1: Option<BlockInfo>,
}

impl<'a> SyncWalker<'a> {
    const fn new(cfg: &'a RollupConfig, client: &'a EngineClient) -> Self {
        Self { cfg, client, l1: None }
    }

    async fn origin(&mut self, head: L2BlockInfo) -> Result<OriginStatus, SyncStartError> {
        if self.l1.is_none_or(|l1| l1.hash != head.l1_origin.hash) {
            // A by-hash read is canonical only when it follows the parent of a known L1 block.
            let id = if self.l1.is_some_and(|l1| l1.parent_hash == head.l1_origin.hash) {
                head.l1_origin.hash.into()
            } else {
                BlockId::number(head.l1_origin.number)
            };
            let block = self.client.get_l1_block(id).await?;
            if block.is_none() && matches!(id, BlockId::Hash(_)) {
                return Err(SyncStartError::ChainViewChanged);
            }
            self.l1 = block.map(|block| BlockInfo {
                hash: block.header.hash,
                number: block.header.number,
                parent_hash: block.header.parent_hash,
                timestamp: block.header.timestamp,
            });
        }
        Ok(match self.l1 {
            None => OriginStatus::Ahead,
            Some(l1) if l1.hash == head.l1_origin.hash => OriginStatus::Canonical,
            Some(_) => OriginStatus::Replaced,
        })
    }

    async fn parent(&self, child: L2BlockInfo) -> Result<L2BlockInfo, SyncStartError> {
        let id = child.block_info.parent_hash.into();
        let block =
            self.client.get_l2_block(id).full().await?.ok_or(SyncStartError::BlockNotFound(id))?;
        let parent =
            L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &self.cfg.genesis)?;
        let canonical_parent =
            self.l1.filter(|l1| l1.hash == child.l1_origin.hash).map(|l1| l1.parent_hash);
        check_epoch_and_sequence(child, parent, canonical_parent)?;
        Ok(parent)
    }
}

// Ported from op-node/rollup/sync/start.go's L1-origin relation check. Compare full origin IDs:
// a same-height hash replacement is an epoch inconsistency, not a same-origin sequence step.
fn check_epoch_and_sequence(
    child: L2BlockInfo,
    parent: L2BlockInfo,
    canonical_parent: Option<B256>,
) -> Result<(), SyncStartError> {
    if parent.l1_origin != child.l1_origin {
        if parent.l1_origin.number.checked_add(1) != Some(child.l1_origin.number) {
            return Err(SyncStartError::L1OriginMismatch);
        }
        if child.seq_num != 0 {
            return Err(SyncStartError::NonZeroSequenceNumber);
        }
        if canonical_parent.is_some_and(|hash| hash != parent.l1_origin.hash) {
            return Err(SyncStartError::L1OriginMismatch);
        }
    } else if parent.seq_num.checked_add(1) != Some(child.seq_num) {
        return Err(SyncStartError::InconsistentSequenceNumber);
    }
    Ok(())
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
mod tests;
