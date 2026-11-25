//! A task making derivation replace the local-safe blocks a cross-chain verifier denied.

use crate::{
    DenyTaskError, EngineClient, EngineState, ExecutableEngineTask, state::EngineSyncStateUpdate,
    task_queue::synchronize,
};
use alloy_eips::{BlockId, BlockNumHash};
use alloy_primitives::B256;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;

/// The [`DenyTask`] makes derivation replace the local-safe blocks a cross-chain verifier denied.
///
/// Derivation replaces a denied block with the deposits-only block wherever derived attributes
/// produce it, see [`crate::ConsolidateTask`]. A denied block at or below the local-safe head was
/// derived before the verifier denied it, so this task moves the local-safe head back to its
/// parent and fails with [`DenyTaskError::Rewound`]: derivation restarts from that parent and
/// replaces the block as it derives it again. Only local-safe moves; the execution layer keeps its
/// head until the replacement reorgs it.
///
/// A denied block at or below the finalized head is never replaced.
#[derive(Debug, Clone, Constructor)]
pub struct DenyTask {
    /// The engine client.
    pub client: Arc<EngineClient>,
    /// The rollup config.
    pub cfg: Arc<RollupConfig>,
    /// The blocks the verifier denied.
    pub denied: Vec<BlockNumHash>,
}

impl DenyTask {
    /// The lowest denied block that is the engine's canonical block at or below the local-safe
    /// head, with its parent's hash.
    async fn lowest_local_safe_denied(
        &self,
        state: &EngineState,
    ) -> Result<Option<(BlockNumHash, B256)>, DenyTaskError> {
        let local_safe = state.sync_state.local_safe_head().block_info.number;
        let mut candidates: Vec<_> =
            self.denied.iter().filter(|denied| denied.number <= local_safe).copied().collect();
        candidates.sort_unstable_by_key(|denied| denied.number);
        for denied in candidates {
            let canonical = self.client.get_l2_block(BlockId::number(denied.number)).await?;
            if let Some(block) = canonical.filter(|block| block.header.hash == denied.hash) {
                return Ok(Some((denied, block.header.parent_hash)));
            }
        }
        Ok(None)
    }
}

#[async_trait]
impl ExecutableEngineTask for DenyTask {
    type Output = ();

    type Error = DenyTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), DenyTaskError> {
        let Some((denied, parent_hash)) = self.lowest_local_safe_denied(state).await? else {
            return Ok(());
        };
        let finalized = state.sync_state.finalized_head().block_info.id();
        if denied.number <= finalized.number {
            error!(target: "engine", ?denied, ?finalized, "Refusing to replace a denied block at or below the finalized head");
            return Err(DenyTaskError::Finalized { denied, finalized });
        }

        let parent = self
            .client
            .get_l2_block(BlockId::hash(parent_hash))
            .full()
            .await?
            .ok_or(DenyTaskError::ParentNotFound(parent_hash))?;
        let parent =
            L2BlockInfo::from_block_and_genesis(&parent.into_consensus(), &self.cfg.genesis)?;

        synchronize(
            self.client.as_ref(),
            state,
            EngineSyncStateUpdate { local_safe_head: Some(parent), ..Default::default() },
        )
        .await?;
        warn!(target: "engine", ?denied, local_safe = ?parent.block_info.id(), "Moved the local-safe head below a denied block to replace it");
        Err(DenyTaskError::Rewound { denied, local_safe: parent.block_info.id() })
    }
}
