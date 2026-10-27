//! A task for finalizing an L2 block.

use crate::{
    EngineClient, EngineState, EngineTaskExt, FinalizeBlockId, FinalizeTaskError,
    state::EngineSyncStateUpdate, task_queue::synchronize,
};
use alloy_eips::BlockId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;

/// The [`FinalizeTask`] fetches the [`L2BlockInfo`] identified by `block_id`, updates the
/// [`EngineState`], and dispatches a forkchoice update to finalize the block.
/// Missing hash-pinned blocks are deferred to the next delegated derivation poll, allowing
/// unsafe imports and other engine tasks to continue while the execution client catches up.
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

        // Sanity check that the block that is being finalized is at least safe.
        if state.sync_state.safe_head().block_info.number < block_number {
            return Err(FinalizeTaskError::BlockNotSafe);
        }

        // Look up by hash when the caller pinned a specific hash (delegated polling supplies
        // `(number, hash)`); otherwise look up by number. Finalization is irreversible, so never
        // substitute the engine's block at the same height if the pinned hash is unavailable.
        let lookup: BlockId = match self.block_id {
            FinalizeBlockId::ByHash(id) => id.hash.into(),
            FinalizeBlockId::ByNumber(n) => n.into(),
        };
        let block = self
            .client
            .get_l2_block(lookup)
            .full()
            .await
            .map_err(FinalizeTaskError::TransportError)?;
        let Some(block) = block else {
            if let FinalizeBlockId::ByHash(id) = self.block_id {
                // Delegated derivation resubmits the target on every poll. Retaining this task
                // for an engine retry would prevent newer imports from making it available.
                debug!(target: "engine", ?id, "Deferring finalization until the block is available");
                return Ok(());
            }
            return Err(FinalizeTaskError::BlockNotFound(block_number));
        };
        let block = block.into_consensus();
        let block_info = L2BlockInfo::from_block_and_genesis(&block, &self.client.cfg().genesis)
            .map_err(FinalizeTaskError::FromBlock)?;

        // An available block must match the complete pinned identity. A malformed response is
        // different from a block that has not downloaded yet and must never be finalized.
        if let FinalizeBlockId::ByHash(id) = self.block_id &&
            block_info.block_info.id() != id
        {
            return Err(FinalizeTaskError::BlockIdentityMismatch {
                expected: id,
                actual: block_info.block_info.id(),
            });
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
