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
/// finalized height, a pinned hash that is not the engine's canonical block, a target above the
/// safe head, or a pinned target the engine has no block for yet fails with an error that skips
/// the task and leaves the engine state unchanged.
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
    /// A pinned target the execution layer answers with null is not available yet, for example
    /// while it catches up: the task is skipped rather than held at the head of the queue ahead
    /// of the imports that let it catch up. A missing by-number target is below the safe head, so
    /// the engine must already have it.
    fn block_missing(&self) -> FinalizeTaskError {
        match self.block_id {
            FinalizeBlockId::ByHash(target) => {
                FinalizeTaskError::NotYetAvailable { target }
            }
            FinalizeBlockId::ByNumber(number) => FinalizeTaskError::BlockNotFound(number),
        }
    }

    /// Whether `target` is the finalized head itself; a different block at its height is an error.
    fn is_already_finalized(
        finalized_head: &L2BlockInfo,
        target: BlockNumHash,
    ) -> Result<bool, FinalizeTaskError> {
        let finalized = finalized_head.block_info.id();
        if target == finalized {
            debug!(target: "engine", ?target, "Target is already finalized");
            return Ok(true);
        }
        // The default head means finality has not been established yet, so any target, including
        // the configured genesis, may initialize it.
        if *finalized_head != L2BlockInfo::default() && target.number == finalized.number {
            return Err(FinalizeTaskError::ConflictsWithFinalizedHead { finalized, target });
        }
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
            .ok_or_else(|| self.block_missing())?;
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
        if target_number < finalized_head.block_info.number {
            return Err(FinalizeTaskError::BehindFinalizedHead {
                finalized: finalized_head.block_info.number,
                target: target_number,
            });
        }
        if let FinalizeBlockId::ByHash(target) = self.block_id &&
            Self::is_already_finalized(&finalized_head, target)?
        {
            return Ok(());
        }

        // Sanity check that the block that is being finalized is at least safe. A later request
        // finalizes a target skipped here, or a newer block.
        let safe_number = state.sync_state.safe_head().block_info.number;
        if safe_number < target_number {
            return Err(FinalizeTaskError::BlockNotSafe { safe: safe_number, target: target_number });
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
