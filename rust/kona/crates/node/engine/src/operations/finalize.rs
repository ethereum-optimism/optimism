//! Hash-pinned finalization with explicit provenance.

use crate::{
    EngineClient, EngineState, EngineSyncStateUpdate, FinalizeError, ForkchoiceStatus, synchronize,
};
use alloy_eips::{BlockId, BlockNumHash};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;

/// Why a particular L2 block may be finalized.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FinalizationSource {
    /// Locally derived data, pinned to its canonical L1 inclusion source.
    Local(BlockNumHash),
    /// An explicitly trusted upstream finalized-block claim.
    Delegated,
}

/// Finalizes the recorded hash only after checking current canonical L1/L2 data.
/// The caller validates the reset epoch before entering this operation.
/// Only a VALID forkchoice publishes the proposed finalized head. SYNCING requires recovery.
pub async fn finalize(
    client: &impl EngineClient,
    cfg: &RollupConfig,
    state: &mut EngineState,
    block: BlockNumHash,
    source: FinalizationSource,
) -> Result<(), FinalizeError> {
    let finalized = state.sync_state.finalized_head().block_info;
    if block.number < finalized.number {
        return Ok(());
    }
    if block.number == finalized.number {
        if block.hash != finalized.hash {
            return Err(FinalizeError::L2Mismatch(block));
        }
        return Ok(());
    }
    if block.number > state.sync_state.safe_head().block_info.number {
        return Err(FinalizeError::NotSafe);
    }
    let id: BlockId = block.number.into();
    let canonical =
        client.get_l2_block(id).full().await?.ok_or(FinalizeError::BlockNotFound(id))?;
    if canonical.header.hash != block.hash || canonical.header.number != block.number {
        return Err(FinalizeError::L2Mismatch(block));
    }
    let info = L2BlockInfo::from_block_and_genesis(&canonical.into_consensus(), &cfg.genesis)?;
    if info.block_info.id() != block {
        return Err(FinalizeError::L2Mismatch(block));
    }
    if let FinalizationSource::Local(source) = source {
        let id: BlockId = source.number.into();
        let canonical = client.get_l1_block(id).await?.ok_or(FinalizeError::BlockNotFound(id))?;
        if canonical.header.hash != source.hash || canonical.header.number != source.number {
            return Err(FinalizeError::L1Mismatch(source));
        }
    }
    let mut proposed = *state;
    let accepted = synchronize(
        client,
        &mut proposed,
        EngineSyncStateUpdate { finalized_head: Some(info), ..Default::default() },
    )
    .await?;
    if accepted == ForkchoiceStatus::Syncing {
        return Err(FinalizeError::Syncing);
    }
    *state = proposed;
    Ok(())
}
