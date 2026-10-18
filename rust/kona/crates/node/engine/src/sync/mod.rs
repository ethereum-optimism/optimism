//! Sync start algorithm for the OP Stack rollup node.

use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_primitives::B256;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::future::Future;

mod forkchoice;
pub use forkchoice::L2ForkchoiceState;

mod error;
pub use error::SyncStartError;

use crate::EngineClient;

/// Read boundary for the walk. Production reads use the existing engine client's providers;
/// tests can describe chain relationships without synthesizing execution payloads.
trait SyncSource: Send + Sync {
    fn l1_block(
        &self,
        id: BlockId,
    ) -> impl Future<Output = Result<Option<BlockInfo>, SyncStartError>> + Send;
    fn l2_block(
        &self,
        id: BlockId,
    ) -> impl Future<Output = Result<Option<L2BlockInfo>, SyncStartError>> + Send;
}

struct EngineSyncSource<'a, C> {
    client: &'a C,
    cfg: &'a RollupConfig,
}

impl<C: EngineClient> SyncSource for EngineSyncSource<'_, C> {
    async fn l1_block(&self, id: BlockId) -> Result<Option<BlockInfo>, SyncStartError> {
        Ok(self.client.get_l1_block(id).await?.map(|block| BlockInfo {
            hash: block.header.hash,
            number: block.header.number,
            parent_hash: block.header.parent_hash,
            timestamp: block.header.timestamp,
        }))
    }

    async fn l2_block(&self, id: BlockId) -> Result<Option<L2BlockInfo>, SyncStartError> {
        let block = match self.client.get_l2_block(id).full().await {
            Err(error)
                if matches!(
                    id,
                    BlockId::Number(BlockNumberOrTag::Safe | BlockNumberOrTag::Finalized)
                ) =>
            {
                // Geth and Erigon use non-standard not-found errors for unset safety labels.
                let message = error.to_string();
                if message.contains("block not found") || message.contains("Unknown block") {
                    return Ok(None);
                }
                return Err(error.into());
            }
            result => result?,
        };
        block
            .map(|block| {
                L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &self.cfg.genesis)
            })
            .transpose()
            .map_err(Into::into)
    }
}

/// Searches for the heads from which to resume derivation.
///
/// Unsafe may extend beyond the L1 visibility captured at the start of this walk. Within that
/// visibility, an origin must agree with a canonical L1 anchor fetched by number. Safe retains
/// Kona's recent-window resume policy: the first block more than a lookback window behind the
/// highest confirmed canonical origin. Finalized remains unchanged and bounds the walk below.
/// Choosing safe below the prior label and at an epoch boundary is deferred to the bootstrap
/// lifecycle change; applying those rules here would walk back to genesis after initial EL sync.
///
/// The canonical anchor is rechecked before returning. A concurrent L1 reorg or unavailable
/// dependency returns an ordinary error: retrying the whole walk belongs to the reset owner.
/// This does not promise that L1 cannot extend or reorg after that final check.
pub async fn find_starting_forkchoice<EngineClient_: EngineClient>(
    cfg: &RollupConfig,
    engine_client: &EngineClient_,
) -> Result<L2ForkchoiceState, SyncStartError> {
    find_heads(cfg, &EngineSyncSource { client: engine_client, cfg }).await
}

