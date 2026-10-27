//! Sync start algorithm for the OP Stack rollup node.

use alloy_eips::BlockId;
use alloy_primitives::B256;
use alloy_provider::ext::EngineApi;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};

mod forkchoice;
pub use forkchoice::L2ForkchoiceState;

mod error;
pub use error::SyncStartError;

use tracing::info;

use crate::EngineClient;

/// Searches for a sync starting point using op-node's epoch and sequence consistency rules.
/// Unsafe origins may be ahead of L1; conservative safe recovery stops at an epoch boundary
/// beyond the sync lookback, or at finality.
pub async fn find_starting_forkchoice(
    cfg: &RollupConfig,
    engine_client: &EngineClient,
) -> Result<L2ForkchoiceState, SyncStartError> {
    find_starting_forkchoice_with_checkpoints(cfg, engine_client, &[]).await
}

/// A completed safe block and an L1 block bounding all of its inherited derivation inputs.
#[derive(Debug, Clone, Copy)]
pub(crate) struct SafeCheckpoint {
    pub safe: L2BlockInfo,
    pub source: BlockInfo,
    /// The retained parent's input, which replay may encounter on a different L1 branch.
    pub replay_source: Option<BlockInfo>,
}

pub(crate) async fn find_starting_forkchoice_with_checkpoints(
    cfg: &RollupConfig,
    engine_client: &EngineClient,
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

    let previous_unsafe = current_fc.un_safe;
    let previous_safe = current_fc.safe;
    let mut cursor = previous_unsafe;
    let mut unsafe_candidate = Some(cursor);
    let mut highest_canonical: Option<L2BlockInfo> = None;
    let mut walker = SyncWalker::new(cfg, engine_client);
    let mut check_checkpoint = !checkpoints.is_empty();
    let lookback = sync_lookback(cfg);

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
        if unsafe_candidate.is_none() {
            unsafe_candidate = Some(cursor);
            if cursor.l1_origin.number.saturating_add(5u64.saturating_mul(lookback)) <
                previous_unsafe.l1_origin.number
            {
                return Err(SyncStartError::ReorgTooDeep);
            }
        }
        match origin {
            OriginStatus::Ahead => highest_canonical = None,
            OriginStatus::Canonical => {
                highest_canonical.get_or_insert(cursor);
            }
            OriginStatus::Replaced => {
                unsafe_candidate = None;
                highest_canonical = None;
                check_checkpoint = !checkpoints.is_empty();
            }
        }
        if cursor.block_info.number == current_fc.finalized.block_info.number {
            current_fc.un_safe = unsafe_candidate.ok_or(SyncStartError::L1OriginMismatch)?;
            current_fc.safe = cursor;
            break;
        }
        if check_checkpoint && origin != OriginStatus::Replaced {
            check_checkpoint = false;
            current_fc.un_safe = unsafe_candidate.ok_or(SyncStartError::L1OriginMismatch)?;
            if let Some(safe) =
                retained_safe_checkpoint(engine_client, &current_fc, checkpoints).await?
            {
                info!(target: "sync_start", safe = %safe, "Retaining completed safe checkpoint with canonical L1 input");
                current_fc.safe = safe;
                return Ok(current_fc);
            }
            // Checkpoint validation may have observed a newer chain view. Restart this origin
            // by number before doing the conservative walk in that view.
            walker.l1 = None;
            highest_canonical = None;
            continue;
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
            current_fc.un_safe = unsafe_candidate.ok_or(SyncStartError::L1OriginMismatch)?;
            current_fc.safe = cursor;
            break;
        }
    }

    // Leave the finalized block as-is, and return the current forkchoice.
    Ok(current_fc)
}

