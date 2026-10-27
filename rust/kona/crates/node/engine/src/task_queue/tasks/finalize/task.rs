//! A task for finalizing an L2 block.

use crate::{
    EngineClient, EngineState, EngineTaskExt, FinalizeBlockId, FinalizeTaskError,
    state::EngineSyncStateUpdate, task_queue::synchronize,
};
use alloy_eips::{BlockId, BlockNumHash};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;

/// The [`FinalizeTask`] fetches the [`L2BlockInfo`] identified by `block_id`, updates the
/// [`EngineState`], and dispatches a forkchoice update to finalize the block.
#[derive(Debug, Clone, Constructor)]
pub struct FinalizeTask {
    /// The engine client.
    pub client: Arc<EngineClient>,
    /// The rollup config.
    pub cfg: Arc<RollupConfig>,
    /// Identifier of the L2 block to finalize.
    pub block_id: FinalizeBlockId,
}

#[async_trait]
impl EngineTaskExt for FinalizeTask {
    type Output = ();

    type Error = FinalizeTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), FinalizeTaskError> {
        let block_number = self.block_id.number();
        let finalized_head = state.sync_state.finalized_head();

        // A delegate can temporarily lag behind our finalized head. Ignore stale targets without
        // asking the execution layer to move its irreversible finalized label backwards.
        if block_number < finalized_head.block_info.number {
            debug!(target: "engine", block_number, ?finalized_head, "Ignoring stale finalized target");
            return Ok(());
        }

        // Sanity check that the block that is being finalized is at least safe.
        if state.sync_state.safe_head().block_info.number < block_number {
            return Err(FinalizeTaskError::BlockNotSafe);
        }

        // A pinned conflicting hash is already enough evidence to reject the request, even when
        // the execution layer has not downloaded that block yet.
        if let FinalizeBlockId::ByHash(id) = self.block_id {
            Self::check_finalized_hash(finalized_head, id)?;
        }

        // Look up by hash when the caller pinned a specific hash (delegated polling supplies
        // `(number, hash)`); otherwise look up by number. Finalization is irreversible, so a stale
        // hash must produce a hard error, not silently finalize whatever the engine has at the
        // same height.
        let lookup: BlockId = match self.block_id {
            FinalizeBlockId::ByHash(id) => id.hash.into(),
            FinalizeBlockId::ByNumber(n) => n.into(),
        };
        let block = self
            .client
            .get_l2_block(lookup)
            .full()
            .await
            .map_err(FinalizeTaskError::TransportError)?
            .ok_or(FinalizeTaskError::BlockNotFound(block_number))?
            .into_consensus();
        let block_info = L2BlockInfo::from_block_and_genesis(&block, &self.client.cfg().genesis)
            .map_err(FinalizeTaskError::FromBlock)?;

        // Validate the resolved height for either identifier form. Otherwise a malformed
        // by-number response could bypass the stale-target guard and regress finality.
        if block_info.block_info.number != block_number {
            return Err(FinalizeTaskError::BlockNotFound(block_number));
        }

        // Identical targets still pass through synchronize so pending EL work can be retried.
        Self::check_finalized_hash(finalized_head, block_info.block_info.id())?;

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

impl FinalizeTask {
    /// Reject a different hash at an established finalized height. The default head represents an
    /// uninitialized label, so it must allow initialization to the configured genesis hash.
    fn check_finalized_hash(
        finalized: L2BlockInfo,
        target: BlockNumHash,
    ) -> Result<(), FinalizeTaskError> {
        if finalized != L2BlockInfo::default() &&
            target.number == finalized.block_info.number &&
            target.hash != finalized.block_info.hash
        {
            return Err(FinalizeTaskError::ConflictingFinalizedBlock {
                number: target.number,
                current_hash: finalized.block_info.hash,
                requested_hash: target.hash,
            });
        }
        Ok(())
    }
}