async fn find_heads(
    cfg: &RollupConfig,
    source: &impl SyncSource,
) -> Result<L2ForkchoiceState, SyncStartError> {
    let initial = L2ForkchoiceState::load(cfg, source).await?;
    info!(target: "sync_start", %initial, "Loaded current L2 EL forkchoice state");
    if initial.finalized.block_info.number > initial.safe.block_info.number ||
        initial.safe.block_info.number > initial.un_safe.block_info.number ||
        initial.finalized.block_info.number < cfg.genesis.l2.number
    {
        return Err(SyncStartError::InvalidHeadOrder);
    }

    let visible = read_l1(source, BlockId::latest())
        .await?
        .ok_or(SyncStartError::BlockNotFound(BlockId::latest()))?;
    // Both windows are measured in L1 blocks. Alt-DA may require a longer lookback.
    let lookback = cfg.alt_da_config.as_ref().map_or(cfg.seq_window_size, |da| {
        cfg.seq_window_size.max(
            da.da_challenge_window
                .unwrap_or_default()
                .saturating_add(da.da_resolve_window.unwrap_or_default()),
        )
    });
    let max_reorg = lookback.saturating_mul(5);
    let mut anchor = None;
    let result = async {
        let mut current = initial.un_safe;
        let mut unsafe_head = Some(current);
        let mut highest_canonical: Option<L2BlockInfo> = None;
        // A single descending L1 cursor caches all reads of a repeated origin. Once anchored by
        // number, following parent hashes cannot splice together different canonical branches.
        let mut canonical: Option<BlockInfo> = None;
        loop {
            debug!(target: "sync_start", block = %current, "Walking reset heads");
            if current.block_info.number < initial.finalized.block_info.number {
                return Err(SyncStartError::MismatchedFinalizedBlock(
                    initial.finalized.block_info.hash,
                    current.block_info.hash,
                ));
            }
            if current.block_info.number == cfg.genesis.l2.number {
                if current.block_info.hash != cfg.genesis.l2.hash {
                    return Err(SyncStartError::InvalidL2GenesisHash(
                        cfg.genesis.l2.hash,
                        current.block_info.hash,
                    ));
                }
                if current.l1_origin != cfg.genesis.l1 {
                    return Err(SyncStartError::L1OriginMismatch);
                }
            }
            if current.block_info.number == initial.finalized.block_info.number &&
                current.block_info.hash != initial.finalized.block_info.hash
            {
                return Err(SyncStartError::MismatchedFinalizedBlock(
                    initial.finalized.block_info.hash,
                    current.block_info.hash,
                ));
            }

            let origin = current.l1_origin;
            let l1 = match canonical {
                Some(block) if block.number == origin.number => Some(block),
                Some(block) => {
                    let parent = read_l1(source, block.parent_hash.into())
                        .await?
                        .ok_or(SyncStartError::BlockNotFound(block.parent_hash.into()))?;
                    if parent.number.checked_add(1) != Some(block.number) ||
                        parent.number != origin.number
                    {
                        return Err(SyncStartError::InvalidL1Parent);
                    }
                    canonical = Some(parent);
                    Some(parent)
                }
                None if origin.number > visible.number => None,
                None => {
                    let block = read_l1(source, origin.number.into())
                        .await?
                        .ok_or(SyncStartError::BlockNotFound(origin.number.into()))?;
                    anchor = Some(block);
                    canonical = Some(block);
                    Some(block)
                }
            };
            match l1 {
                Some(block)
                    if current.block_info.number == cfg.genesis.l2.number &&
                        block.hash != cfg.genesis.l1.hash =>
                {
                    return Err(SyncStartError::InvalidL1GenesisHash(
                        cfg.genesis.l1.hash,
                        block.hash,
                    ));
                }
                _ => {}
            }

            // This limit concerns replacement of unsafe, not the later traversal to safe.
            if unsafe_head.is_none() {
                if initial.un_safe.l1_origin.number.saturating_sub(origin.number) > max_reorg {
                    return Err(SyncStartError::ReorgTooDeep);
                }
                unsafe_head = Some(current);
            }
            match l1 {
                None => highest_canonical = None,
                Some(block) if block.hash == origin.hash => {
                    highest_canonical.get_or_insert(current);
                }
                Some(_) => {
                    unsafe_head = None;
                    highest_canonical = None;
                }
            }

            if current.block_info.number == initial.finalized.block_info.number &&
                unsafe_head.is_none()
            {
                return Err(SyncStartError::FinalizedL1OriginMismatch);
            }
            // Keep the existing bounded recent-window resume policy until bootstrap can
            // establish a useful safe checkpoint. Invisible origins never complete this window.
            let behind_window = highest_canonical.is_some_and(|highest| {
                highest.l1_origin.number.saturating_sub(origin.number) > lookback
            });
            if behind_window || current.block_info.number == initial.finalized.block_info.number {
                return Ok(L2ForkchoiceState {
                    un_safe: unsafe_head.ok_or(SyncStartError::FinalizedL1OriginMismatch)?,
                    safe: current,
                    finalized: initial.finalized,
                });
            }

            let parent_id = current.block_info.parent_hash.into();
            let parent = read_l2(source, parent_id)
                .await?
                .ok_or(SyncStartError::BlockNotFound(parent_id))?;
            if parent.block_info.number.checked_add(1) != Some(current.block_info.number) {
                return Err(SyncStartError::InvalidL2Parent);
            }
            if parent.l1_origin != origin {
                if parent.l1_origin.number.checked_add(1) != Some(origin.number) {
                    return Err(SyncStartError::L1OriginMismatch);
                }
                if current.seq_num != 0 {
                    return Err(SyncStartError::NonZeroSequenceNumber);
                }
                if l1.is_some_and(|block| {
                    block.hash == origin.hash && block.parent_hash != parent.l1_origin.hash
                }) {
                    return Err(SyncStartError::L1OriginMismatch);
                }
            } else if parent.seq_num.checked_add(1) != Some(current.seq_num) {
                return Err(SyncStartError::InconsistentSequenceNumber);
            }
            current = parent;
        }
    }
    .await;

    // Reorg-sensitive failures (including genesis/finality mismatches) must not become fatal
    // merely because an old and a new L1 view were observed during this attempt.
    if let Some(anchor) = anchor {
        let confirmed = read_l1(source, anchor.number.into()).await?;
        if confirmed.is_none_or(|block| block.hash != anchor.hash) {
            return Err(SyncStartError::L1ChainChanged);
        }
    }
    result
}

async fn read_l1(
    source: &impl SyncSource,
    id: BlockId,
) -> Result<Option<BlockInfo>, SyncStartError> {
    let block = source.l1_block(id).await?;
    if let Some(block) = block &&
        !response_matches(id, block.hash, block.number)
    {
        return Err(SyncStartError::InvalidL1Response(id));
    }
    Ok(block)
}

async fn read_l2(
    source: &impl SyncSource,
    id: BlockId,
) -> Result<Option<L2BlockInfo>, SyncStartError> {
    let block = source.l2_block(id).await?;
    if let Some(block) = block &&
        !response_matches(id, block.block_info.hash, block.block_info.number)
    {
        return Err(SyncStartError::InvalidL2Response(id));
    }
    Ok(block)
}

fn response_matches(id: BlockId, hash: B256, number: u64) -> bool {
    match id {
        BlockId::Hash(expected) => expected.block_hash == hash,
        BlockId::Number(BlockNumberOrTag::Number(expected)) => expected == number,
        BlockId::Number(_) => true,
    }
}

#[cfg(test)]
mod tests;

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
