//! A task moving the safe label to a cross-chain verifier's head.

use crate::{
    EngineClient, EngineState, ExecutableEngineTask, PromoteTaskError, VerifierHead,
    state::EngineSyncStateUpdate, task_queue::synchronize,
};
use alloy_eips::BlockNumHash;
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
///   height, not ahead of the local-safe head, and the L1 block the verification depended on is
///   canonical on the node's L1 chain. Otherwise the label holds where it is; it never falls back
///   to local-safe. An L1 block the node has not seen yet only means its L1 view lags the
///   verifier's for a moment, so it holds the label too.
///
/// The label then stays between the finalized and the local-safe head, and is the local-safe head
/// while that is before the verifier's activation, whatever the verifier answers, see
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
            VerifierHead::Verified { block, l1, .. } => {
                if block.number > local_safe.block_info.number {
                    debug!(target: "engine", verified = ?block, local_safe = %local_safe, "Verified head is ahead of the local-safe head; holding the safe label");
                    return Ok(None);
                }
                if !l1_block_is_canonical(&self.client, l1).await {
                    debug!(target: "engine", verified = ?block, ?l1, "The verified head's L1 block is not canonical on the node's L1 chain; holding the safe label");
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

/// Whether `l1` is the canonical L1 block at its height on the L1 chain `client` reads. An L1
/// block the node has not seen yet, or a failed lookup, is not. Genesis carries no L1 block.
pub async fn l1_block_is_canonical(client: &EngineClient, l1: BlockNumHash) -> bool {
    if l1.hash.is_zero() {
        return true;
    }
    match client.l1_block_hash_by_number(l1.number).await {
        Ok(hash) => hash == Some(l1.hash),
        Err(err) => {
            debug!(target: "engine", %err, ?l1, "Failed to look up an L1 block a verifier reported");
            false
        }
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
