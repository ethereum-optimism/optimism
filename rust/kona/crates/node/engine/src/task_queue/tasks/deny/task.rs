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

/// The [`DenyTask`] makes derivation derive again the local-safe heights a cross-chain verifier's
/// denials decide: replace the blocks it denied, and bring back the blocks the node replaced on a
/// denial a reorg voided.
///
/// Derivation replaces a denied block with the deposits-only block wherever derived attributes
/// produce it, see [`crate::ConsolidateTask`]. A denied block at or below the local-safe head was
/// derived before the verifier denied it, so this task moves the local-safe head back to its
/// parent and fails with [`DenyTaskError::Rewound`]: derivation restarts from that parent and
/// replaces the block as it derives it again. Only local-safe moves; the execution layer keeps its
/// head until the replacement reorgs it.
///
/// A `reorged` denial was decided on an L1 block a reorg removed, so it does not hold. Where the
/// engine's block at its height is not the denied block, the node replaced the block on it, and
/// this task moves the local-safe head back below that height in the same way, so derivation
/// derives the height again without the denial. The lowest height either list asks for goes
/// first.
///
/// A height at or below the finalized head is never derived again.
#[derive(Debug, Clone, Constructor)]
pub struct DenyTask {
    /// The engine client.
    pub client: Arc<EngineClient>,
    /// The rollup config.
    pub cfg: Arc<RollupConfig>,
    /// The blocks the verifier denied.
    pub denied: Vec<BlockNumHash>,
    /// The blocks the verifier denied on L1 blocks a reorg removed.
    pub reorged: Vec<BlockNumHash>,
}

impl DenyTask {
    /// The engine's lowest block at or below the local-safe head that derivation must derive
    /// again, with its parent's hash and whether it is denied: a denied block that is the engine's
    /// canonical block, or the engine's block at the height of a reorged denial if that is
    /// another block.
    async fn lowest_to_derive_again(
        &self,
        state: &EngineState,
    ) -> Result<Option<(BlockNumHash, B256, bool)>, DenyTaskError> {
        let local_safe = state.sync_state.local_safe_head().block_info.number;
        let mut candidates: Vec<_> = self
            .denied
            .iter()
            .map(|block| (*block, true))
            .chain(self.reorged.iter().map(|block| (*block, false)))
            .filter(|(block, _)| block.number <= local_safe)
            .collect();
        candidates.sort_unstable_by_key(|(block, _)| block.number);
        for (block, denied) in candidates {
            let Some(canonical) = self.client.get_l2_block(BlockId::number(block.number)).await?
            else {
                continue;
            };
            let id = BlockNumHash { number: block.number, hash: canonical.header.hash };
            if (id == block) == denied {
                return Ok(Some((id, canonical.header.parent_hash, denied)));
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
        let Some((block, parent_hash, denied)) = self.lowest_to_derive_again(state).await? else {
            return Ok(());
        };
        let finalized = state.sync_state.finalized_head().block_info.id();
        if block.number <= finalized.number {
            error!(target: "engine", ?block, denied, ?finalized, "Refusing to derive again a height at or below the finalized head");
            return Err(DenyTaskError::Finalized { denied: block, finalized });
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
        let local_safe = parent.block_info.id();
        if denied {
            warn!(target: "engine", denied = ?block, ?local_safe, "Moved the local-safe head below a denied block to replace it");
            return Err(DenyTaskError::Rewound { denied: block, local_safe });
        }
        warn!(target: "engine", replacement = ?block, ?local_safe, "Moved the local-safe head below a block replaced on a denial a reorg voided, to derive it again");
        Err(DenyTaskError::Restored { replacement: block, local_safe })
    }
}
