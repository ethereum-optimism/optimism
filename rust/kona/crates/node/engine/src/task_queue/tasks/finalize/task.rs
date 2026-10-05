//! A task for finalizing an L2 block.

use crate::{
    EngineClient, EngineState, EngineTaskExt, FinalizeBlockId, FinalizeTaskError, SynchronizeTask,
    state::EngineSyncStateUpdate,
};
use alloy_eips::BlockId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::{sync::Arc, time::Instant};

/// The [`FinalizeTask`] fetches the [`L2BlockInfo`] identified by `block_id`, updates the
/// [`EngineState`], and dispatches a forkchoice update to finalize the block.
#[derive(Debug, Clone, Constructor)]
pub struct FinalizeTask<EngineClient_: EngineClient> {
    /// The engine client.
    pub client: Arc<EngineClient_>,
    /// The rollup config.
    pub cfg: Arc<RollupConfig>,
    /// Identifier of the L2 block to finalize.
    pub block_id: FinalizeBlockId,
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for FinalizeTask<EngineClient_> {
    type Output = ();

    type Error = FinalizeTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), FinalizeTaskError> {
        let block_number = self.block_id.number();

        // Sanity check that the block that is being finalized is at least safe.
        if state.sync_state.safe_head().block_info.number < block_number {
            return Err(FinalizeTaskError::BlockNotSafe);
        }

        // Look up by hash when the caller pinned a specific hash (delegated polling supplies
        // `(number, hash)`); otherwise look up by number. Finalization is irreversible, so a stale
        // hash must produce a hard error, not silently finalize whatever the engine has at the
        // same height.
        let lookup: BlockId = match self.block_id {
            FinalizeBlockId::ByHash(id) => id.hash.into(),
            FinalizeBlockId::ByNumber(n) => n.into(),
        };

        let block_fetch_start = Instant::now();
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

        // For ByHash, also assert the returned block's height matches the caller's claim. The
        // engine should never serve a block at a different height for a given hash, but a height
        // mismatch indicates either an upstream protocol bug or a misuse of the API and must
        // never lead to silent finalization.
        if let FinalizeBlockId::ByHash(id) = self.block_id &&
            block_info.block_info.number != id.number
        {
            return Err(FinalizeTaskError::BlockNotFound(id.number));
        }

        let block_fetch_duration = block_fetch_start.elapsed();

        // Dispatch a forkchoice update.
        let fcu_start = Instant::now();
        SynchronizeTask::new(
            self.client.clone(),
            self.cfg.clone(),
            EngineSyncStateUpdate { finalized_head: Some(block_info), ..Default::default() },
        )
        .execute(state)
        .await?;
        let fcu_duration = fcu_start.elapsed();

        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            ?block_fetch_duration,
            ?fcu_duration,
            "Updated finalized head"
        );

        Ok(())
    }
}
