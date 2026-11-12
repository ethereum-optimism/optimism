//! A task for finalizing an L2 block.

use crate::{
    EngineClient, EngineState, ExecutableEngineTask, FinalizeBlockId, FinalizeTaskError,
    state::EngineSyncStateUpdate, task_queue::synchronize,
};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;

/// The [`FinalizeTask`] fetches the [`L2BlockInfo`] identified by `block_id`, updates the
/// [`EngineState`], and dispatches a forkchoice update to finalize the block.
///
/// Finality only moves forward. A target behind the finalized head, a different block at the
/// finalized height, or a pinned hash that is not the engine's canonical block fails with an error
/// that skips the task and leaves the engine state unchanged.
#[derive(Debug, Clone, Constructor)]
pub struct FinalizeTask {
    /// The engine client.
    pub client: Arc<EngineClient>,
    /// The rollup config.
    pub cfg: Arc<RollupConfig>,
    /// Identifier of the L2 block to finalize.
    pub block_id: FinalizeBlockId,
}

impl FinalizeTask {
    /// Rejects a target that would move finality backward.
    const fn ensure_not_behind_finalized_head(
        finalized_head: &L2BlockInfo,
        target_number: u64,
    ) -> Result<(), FinalizeTaskError> {
        if target_number < finalized_head.block_info.number {
            return Err(FinalizeTaskError::BehindFinalizedHead {
                finalized: finalized_head.block_info.number,
                target: target_number,
            });
        }
        Ok(())
    }

    /// Rejects a different block at the finalized head's height.
    ///
    /// The default head means finality has not been established yet, so any target, including the
    /// configured genesis, may initialize it.
    fn ensure_no_conflict_with_finalized_head(
        finalized_head: &L2BlockInfo,
        target: BlockNumHash,
    ) -> Result<(), FinalizeTaskError> {
        let finalized = finalized_head.block_info.id();
        if *finalized_head != L2BlockInfo::default() &&
            target.number == finalized.number &&
            target.hash != finalized.hash
        {
            return Err(FinalizeTaskError::ConflictsWithFinalizedHead { finalized, target });
        }
        Ok(())
    }

    /// Whether `target` is the finalized head itself; a different block at its height is an error.
    fn is_already_finalized(
        finalized_head: &L2BlockInfo,
        target: BlockNumHash,
    ) -> Result<bool, FinalizeTaskError> {
        if target == finalized_head.block_info.id() {
            debug!(target: "engine", ?target, "Target is already finalized");
            return Ok(true);
        }
        Self::ensure_no_conflict_with_finalized_head(finalized_head, target)?;
        Ok(false)
    }

    /// Fetches the engine's canonical block at the target's height.
    ///
    /// A pinned hash must be that block: a forkchoice update cannot finalize a block off the
    /// canonical chain, and finalizing the canonical block instead would finalize a block the
    /// caller never named. The block's identity is checked before it is decoded, so a mismatch is
    /// reported as such rather than as a decoding failure.
    async fn fetch_canonical_block_info(&self) -> Result<L2BlockInfo, FinalizeTaskError> {
        let requested = self.block_id.number();
        debug!(target: "engine", block_id = ?self.block_id, "Fetching the block to finalize");
        let block = self
            .client
            .get_l2_block(requested.into())
            .full()
            .await
            .map_err(FinalizeTaskError::TransportError)?
            .ok_or(FinalizeTaskError::BlockNotFound(requested))?;
        let canonical = BlockNumHash { hash: block.header.hash, number: block.header.number };
        debug!(target: "engine", ?canonical, "Fetched the block to finalize");

        if canonical.number != requested {
            return Err(FinalizeTaskError::BlockNumberMismatch {
                requested,
                returned: canonical.number,
            });
        }
        if let FinalizeBlockId::ByHash(target) = self.block_id &&
            target.hash != canonical.hash
        {
            return Err(FinalizeTaskError::NotCanonical { canonical, target });
        }

        L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &self.cfg.genesis)
            .map_err(FinalizeTaskError::FromBlock)
    }
}

#[async_trait]
impl ExecutableEngineTask for FinalizeTask {
    type Output = ();

    type Error = FinalizeTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), FinalizeTaskError> {
        let finalized_head = state.sync_state.finalized_head();
        let target_number = self.block_id.number();

        // A lagging source can report an older finalized block, or one from another chain. A
        // pinned hash is enough to detect either without asking the execution layer.
        Self::ensure_not_behind_finalized_head(&finalized_head, target_number)?;
        if let FinalizeBlockId::ByHash(target) = self.block_id &&
            Self::is_already_finalized(&finalized_head, target)?
        {
            return Ok(());
        }

        // Sanity check that the block that is being finalized is at least safe.
        if state.sync_state.safe_head().block_info.number < target_number {
            return Err(FinalizeTaskError::BlockNotSafe);
        }

        let block_info = self.fetch_canonical_block_info().await?;

        // A by-number target only learns its hash from the lookup.
        if matches!(self.block_id, FinalizeBlockId::ByNumber(_)) &&
            Self::is_already_finalized(&finalized_head, block_info.block_info.id())?
        {
            return Ok(());
        }

        // Dispatch a forkchoice update.
        synchronize(
            self.client.as_ref(),
            state,
            EngineSyncStateUpdate { finalized_head: Some(block_info), ..Default::default() },
        )
        .await?;

        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            "Updated finalized head"
        );

        Ok(())
    }
}
