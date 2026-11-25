//! A task moving the safe label to a cross-chain verifier's head.

use crate::{
    EngineClient, EngineState, ExecutableEngineTask, PromoteTaskError, VerifierHead,
    state::EngineSyncStateUpdate, task_queue::synchronize,
};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;

/// The [`PromoteTask`] moves the safe label to the block a cross-chain verifier reports, as
/// op-node's `SafeL2Head` does with a super authority:
///
/// - [`VerifierHead::PreActivation`]: the safe label is the local-safe head.
/// - [`VerifierHead::Anchor`]: the canonical block at the anchor timestamp, or the local-safe head
///   if derivation has not reached it.
/// - [`VerifierHead::Verified`]: the verified block, if it is the engine's canonical block at its
///   height and not ahead of the local-safe head. Otherwise the label holds where it is; it never
///   falls back to local-safe.
///
/// The label then stays between the finalized and the local-safe head, see
/// [`crate::SafeLabel::Verified`].
#[derive(Debug, Clone, Constructor)]
pub struct PromoteTask {
    /// The engine client.
    pub client: Arc<EngineClient>,
    /// The rollup config.
    pub cfg: Arc<RollupConfig>,
    /// The verifier's head for this chain.
    pub head: VerifierHead,
}

impl PromoteTask {
    /// The block the safe label moves to, or `None` to hold it.
    async fn target(&self, state: &EngineState) -> Result<Option<L2BlockInfo>, PromoteTaskError> {
        let local_safe = state.sync_state.local_safe_head();
        match self.head {
            VerifierHead::PreActivation => Ok(Some(local_safe)),
            VerifierHead::Anchor { timestamp } => {
                let number = self.cfg.block_number_from_timestamp(timestamp);
                if number >= local_safe.block_info.number {
                    return Ok(Some(local_safe));
                }
                self.canonical_block(number).await.map(Some)
            }
            VerifierHead::Verified { block, .. } => {
                if block.number > local_safe.block_info.number {
                    debug!(target: "engine", verified = ?block, local_safe = %local_safe, "Verified head is ahead of the local-safe head; holding the safe label");
                    return Ok(None);
                }
                if block == local_safe.block_info.id() {
                    return Ok(Some(local_safe));
                }
                let canonical = self.canonical_block(block.number).await?;
                if canonical.block_info.hash != block.hash {
                    warn!(target: "engine", verified = ?block, canonical = %canonical, "Verified head is not canonical; holding the safe label");
                    return Ok(None);
                }
                Ok(Some(canonical))
            }
        }
    }

    /// The engine's canonical block at `number`, at or below the local-safe head.
    async fn canonical_block(&self, number: u64) -> Result<L2BlockInfo, PromoteTaskError> {
        let block = self
            .client
            .get_l2_block(number.into())
            .full()
            .await?
            .ok_or(PromoteTaskError::BlockNotFound(number))?;
        Ok(L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &self.cfg.genesis)?)
    }
}

#[async_trait]
impl ExecutableEngineTask for PromoteTask {
    type Output = ();

    type Error = PromoteTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), PromoteTaskError> {
        let Some(target) = self.target(state).await? else {
            return Ok(());
        };
        let safe_before = state.sync_state.safe_head();
        synchronize(
            self.client.as_ref(),
            state,
            EngineSyncStateUpdate { safe_head: Some(target), ..Default::default() },
        )
        .await?;
        let safe_after = state.sync_state.safe_head();
        if safe_after != safe_before {
            info!(
                target: "engine",
                head = ?self.head,
                number = safe_after.block_info.number,
                hash = %safe_after.block_info.hash,
                "Moved the safe label to the verifier's head"
            );
        }
        Ok(())
    }
}