/// Matches op-node Config.SyncLookback, including the longer Alt-DA challenge/resolve window.
pub(crate) fn sync_lookback(cfg: &RollupConfig) -> u64 {
    let alt_da = cfg.alt_da_config.as_ref();
    alt_da.map_or(cfg.seq_window_size, |da| {
        cfg.seq_window_size.max(
            da.da_challenge_window
                .unwrap_or_default()
                .saturating_add(da.da_resolve_window.unwrap_or_default()),
        )
    })
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

/// A canonical origin alone does not prove safety: the batch may have been submitted later.
/// Retain only completed blocks whose actual L1 input and L2 ancestry are still canonical.
async fn retained_safe_checkpoint(
    client: &EngineClient,
    heads: &L2ForkchoiceState,
    checkpoints: &[SafeCheckpoint],
) -> Result<Option<L2BlockInfo>, SyncStartError> {
    if checkpoints.is_empty() {
        return Ok(None);
    }
    let anchor =
        client.get_l1_block(BlockId::latest()).await?.ok_or(SyncStartError::ChainViewChanged)?;
    // The L1 chain may have changed since the unsafe walk, before this anchor was read.
    let mut walker = SyncWalker::new(client.cfg(), client);
    if heads.un_safe != heads.finalized {
        let status = walker.origin(heads.un_safe).await?;
        if status == OriginStatus::Replaced ||
            (status == OriginStatus::Ahead &&
                heads.un_safe.l1_origin.number <= anchor.header.number)
        {
            return Err(SyncStartError::ChainViewChanged);
        }
    }
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
        if let Some(parent_input) =
            checkpoint.replay_source.filter(|input| *input != checkpoint.source)
        {
            let parent = client.get_l1_block(parent_input.number.into()).await?;
            if parent.is_none_or(|block| block.header.hash != parent_input.hash) {
                continue;
            }
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
        // Numeric ancestry alone does not establish epoch/sequence consistency. Apply the
        // same edge checks as the conservative walk before accepting the retained checkpoint.
        let mut cursor = heads.un_safe;
        let mut ancestry = SyncWalker { cfg: walker.cfg, client, l1: walker.l1 };
        while cursor.block_info.number > checkpoint.safe.block_info.number {
            cursor = ancestry.parent(cursor).await?;
            if cursor.block_info.number > checkpoint.safe.block_info.number &&
                ancestry.origin(cursor).await? == OriginStatus::Replaced
            {
                return Ok(None);
            }
        }
        if cursor != checkpoint.safe {
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
    use crate::test_utils::{test_block_info, test_engine_client};
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadStatus, PayloadStatusEnum};
    use serde_json::json;
    use std::sync::Arc;

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
        CanonicalReplay,
        ReorgedReplay,
        ReorgedUnsafe,
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
    #[case(Scenario::CanonicalReplay)]
    #[case(Scenario::ReorgedReplay)]
    #[case(Scenario::ReorgedUnsafe)]
    #[tokio::test]
    async fn checkpoint_requires_canonical_batch_input_and_l2_ancestry(#[case] scenario: Scenario) {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let safe = test_block_info(117029);
        let heads = L2ForkchoiceState { safe, un_safe: safe, finalized: test_block_info(44302) };
        let mut source = alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default();
        source.header.inner.number = 10679;
        source.header.hash = B256::repeat_byte(1);
        let mut checkpoint = SafeCheckpoint {
            safe,
            source: BlockInfo { number: 10679, hash: source.header.hash, ..Default::default() },
            replay_source: None,
        };
        if matches!(scenario, Scenario::CanonicalReplay | Scenario::ReorgedReplay) {
            checkpoint.replay_source =
                Some(BlockInfo { number: 10678, hash: B256::repeat_byte(8), ..Default::default() });
        }
        let mut anchor = source.clone();
        anchor.header.inner.number = 10690;
        anchor.header.hash = B256::repeat_byte(2);
        let mut safe_block =
            alloy_rpc_types_eth::Block::<op_alloy_rpc_types::Transaction>::default();
        safe_block.header.hash = safe.hash();
        let mut unsafe_block = safe_block.clone();
        unsafe_block.header.hash = heads.un_safe.hash();
        let mut checkpoints = vec![checkpoint];
        if matches!(scenario, Scenario::NewerInvalid) {
            checkpoints.push(SafeCheckpoint {
                safe,
                source: BlockInfo { number: 10680, ..checkpoint.source },
                replay_source: None,
            });
        }
        let attempts = if matches!(scenario, Scenario::SyncingAfterValid) { 2 } else { 1 };
        for attempt in 0..attempts {
            l1.expect_params(
                "eth_getBlockByNumber",
                json!(["latest", false]),
                (!matches!(scenario, Scenario::MissingHead)).then_some(&anchor),
            );
            if matches!(scenario, Scenario::MissingHead) {
                break;
            }
            let mut origin = source.clone();
            origin.header.hash = if matches!(scenario, Scenario::ReorgedUnsafe) {
                B256::repeat_byte(9)
            } else {
                heads.un_safe.l1_origin.hash
            };
            l1.expect_params("eth_getBlockByNumber", json!(["0x0", false]), origin);
            if matches!(scenario, Scenario::ReorgedUnsafe) {
                break;
            }
            if matches!(scenario, Scenario::NewerInvalid) {
                l1.expect_params(
                    "eth_getBlockByNumber",
                    json!(["0x29b8", false]),
                    Option::<alloy_rpc_types_eth::Block>::None,
                );
            }
            if matches!(scenario, Scenario::InvalidInput) {
                source.header.hash = B256::ZERO;
            }
            l1.expect_params("eth_getBlockByNumber", json!(["0x29b7", false]), &source);
            if matches!(scenario, Scenario::InvalidInput) {
                break;
            }
            if let Some(parent_input) = checkpoint.replay_source {
                let mut parent = source.clone();
                parent.header.hash = if matches!(scenario, Scenario::ReorgedReplay) {
                    B256::ZERO
                } else {
                    parent_input.hash
                };
                l1.expect_params("eth_getBlockByNumber", json!(["0x29b6", false]), parent);
                if matches!(scenario, Scenario::ReorgedReplay) {
                    break;
                }
            }
            if matches!(scenario, Scenario::InvalidL2) {
                safe_block.header.hash = B256::ZERO;
            }
            l2.expect_params(
                "eth_getBlockByNumber",
                json!([format!("{:#x}", safe.block_info.number), false]),
                &safe_block,
            );
            if matches!(scenario, Scenario::InvalidL2) {
                break;
            }
            if matches!(scenario, Scenario::ChangedL2) {
                unsafe_block.header.hash = B256::ZERO;
            }
            l2.expect_params(
                "eth_getBlockByNumber",
                json!([format!("{:#x}", heads.un_safe.block_info.number), false]),
                &unsafe_block,
            );
            if matches!(scenario, Scenario::ChangedL2) {
                break;
            }
            let status =
                if attempt == 1 { PayloadStatusEnum::Syncing } else { PayloadStatusEnum::Valid };
            l2.expect_params("engine_forkchoiceUpdatedV3", json!([{
                "headBlockHash": heads.un_safe.hash(), "safeBlockHash": safe.hash(), "finalizedBlockHash": heads.finalized.hash()
            }, null]), ForkchoiceUpdated::new(PayloadStatus::from_status(status)));
            if attempt == 1 {
                break;
            }
            let mut recheck = anchor.clone();
            if matches!(scenario, Scenario::ChangedL1) {
                recheck.header.hash = B256::ZERO;
            }
            l1.expect_params("eth_getBlockByNumber", json!(["0x29c2", false]), recheck);
        }
        for attempt in 0..attempts {
            let selected = retained_safe_checkpoint(&client, &heads, &checkpoints).await;
            match scenario {
                Scenario::SyncingAfterValid if attempt == 1 => {
                    assert!(matches!(selected, Err(SyncStartError::CheckpointNotValidated)))
                }
                Scenario::ChangedL1 |
                Scenario::ChangedL2 |
                Scenario::MissingHead |
                Scenario::ReorgedUnsafe => {
                    assert!(matches!(selected, Err(SyncStartError::ChainViewChanged)))
                }
                Scenario::InvalidInput | Scenario::InvalidL2 | Scenario::ReorgedReplay => {
                    assert_eq!(selected.unwrap(), None)
                }
                _ => assert_eq!(selected.unwrap(), Some(safe)),
            }
        }
        l1.assert_finished();
        l2.assert_finished();
    }

    #[tokio::test]
    async fn startup_without_accepted_checkpoints_uses_conservative_walkback() {
        let (client, l1, l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let block = test_block_info(1);
        let heads = L2ForkchoiceState { safe: block, un_safe: block, finalized: block };
        assert_eq!(retained_safe_checkpoint(&client, &heads, &[]).await.unwrap(), None);
        l1.assert_finished();
        l2.assert_finished();
    }
}

#[cfg(test)]
mod tests;
