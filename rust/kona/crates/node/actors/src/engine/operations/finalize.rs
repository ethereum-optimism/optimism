//! Finalizing an L2 block.

use super::Engine;
use crate::engine::{FinalizeBlockId, FinalizeTaskError};
use alloy_eips::BlockId;
use kona_engine::EngineSyncStateUpdate;
use kona_protocol::L2BlockInfo;

impl Engine {
    /// Fetches the [`L2BlockInfo`] identified by `block_id`, makes it the finalized head, and
    /// dispatches a forkchoice update to finalize the block.
    pub(in crate::engine) async fn finalize_block(
        &mut self,
        block_id: FinalizeBlockId,
    ) -> Result<(), FinalizeTaskError> {
        let block_number = block_id.number();

        // Sanity check that the block that is being finalized is at least safe.
        if self.state.sync_state.safe_head().block_info.number < block_number {
            return Err(FinalizeTaskError::BlockNotSafe);
        }

        // Look up by hash when the caller pinned a specific hash (delegated polling supplies
        // `(number, hash)`); otherwise look up by number. Finalization is irreversible, so a stale
        // hash must produce a hard error, not silently finalize whatever the engine has at the
        // same height.
        let lookup: BlockId = match block_id {
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

        // For ByHash, also assert the returned block's height matches the caller's claim. The
        // engine should never serve a block at a different height for a given hash, but a height
        // mismatch indicates either an upstream protocol bug or a misuse of the API and must
        // never lead to silent finalization.
        if let FinalizeBlockId::ByHash(id) = block_id &&
            block_info.block_info.number != id.number
        {
            return Err(FinalizeTaskError::BlockNotFound(id.number));
        }

        // Dispatch a forkchoice update.
        self.synchronize(EngineSyncStateUpdate {
            finalized_head: Some(block_info),
            ..Default::default()
        })
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

#[cfg(test)]
mod tests;
